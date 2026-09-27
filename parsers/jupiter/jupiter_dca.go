package jupiter

import (
	"bytes"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// JupiterDCAParser parses Jupiter DCA transactions
type JupiterDCAParser struct {
	*parsers.BaseParser
}

// NewJupiterDCAParser creates a new Jupiter DCA parser
func NewJupiterDCAParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *JupiterDCAParser {
	return &JupiterDCAParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// ProcessTrades parses Jupiter DCA trades
func (p *JupiterDCAParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.JUPITER_DCA.ID {
			data := p.Adapter.GetInstructionData(ci.Instruction)
			if len(data) >= 16 && bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER_DCA.FILLED) {
				trade := p.parseFullFilled(ci.Instruction, utils.FormatIdx(ci.OuterIndex, ci.InnerIndex))
				if trade != nil {
					trades = append(trades, *trade)
				}
			}
		}
	}

	return trades
}

// parseFullFilled parses DCA filled event
func (p *JupiterDCAParser) parseFullFilled(instruction interface{}, idx string) *types.TradeInfo {
	data := p.Adapter.GetInstructionData(instruction)
	if len(data) < 16 {
		return nil
	}

	eventData := data[16:]
	layout, err := ParseJupiterDCAFilledLayout(eventData)
	if err != nil {
		return nil
	}

	event := layout.ToObject()

	inputDecimal := p.Adapter.GetTokenDecimals(event.InputMint)
	outputDecimal := p.Adapter.GetTokenDecimals(event.OutputMint)
	feeDecimal := p.Adapter.GetTokenDecimals(event.FeeMint)

	inUIAmount := types.ConvertToUIAmount(event.InAmount, inputDecimal)
	outUIAmount := types.ConvertToUIAmount(event.OutAmount, outputDecimal)
	feeUIAmount := types.ConvertToUIAmount(event.Fee, feeDecimal)

	trade := &types.TradeInfo{
		Type: utils.GetTradeType(event.InputMint, event.OutputMint),
		InputToken: types.TokenInfo{
			Mint:      event.InputMint,
			Amount:    inUIAmount,
			AmountRaw: event.InAmount.String(),
			Decimals:  inputDecimal,
		},
		OutputToken: types.TokenInfo{
			Mint:      event.OutputMint,
			Amount:    outUIAmount,
			AmountRaw: event.OutAmount.String(),
			Decimals:  outputDecimal,
		},
		// The DCA program's fee, taken from the output (Filled.fee)
		Fee: &types.FeeInfo{
			Mint:      event.FeeMint,
			Amount:    feeUIAmount,
			AmountRaw: event.Fee.String(),
			Decimals:  feeDecimal,
			Dex:       constants.DEX_PROGRAMS.JUPITER_DCA.Name,
			Type:      "protocol",
		},
		User:      event.UserKey,
		ProgramId: constants.DEX_PROGRAMS.JUPITER_DCA.ID,
		AMM:       p.getAMM(),
		Route:     p.DexInfo.Route,
		Slot:      p.Adapter.Slot(),
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
		Idx:       idx,
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// getAMM gets the AMM name from transfer actions
func (p *JupiterDCAParser) getAMM() string {
	amms := utils.GetAMMs(p.getTransferActionKeys())
	if len(amms) > 0 {
		return amms[0]
	}
	if p.DexInfo.AMM != "" {
		return p.DexInfo.AMM
	}
	return constants.DEX_PROGRAMS.JUPITER_DCA.Name
}

// getTransferActionKeys returns the transfer action keys in execution order,
// so the AMM chosen from them does not depend on map iteration order
func (p *JupiterDCAParser) getTransferActionKeys() []string {
	return utils.SortedTransferKeys(p.TransferActions)
}

// DCA self-CPI events (DCA IDL): the Anchor event prefix followed by
// sha256("event:Opened")[:8] / sha256("event:Closed")[:8]
var (
	dcaOpenedEvent = []byte{228, 69, 165, 46, 81, 203, 154, 29, 166, 172, 97, 9, 77, 76, 189, 109}
	dcaClosedEvent = []byte{228, 69, 165, 46, 81, 203, 154, 29, 50, 31, 87, 155, 135, 220, 195, 239}
)

// ProcessTransfers reports DCA (Recurring) deposits and refunds from the DCA
// program's events. Opened (open_dca, open_dca_v2) is the user's deposit of
// in_deposited input tokens. Closed (close_dca by the user, end_and_close by a
// keeper) returns what was left in the DCA account: the close instruction's
// token transfers of the input or output mint, or, when none is found, the
// event's unfilled input amount.
func (p *JupiterDCAParser) ProcessTransfers() []types.TransferData {
	var transfers []types.TransferData

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.JUPITER_DCA.ID {
			continue
		}
		data := p.Adapter.GetInstructionData(ci.Instruction)
		if len(data) < 16 {
			continue
		}
		switch {
		case bytes.Equal(data[:16], dcaOpenedEvent):
			if event, err := ParseJupiterDCAOpenedEvent(data[16:]); err == nil {
				transfers = append(transfers, p.parseOpened(event, ci)...)
			}
		case bytes.Equal(data[:16], dcaClosedEvent):
			if event, err := ParseJupiterDCAClosedEvent(data[16:]); err == nil {
				transfers = append(transfers, p.parseClosed(event, ci)...)
			}
		}
	}

	return transfers
}

// orderTransfers returns the token transfers of the DCA instruction that
// emitted the event ci
func (p *JupiterDCAParser) orderTransfers(ci types.ClassifiedInstruction) (*types.ClassifiedInstruction, []types.TransferData) {
	order := findEmittingInstruction(p.Adapter, p.ClassifiedInstructions, ci)
	if order == nil {
		return nil, nil
	}
	return order, p.GetTransfersForInstruction(order.ProgramId, order.OuterIndex, order.InnerIndex, nil)
}

// parseOpened reports the deposit of an Opened event: in_deposited of the
// input mint, from the user into the DCA's input account
func (p *JupiterDCAParser) parseOpened(event *JupiterDCAOrderEvent, ci types.ClassifiedInstruction) []types.TransferData {
	order, instTransfers := p.orderTransfers(ci)
	idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	if order != nil {
		idx = utils.FormatIdx(order.OuterIndex, order.InnerIndex)
	}

	transfer := p.eventTransfer("OpenDca", event.InputMint, event.InDeposited, event.UserKey, event.DCAKey, idx)
	transfer.Info.Authority = event.UserKey
	for i := range instTransfers {
		if t := instTransfers[i]; t.Info.Mint == event.InputMint && t.Info.TokenAmount.Amount == event.InDeposited.String() {
			transfer.Info.Source = t.Info.Source
			transfer.Info.Destination = t.Info.Destination
			transfer.Info.DestinationOwner = t.Info.DestinationOwner
			transfer.Info.SourceBalance = t.Info.SourceBalance
			transfer.Info.SourcePreBalance = t.Info.SourcePreBalance
			transfer.Info.DestinationBalance = t.Info.DestinationBalance
			transfer.Info.DestinationPreBalance = t.Info.DestinationPreBalance
			break
		}
	}
	return []types.TransferData{transfer}
}

// parseClosed reports what a close returned to the user: the close
// instruction's transfers of the order's input or output mint, else the
// event's unfilled input amount
func (p *JupiterDCAParser) parseClosed(event *JupiterDCAOrderEvent, ci types.ClassifiedInstruction) []types.TransferData {
	var transfers []types.TransferData
	order, instTransfers := p.orderTransfers(ci)
	idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	if order != nil {
		idx = utils.FormatIdx(order.OuterIndex, order.InnerIndex)
	}

	for _, t := range instTransfers {
		if t.Info.Mint != event.InputMint && t.Info.Mint != event.OutputMint {
			continue
		}
		// Reported as a DCA transfer like OpenDca: the order program's id,
		// not the token program's
		t.Type = "CloseDca"
		t.ProgramId = constants.DEX_PROGRAMS.JUPITER_DCA.ID
		t.Idx = idx
		transfers = append(transfers, t)
	}
	if len(transfers) == 0 && event.UnfilledAmount.Sign() > 0 {
		transfer := p.eventTransfer("CloseDca", event.InputMint, event.UnfilledAmount, event.DCAKey, event.UserKey, idx)
		transfer.Info.Authority = event.DCAKey
		transfers = append(transfers, transfer)
	}
	return transfers
}

// eventTransfer builds a transfer of amount of mint from source to destination
func (p *JupiterDCAParser) eventTransfer(transferType, mint string, amount *big.Int, source, destination, idx string) types.TransferData {
	decimals := p.Adapter.GetTokenDecimals(mint)
	uiAmount := types.ConvertToUIAmount(amount, decimals)
	return types.TransferData{
		Type:      transferType,
		ProgramId: constants.DEX_PROGRAMS.JUPITER_DCA.ID,
		Info: types.TransferDataInfo{
			Source:      source,
			Destination: destination,
			Mint:        mint,
			TokenAmount: types.TokenAmount{
				Amount:   amount.String(),
				UIAmount: &uiAmount,
				Decimals: decimals,
			},
		},
		Idx:       idx,
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
	}
}

// GetAMMs extracts AMM names from transfer action keys
func GetAMMs(keys []string) []string {
	return utils.GetAMMs(keys)
}
