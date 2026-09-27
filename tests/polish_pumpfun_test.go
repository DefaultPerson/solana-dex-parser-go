package tests

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/mr-tron/base58"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// Pump.fun buys of the program version before the TradeEvent fee fields
// (121-byte events) paid the protocol fee by a System transfer from the user
// to the buy's fee_recipient account (IDL account 1). The trade reported only
// the bonding-curve amount, without that fee (docs-v1-1: the Quick Start
// example showed 2000000000 for a buy that cost 2020000000).
//
// Truth, decoded from the raw fixtures: the event's sol_amount, the buy
// instruction's fee_recipient and the user's System transfers to it inside
// the same outer instruction. The fee is also 1% of sol_amount (the
// fee_basis_points of 100 the Global account held then).
func TestPolishPumpfunLegacyBuyFee(t *testing.T) {
	cases := []struct {
		sig, idx string
		outer    int
		inner    int // inner index of the TradeEvent
	}{
		{"4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz", "5-3", 5, 3},
		{sigPumpLegacyEvent, "4-3", 4, 3},
		{"5fuBdjC7G3ABez84ZBppPz4SXg1EmhKFToXhfPpr8qGddVTnUaAvrgZ5UPfN3PXJdAXEnWAeoZfDXjG23u25trZB", "1-3", 1, 3},
		{"648cwSysqKXnb3XLPy577Lu4oBk7jimaY8p95JGfS9QUNabYar5pzfcRdu518TWw3dbopquJnMne9qx22xuf8xqn", "7-5", 7, 5},
		{"U9K99asspxi8WzTHhzmvBZZ5BtaZsFijgrgW3zYLSkjGTEXLzHjtwffe3X85LPxqRQ8NvSdS3trhag5qptFASRj", "4-5", 4, 5},
		{"v8s37Srj6QPMtRC1HfJcrSenCHvYebHiGkHVuFFiQ6UviqHnoVx4U77M3TZhQQXewXadHYh5t35LkesJi3ztPZZ", "5-4", 5, 4},
	}
	for _, c := range cases {
		raw := loadMemeRaw(t, c.sig)
		payload := raw.data(c.outer, c.inner)
		if len(payload) != 16+121 || !bytes.Equal(payload[:16], constants.DISCRIMINATORS.PUMPFUN.TRADE_EVENT) {
			t.Fatalf("%.8s: %s is not a 121-byte TradeEvent (%d bytes)", c.sig, c.idx, len(payload))
		}
		payload = payload[16:]
		solAmount := binary.LittleEndian.Uint64(payload[32:])
		user := base58.Encode(payload[49:81])

		// the buy instruction: the legacy buy in this outer instruction
		recipient := ""
		for inner := -1; inner < len(raw.inner[c.outer]); inner++ {
			if raw.program(c.outer, inner) == constants.DEX_PROGRAMS.PUMP_FUN.ID &&
				bytes.HasPrefix(raw.data(c.outer, inner), constants.DISCRIMINATORS.PUMPFUN.BUY) {
				recipient = raw.accounts(c.outer, inner)[1]
			}
		}
		if recipient == "" {
			t.Fatalf("%.8s: no legacy buy in outer instruction %d", c.sig, c.outer)
		}
		fee := raw.sumTransfers(c.outer, -1, len(raw.inner[c.outer]), func(tr memeRawTransfer) bool {
			return tr.Program == memeSystemProgram && tr.From == user && tr.To == recipient
		})
		if fee.Sign() == 0 || fee.Uint64() != solAmount/100 {
			t.Fatalf("%.8s: fee transfer %s, want 1%% of %d", c.sig, fee, solAmount)
		}
		paid := new(big.Int).Add(fee, new(big.Int).SetUint64(solAmount))

		r := memeParse(t, c.sig)
		tr := memeTradeAt(t, r, c.idx)
		if tr.InputToken.Mint != solMint || tr.InputToken.AmountRaw != paid.String() {
			t.Errorf("%.8s: input %s %s, want %s SOL (sol_amount %d + fee %s)", c.sig, tr.InputToken.Mint, tr.InputToken.AmountRaw, paid, solAmount, fee)
		}
		if tr.Fee == nil || tr.Fee.AmountRaw != fee.String() || tr.Fee.Mint != solMint {
			t.Errorf("%.8s: Fee %+v, want %s SOL", c.sig, tr.Fee, fee)
		}
		if len(tr.Fees) != 1 || tr.Fees[0].Type != "protocol" || tr.Fees[0].AmountRaw != fee.String() || tr.Fees[0].Recipient != recipient {
			t.Errorf("%.8s: Fees %+v, want one protocol fee of %s to %s", c.sig, tr.Fees, fee, recipient)
		}
		if e := memeEventAt(t, r, c.idx); e.InputToken == nil || e.InputToken.AmountRaw != paid.String() || feeOf(e.Fees, "protocol") != fee.String() {
			t.Errorf("%.8s: meme event input %+v fees %+v", c.sig, e.InputToken, e.Fees)
		}
	}
}
