package tests

import (
	"testing"

	"github.com/solana-dex-parser-go/utils"
)

func TestBinaryReaderU64(t *testing.T) {
	// Little-endian encoding of 12345678901234567890
	data := []byte{
		0xD2, 0x0A, 0x3F, 0xCE, 0x96, 0x5F, 0xAB, 0xAB, // u64
	}

	reader := utils.NewBinaryReader(data)
	val, err := reader.ReadU64()
	if err != nil {
		t.Fatalf("Failed to read u64: %v", err)
	}

	expected := uint64(0xABAB5F96CE3F0AD2)
	if val != expected {
		t.Errorf("Expected %d, got %d", expected, val)
	}
}

func TestBinaryReaderU32(t *testing.T) {
	data := []byte{0x78, 0x56, 0x34, 0x12}

	reader := utils.NewBinaryReader(data)
	val, err := reader.ReadU32()
	if err != nil {
		t.Fatalf("Failed to read u32: %v", err)
	}

	if val != 0x12345678 {
		t.Errorf("Expected 0x12345678, got 0x%X", val)
	}
}

func TestBinaryReaderU16(t *testing.T) {
	data := []byte{0x34, 0x12}

	reader := utils.NewBinaryReader(data)
	val, err := reader.ReadU16()
	if err != nil {
		t.Fatalf("Failed to read u16: %v", err)
	}

	if val != 0x1234 {
		t.Errorf("Expected 0x1234, got 0x%X", val)
	}
}

func TestBinaryReaderU8(t *testing.T) {
	data := []byte{0x42}

	reader := utils.NewBinaryReader(data)
	val, err := reader.ReadU8()
	if err != nil {
		t.Fatalf("Failed to read u8: %v", err)
	}

	if val != 0x42 {
		t.Errorf("Expected 0x42, got 0x%X", val)
	}
}

func TestBinaryReaderI64(t *testing.T) {
	// -1 in little-endian
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	reader := utils.NewBinaryReader(data)
	val, err := reader.ReadI64()
	if err != nil {
		t.Fatalf("Failed to read i64: %v", err)
	}

	if val != -1 {
		t.Errorf("Expected -1, got %d", val)
	}
}

func TestBinaryReaderString(t *testing.T) {
	// String with 4-byte length prefix + data "test"
	data := []byte{
		0x04, 0x00, 0x00, 0x00, // length = 4
		't', 'e', 's', 't', // "test"
	}

	reader := utils.NewBinaryReader(data)
	str, err := reader.ReadString()
	if err != nil {
		t.Fatalf("Failed to read string: %v", err)
	}

	if str != "test" {
		t.Errorf("Expected 'test', got '%s'", str)
	}
}

func TestBinaryReaderPubkey(t *testing.T) {
	// 32-byte pubkey (all zeros for simplicity)
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(i)
	}

	reader := utils.NewBinaryReader(data)
	pubkey, err := reader.ReadPubkey()
	if err != nil {
		t.Fatalf("Failed to read pubkey: %v", err)
	}

	// Should be base58 encoded
	if pubkey == "" {
		t.Error("Pubkey should not be empty")
	}
	t.Logf("Pubkey: %s", pubkey)
}

func TestBinaryReaderBool(t *testing.T) {
	data := []byte{0x01, 0x00}

	reader := utils.NewBinaryReader(data)

	val1, err := reader.ReadBool()
	if err != nil {
		t.Fatalf("Failed to read bool: %v", err)
	}
	if !val1 {
		t.Error("Expected true, got false")
	}

	val2, err := reader.ReadBool()
	if err != nil {
		t.Fatalf("Failed to read bool: %v", err)
	}
	if val2 {
		t.Error("Expected false, got true")
	}
}

func TestBinaryReaderSkip(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	reader := utils.NewBinaryReader(data)
	reader.Skip(3)

	val, err := reader.ReadU8()
	if err != nil {
		t.Fatalf("Failed to read u8: %v", err)
	}

	if val != 0x04 {
		t.Errorf("Expected 0x04, got 0x%X", val)
	}
}

func TestBinaryReaderRemaining(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	reader := utils.NewBinaryReader(data)

	if reader.Remaining() != 5 {
		t.Errorf("Expected 5 remaining, got %d", reader.Remaining())
	}

	reader.Skip(2)
	if reader.Remaining() != 3 {
		t.Errorf("Expected 3 remaining, got %d", reader.Remaining())
	}
}

func TestBinaryReaderBoundsCheck(t *testing.T) {
	data := []byte{0x01, 0x02}

	reader := utils.NewBinaryReader(data)

	// Read within bounds
	_, err1 := reader.ReadU8()
	if err1 != nil {
		t.Error("Should not have error after valid read 1")
	}

	_, err2 := reader.ReadU8()
	if err2 != nil {
		t.Error("Should not have error after valid read 2")
	}

	// Read past bounds - should return an error
	_, err3 := reader.ReadU8()
	if err3 == nil {
		t.Error("Should have error after reading past bounds")
	}
}

func TestBinaryReaderU64AsBigInt(t *testing.T) {
	// Max u64 value
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	reader := utils.NewBinaryReader(data)
	val := reader.ReadU64AsBigInt()

	expected := "18446744073709551615"
	if val.String() != expected {
		t.Errorf("Expected %s, got %s", expected, val.String())
	}
}

func TestBinaryReaderSequentialReads(t *testing.T) {
	data := []byte{
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // u64: 1
		0x02, 0x00,                                     // u16: 2
		0x03,                                           // u8: 3
		0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // i64: 4
	}

	reader := utils.NewBinaryReader(data)

	u64Val, _ := reader.ReadU64()
	if u64Val != 1 {
		t.Errorf("Expected u64=1, got %d", u64Val)
	}

	u16Val, _ := reader.ReadU16()
	if u16Val != 2 {
		t.Errorf("Expected u16=2, got %d", u16Val)
	}

	u8Val, _ := reader.ReadU8()
	if u8Val != 3 {
		t.Errorf("Expected u8=3, got %d", u8Val)
	}

	i64Val, _ := reader.ReadI64()
	if i64Val != 4 {
		t.Errorf("Expected i64=4, got %d", i64Val)
	}
}

func BenchmarkBinaryReaderU64(b *testing.B) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := utils.NewBinaryReader(data)
		reader.ReadU64()
	}
}
