package constants

// BOT_FEE_ACCOUNTS maps bot names to their fee account addresses
// Trading bots are detected by SOL transfers to these fee accounts
var BOT_FEE_ACCOUNTS = map[string][]string{
	"Trojan": {
		"9yMwSPk9mrXSN7yDHUuZurAh1sjbJsfpUqjZ7SvVtdco",
	},
	"BONKbot": {
		"ZG98FUCjb8mJ824Gbs6RsgVmr1FhXb2oNiJHa2dwmPd",
	},
	"Axiom": {
		"7LCZckF6XXGQ1hDY6HFXBKWAtiUgL9QY5vj1C4Bn1Qjj",
		"4V65jvcDG9DSQioUVqVPiUcUY9v6sb6HKtMnsxSKEz5S",
		"CeA3sPZfWWToFEBmw5n1Y93tnV66Vmp8LacLzsVprgxZ",
		"AaG6of1gbj1pbDumvbSiTuJhRCRkkUNaWVxijSbWvTJW",
		"7oi1L8U9MRu5zDz5syFahsiLUric47LzvJBQX6r827ws",
		"9kPrgLggBJ69tx1czYAbp7fezuUmL337BsqQTKETUEhP",
		"DKyUs1xXMDy8Z11zNsLnUg3dy9HZf6hYZidB6WodcaGy",
		"4FobGn5ZWYquoJkxMzh2VUAWvV36xMgxQ3M7uG1pGGhd",
		"76sxKrPtgoJHDJvxwFHqb3cAXWfRHFLe3VpKcLCAHSEf",
		"H2cDR3EkJjtTKDQKk8SJS48du9mhsdzQhy8xJx5UMqQK",
		"8m5GkL7nVy95G4YVUbs79z873oVKqg2afgKRmqxsiiRm",
		"4kuG6NsAFJNwqEkac8GFDMMheCGKUPEbaRVHHyFHSwWz",
		"8vFGAKdwpn4hk7kc1cBgfWZzpyW3MEMDATDzVZhddeQb",
		"86Vh4XGLW2b6nvWbRyDs4ScgMXbuvRCHT7WbUT3RFxKG",
		"DZfEurFKFtSbdWZsKSDTqpqsQgvXxmESpvRtXkAdgLwM",
		"5L2QKqDn5ukJSWGyqR4RPvFvwnBabKWqAqMzH4heaQNB",
		"DYVeNgXGLAhZdeLMMYnCw1nPnMxkBN7fJnNpHmizTrrF",
		"Hbj6XdxX6eV4nfbYTseysibp4zZJtVRRPn2J3BhGRuK9",
		"846ah7iBSu9ApuCyEhA5xpnjHHX7d4QJKetWLbwzmJZ8",
		"5BqYhuD4q1YD3DMAYkc1FeTu9vqQVYYdfBAmkZjamyZg",
	},
	"GMGN": {
		"BB5dnY55FXS1e1NXqZDwCzgdYJdMCj3B92PU6Q5Fb6DT",
		"7sHXjs1j7sDJGVSMSPjD1b4v3FD6uRSvRWfhRdfv5BiA",
		"HeZVpHj9jLwTVtMMbzQRf6mLtFPkWNSg11o68qrbUBa3",
		"ByRRgnZenY6W2sddo1VJzX9o4sMU4gPDUkcmgrpGBxRy",
		"DXfkEGoo6WFsdL7x6gLZ7r6Hw2S6HrtrAQVPWYx2A1s9",
		"3t9EKmRiAUcQUYzTZpNojzeGP1KBAVEEbDNmy6wECQpK",
		"DymeoWc5WLNiQBaoLuxrxDnDRvLgGZ1QGsEoCAM7Jsrx",
		"dBhdrmwBkRa66XxBuAK4WZeZnsZ6bHeHCCLXa3a8bTJ",
		"6TxjC5wJzuuZgTtnTMipwwULEbMPx5JPW3QwWkdTGnrn",
	},
	"BullX": {
		"9RYJ3qr5eU5xAooqVcbmdeusjcViL5Nkiq7Gske3tiKq",
		"F4hJ3Ee3c5UuaorKAMfELBjYCjiiLH75haZTKqTywRP3",
	},
	"Maestro": {
		"MaestroUL88UBnZr3wfoN7hqmNWFi3ZYCGqZoJJHE36",
		"FRMxAnZgkW58zbYcE7Bxqsg99VWpJh6sMP5xLzAWNabN",
	},
	"Bloom": {
		"7HeD6sLLqAnKVRuSfc1Ko3BSPMNKWgGTiWLKXJF31vKM",
	},
	"BananaGun": {
		"47hEzz83VFR23rLTEeVm9A7eFzjJwjvdupPPmX3cePqF",
		"4BBNEVRgrxVKv9f7pMNE788XM1tt379X9vNjpDH2KCL7",
		"8r2hZoDfk5hDWJ1sDujAi2Qr45ZyZw5EQxAXiMZWLKh2",
	},
	"Raybot": {
		"4mih95RmBqfHYvEfqq6uGGLp1Fr3gVS3VNSEa3JVRfQK",
	},
}

// botFeeAccountMap is a reverse lookup map for fast fee account detection
var botFeeAccountMap map[string]string

func init() {
	botFeeAccountMap = make(map[string]string)
	for botName, accounts := range BOT_FEE_ACCOUNTS {
		for _, account := range accounts {
			botFeeAccountMap[account] = botName
		}
	}
}

// GetBotName returns the bot name if the account is a known bot fee account
// Returns empty string if not found
func GetBotName(account string) string {
	if botName, ok := botFeeAccountMap[account]; ok {
		return botName
	}
	return ""
}

// IsBotFeeAccount checks if an account is a known bot fee account
func IsBotFeeAccount(account string) bool {
	_, ok := botFeeAccountMap[account]
	return ok
}

// GetAllBotFeeAccounts returns all known bot fee accounts as a slice
func GetAllBotFeeAccounts() []string {
	accounts := make([]string, 0, len(botFeeAccountMap))
	for account := range botFeeAccountMap {
		accounts = append(accounts, account)
	}
	return accounts
}

// GetBotNames returns all known bot names
func GetBotNames() []string {
	names := make([]string, 0, len(BOT_FEE_ACCOUNTS))
	for name := range BOT_FEE_ACCOUNTS {
		names = append(names, name)
	}
	return names
}
