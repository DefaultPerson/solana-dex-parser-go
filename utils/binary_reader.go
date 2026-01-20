package utils

import (
	"encoding/binary"
	"errors"
	"math/big"
	"sync"

	"github.com/mr-tron/base58"
)

// binaryReaderPool is a sync.Pool for BinaryReader instances
var binaryReaderPool = sync.Pool{
	New: func() interface{} {
		return &BinaryReader{}
	},
}

// BinaryReader provides methods for reading binary data with offset tracking
type BinaryReader struct {
	buffer []byte
	offset int
	err    error
}

// NewBinaryReader creates a new BinaryReader from a byte slice
func NewBinaryReader(buffer []byte) *BinaryReader {
	return &BinaryReader{
		buffer: buffer,
		offset: 0,
	}
}

// GetBinaryReader gets a BinaryReader from the pool and initializes it
func GetBinaryReader(buffer []byte) *BinaryReader {
	r := binaryReaderPool.Get().(*BinaryReader)
	r.buffer = buffer
	r.offset = 0
	r.err = nil
	return r
}

// Release returns the BinaryReader to the pool
func (r *BinaryReader) Release() {
	r.buffer = nil // Release buffer reference to allow GC
	binaryReaderPool.Put(r)
}

// ReadFixedArray reads a fixed-length byte array
func (r *BinaryReader) ReadFixedArray(length int) ([]byte, error) {
	if err := r.checkBounds(length); err != nil {
		return nil, err
	}
	arr := make([]byte, length)
	copy(arr, r.buffer[r.offset:r.offset+length])
	r.offset += length
	return arr, nil
}

// ReadU8 reads an unsigned 8-bit integer
func (r *BinaryReader) ReadU8() (uint8, error) {
	if err := r.checkBounds(1); err != nil {
		return 0, err
	}
	value := r.buffer[r.offset]
	r.offset++
	return value, nil
}

// ReadU16 reads an unsigned 16-bit little-endian integer
func (r *BinaryReader) ReadU16() (uint16, error) {
	if err := r.checkBounds(2); err != nil {
		return 0, err
	}
	value := binary.LittleEndian.Uint16(r.buffer[r.offset:])
	r.offset += 2
	return value, nil
}

// ReadU32 reads an unsigned 32-bit little-endian integer
func (r *BinaryReader) ReadU32() (uint32, error) {
	if err := r.checkBounds(4); err != nil {
		return 0, err
	}
	value := binary.LittleEndian.Uint32(r.buffer[r.offset:])
	r.offset += 4
	return value, nil
}

// ReadU64 reads an unsigned 64-bit little-endian integer
func (r *BinaryReader) ReadU64() (uint64, error) {
	if err := r.checkBounds(8); err != nil {
		return 0, err
	}
	value := binary.LittleEndian.Uint64(r.buffer[r.offset:])
	r.offset += 8
	return value, nil
}

// ReadI64 reads a signed 64-bit little-endian integer
func (r *BinaryReader) ReadI64() (int64, error) {
	if err := r.checkBounds(8); err != nil {
		return 0, err
	}
	value := int64(binary.LittleEndian.Uint64(r.buffer[r.offset:]))
	r.offset += 8
	return value, nil
}

// ReadI128 reads a signed 128-bit little-endian integer as two int64
func (r *BinaryReader) ReadI128() (lo uint64, hi int64, err error) {
	if err = r.checkBounds(16); err != nil {
		return 0, 0, err
	}
	lo = binary.LittleEndian.Uint64(r.buffer[r.offset:])
	hi = int64(binary.LittleEndian.Uint64(r.buffer[r.offset+8:]))
	r.offset += 16
	return lo, hi, nil
}

// ReadU128 reads an unsigned 128-bit little-endian integer as two uint64
func (r *BinaryReader) ReadU128() (lo, hi uint64, err error) {
	if err = r.checkBounds(16); err != nil {
		return 0, 0, err
	}
	lo = binary.LittleEndian.Uint64(r.buffer[r.offset:])
	hi = binary.LittleEndian.Uint64(r.buffer[r.offset+8:])
	r.offset += 16
	return lo, hi, nil
}

// ReadString reads a Borsh-encoded string (4-byte length prefix + UTF-8 data)
func (r *BinaryReader) ReadString() (string, error) {
	length, err := r.ReadU32()
	if err != nil {
		return "", err
	}

	if err := r.checkBounds(int(length)); err != nil {
		return "", err
	}

	str := string(r.buffer[r.offset : r.offset+int(length)])
	r.offset += int(length)
	return str, nil
}

// ReadPubkey reads a 32-byte public key and returns it as base58 string
func (r *BinaryReader) ReadPubkey() (string, error) {
	bytes, err := r.ReadFixedArray(32)
	if err != nil {
		return "", err
	}
	return base58.Encode(bytes), nil
}

// ReadBool reads a boolean value (1 byte)
func (r *BinaryReader) ReadBool() (bool, error) {
	b, err := r.ReadU8()
	if err != nil {
		return false, err
	}
	return b != 0, nil
}

// Skip advances the offset by n bytes
func (r *BinaryReader) Skip(n int) error {
	if err := r.checkBounds(n); err != nil {
		return err
	}
	r.offset += n
	return nil
}

// Remaining returns the number of unread bytes
func (r *BinaryReader) Remaining() int {
	return len(r.buffer) - r.offset
}

// GetOffset returns the current read position
func (r *BinaryReader) GetOffset() int {
	return r.offset
}

// SetOffset sets the read position
func (r *BinaryReader) SetOffset(offset int) error {
	if offset < 0 || offset > len(r.buffer) {
		return errors.New("offset out of bounds")
	}
	r.offset = offset
	return nil
}

// GetBuffer returns the underlying buffer
func (r *BinaryReader) GetBuffer() []byte {
	return r.buffer
}

// Slice returns a slice of the buffer from current offset
func (r *BinaryReader) Slice(length int) ([]byte, error) {
	if err := r.checkBounds(length); err != nil {
		return nil, err
	}
	return r.buffer[r.offset : r.offset+length], nil
}

// checkBounds verifies that length bytes can be read from current offset
func (r *BinaryReader) checkBounds(length int) error {
	if r.offset+length > len(r.buffer) {
		return errors.New("buffer overflow: trying to read beyond buffer length")
	}
	return nil
}

// Error returns any accumulated error
func (r *BinaryReader) Error() error {
	return r.err
}

// HasError returns true if an error has occurred
func (r *BinaryReader) HasError() bool {
	return r.err != nil
}

// ReadU64AsBigInt reads a u64 and returns it as *big.Int (convenience method)
func (r *BinaryReader) ReadU64AsBigInt() *big.Int {
	if r.err != nil {
		return big.NewInt(0)
	}
	val, err := r.ReadU64()
	if err != nil {
		r.err = err
		return big.NewInt(0)
	}
	return new(big.Int).SetUint64(val)
}

// ReadU128AsBigInt reads a u128 and returns it as *big.Int (convenience method)
func (r *BinaryReader) ReadU128AsBigInt() *big.Int {
	if r.err != nil {
		return big.NewInt(0)
	}
	lo, hi, err := r.ReadU128()
	if err != nil {
		r.err = err
		return big.NewInt(0)
	}
	result := new(big.Int).SetUint64(hi)
	result.Lsh(result, 64)
	result.Or(result, new(big.Int).SetUint64(lo))
	return result
}
