package pumpfun

import (
	"encoding/binary"
	"math/big"

	"github.com/mr-tron/base58"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// PumpFeeClaimParser reports the fees and rewards Pump.fun and PumpSwap pay
// out of their vaults as transfers with a Type naming the payout. They are
// not MemeEvents: MemeEvent types follow a coin's life (create, trade,
// complete, migrate) and are shared with TradeInfo.Type, while these move
// value from a program vault to a creator, a user or a holder-rewards
// account, like the deposits and withdrawals of Jupiter orders.
//
//   - collectCreatorFee: Pump.fun collect_creator_fee(_v2)
//     (CollectCreatorFeeEvent: creator, creator_fee, quote_mint)
//   - collectCoinCreatorFee: PumpSwap collect_coin_creator_fee
//     (CollectCoinCreatorFeeEvent: coin_creator, coin_creator_fee, vault ATA,
//     creator token account)
//   - claimCashback: claim_cashback(_v2) of either program (ClaimCashbackEvent:
//     user, amount)
//   - claimTokenIncentives: claim_token_incentives of either program
//     (ClaimTokenIncentivesEvent: user, mint, amount)
//   - distributeCreatorFees: Pump.fun distribute_creator_fees(_v2), the
//     transfers to the coin's fee shareholders
//   - distributeFeeToHolders: Pump.fun distribute_fee_to_holders, the transfers
//     out of the coin's holder-rewards account
//   - transferCreatorFeesToPump: PumpSwap transfer_creator_fees_to_pump(_v2),
//     a coin creator's PumpSwap fees moved to its Pump.fun creator vault
//
// A single payout is the emitting instruction's transfer (token or System) of
// the event amount; when the program moves lamports without a transfer
// instruction (e.g. claim_cashback), it is built from the event, with the
// source and mint from the instruction's accounts where the IDL names them.
// Distributions report every transfer inside the instruction.
type PumpFeeClaimParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
	txUtils         *utils.TransactionUtils
}

// NewPumpFeeClaimParser creates a parser of Pump.fun and PumpSwap fee payouts
// (the TransferParserFactory signature; all instructions of both programs
// are read, so a payout of the other program in the same transaction is
// reported too)
func NewPumpFeeClaimParser(
	adapter *adapter.TransactionAdapter,
	_ types.DexInfo,
	transferActions map[string][]types.TransferData,
	_ []types.ClassifiedInstruction,
) *PumpFeeClaimParser {
	return &PumpFeeClaimParser{
		adapter:         adapter,
		transferActions: transferActions,
		txUtils:         utils.NewTransactionUtils(adapter),
	}
}

// feePayout is a single payout decoded from an event
type feePayout struct {
	transferType string
	amount       *big.Int
	mint         string // "" when the event does not name it
	source       string // "" when unknown
	destination  string
}

// ProcessTransfers returns the fee payouts of the transaction in execution
// order
func (p *PumpFeeClaimParser) ProcessTransfers() []types.TransferData {
	pf, ps := constants.DISCRIMINATORS.PUMPFUN, constants.DISCRIMINATORS.PUMPSWAP
	instructions := utils.ProgramInstructions(p.adapter, constants.DEX_PROGRAMS.PUMP_FUN.ID, constants.DEX_PROGRAMS.PUMP_SWAP.ID)
	utils.SortInstructionsByExecution(instructions)

	var transfers []types.TransferData
	for _, ci := range instructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		isPumpfun := ci.ProgramId == constants.DEX_PROGRAMS.PUMP_FUN.ID
		switch {
		case isPumpfun && constants.MatchDiscriminator(data, pf.COLLECT_CREATOR_FEE_EVENT):
			transfers = append(transfers, p.single(instructions, ci, p.collectCreatorFee(data[16:]))...)
		case !isPumpfun && constants.MatchDiscriminator(data, ps.COLLECT_COIN_CREATOR_FEE_EVENT):
			transfers = append(transfers, p.single(instructions, ci, p.collectCoinCreatorFee(data[16:]))...)
		case constants.MatchDiscriminator(data, pf.CLAIM_CASHBACK_EVENT): // same event in both programs
			transfers = append(transfers, p.single(instructions, ci, p.claimCashback(data[16:], instructions, ci))...)
		case constants.MatchDiscriminator(data, pf.CLAIM_TOKEN_INCENTIVES_EVENT):
			transfers = append(transfers, p.single(instructions, ci, p.claimTokenIncentives(data[16:]))...)
		case isPumpfun && constants.MatchDiscriminator(data, pf.DISTRIBUTE_CREATOR_FEES_EVENT):
			transfers = append(transfers, p.all(instructions, ci, "distributeCreatorFees")...)
		case isPumpfun && constants.MatchDiscriminator(data, pf.DISTRIBUTE_FEE_TO_HOLDERS_EVENT):
			transfers = append(transfers, p.all(instructions, ci, "distributeFeeToHolders")...)
		case !isPumpfun && (constants.MatchDiscriminator(data, ps.TRANSFER_CREATOR_FEES_TO_PUMP) ||
			constants.MatchDiscriminator(data, ps.TRANSFER_CREATOR_FEES_TO_PUMP_V2)):
			transfers = append(transfers, p.label(p.instructionTransfers(ci), ci, "transferCreatorFeesToPump")...)
		}
	}
	return transfers
}

// collectCreatorFee decodes CollectCreatorFeeEvent: timestamp i64, creator,
// creator_fee u64, quote_mint (absent in old events: SOL)
func (p *PumpFeeClaimParser) collectCreatorFee(data []byte) *feePayout {
	if len(data) < 48 {
		return nil
	}
	mint := constants.TOKENS.SOL
	if len(data) >= 80 {
		mint = normalizeQuoteMint(base58.Encode(data[48:80]))
	}
	return &feePayout{
		transferType: "collectCreatorFee",
		amount:       new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[40:48])),
		mint:         mint,
		destination:  base58.Encode(data[8:40]),
	}
}

// collectCoinCreatorFee decodes CollectCoinCreatorFeeEvent: timestamp i64,
// coin_creator, coin_creator_fee u64, coin_creator_vault_ata,
// coin_creator_token_account
func (p *PumpFeeClaimParser) collectCoinCreatorFee(data []byte) *feePayout {
	if len(data) < 112 {
		return nil
	}
	source := base58.Encode(data[48:80])
	return &feePayout{
		transferType: "collectCoinCreatorFee",
		amount:       new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[40:48])),
		mint:         p.adapter.KnownTokenAccountMint(source),
		source:       source,
		destination:  base58.Encode(data[80:112]),
	}
}

// claimCashback decodes ClaimCashbackEvent: user, amount u64, ... The quote
// mint and the paying account come from the emitting instruction (IDL):
// Pump.fun claim_cashback pays SOL from the user volume accumulator (account
// 1); claim_cashback_v2 and PumpSwap claim_cashback name the quote mint at
// account 2 (Pump.fun v2 pays SOL from account 1, other quotes from account
// 5 to 6; PumpSwap pays from account 4 to 5).
func (p *PumpFeeClaimParser) claimCashback(data []byte, instructions []types.ClassifiedInstruction, event types.ClassifiedInstruction) *feePayout {
	if len(data) < 40 {
		return nil
	}
	payout := &feePayout{
		transferType: "claimCashback",
		amount:       new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[32:40])),
		mint:         constants.TOKENS.SOL,
		destination:  base58.Encode(data[0:32]),
	}
	emitter := utils.FindEventEmitter(p.adapter, instructions, event, nil)
	if emitter == nil {
		return payout
	}
	accounts := p.adapter.GetInstructionAccounts(emitter.Instruction)
	ixData := p.adapter.GetInstructionData(emitter.Instruction)
	pf, ps := constants.DISCRIMINATORS.PUMPFUN, constants.DISCRIMINATORS.PUMPSWAP
	switch {
	case emitter.ProgramId == constants.DEX_PROGRAMS.PUMP_FUN.ID && constants.MatchDiscriminator(ixData, pf.CLAIM_CASHBACK) && len(accounts) > 1:
		payout.source = accounts[1]
	case emitter.ProgramId == constants.DEX_PROGRAMS.PUMP_FUN.ID && constants.MatchDiscriminator(ixData, pf.CLAIM_CASHBACK_V2) && len(accounts) > 6:
		payout.mint = normalizeQuoteMint(accounts[2])
		payout.source = accounts[1]
		if payout.mint != constants.TOKENS.SOL {
			payout.source, payout.destination = accounts[5], accounts[6]
		}
	case emitter.ProgramId == constants.DEX_PROGRAMS.PUMP_SWAP.ID && constants.MatchDiscriminator(ixData, ps.CLAIM_CASHBACK) && len(accounts) > 5:
		payout.mint = normalizeQuoteMint(accounts[2])
		payout.source, payout.destination = accounts[4], accounts[5]
	}
	return payout
}

// claimTokenIncentives decodes ClaimTokenIncentivesEvent: user, mint,
// amount u64, ...
func (p *PumpFeeClaimParser) claimTokenIncentives(data []byte) *feePayout {
	if len(data) < 72 {
		return nil
	}
	return &feePayout{
		transferType: "claimTokenIncentives",
		amount:       new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[64:72])),
		mint:         base58.Encode(data[32:64]),
		destination:  base58.Encode(data[0:32]),
	}
}

// single reports one payout: the emitting instruction's transfer of the
// event amount (and mint, when known), else a transfer built from the event
func (p *PumpFeeClaimParser) single(instructions []types.ClassifiedInstruction, event types.ClassifiedInstruction, payout *feePayout) []types.TransferData {
	if payout == nil || payout.amount.Sign() <= 0 {
		return nil
	}
	emitter := utils.FindEventEmitter(p.adapter, instructions, event, nil)
	if emitter == nil {
		emitter = &event
	}
	for _, t := range p.instructionTransfers(*emitter) {
		if t.Info.TokenAmount.Amount == payout.amount.String() && (payout.mint == "" || t.Info.Mint == payout.mint) {
			return p.label([]types.TransferData{t}, *emitter, payout.transferType)
		}
	}
	if payout.mint == "" {
		return nil
	}
	decimals := p.adapter.GetTokenDecimals(payout.mint)
	uiAmount := types.ConvertToUIAmount(payout.amount, decimals)
	transfer := types.TransferData{
		Type:      payout.transferType,
		ProgramId: emitter.ProgramId,
		Info: types.TransferDataInfo{
			Authority:   payout.source,
			Source:      payout.source,
			Destination: payout.destination,
			Mint:        payout.mint,
			TokenAmount: types.TokenAmount{
				Amount:   payout.amount.String(),
				UIAmount: &uiAmount,
				Decimals: decimals,
			},
		},
		Idx:       utils.FormatIdx(emitter.OuterIndex, emitter.InnerIndex),
		Timestamp: p.adapter.BlockTime(),
		Signature: p.adapter.Signature(),
	}
	if payout.mint == constants.TOKENS.SOL {
		if payout.source != "" {
			transfer.Info.SourceBalance = p.adapter.GetAccountBalance([]string{payout.source})[0]
			transfer.Info.SourcePreBalance = p.adapter.GetAccountPreBalance([]string{payout.source})[0]
		}
		transfer.Info.DestinationBalance = p.adapter.GetAccountBalance([]string{payout.destination})[0]
		transfer.Info.DestinationPreBalance = p.adapter.GetAccountPreBalance([]string{payout.destination})[0]
	}
	return []types.TransferData{transfer}
}

// all reports every transfer inside the instruction that emitted event
func (p *PumpFeeClaimParser) all(instructions []types.ClassifiedInstruction, event types.ClassifiedInstruction, transferType string) []types.TransferData {
	emitter := utils.FindEventEmitter(p.adapter, instructions, event, nil)
	if emitter == nil {
		return nil
	}
	return p.label(p.instructionTransfers(*emitter), *emitter, transferType)
}

// instructionTransfers returns the token and System (SOL) transfers made
// inside ci, in execution order
func (p *PumpFeeClaimParser) instructionTransfers(ci types.ClassifiedInstruction) []types.TransferData {
	first, last := utils.CPIGroup(p.adapter, ci)
	var result []types.TransferData
	for _, t := range utils.SortedTransfers(p.transferActions) {
		if t.Type != "transfer" && t.Type != "transferChecked" {
			continue
		}
		if outer, inner := utils.SplitIdx(t.Idx); outer == ci.OuterIndex && inner >= first && inner <= last {
			result = append(result, t)
		}
	}
	return result
}

// label sets the payout type, the paying program and ci's idx on transfers
func (p *PumpFeeClaimParser) label(transfers []types.TransferData, ci types.ClassifiedInstruction, transferType string) []types.TransferData {
	for i := range transfers {
		transfers[i].Type = transferType
		transfers[i].ProgramId = ci.ProgramId
		transfers[i].Idx = utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	}
	return transfers
}
