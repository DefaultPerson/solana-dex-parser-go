package propamm

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// disc holds the discriminators the decoders match
var disc = constants.DISCRIMINATORS

// hasTag reports whether data starts with tag and has one of the lengths
func hasTag(data, tag []byte, lengths ...int) bool {
	if !constants.MatchDiscriminator(data, tag) {
		return false
	}
	if len(lengths) == 0 {
		return true
	}
	for _, n := range lengths {
		if len(data) == n {
			return true
		}
	}
	return false
}

// swapLegs names, by instruction account index, the two token movements of
// one swap direction: user input account -> pool input vault, and pool output
// vault -> user output account.
type swapLegs struct {
	userIn, vaultIn, vaultOut, userOut int
}

// bothDirections returns the legs of a pool with sides A and B for either
// direction, A in first
func bothDirections(userA, userB, vaultA, vaultB int) []swapLegs {
	return []swapLegs{
		{userIn: userA, vaultIn: vaultA, vaultOut: vaultB, userOut: userB},
		{userIn: userB, vaultIn: vaultB, vaultOut: vaultA, userOut: userA},
	}
}

// swapLayout describes a recognised swap instruction
type swapLayout struct {
	pool int        // account index of the pool or market
	legs []swapLegs // candidate directions
}

// swapDecoder recognises a venue's swap instruction from its data and its
// number of accounts. It returns nil for every other instruction.
type swapDecoder func(data []byte, accountCount int) *swapLayout

// VenueParser is the trade parser shared by the venues whose swap moves one
// token transfer from the user into a pool vault and one from the other pool
// vault back to the user, inside the swap instruction's own CPI group (prop
// AMMs, Manifest, Byreal, Saros DLMM, Obric).
//
// A trade is emitted only for an instruction of the venue's program that the
// venue's decoder recognises as a swap and whose CPI group holds both
// transfers between the accounts named by the swap layout. Amounts are those
// two transfers (what the pool received and paid out), never user balance
// deltas: when the venue is a hop of an aggregator route, the counterparty is
// the aggregator's intermediate account. User is the transaction signer, as
// for the other AMM parsers.
type VenueParser struct {
	adapter                *adapter.TransactionAdapter
	dexInfo                types.DexInfo
	transferActions        map[string][]types.TransferData
	classifiedInstructions []types.ClassifiedInstruction
	txUtils                *utils.TransactionUtils
	program                constants.DexProgram
	decode                 swapDecoder
}

func newVenueParser(
	program constants.DexProgram,
	decode swapDecoder,
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *VenueParser {
	return &VenueParser{
		adapter:                adapter,
		dexInfo:                dexInfo,
		transferActions:        transferActions,
		classifiedInstructions: classifiedInstructions,
		txUtils:                utils.NewTransactionUtils(adapter),
		program:                program,
		decode:                 decode,
	}
}

// ProcessTrades returns one trade per swap instruction of the venue
func (p *VenueParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.classifiedInstructions {
		if ci.ProgramId != p.program.ID {
			continue
		}
		accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
		layout := p.decode(p.adapter.GetInstructionData(ci.Instruction), len(accounts))
		if layout == nil || layout.pool >= len(accounts) {
			continue
		}
		if trade := p.parseSwap(ci, accounts, layout); trade != nil {
			trades = append(trades, *trade)
		}
	}

	return trades
}

// parseSwap builds the trade of a recognised swap from the transfers of its
// CPI group, trying the layout's directions in order
func (p *VenueParser) parseSwap(ci types.ClassifiedInstruction, accounts []string, layout *swapLayout) *types.TradeInfo {
	transfers := p.cpiTransfers(ci)
	if len(transfers) < 2 {
		return nil
	}

	for _, legs := range layout.legs {
		if legs.userIn >= len(accounts) || legs.vaultIn >= len(accounts) ||
			legs.vaultOut >= len(accounts) || legs.userOut >= len(accounts) {
			continue
		}
		in := findTransfer(transfers, accounts[legs.userIn], accounts[legs.vaultIn])
		out := findTransfer(transfers, accounts[legs.vaultOut], accounts[legs.userOut])
		if in != nil && out != nil {
			return p.buildTrade(ci, accounts[layout.pool], in, out)
		}
	}

	return nil
}

// findTransfer returns the first transfer from source to destination
func findTransfer(transfers []types.TransferData, source, destination string) *types.TransferData {
	for i := range transfers {
		if transfers[i].Info.Source == source && transfers[i].Info.Destination == destination {
			return &transfers[i]
		}
	}
	return nil
}

// buildTrade makes the trade of one swap from its input and output transfers
func (p *VenueParser) buildTrade(ci types.ClassifiedInstruction, pool string, in, out *types.TransferData) *types.TradeInfo {
	inputToken := *p.txUtils.GetTransferTokenInfo(in)
	outputToken := *p.txUtils.GetTransferTokenInfo(out)
	if inputToken.Mint == "" || outputToken.Mint == "" || inputToken.Mint == outputToken.Mint {
		return nil
	}

	trade := &types.TradeInfo{
		Type:        utils.GetTradeType(inputToken.Mint, outputToken.Mint),
		Pool:        []string{pool},
		InputToken:  inputToken,
		OutputToken: outputToken,
		User:        p.adapter.Signer(),
		ProgramId:   p.program.ID,
		AMM:         p.program.Name,
		Route:       p.dexInfo.Route,
		Slot:        p.adapter.Slot(),
		Timestamp:   p.adapter.BlockTime(),
		Signature:   p.adapter.Signature(),
		Idx:         utils.FormatIdx(ci.OuterIndex, ci.InnerIndex),
	}
	trade = p.txUtils.AttachTokenTransferInfo(trade, p.transferActions)

	// AttachTokenTransferInfo takes the first transfers of the transaction
	// with the same mint and amount; keep this swap's own transfer details
	inputChange, outputChange := trade.InputToken.BalanceChange, trade.OutputToken.BalanceChange
	trade.InputToken, trade.OutputToken = inputToken, outputToken
	trade.InputToken.BalanceChange, trade.OutputToken.BalanceChange = inputChange, outputChange

	return trade
}

// cpiTransfers returns the SPL Token and Token-2022 transfers made inside the
// CPI group of ci (see cpiGroup)
func (p *VenueParser) cpiTransfers(ci types.ClassifiedInstruction) []types.TransferData {
	return groupTransfers(p.adapter, p.txUtils, ci, false)
}

// cpiGroup returns the inner instructions of the CPI group of ci
// (utils.CPIGroup), with their inner indexes
func cpiGroup(a *adapter.TransactionAdapter, ci types.ClassifiedInstruction) (indexes []int, instructions []interface{}) {
	first, last := utils.CPIGroup(a, ci)
	for j := first; j <= last; j++ {
		indexes = append(indexes, j)
		instructions = append(instructions, a.GetInnerInstruction(ci.OuterIndex, j))
	}
	return indexes, instructions
}

// groupTransfers returns the SPL Token and Token-2022 transfers of the CPI
// group of ci, and System program transfers too when withNative is set
func groupTransfers(a *adapter.TransactionAdapter, tu *utils.TransactionUtils, ci types.ClassifiedInstruction, withNative bool) []types.TransferData {
	indexes, instructions := cpiGroup(a, ci)
	var transfers []types.TransferData
	for i, ix := range instructions {
		programId := a.GetInstructionProgramId(ix)
		if programId != constants.TOKEN_PROGRAM_ID && programId != constants.TOKEN_2022_PROGRAM_ID &&
			!(withNative && programId == constants.SYSTEM_PROGRAM_ID) {
			continue
		}
		t := tu.ParseInstructionAction(ix, utils.FormatIdx(ci.OuterIndex, indexes[i]), nil)
		if t != nil && (t.Type == "transfer" || t.Type == "transferChecked") {
			transfers = append(transfers, *t)
		}
	}
	return transfers
}
