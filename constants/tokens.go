package constants

// TOKENS contains known token addresses
var TOKENS = struct {
	NATIVE string
	SOL    string
	USDC   string
	USDT   string
	USD1   string
	USDG   string
	PYUSD  string
	EURC   string
	USDY   string
	FDUSD  string
}{
	NATIVE: "11111111111111111111111111111111",
	SOL:    "So11111111111111111111111111111111111111112", // Wrapped SOL
	USDC:   "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
	USDT:   "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB",
	USD1:   "USD1ttGY1N17NEEHLmELoaybftRBUSErhqYiQzvEmuB",
	USDG:   "2u1tszSeqZ3qBWF3uNGPFc8TzMk2tdiwknnRMWGWjGWH",
	PYUSD:  "2b1kV6DkPAnxd5ixfnxCpjxmKwqjjaYmCZfHsFu24GXo",
	EURC:   "HzwqbKZw8HxMN6bF2yFZNrht3c2iXXzpKcFu7uBEDKtr",
	USDY:   "A1KLoBrKBde8Ty9qtNQUtq3C2ortoC3u7twggz7sEto6",
	FDUSD:  "9zNQRsGLjNKwCUU5Gq5LR8beUCPzQMVMqKAi3SSZh54u",
}

// TOKEN_DECIMALS maps token addresses to their decimal precision
var TOKEN_DECIMALS = map[string]uint8{
	"So11111111111111111111111111111111111111112":   9, // SOL
	"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": 6, // USDC
	"Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB": 6, // USDT
	"USD1ttGY1N17NEEHLmELoaybftRBUSErhqYiQzvEmuB":  6, // USD1
	"2u1tszSeqZ3qBWF3uNGPFc8TzMk2tdiwknnRMWGWjGWH": 6, // USDG
	"2b1kV6DkPAnxd5ixfnxCpjxmKwqjjaYmCZfHsFu24GXo": 6, // PYUSD
	"HzwqbKZw8HxMN6bF2yFZNrht3c2iXXzpKcFu7uBEDKtr": 6, // EURC
	"A1KLoBrKBde8Ty9qtNQUtq3C2ortoC3u7twggz7sEto6": 6, // USDY
	"9zNQRsGLjNKwCUU5Gq5LR8beUCPzQMVMqKAi3SSZh54u": 6, // FDUSD
}

// IsStablecoin checks if a token is a known stablecoin
func IsStablecoin(mint string) bool {
	switch mint {
	case TOKENS.USDC, TOKENS.USDT, TOKENS.USD1, TOKENS.USDG,
		TOKENS.PYUSD, TOKENS.EURC, TOKENS.USDY, TOKENS.FDUSD:
		return true
	default:
		return false
	}
}

// IsSOL checks if a token is SOL (native or wrapped)
func IsSOL(mint string) bool {
	return mint == TOKENS.SOL || mint == TOKENS.NATIVE
}

// IsQuoteToken checks if a token is a quote token (SOL or stablecoin)
func IsQuoteToken(mint string) bool {
	return IsSOL(mint) || IsStablecoin(mint)
}

// GetTokenDecimals returns the decimals for a known token, or 0 if unknown
func GetTokenDecimals(mint string) (uint8, bool) {
	decimals, ok := TOKEN_DECIMALS[mint]
	return decimals, ok
}
