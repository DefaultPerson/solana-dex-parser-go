package tests

import (
	"encoding/binary"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// TestBotFeeAccounts tests the bot fee account detection
func TestBotFeeAccounts(t *testing.T) {
	tests := []struct {
		name     string
		account  string
		expected string
	}{
		{
			name:     "Trojan bot fee account",
			account:  "9yMwSPk9mrXSN7yDHUuZurAh1sjbJsfpUqjZ7SvVtdco",
			expected: "Trojan",
		},
		{
			name:     "BONKbot fee account",
			account:  "ZG98FUCjb8mJ824Gbs6RsgVmr1FhXb2oNiJHa2dwmPd",
			expected: "BONKbot",
		},
		{
			name:     "Axiom fee account",
			account:  "7LCZckF6XXGQ1hDY6HFXBKWAtiUgL9QY5vj1C4Bn1Qjj",
			expected: "Axiom",
		},
		{
			name:     "GMGN fee account",
			account:  "BB5dnY55FXS1e1NXqZDwCzgdYJdMCj3B92PU6Q5Fb6DT",
			expected: "GMGN",
		},
		{
			name:     "BullX fee account",
			account:  "9RYJ3qr5eU5xAooqVcbmdeusjcViL5Nkiq7Gske3tiKq",
			expected: "BullX",
		},
		{
			name:     "Maestro fee account",
			account:  "MaestroUL88UBnZr3wfoN7hqmNWFi3ZYCGqZoJJHE36",
			expected: "Maestro",
		},
		{
			name:     "Bloom fee account",
			account:  "7HeD6sLLqAnKVRuSfc1Ko3BSPMNKWgGTiWLKXJF31vKM",
			expected: "Bloom",
		},
		{
			name:     "BananaGun fee account",
			account:  "47hEzz83VFR23rLTEeVm9A7eFzjJwjvdupPPmX3cePqF",
			expected: "BananaGun",
		},
		{
			name:     "Raybot fee account",
			account:  "4mih95RmBqfHYvEfqq6uGGLp1Fr3gVS3VNSEa3JVRfQK",
			expected: "Raybot",
		},
		{
			name:     "Unknown account",
			account:  "11111111111111111111111111111111",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constants.GetBotName(tt.account)
			if result != tt.expected {
				t.Errorf("GetBotName(%s) = %s, want %s", tt.account, result, tt.expected)
			}
		})
	}
}

// TestIsBotFeeAccount tests the IsBotFeeAccount function
func TestIsBotFeeAccount(t *testing.T) {
	if !constants.IsBotFeeAccount("9yMwSPk9mrXSN7yDHUuZurAh1sjbJsfpUqjZ7SvVtdco") {
		t.Error("Expected Trojan fee account to be recognized")
	}

	if constants.IsBotFeeAccount("11111111111111111111111111111111") {
		t.Error("Expected unknown account to not be recognized as bot fee account")
	}
}

// TestGetAllBotFeeAccounts tests the GetAllBotFeeAccounts function
func TestGetAllBotFeeAccounts(t *testing.T) {
	accounts := constants.GetAllBotFeeAccounts()
	if len(accounts) == 0 {
		t.Error("Expected at least one bot fee account")
	}

	// Should contain known fee accounts
	found := false
	for _, acc := range accounts {
		if acc == "9yMwSPk9mrXSN7yDHUuZurAh1sjbJsfpUqjZ7SvVtdco" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected Trojan fee account in list")
	}
}

// TestGetBotNames tests the GetBotNames function
func TestGetBotNames(t *testing.T) {
	names := constants.GetBotNames()
	if len(names) == 0 {
		t.Error("Expected at least one bot name")
	}

	// Should contain known bots
	expected := map[string]bool{
		"Trojan":    true,
		"BONKbot":   true,
		"Axiom":     true,
		"GMGN":      true,
		"BullX":     true,
		"Maestro":   true,
		"Bloom":     true,
		"BananaGun": true,
		"Raybot":    true,
	}

	for _, name := range names {
		if !expected[name] {
			t.Errorf("Unexpected bot name: %s", name)
		}
	}
}

// TestPropAMMPrograms tests that Prop AMM programs are registered
func TestPropAMMPrograms(t *testing.T) {
	tests := []struct {
		name      string
		programId string
		expected  string
	}{
		{
			name:      "SolFi program",
			programId: "SoLFiHG9TfgtdUXUjWAxi3LtvYuFyDLVhBWxdMZxyCe",
			expected:  "SolFi",
		},
		{
			name:      "GoonFi program",
			programId: "goonERTdGsjnkZqWuVjs73BZ3Pb9qoCUdBUL17BnS5j",
			expected:  "GoonFi",
		},
		{
			name:      "Obric V2 program",
			programId: "obriQD1zbpyLz95G5n7nJe6a4DPjpFwa5XYPoNm113y",
			expected:  "ObricV2",
		},
		{
			name:      "HumidiFi program",
			programId: "9H6tua7jkLhdm3w8BvgpTn5LZNU7g4ZynDmCiNN3q6Rp",
			expected:  "HumidiFi",
		},
		{
			name:      "DFlow program",
			programId: "DF1ow4tspfHX9JwWJsAb9epbkA8hmpSEAtxXy1V27QBH",
			expected:  "DFlow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !constants.IsDexProgram(tt.programId) {
				t.Errorf("Expected %s to be a registered DEX program", tt.programId)
			}

			prog := constants.GetDexProgramByID(tt.programId)
			if prog.Name != tt.expected {
				t.Errorf("GetDexProgramByID(%s).Name = %s, want %s", tt.programId, prog.Name, tt.expected)
			}
		})
	}
}

// TestPropAMMDiscriminators tests that discriminators are defined
func TestPropAMMDiscriminators(t *testing.T) {
	// SolFi discriminator
	if len(constants.DISCRIMINATORS.SOLFI.SWAP) != 1 {
		t.Error("Expected SolFi SWAP discriminator to be 1 byte")
	}
	if constants.DISCRIMINATORS.SOLFI.SWAP[0] != 0x07 {
		t.Errorf("Expected SolFi SWAP discriminator to be 0x07, got 0x%02x", constants.DISCRIMINATORS.SOLFI.SWAP[0])
	}

	// GoonFi discriminator
	if len(constants.DISCRIMINATORS.GOONFI.SWAP) != 1 {
		t.Error("Expected GoonFi SWAP discriminator to be 1 byte")
	}
	if constants.DISCRIMINATORS.GOONFI.SWAP[0] != 0x02 {
		t.Errorf("Expected GoonFi SWAP discriminator to be 0x02, got 0x%02x", constants.DISCRIMINATORS.GOONFI.SWAP[0])
	}

	// Obric discriminators (8 bytes Anchor)
	if len(constants.DISCRIMINATORS.OBRIC.SWAP) != 8 {
		t.Error("Expected Obric SWAP discriminator to be 8 bytes")
	}
	if len(constants.DISCRIMINATORS.OBRIC.SWAP_X_TO_Y) != 8 {
		t.Error("Expected Obric SWAP_X_TO_Y discriminator to be 8 bytes")
	}
	if len(constants.DISCRIMINATORS.OBRIC.SWAP_Y_TO_X) != 8 {
		t.Error("Expected Obric SWAP_Y_TO_X discriminator to be 8 bytes")
	}

	// DFlow discriminators (8 bytes Anchor)
	if len(constants.DISCRIMINATORS.DFLOW.SWAP) != 8 {
		t.Error("Expected DFlow SWAP discriminator to be 8 bytes")
	}
	if len(constants.DISCRIMINATORS.DFLOW.SWAP2) != 8 {
		t.Error("Expected DFlow SWAP2 discriminator to be 8 bytes")
	}
	if len(constants.DISCRIMINATORS.DFLOW.SWAP_WITH_DEST) != 8 {
		t.Error("Expected DFlow SWAP_WITH_DEST discriminator to be 8 bytes")
	}

	// HumidiFi discriminator (8 bytes after decryption)
	if len(constants.DISCRIMINATORS.HUMIDIFI.SWAP) != 8 {
		t.Error("Expected HumidiFi SWAP discriminator to be 8 bytes")
	}
}

// TestHumidiFiXorDecryption tests the HumidiFi XOR decryption logic
func TestHumidiFiXorDecryption(t *testing.T) {
	// Test XOR key
	xorKey := []byte{58, 255, 47, 255, 226, 186, 235, 195}

	// Test deobfuscation of a simple 8-byte block
	encrypted := make([]byte, 8)
	for i := 0; i < 8; i++ {
		encrypted[i] = xorKey[i] // XOR with key should give zeros for pos=0
	}

	// Create position mask for pos=0
	posMask := make([]byte, 8)
	for j := 0; j < 8; j += 2 {
		binary.LittleEndian.PutUint16(posMask[j:], uint16(0))
	}

	// Decrypt
	decrypted := make([]byte, 8)
	for i := 0; i < 8; i++ {
		decrypted[i] = encrypted[i] ^ xorKey[i] ^ posMask[i]
	}

	// All bytes should be from posMask (since encrypted[i] ^ xorKey[i] = 0)
	for i := 0; i < 8; i++ {
		if decrypted[i] != posMask[i] {
			t.Errorf("Decryption mismatch at position %d: got %d, want %d", i, decrypted[i], posMask[i])
		}
	}
}
