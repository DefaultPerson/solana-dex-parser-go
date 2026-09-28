package adapter

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/goccy/go-json"
	"github.com/mr-tron/base58"
)

// This file holds the JSON decoding that lets SolanaTransaction accept, besides
// the standard RPC "json"/"jsonParsed" encodings, the form produced by Node.js
// tooling: public keys, signatures and instruction data as Buffer objects
// {"type":"Buffer","data":[...]} or plain byte arrays, account index lists as
// Buffers, and the slot as a string. Instructions are decoded with UseNumber so
// numeric fields of parsed instructions (e.g. system transfer lamports) keep
// their exact integer value.

// UnmarshalJSON accepts the slot as a JSON number or a decimal string.
func (t *SolanaTransaction) UnmarshalJSON(data []byte) error {
	type alias SolanaTransaction
	aux := struct {
		alias
		Slot json.RawMessage `json:"slot"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*t = SolanaTransaction(aux.alias)
	t.Slot = 0
	if raw := bytes.Trim(bytes.TrimSpace(aux.Slot), `"`); len(raw) > 0 && string(raw) != "null" {
		slot, err := strconv.ParseUint(string(raw), 10, 64)
		if err != nil {
			return fmt.Errorf("slot: %w", err)
		}
		t.Slot = slot
	}
	return nil
}

// UnmarshalJSON accepts signatures as base58 strings, Buffer objects or byte arrays.
func (d *TransactionData) UnmarshalJSON(data []byte) error {
	type alias TransactionData
	aux := struct {
		alias
		Signatures []json.RawMessage `json:"signatures"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*d = TransactionData(aux.alias)
	d.Signatures = nil
	if aux.Signatures != nil {
		d.Signatures = make([]string, len(aux.Signatures))
		for i, raw := range aux.Signatures {
			sig, err := keyFromJSON(raw)
			if err != nil {
				return fmt.Errorf("signatures[%d]: %w", i, err)
			}
			d.Signatures[i] = sig
		}
	}
	return nil
}

// UnmarshalJSON accepts static account keys as strings or Buffers and decodes
// instructions with exact numbers.
func (m *TransactionMessage) UnmarshalJSON(data []byte) error {
	type alias TransactionMessage
	aux := struct {
		alias
		StaticAccountKeys []json.RawMessage `json:"staticAccountKeys,omitempty"`
		Instructions      json.RawMessage   `json:"instructions,omitempty"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*m = TransactionMessage(aux.alias)
	m.StaticAccountKeys = nil
	if aux.StaticAccountKeys != nil {
		m.StaticAccountKeys = make([]string, len(aux.StaticAccountKeys))
		for i, raw := range aux.StaticAccountKeys {
			key, err := keyFromJSON(raw)
			if err != nil {
				return fmt.Errorf("staticAccountKeys[%d]: %w", i, err)
			}
			m.StaticAccountKeys[i] = key
		}
	}
	instructions, err := decodeInstructionList(aux.Instructions)
	if err != nil {
		return fmt.Errorf("instructions: %w", err)
	}
	m.Instructions = instructions
	return nil
}

// UnmarshalJSON decodes inner instructions with exact numbers.
func (s *InnerInstructionSet) UnmarshalJSON(data []byte) error {
	aux := struct {
		Index        int             `json:"index"`
		Instructions json.RawMessage `json:"instructions"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	instructions, err := decodeInstructionList(aux.Instructions)
	if err != nil {
		return fmt.Errorf("instructions: %w", err)
	}
	s.Index = aux.Index
	s.Instructions = instructions
	return nil
}

// UnmarshalJSON accepts data as a base58 string, Buffer or byte array and
// account index lists as arrays, Buffers or base58 strings.
func (c *CompiledInstruction) UnmarshalJSON(data []byte) error {
	aux := struct {
		ProgramIdIndex    int             `json:"programIdIndex"`
		Accounts          json.RawMessage `json:"accounts"`
		AccountKeyIndexes json.RawMessage `json:"accountKeyIndexes"`
		Data              json.RawMessage `json:"data"`
		StackHeight       int             `json:"stackHeight"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	accounts, err := indexListFromJSON(aux.Accounts)
	if err != nil {
		return fmt.Errorf("accounts: %w", err)
	}
	accountKeyIndexes, err := indexListFromJSON(aux.AccountKeyIndexes)
	if err != nil {
		return fmt.Errorf("accountKeyIndexes: %w", err)
	}
	*c = CompiledInstruction{
		ProgramIdIndex:    aux.ProgramIdIndex,
		Accounts:          accounts,
		AccountKeyIndexes: accountKeyIndexes,
		StackHeight:       aux.StackHeight,
	}
	var s string
	if err := json.Unmarshal(aux.Data, &s); err == nil {
		c.Data = s
		return nil
	}
	if b, ok, err := bytesFromJSON(aux.Data); err != nil {
		return fmt.Errorf("data: %w", err)
	} else if ok {
		c.DataBytes = b
	}
	return nil
}

// UnmarshalJSON accepts addresses as strings or Buffers.
func (l *LoadedAddresses) UnmarshalJSON(data []byte) error {
	aux := struct {
		Writable []json.RawMessage `json:"writable"`
		Readonly []json.RawMessage `json:"readonly"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	if l.Writable, err = keyListFromJSON(aux.Writable); err != nil {
		return fmt.Errorf("writable: %w", err)
	}
	if l.Readonly, err = keyListFromJSON(aux.Readonly); err != nil {
		return fmt.Errorf("readonly: %w", err)
	}
	return nil
}

// UnmarshalJSON accepts the table address as a string or Buffer and index
// lists as arrays or Buffers.
func (l *AddressTableLookup) UnmarshalJSON(data []byte) error {
	aux := struct {
		AccountKey      json.RawMessage `json:"accountKey"`
		WritableIndexes json.RawMessage `json:"writableIndexes"`
		ReadonlyIndexes json.RawMessage `json:"readonlyIndexes"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	key, err := keyFromJSON(aux.AccountKey)
	if err != nil {
		return fmt.Errorf("accountKey: %w", err)
	}
	writable, err := indexListFromJSON(aux.WritableIndexes)
	if err != nil {
		return fmt.Errorf("writableIndexes: %w", err)
	}
	readonly, err := indexListFromJSON(aux.ReadonlyIndexes)
	if err != nil {
		return fmt.Errorf("readonlyIndexes: %w", err)
	}
	*l = AddressTableLookup{AccountKey: key, WritableIndexes: writable, ReadonlyIndexes: readonly}
	return nil
}

func decodeInstructionList(raw json.RawMessage) ([]interface{}, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var list []interface{}
	if err := dec.Decode(&list); err != nil {
		return nil, err
	}
	return list, nil
}

func keyListFromJSON(raws []json.RawMessage) ([]string, error) {
	if raws == nil {
		return nil, nil
	}
	keys := make([]string, len(raws))
	for i, raw := range raws {
		key, err := keyFromJSON(raw)
		if err != nil {
			return nil, err
		}
		keys[i] = key
	}
	return keys, nil
}

// keyFromJSON decodes a public key or signature given as a base58 string, a
// Buffer object or a byte array. null decodes to "".
func keyFromJSON(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	if raw[0] == '"' {
		var s string
		err := json.Unmarshal(raw, &s)
		return s, err
	}
	b, ok, err := bytesFromJSON(raw)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("unsupported key encoding %.40s", raw)
	}
	return base58.Encode(b), nil
}

// bytesFromJSON decodes a Buffer object or a JSON byte array. ok is false for
// null, an empty object or other shapes.
func bytesFromJSON(raw json.RawMessage) ([]byte, bool, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, false, nil
	}
	switch raw[0] {
	case '[':
		var ints []int
		if err := json.Unmarshal(raw, &ints); err != nil {
			return nil, false, err
		}
		b, err := intsToBytes(ints)
		return b, err == nil, err
	case '{':
		var obj struct {
			Type string `json:"type"`
			Data []int  `json:"data"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, false, err
		}
		if obj.Data == nil {
			return nil, false, nil
		}
		b, err := intsToBytes(obj.Data)
		return b, err == nil, err
	}
	return nil, false, nil
}

// indexListFromJSON decodes an account index list given as an array of
// numbers, a Buffer object or a base58 string of index bytes.
func indexListFromJSON(raw json.RawMessage) ([]int, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	switch raw[0] {
	case '[':
		var ints []int
		err := json.Unmarshal(raw, &ints)
		return ints, err
	case '"':
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		b, err := base58.Decode(s)
		if err != nil {
			return nil, err
		}
		return bytesToInts(b), nil
	}
	b, ok, err := bytesFromJSON(raw)
	if err != nil || !ok {
		return nil, err
	}
	return bytesToInts(b), nil
}

func intsToBytes(ints []int) ([]byte, error) {
	b := make([]byte, len(ints))
	for i, v := range ints {
		if v < 0 || v > 255 {
			return nil, errors.New("byte value out of range")
		}
		b[i] = byte(v)
	}
	return b, nil
}

func bytesToInts(b []byte) []int {
	ints := make([]int, len(b))
	for i, v := range b {
		ints[i] = int(v)
	}
	return ints
}

// intFromValue converts a decoded JSON number (json.Number, float64) or a Go
// integer to int.
func intFromValue(v interface{}) (int, bool) {
	switch n := v.(type) {
	case json.Number:
		i, err := strconv.Atoi(string(n))
		return i, err == nil
	case float64:
		if n != math.Trunc(n) || n < float64(math.MinInt) || n >= float64(math.MaxInt) {
			return 0, false
		}
		return int(n), true
	case int:
		return n, true
	case int64:
		if n < math.MinInt || n > math.MaxInt {
			return 0, false
		}
		return int(n), true
	case int32:
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		if uint64(n) > math.MaxInt {
			return 0, false
		}
		return int(n), true
	case uint64:
		if n > math.MaxInt {
			return 0, false
		}
		return int(n), true
	}
	return 0, false
}

// bytesFromValue converts a decoded Buffer object, a JSON array of numbers or
// a Go byte slice to bytes.
func bytesFromValue(v interface{}) ([]byte, bool) {
	switch b := v.(type) {
	case []byte:
		return b, true
	case []interface{}:
		out := make([]byte, len(b))
		for i, e := range b {
			n, ok := intFromValue(e)
			if !ok || n < 0 || n > 255 {
				return nil, false
			}
			out[i] = byte(n)
		}
		return out, true
	case []int:
		out, err := intsToBytes(b)
		return out, err == nil
	case map[string]interface{}:
		if data, ok := b["data"]; ok {
			return bytesFromValue(data)
		}
	}
	return nil, false
}

// keyFromValue converts a decoded key (string, Buffer object, byte array) to base58.
func keyFromValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := bytesFromValue(v); ok && len(b) > 0 {
		return base58.Encode(b)
	}
	return ""
}
