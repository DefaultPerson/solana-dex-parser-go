package parsers

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// TradeParser interface for DEX trade parsers
type TradeParser interface {
	ProcessTrades() []types.TradeInfo
}

// LiquidityParser interface for liquidity pool parsers
type LiquidityParser interface {
	ProcessLiquidity() []types.PoolEvent
}

// EventParser interface for meme event parsers
type EventParser interface {
	ProcessEvents() []types.MemeEvent
}

// TransferParser interface for transfer parsers
type TransferParser interface {
	ProcessTransfers() []types.TransferData
}

// BaseParser provides common functionality for trade parsers
type BaseParser struct {
	Adapter                *adapter.TransactionAdapter
	DexInfo                types.DexInfo
	TransferActions        map[string][]types.TransferData
	ClassifiedInstructions []types.ClassifiedInstruction
	Utils                  *utils.TransactionUtils
}

// NewBaseParser creates a new BaseParser
func NewBaseParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *BaseParser {
	return &BaseParser{
		Adapter:                adapter,
		DexInfo:                dexInfo,
		TransferActions:        transferActions,
		ClassifiedInstructions: classifiedInstructions,
		Utils:                  utils.NewTransactionUtils(adapter),
	}
}

// GetTransfersForInstruction returns transfers for a specific instruction
func (bp *BaseParser) GetTransfersForInstruction(programId string, outerIndex int, innerIndex int, extraTypes []string) []types.TransferData {
	key := utils.FormatTransferKey(programId, outerIndex, innerIndex)

	transfers, ok := bp.TransferActions[key]
	if !ok {
		return nil
	}

	// Filter by allowed types
	allowedTypes := map[string]bool{
		"transfer":        true,
		"transferChecked": true,
	}
	for _, t := range extraTypes {
		allowedTypes[t] = true
	}

	var result []types.TransferData
	for _, t := range transfers {
		if allowedTypes[t.Type] {
			result = append(result, t)
		}
	}

	return result
}

// GetInstructionData returns instruction data
func (bp *BaseParser) GetInstructionData(instruction interface{}) []byte {
	return bp.Adapter.GetInstructionData(instruction)
}

// BaseLiquidityParser provides common functionality for liquidity parsers
type BaseLiquidityParser struct {
	Adapter                *adapter.TransactionAdapter
	TransferActions        map[string][]types.TransferData
	ClassifiedInstructions []types.ClassifiedInstruction
	Utils                  *utils.TransactionUtils
}

// NewBaseLiquidityParser creates a new BaseLiquidityParser
func NewBaseLiquidityParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *BaseLiquidityParser {
	return &BaseLiquidityParser{
		Adapter:                adapter,
		TransferActions:        transferActions,
		ClassifiedInstructions: classifiedInstructions,
		Utils:                  utils.NewTransactionUtils(adapter),
	}
}

// GetTransfersForInstruction returns transfers for a specific instruction
func (bp *BaseLiquidityParser) GetTransfersForInstruction(programId string, outerIndex int, innerIndex int, filterTypes []string) []types.TransferData {
	key := utils.FormatTransferKey(programId, outerIndex, innerIndex)

	transfers, ok := bp.TransferActions[key]
	if !ok {
		return nil
	}

	if len(filterTypes) == 0 {
		return transfers
	}

	// Filter by types
	allowedTypes := make(map[string]bool)
	for _, t := range filterTypes {
		allowedTypes[t] = true
	}

	var result []types.TransferData
	for _, t := range transfers {
		if allowedTypes[t.Type] {
			result = append(result, t)
		}
	}

	return result
}

// GetInstructionByDiscriminator finds an instruction by discriminator
func (bp *BaseLiquidityParser) GetInstructionByDiscriminator(discriminator []byte, slice int) *types.ClassifiedInstruction {
	for i := range bp.ClassifiedInstructions {
		inst := &bp.ClassifiedInstructions[i]
		data := bp.Adapter.GetInstructionData(inst.Instruction)
		if len(data) >= slice && bytes.Equal(data[:slice], discriminator) {
			return inst
		}
	}
	return nil
}

// GetInstructionData returns instruction data
func (bp *BaseLiquidityParser) GetInstructionData(instruction interface{}) []byte {
	return bp.Adapter.GetInstructionData(instruction)
}

// BaseEventParser provides common functionality for event parsers
type BaseEventParser struct {
	Adapter         *adapter.TransactionAdapter
	TransferActions map[string][]types.TransferData
	Utils           *utils.TransactionUtils
}

// NewBaseEventParser creates a new BaseEventParser
func NewBaseEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *BaseEventParser {
	return &BaseEventParser{
		Adapter:         adapter,
		TransferActions: transferActions,
		Utils:           utils.NewTransactionUtils(adapter),
	}
}

// GetTransfersForInstruction returns transfers for a specific instruction
func (bp *BaseEventParser) GetTransfersForInstruction(programId string, outerIndex int, innerIndex int) []types.TransferData {
	key := utils.FormatTransferKey(programId, outerIndex, innerIndex)

	transfers, ok := bp.TransferActions[key]
	if !ok {
		return nil
	}

	// Filter by transfer types
	var result []types.TransferData
	for _, t := range transfers {
		if t.Type == "transfer" || t.Type == "transferChecked" {
			result = append(result, t)
		}
	}

	return result
}

// GetInstructionData returns instruction data
func (bp *BaseEventParser) GetInstructionData(instruction interface{}) []byte {
	return bp.Adapter.GetInstructionData(instruction)
}

// Helper functions for parsers

// MatchDiscriminator checks if data starts with discriminator
func MatchDiscriminator(data []byte, discriminator []byte) bool {
	if len(data) < len(discriminator) {
		return false
	}
	return bytes.Equal(data[:len(discriminator)], discriminator)
}

// MatchAnyDiscriminator checks if data matches any discriminator in the map
func MatchAnyDiscriminator(data []byte, discriminators map[string][]byte) (string, bool) {
	for name, disc := range discriminators {
		if MatchDiscriminator(data, disc) {
			return name, true
		}
	}
	return "", false
}

// FilterByDiscriminators filters instructions by discriminator
func FilterByDiscriminators(instructions []types.ClassifiedInstruction, adapter *adapter.TransactionAdapter, excludeDiscriminators [][]byte) []types.ClassifiedInstruction {
	var result []types.ClassifiedInstruction
	for _, inst := range instructions {
		data := adapter.GetInstructionData(inst.Instruction)
		excluded := false
		for _, disc := range excludeDiscriminators {
			if MatchDiscriminator(data, disc) {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, inst)
		}
	}
	return result
}
