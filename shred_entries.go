package dexparser

import (
	"errors"
	"fmt"

	"github.com/mr-tron/base58"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// DecodeShredEntries decodes the entries of a Jito ShredStream Entry message
// (shredstream.Entry.entries: a bincode Vec<solana_entry::Entry>, each entry
// {num_hashes u64, hash [32]u8, transactions Vec<VersionedTransaction>}) into
// pre-execution transactions for ShredParser. Legacy, v0 and v1 transactions
// are supported.
//
// The transactions have no meta: only outer instructions exist, v0 lookup
// table accounts are resolved from ParseConfig.AddressLookupTables /
// ALTsFetcher when parsing, and Slot is 0 (set it from Entry.slot). The index
// of a transaction in the result is its index within this message
// (entry-local), not its index in the slot. A structurally malformed
// transaction stops decoding: the transactions before it are returned with the
// error.
func DecodeShredEntries(data []byte) ([]*adapter.SolanaTransaction, error) {
	r := utils.NewBinaryReader(data)

	entryCount, err := r.ReadU64()
	if err != nil {
		return nil, fmt.Errorf("entries: %w", err)
	}
	const minEntrySize = 8 + 32 + 8 // num_hashes, hash, transaction count
	if entryCount > uint64(r.Remaining()/minEntrySize) {
		return nil, errors.New("entries: count exceeds the data")
	}

	var txs []*adapter.SolanaTransaction
	for i := uint64(0); i < entryCount; i++ {
		r.Skip(8 + 32) // num_hashes, hash
		txCount, err := r.ReadU64()
		if err != nil {
			return txs, fmt.Errorf("entry %d: %w", i, err)
		}
		if txCount > uint64(r.Remaining()) {
			return txs, fmt.Errorf("entry %d: transaction count exceeds the data", i)
		}
		for j := uint64(0); j < txCount; j++ {
			tx, err := decodeVersionedTransaction(r)
			if err != nil {
				return txs, fmt.Errorf("entry %d transaction %d: %w", i, j, err)
			}
			txs = append(txs, tx)
		}
	}
	return txs, nil
}

// wire format constants of versioned transactions
const (
	messageVersionPrefix = 0x80
	v1MessagePrefix      = messageVersionPrefix | 1
	signatureLength      = 64
	pubkeyLength         = 32
	// v1 TransactionConfigMask bits
	v1ConfigPriorityFee      = 0b11
	v1ConfigComputeUnitLimit = 0b100
	v1ConfigLoadedDataSize   = 0b1000
	v1ConfigHeapSize         = 0b10000
)

var errShredMalformed = errors.New("malformed transaction")

// decodeVersionedTransaction decodes one VersionedTransaction: legacy and v0
// transactions start with the short_vec signature count, v1 transactions
// with the 0x81 version byte and carry their signatures after the message
func decodeVersionedTransaction(r *utils.BinaryReader) (*adapter.SolanaTransaction, error) {
	first, err := r.ReadU8()
	if err != nil {
		return nil, err
	}

	if first&messageVersionPrefix == 0 {
		// Legacy or v0: first is the one-byte short_vec signature count
		signatures, err := readSignatures(r, int(first))
		if err != nil {
			return nil, err
		}
		prefix, err := r.ReadU8()
		if err != nil {
			return nil, err
		}
		tx := &adapter.SolanaTransaction{}
		tx.Transaction.Signatures = signatures
		var header [3]byte
		if prefix&messageVersionPrefix != 0 {
			if prefix != messageVersionPrefix {
				return nil, fmt.Errorf("unsupported message version %d in a legacy/v0 transaction", prefix&^messageVersionPrefix)
			}
			tx.Version = 0
			if _, err := readInto(r, header[:]); err != nil {
				return nil, err
			}
		} else {
			tx.Version = "legacy"
			header[0] = prefix
			if _, err := readInto(r, header[1:]); err != nil {
				return nil, err
			}
		}
		msg := &tx.Transaction.Message
		msg.Header = messageHeader(header)
		if msg.AccountKeys, err = readShortVecKeys(r); err != nil {
			return nil, err
		}
		if err := r.Skip(32); err != nil { // recent blockhash
			return nil, err
		}
		if msg.Instructions, err = readCompiledInstructions(r); err != nil {
			return nil, err
		}
		if tx.Version == 0 {
			if msg.AddressTableLookups, err = readLookups(r); err != nil {
				return nil, err
			}
		}
		return tx, nil
	}

	if first != v1MessagePrefix {
		return nil, fmt.Errorf("unsupported transaction version byte %#x", first)
	}
	return decodeV1Transaction(r)
}

// decodeV1Transaction decodes a v1 (SIMD-0385) transaction after its version
// byte: legacy header, u32 config mask, blockhash, u8 instruction count, u8
// address count, addresses, config values, 4-byte instruction headers
// {program_id_index, num_accounts, data_len u16}, instruction payloads, then
// num_required_signatures signatures
func decodeV1Transaction(r *utils.BinaryReader) (*adapter.SolanaTransaction, error) {
	var header [3]byte
	if _, err := readInto(r, header[:]); err != nil {
		return nil, err
	}
	mask, err := r.ReadU32()
	if err != nil {
		return nil, err
	}
	if mask&^uint32(0b11111) != 0 || (mask&v1ConfigPriorityFee != 0 && mask&v1ConfigPriorityFee != v1ConfigPriorityFee) {
		return nil, fmt.Errorf("invalid v1 config mask %#x", mask)
	}
	if err := r.Skip(32); err != nil { // lifetime specifier (blockhash)
		return nil, err
	}
	numInstructions, _ := r.ReadU8()
	numAddresses, err := r.ReadU8()
	if err != nil {
		return nil, err
	}

	tx := &adapter.SolanaTransaction{Version: 1}
	msg := &tx.Transaction.Message
	msg.Header = messageHeader(header)
	if msg.AccountKeys, err = readKeys(r, int(numAddresses)); err != nil {
		return nil, err
	}

	config := &adapter.TransactionConfig{}
	if mask&v1ConfigPriorityFee != 0 {
		v, err := r.ReadU64()
		if err != nil {
			return nil, err
		}
		config.PriorityFee = &v
	}
	for _, field := range []struct {
		bit uint32
		dst **uint64
	}{
		{v1ConfigComputeUnitLimit, &config.ComputeUnitLimit},
		{v1ConfigLoadedDataSize, &config.LoadedAccountsDataSizeLimit},
		{v1ConfigHeapSize, &config.HeapSize},
	} {
		if mask&field.bit != 0 {
			v, err := r.ReadU32()
			if err != nil {
				return nil, err
			}
			w := uint64(v)
			*field.dst = &w
		}
	}
	msg.TransactionConfig = config

	type ixHeader struct {
		programIdIndex, numAccounts int
		dataLen                     int
	}
	headers := make([]ixHeader, int(numInstructions))
	if r.Remaining() < 4*len(headers) {
		return nil, errShredMalformed
	}
	for i := range headers {
		programIdIndex, _ := r.ReadU8()
		numAccounts, _ := r.ReadU8()
		dataLen, err := r.ReadU16()
		if err != nil {
			return nil, err
		}
		headers[i] = ixHeader{int(programIdIndex), int(numAccounts), int(dataLen)}
	}
	msg.Instructions = make([]interface{}, len(headers))
	for i, h := range headers {
		accounts, err := r.ReadFixedArray(h.numAccounts)
		if err != nil {
			return nil, err
		}
		data, err := r.ReadFixedArray(h.dataLen)
		if err != nil {
			return nil, err
		}
		msg.Instructions[i] = compiledInstructionMap(h.programIdIndex, accounts, data)
	}

	if tx.Transaction.Signatures, err = readSignatures(r, int(header[0])); err != nil {
		return nil, err
	}
	return tx, nil
}

func messageHeader(h [3]byte) *adapter.MessageHeader {
	return &adapter.MessageHeader{
		NumRequiredSignatures:       int(h[0]),
		NumReadonlySignedAccounts:   int(h[1]),
		NumReadonlyUnsignedAccounts: int(h[2]),
	}
}

// compiledInstructionMap builds an instruction in the form of the "json"
// encoding, with index and data bytes the adapter decodes directly
func compiledInstructionMap(programIdIndex int, accounts, data []byte) map[string]interface{} {
	return map[string]interface{}{
		"programIdIndex": programIdIndex,
		"accounts":       byteIndexes(accounts),
		"data":           data,
	}
}

func readInto(r *utils.BinaryReader, dst []byte) (int, error) {
	b, err := r.ReadFixedArray(len(dst))
	if err != nil {
		return 0, err
	}
	return copy(dst, b), nil
}

// readShortVecLength reads a compact-u16 length and checks that that many
// elements of elemSize bytes fit in the remaining data
func readShortVecLength(r *utils.BinaryReader, elemSize int) (int, error) {
	n := 0
	for shift := 0; ; shift += 7 {
		b, err := r.ReadU8()
		if err != nil {
			return 0, err
		}
		if shift == 14 && b > 3 {
			return 0, errShredMalformed
		}
		n |= int(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		if shift == 14 {
			return 0, errShredMalformed
		}
	}
	if n*elemSize > r.Remaining() {
		return 0, errShredMalformed
	}
	return n, nil
}

func readSignatures(r *utils.BinaryReader, count int) ([]string, error) {
	if count*signatureLength > r.Remaining() {
		return nil, errShredMalformed
	}
	signatures := make([]string, count)
	for i := range signatures {
		sig, err := r.ReadFixedArray(signatureLength)
		if err != nil {
			return nil, err
		}
		signatures[i] = base58.Encode(sig)
	}
	return signatures, nil
}

func readShortVecKeys(r *utils.BinaryReader) ([]adapter.AccountKey, error) {
	n, err := readShortVecLength(r, pubkeyLength)
	if err != nil {
		return nil, err
	}
	return readKeys(r, n)
}

func readKeys(r *utils.BinaryReader, n int) ([]adapter.AccountKey, error) {
	if n*pubkeyLength > r.Remaining() {
		return nil, errShredMalformed
	}
	keys := make([]adapter.AccountKey, n)
	for i := range keys {
		key, err := r.ReadPubkey()
		if err != nil {
			return nil, err
		}
		keys[i] = adapter.AccountKey{Pubkey: key}
	}
	return keys, nil
}

func readShortVecBytes(r *utils.BinaryReader) ([]byte, error) {
	n, err := readShortVecLength(r, 1)
	if err != nil {
		return nil, err
	}
	return r.ReadFixedArray(n)
}

// readCompiledInstructions reads short_vec<CompiledInstruction{program_id_index
// u8, accounts short_vec<u8>, data short_vec<u8>}>
func readCompiledInstructions(r *utils.BinaryReader) ([]interface{}, error) {
	n, err := readShortVecLength(r, 3)
	if err != nil {
		return nil, err
	}
	instructions := make([]interface{}, n)
	for i := range instructions {
		programIdIndex, err := r.ReadU8()
		if err != nil {
			return nil, err
		}
		accounts, err := readShortVecBytes(r)
		if err != nil {
			return nil, err
		}
		data, err := readShortVecBytes(r)
		if err != nil {
			return nil, err
		}
		instructions[i] = compiledInstructionMap(int(programIdIndex), accounts, data)
	}
	return instructions, nil
}

// readLookups reads short_vec<MessageAddressTableLookup{account_key,
// writable_indexes short_vec<u8>, readonly_indexes short_vec<u8>}>
func readLookups(r *utils.BinaryReader) ([]adapter.AddressTableLookup, error) {
	n, err := readShortVecLength(r, pubkeyLength+2)
	if err != nil {
		return nil, err
	}
	lookups := make([]adapter.AddressTableLookup, n)
	for i := range lookups {
		key, err := r.ReadPubkey()
		if err != nil {
			return nil, err
		}
		writable, err := readShortVecBytes(r)
		if err != nil {
			return nil, err
		}
		readonly, err := readShortVecBytes(r)
		if err != nil {
			return nil, err
		}
		lookups[i] = adapter.AddressTableLookup{
			AccountKey:      key,
			WritableIndexes: byteIndexes(writable),
			ReadonlyIndexes: byteIndexes(readonly),
		}
	}
	return lookups, nil
}
