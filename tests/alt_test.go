package tests

import (
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/alt"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

func TestAltEventParser(t *testing.T) {
	// Test with empty instructions
	txAdapter := &adapter.TransactionAdapter{}
	classifiedInstructions := []types.ClassifiedInstruction{}

	parser := alt.NewAltEventParser(txAdapter, classifiedInstructions)
	events := parser.ProcessEvents()

	if len(events) != 0 {
		t.Errorf("Expected 0 events for empty instructions, got %d", len(events))
	}
}

func TestAltInstructionTypes(t *testing.T) {
	// Verify ALT instruction type constants
	tests := []struct {
		name     string
		expected uint32
	}{
		{"CreateLookupTable", 0},
		{"FreezeLookupTable", 1},
		{"ExtendLookupTable", 2},
		{"DeactivateLookupTable", 3},
		{"CloseLookupTable", 4},
	}

	for i, tt := range tests {
		if uint32(i) != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.name, tt.expected, i)
		}
	}
}

func TestAltEventStruct(t *testing.T) {
	event := types.AltEvent{
		Type:         "CreateLookupTable",
		AltAccount:   "ALTAccount123",
		AltAuthority: "Authority456",
		PayerAccount: "Payer789",
		RecentSlot:   12345678,
		Idx:          "0",
	}

	if event.Type != "CreateLookupTable" {
		t.Errorf("Expected Type 'CreateLookupTable', got '%s'", event.Type)
	}
	if event.AltAccount != "ALTAccount123" {
		t.Errorf("Expected AltAccount 'ALTAccount123', got '%s'", event.AltAccount)
	}
	if event.RecentSlot != 12345678 {
		t.Errorf("Expected RecentSlot 12345678, got %d", event.RecentSlot)
	}
}

func TestAltEventExtend(t *testing.T) {
	event := types.AltEvent{
		Type:         "ExtendLookupTable",
		AltAccount:   "ALTAccount123",
		AltAuthority: "Authority456",
		NewAddresses: []string{"addr1", "addr2", "addr3"},
	}

	if len(event.NewAddresses) != 3 {
		t.Errorf("Expected 3 NewAddresses, got %d", len(event.NewAddresses))
	}
	if event.NewAddresses[0] != "addr1" {
		t.Errorf("Expected first address 'addr1', got '%s'", event.NewAddresses[0])
	}
}

func TestAltEventClose(t *testing.T) {
	event := types.AltEvent{
		Type:         "CloseLookupTable",
		AltAccount:   "ALTAccount123",
		AltAuthority: "Authority456",
		Recipient:    "RecipientWallet",
	}

	if event.Recipient != "RecipientWallet" {
		t.Errorf("Expected Recipient 'RecipientWallet', got '%s'", event.Recipient)
	}
}
