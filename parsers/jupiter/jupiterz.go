package jupiter

import (
	"encoding/binary"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// JupiterZParser parses fills of the Jupiter Z RFQ order engine
// (61DFfeTKM7trxYcPQCM78bJ794ddZprZpAwAnLiwTpYH). In a fill the taker (the
// user) sends input_amount of input_mint to the maker (the market maker, who
// usually pays the transaction fee) and receives output_amount of output_mint.
type JupiterZParser struct {
	*parsers.BaseParser
}

// NewJupiterZParser creates a new Jupiter Z parser
func NewJupiterZParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *JupiterZParser {
	return &JupiterZParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// JupiterZFill is a decoded order_engine fill instruction
type JupiterZFill struct {
	Taker        string
	Maker        string
	InputMint    string
	OutputMint   string
	InputAmount  *big.Int
	OutputAmount *big.Int
	ExpireAt     int64
}

// jupiterZFillMinAccounts is the account count of fill in the on-chain IDL:
// taker, maker, taker_input_mint_token_account, maker_input_mint_token_account,
// taker_output_mint_token_account, maker_output_mint_token_account,
// input_mint, input_token_program, output_mint, output_token_program,
// system_program
const jupiterZFillMinAccounts = 11

// ParseJupiterZFill decodes a fill instruction from its data (including the
// 8-byte discriminator) and accounts. Layout after the discriminator:
// input_amount(u64) + output_amount(u64) + expire_at(i64); real fills carry a
// few more bytes that the IDL does not describe, which are ignored.
func ParseJupiterZFill(data []byte, accounts []string) (*JupiterZFill, error) {
	if len(data) < 32 || len(accounts) < jupiterZFillMinAccounts {
		return nil, ErrInsufficientData
	}
	return &JupiterZFill{
		Taker:        accounts[0],
		Maker:        accounts[1],
		InputMint:    accounts[6],
		OutputMint:   accounts[8],
		InputAmount:  new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[8:16])),
		OutputAmount: new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[16:24])),
		ExpireAt:     int64(binary.LittleEndian.Uint64(data[24:32])),
	}, nil
}

// ProcessTrades returns one trade per fill: user = taker, input = what the
// taker paid, output = what the taker received (the instruction amounts, which
// the program transfers exactly)
func (p *JupiterZParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.JUPITER_Z.ID {
			continue
		}
		data := p.Adapter.GetInstructionData(ci.Instruction)
		if !parsers.MatchDiscriminator(data, constants.DISCRIMINATORS.JUPITER_Z.FILL) {
			continue
		}
		fill, err := ParseJupiterZFill(data, p.Adapter.GetInstructionAccounts(ci.Instruction))
		if err != nil {
			continue
		}

		inDecimals := p.Adapter.GetTokenDecimals(fill.InputMint)
		outDecimals := p.Adapter.GetTokenDecimals(fill.OutputMint)
		trade := &types.TradeInfo{
			Type: utils.GetTradeType(fill.InputMint, fill.OutputMint),
			InputToken: types.TokenInfo{
				Mint:      fill.InputMint,
				Amount:    types.ConvertToUIAmount(fill.InputAmount, inDecimals),
				AmountRaw: fill.InputAmount.String(),
				Decimals:  inDecimals,
			},
			OutputToken: types.TokenInfo{
				Mint:      fill.OutputMint,
				Amount:    types.ConvertToUIAmount(fill.OutputAmount, outDecimals),
				AmountRaw: fill.OutputAmount.String(),
				Decimals:  outDecimals,
			},
			User:      fill.Taker,
			ProgramId: constants.DEX_PROGRAMS.JUPITER_Z.ID,
			AMM:       constants.DEX_PROGRAMS.JUPITER_Z.Name,
			Route:     p.DexInfo.Route,
			Slot:      p.Adapter.Slot(),
			Timestamp: p.Adapter.BlockTime(),
			Signature: p.Adapter.Signature(),
			Idx:       utils.FormatIdx(ci.OuterIndex, ci.InnerIndex),
		}
		trades = append(trades, *p.Utils.AttachTokenTransferInfo(trade, p.TransferActions))
	}

	return trades
}
