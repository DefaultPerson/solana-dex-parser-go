package tests

import (
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// TestCoreClassifiedInstructionGetIdx: indices >= 10 used to become garbage
// runes ("<", "2-?", "{-1").
func TestCoreClassifiedInstructionGetIdx(t *testing.T) {
	cases := []struct {
		outer, inner int
		want         string
	}{
		{0, -1, "0"},
		{12, -1, "12"},
		{2, 15, "2-15"},
		{75, 1, "75-1"},
		{3, 0, "3-0"},
	}
	for _, c := range cases {
		ci := types.ClassifiedInstruction{OuterIndex: c.outer, InnerIndex: c.inner}
		if got := ci.GetIdx(); got != c.want {
			t.Errorf("GetIdx(%d,%d) = %q, want %q", c.outer, c.inner, got, c.want)
		}
		if got := utils.FormatIdx(c.outer, c.inner); got != c.want {
			t.Errorf("FormatIdx(%d,%d) = %q, want %q", c.outer, c.inner, got, c.want)
		}
	}
}

func TestCoreConvertToUIAmountNil(t *testing.T) {
	if got := types.ConvertToUIAmount(nil, 9); got != 0 {
		t.Errorf("ConvertToUIAmount(nil) = %v, want 0", got)
	}
}

// TestCoreBinaryReaderStickyErrors: a failed read must set the error that
// HasError reports, and later reads must not resume at a misaligned offset.
func TestCoreBinaryReaderStickyErrors(t *testing.T) {
	r := utils.NewBinaryReader([]byte{1, 2, 3, 4})
	if _, err := r.ReadU64(); err == nil {
		t.Fatal("ReadU64 on 4 bytes: expected error")
	}
	if !r.HasError() {
		t.Error("HasError() = false after a failed ReadU64")
	}
	if v, err := r.ReadU32(); err == nil || v != 0 {
		t.Errorf("ReadU32 after failure = %d, %v; want 0 and an error", v, err)
	}

	for name, read := range map[string]func(r *utils.BinaryReader) error{
		"ReadU8":   func(r *utils.BinaryReader) error { _, err := r.ReadU8(); return err },
		"ReadU16":  func(r *utils.BinaryReader) error { _, err := r.ReadU16(); return err },
		"ReadU32":  func(r *utils.BinaryReader) error { _, err := r.ReadU32(); return err },
		"ReadI64":  func(r *utils.BinaryReader) error { _, err := r.ReadI64(); return err },
		"ReadU128": func(r *utils.BinaryReader) error { _, _, err := r.ReadU128(); return err },
		"ReadPub":  func(r *utils.BinaryReader) error { _, err := r.ReadPubkey(); return err },
		"ReadStr":  func(r *utils.BinaryReader) error { _, err := r.ReadString(); return err },
	} {
		r := utils.NewBinaryReader(nil)
		if err := read(r); err == nil || !r.HasError() {
			t.Errorf("%s on empty buffer: err=%v HasError=%v, want error recorded", name, err, r.HasError())
		}
	}
}

func TestCoreBinaryReaderNegativeLength(t *testing.T) {
	r := utils.NewBinaryReader([]byte{1, 2, 3})
	if err := r.Skip(-1); err == nil {
		t.Error("Skip(-1) accepted")
	}
	r = utils.NewBinaryReader([]byte{1, 2, 3})
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Errorf("Slice(-2) panicked: %v", p)
			}
		}()
		if _, err := r.Slice(-2); err == nil {
			t.Error("Slice(-2) accepted")
		}
	}()
	r = utils.NewBinaryReader([]byte{1, 2, 3})
	if _, err := r.ReadFixedArray(-1); err == nil {
		t.Error("ReadFixedArray(-1) accepted")
	}
}

// TestCoreBinaryReaderVecLength rejects vector counts that cannot fit in the
// remaining bytes (a huge count must not drive allocations or loops).
func TestCoreBinaryReaderVecLength(t *testing.T) {
	r := utils.NewBinaryReader([]byte{2, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8})
	if n, err := r.ReadVecLength(4); err != nil || n != 2 {
		t.Errorf("ReadVecLength(4) = %d, %v; want 2", n, err)
	}
	r = utils.NewBinaryReader([]byte{0xff, 0xff, 0xff, 0xff, 1, 2})
	if _, err := r.ReadVecLength(32); err == nil || !r.HasError() {
		t.Error("ReadVecLength accepted a count larger than the buffer")
	}
}
