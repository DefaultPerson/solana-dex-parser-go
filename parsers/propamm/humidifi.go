package propamm

import (
	"encoding/binary"
	"fmt"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// HumidiFi XOR encryption key
var humidiFiXorKey = []byte{58, 255, 47, 255, 226, 186, 235, 195}

// HumidiFiParser parses HumidiFi DEX transactions
type HumidiFiParser struct {
	adapter                *adapter.TransactionAdapter
	dexInfo                types.DexInfo
	transferActions        map[string][]types.TransferData
	classifiedInstructions []types.ClassifiedInstruction
	txUtils                *utils.TransactionUtils
}

// NewHumidiFiParser creates a new HumidiFi parser
func NewHumidiFiParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *HumidiFiParser {
	return &HumidiFiParser{
		adapter:                adapter,
		dexInfo:                dexInfo,
		transferActions:        transferActions,
		classifiedInstructions: classifiedInstructions,
		txUtils:                utils.NewTransactionUtils(adapter),
	}
}

// deobfuscateHumidiFi decrypts XOR-obfuscated HumidiFi instruction data
func deobfuscateHumidiFi(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	result := make([]byte, len(data))
	copy(result, data)

	pos := 0
	for i := 0; i < len(result); i += 8 {
		chunkSize := 8
		if len(result)-i < 8 {
			chunkSize = len(result) - i
		}

		// Create position mask
		posMask := make([]byte, 8)
		for j := 0; j < 8; j += 2 {
			binary.LittleEndian.PutUint16(posMask[j:], uint16(pos))
		}

		// XOR each byte
		for j := 0; j < chunkSize; j++ {
			result[i+j] ^= humidiFiXorKey[j] ^ posMask[j]
		}
		pos++
	}

	return result
}

// ProcessTrades processes HumidiFi trades
func (p *HumidiFiParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.classifiedInstructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 17 { // Minimum: 8 header + 8 amountIn + 1 direction
			continue
		}

		// Deobfuscate the instruction data
		decrypted := deobfuscateHumidiFi(data)

		trade := p.parseSwap(ci, decrypted)
		if trade != nil {
			trades = append(trades, *trade)
		}
	}

	return trades
}

// parseSwap parses a HumidiFi swap instruction
func (p *HumidiFiParser) parseSwap(ci types.ClassifiedInstruction, decryptedData []byte) *types.TradeInfo {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	// HumidiFi swap account layout:
	// 0: Signer
	// 1: Pool
	// 2: Pool Base Token Account
	// 3: Pool Quote Token Account
	// 4: User Base Token Account
	// 5: User Quote Token Account
	if len(accounts) < 6 {
		return nil
	}

	// Parse instruction data
	// 0-7:   Header
	// 8-15:  amountIn (u64 LE)
	// 16:    isBaseToQuote (0=quote-to-base, 1=base-to-quote)
	if len(decryptedData) < 17 {
		return nil
	}

	amountIn := binary.LittleEndian.Uint64(decryptedData[8:16])
	isBaseToQuote := decryptedData[16] == 1

	innerIdx := ci.InnerIndex
	if innerIdx < 0 {
		innerIdx = 0
	}

	// Get transfers for this instruction
	transfers := p.txUtils.GetTransfersForInstruction(
		p.transferActions,
		ci.ProgramId,
		ci.OuterIndex,
		ci.InnerIndex,
		nil,
	)

	if len(transfers) < 2 {
		// Fallback: try to create trade from account data
		return p.createTradeFromAccounts(ci, accounts, amountIn, isBaseToQuote, innerIdx)
	}

	dexInfo := types.DexInfo{
		ProgramId: constants.DEX_PROGRAMS.HUMIDIFI.ID,
		AMM:       constants.DEX_PROGRAMS.HUMIDIFI.Name,
		Route:     p.dexInfo.Route,
	}

	trade := p.txUtils.ProcessSwapData(transfers, dexInfo, false)
	if trade != nil {
		trade.Pool = []string{accounts[1]} // Pool account
		trade.Idx = utils.FormatIdx(ci.OuterIndex, innerIdx)
		trade = p.txUtils.AttachTokenTransferInfo(trade, p.transferActions)
	}

	return trade
}

// createTradeFromAccounts creates a trade from account data when no transfers are found
func (p *HumidiFiParser) createTradeFromAccounts(
	ci types.ClassifiedInstruction,
	accounts []string,
	amountIn uint64,
	isBaseToQuote bool,
	innerIdx int,
) *types.TradeInfo {
	if len(accounts) < 6 {
		return nil
	}

	// Default decimals (will be corrected by transfer info if available)
	decimals := uint8(9)

	var inputToken, outputToken types.TokenInfo

	if isBaseToQuote {
		// Selling base token for quote token
		inputToken = types.TokenInfo{
			Amount:    types.ConvertToUIAmountUint64(amountIn, decimals),
			AmountRaw: fmt.Sprintf("%d", amountIn),
			Decimals:  decimals,
		}
		outputToken = types.TokenInfo{
			Decimals: decimals,
		}
	} else {
		// Selling quote token for base token
		inputToken = types.TokenInfo{
			Amount:    types.ConvertToUIAmountUint64(amountIn, decimals),
			AmountRaw: fmt.Sprintf("%d", amountIn),
			Decimals:  decimals,
		}
		outputToken = types.TokenInfo{
			Decimals: decimals,
		}
	}

	return &types.TradeInfo{
		Type:        types.TradeTypeSwap,
		Pool:        []string{accounts[1]},
		User:        accounts[0],
		InputToken:  inputToken,
		OutputToken: outputToken,
		ProgramId:   constants.DEX_PROGRAMS.HUMIDIFI.ID,
		AMM:         constants.DEX_PROGRAMS.HUMIDIFI.Name,
		Route:       p.dexInfo.Route,
		Slot:        p.adapter.Slot(),
		Timestamp:   p.adapter.BlockTime(),
		Signature:   p.adapter.Signature(),
		Idx:         utils.FormatIdx(ci.OuterIndex, innerIdx),
	}
}
