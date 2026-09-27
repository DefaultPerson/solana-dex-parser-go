package constants

// TIP_ACCOUNTS maps transaction-landing (SWQoS / relay) providers to the
// accounts that receive their tips. A tip pays for landing the transaction, not
// for the trade, so it is never booked as a trade fee.
//
// Jito: verified. The 8 accounts are owned by the Jito tip-payment program
// T1pyyaTNZsKv2WcRAB8oVnk93mLJw2XzjtVYqCsaHqt (getMultipleAccounts, 2026-09-27).
// All other providers: unverified. The provider attribution comes from
// sol-parser-sdk 0.7.5 src/transaction_cost.rs (mirrored from sol-trade-sdk);
// every account exists on-chain as a system wallet (checked 2026-09-27).
var TIP_ACCOUNTS = map[string][]string{
	"Jito": {
		"96gYZGLnJYVFmbjzopPSU6QiEV5fGqZNyN9nmNhvrZU5",
		"HFqU5x63VTqvQss8hp11i4wVV8bD44PvwucfZ2bU7gRe",
		"Cw8CFyM9FkoMi7K7Crf6HNQqf4uEMzpKw6QNghXLvLkY",
		"ADaUMid9yfUytqMBgopwjb2DTLSokTSzL1zt6iGPaS49",
		"DfXygSm4jCyNCybVYYK6DwvWqjKee8pbDmJGcLWNDXjh",
		"ADuUkR4vqLUMWXxW9gh6D6L8pMSawimctcNZ5pGwDcEt",
		"DttWaMuVvTiduZRnguLF7jNxTgiMBZ1hyAumKUiL2KRL",
		"3AVi9Tg9Uo68tJfuvoKvqKNWKkC5wPdSSdeBnizKZ6jT",
	},
	"Helius": { // unverified
		"4ACfpUFoaSD9bfPdeu6DBt89gB6ENTeHBXCAi87NhDEE",
		"D2L6yPZ2FmmmTKPgzaMKdhu6EWZcTpLy1Vhx8uvZe7NZ",
		"9bnz4RShgq1hAnLnZbP8kbgBg1kEmcJBYQq3gQbmnSta",
		"5VY91ws6B2hMmBFRsXkoAAdsPHBJwRfBht4DXox3xkwn",
		"2nyhqdwKcJZR2vcqCyrYsaPVdAnFoJjiksCXJ7hfEYgD",
		"2q5pghRs6arqVjRvT5gfgWfWcHWmw1ZuCzphgd5KfWGJ",
		"wyvPkWjVZz1M8fHQnMMCDTQDbkManefNNhweYk5WkcF",
		"3KCKozbAaF75qEU33jtzozcJ29yJuaLJTy2jFdzUY8bT",
		"4vieeGHPYPG2MmyPRcYjdiDmmhN3ww7hsFNap8pVN3Ey",
		"4TQLFNWK8AovT1gFvda5jfw2oJeRMKEmw7aH6MGBJ3or",
	},
	"NextBlock": { // unverified
		"NextbLoCkVtMGcV47JzewQdvBpLqT9TxQFozQkN98pE",
		"NexTbLoCkWykbLuB1NkjXgFWkX9oAtcoagQegygXXA2",
		"NeXTBLoCKs9F1y5PJS9CKrFNNLU1keHW71rfh7KgA1X",
		"NexTBLockJYZ7QD7p2byrUa6df8ndV2WSd8GkbWqfbb",
		"neXtBLock1LeC67jYd1QdAa32kbVeubsfPNTJC1V5At",
		"nEXTBLockYgngeRmRrjDV31mGSekVPqZoMGhQEZtPVG",
		"NEXTbLoCkB51HpLBLojQfpyVAMorm3zzKg7w9NFdqid",
		"nextBLoCkPMgmG8ZgJtABeScP35qLa2AMCNKntAP7Xc",
	},
	"ZeroSlot": { // unverified
		"Eb2KpSC8uMt9GmzyAEm5Eb1AAAgTjRaXWFjKyFXHZxF3",
		"FCjUJZ1qozm1e8romw216qyfQMaaWKxWsuySnumVCCNe",
		"ENxTEjSQ1YabmUpXAdCgevnHQ9MHdLv8tzFiuiYJqa13",
		"6rYLG55Q9RpsPGvqdPNJs4z5WTxJVatMB8zV3WJhs5EK",
		"Cix2bHfqPcKcM233mzxbLk14kSggUUiz2A87fJtGivXr",
	},
	"Temporal": { // unverified
		"TEMPaMeCRFAS9EKF53Jd6KpHxgL47uWLcpFArU1Fanq",
		"noz3jAjPiHuBPqiSPkkugaJDkJscPuRhYnSpbi8UvC4",
		"noz3str9KXfpKknefHji8L1mPgimezaiUyCHYMDv1GE",
		"noz6uoYCDijhu1V7cutCpwxNiSovEwLdRHPwmgCGDNo",
		"noz9EPNcT7WH6Sou3sr3GGjHQYVkN3DNirpbvDkv9YJ",
		"nozc5yT15LazbLTFVZzoNZCwjh3yUtW86LoUyqsBu4L",
		"nozFrhfnNGoyqwVuwPAW4aaGqempx4PU6g6D9CJMv7Z",
		"nozievPk7HyK1Rqy1MPJwVQ7qQg2QoJGyP71oeDwbsu",
		"noznbgwYnBLDHu8wcQVCEw6kDrXkPdKkydGJGNXGvL7",
		"nozNVWs5N8mgzuD3qigrCG2UoKxZttxzZ85pvAQVrbP",
		"nozpEGbwx4BcGp6pvEdAh1JoC2CQGZdU6HbNP1v2p6P",
		"nozrhjhkCr3zXT3BiT4WCodYCUFeQvcdUkM7MqhKqge",
		"nozrwQtWhEdrA6W8dkbt9gnUaMs52PdAv5byipnadq3",
		"nozUacTVWub3cL4mJmGCYjKZTnE9RbdY5AP46iQgbPJ",
		"nozWCyTPppJjRuw2fpzDhhWbW355fzosWSzrrMYB1Qk",
		"nozWNju6dY353eMkMqURqwQEoM3SFgEKC6psLCSfUne",
		"nozxNBgWohjR75vdspfxR5H9ceC7XXH99xpxhVGt3Bb",
	},
	"bloXroute": { // unverified
		"HWEoBxYs7ssKuudEjzjmpfJVX7Dvi7wescFsVx2L5yoY",
		"95cfoy472fcQHaw4tPGBTKpn6ZQnfEPfBgDQx6gcRmRg",
		"3UQUKjhMKaY2S6bjcQD6yHB7utcZt5bfarRCmctpRtUd",
		"FogxVNs6Mm2w9rnGL1vkARSwJxvLE8mujTv3LK8RnUhF",
	},
	"Node1": { // unverified
		"node1PqAa3BWWzUnTHVbw8NJHC874zn9ngAkXjgWEej",
		"node1UzzTxAAeBTpfZkQPJXBAqixsbdth11ba1NXLBG",
		"node1Qm1bV4fwYnCurP8otJ9s5yrkPq7SPZ5uhj3Tsv",
		"node1PUber6SFmSQgvf2ECmXsHP5o3boRSGhvJyPMX1",
		"node1AyMbeqiVN6eoQzEAwCA6Pk826hrdqdAHR7cdJ3",
		"node1YtWCoTwwVYTFLfS19zquRQzYX332hs1HEuRBjC",
	},
	"FlashBlock": { // unverified
		"FLaShB3iXXTWE1vu9wQsChUKq3HFtpMAhb8kAh1pf1wi",
		"FLashhsorBmM9dLpuq6qATawcpqk1Y2aqaZfkd48iT3W",
		"FLaSHJNm5dWYzEgnHJWWJP5ccu128Mu61NJLxUf7mUXU",
		"FLaSHR4Vv7sttd6TyDF4yR1bJyAxRwWKbohDytEMu3wL",
		"FLASHRzANfcAKDuQ3RXv9hbkBy4WVEKDzoAgxJ56DiE4",
		"FLasHstqx11M8W56zrSEqkCyhMCCpr6ze6Mjdvqope5s",
		"FLAShWTjcweNT4NSotpjpxAkwxUr2we3eXQGhpTVzRwy",
		"FLasHXTqrbNvpWFB6grN47HGZfK6pze9HLNTgbukfPSk",
		"FLAshyAyBcKb39KPxSzXcepiS8iDYUhDGwJcJDPX4g2B",
		"FLAsHZTRcf3Dy1APaz6j74ebdMC6Xx4g6i9YxjyrDybR",
	},
	"BlockRazor": { // unverified
		"FjmZZrFvhnqqb9ThCuMVnENaM3JGVuGWNyCAxRJcFpg9",
		"6No2i3aawzHsjtThw81iq1EXPJN6rh8eSJCLaYZfKDTG",
		"A9cWowVAiHe9pJfKAj3TJiN9VpbzMUq6E4kEvf5mUT22",
		"Gywj98ophM7GmkDdaWs4isqZnDdFCW7B46TXmKfvyqSm",
		"68Pwb4jS7eZATjDfhmTXgRJjCiZmw1L7Huy4HNpnxJ3o",
		"4ABhJh5rZPjv63RBJBuyWzBK3g9gWMUQdTZP2kiW31V9",
		"B2M4NG5eyZp5SBQrSdtemzk5TqVuaWGQnowGaCBt8GyM",
		"5jA59cXMKQqZAVdtopv8q3yyw9SYfiE3vUCbt7p8MfVf",
		"5YktoWygr1Bp9wiS1xtMtUki1PeYuuzuCF98tqwYxf61",
		"295Avbam4qGShBYK7E9H5Ldew4B3WyJGmgmXfiWdeeyV",
		"EDi4rSy2LZgKJX74mbLTFk4mxoTgT6F7HxxzG2HBAFyK",
		"BnGKHAC386n4Qmv9xtpBVbRaUTKixjBe3oagkPFKtoy6",
		"Dd7K2Fp7AtoN8xCghKDRmyqr5U169t48Tw5fEd3wT9mq",
		"AP6qExwrbRgBAVaehg4b5xHENX815sMabtBzUzVB4v8S",
	},
	"Astralane": { // unverified
		"astrazznxsGUhWShqgNtAdfrzP2G83DzcWVJDxwV9bF",
		"astra4uejePWneqNaJKuFFA8oonqCE1sqF6b45kDMZm",
		"astra9xWY93QyfG6yM8zwsKsRodscjQ2uU2HKNL5prk",
		"astraRVUuTHjpwEVvNBeQEgwYx9w9CFyfxjYoobCZhL",
		"astraEJ2fEj8Xmy6KLG7B3VfbKfsHXhHrNdCQx7iGJK",
		"astraubkDw81n4LuutzSQ8uzHCv4BhPVhfvTcYv8SKC",
		"astraZW5GLFefxNPAatceHhYjfA1ciq9gvfEg2S47xk",
		"astrawVNP4xDBKT7rAdxrLYiTSTdqtUr63fSMduivXK",
		"AstrA1ejL4UeXC2SBP4cpeEmtcFPZVLxx3XGKXyCW6to",
		"AsTra79FET4aCKWspPqeSFvjJNyp96SvAnrmyAxqg5b7",
		"AstrABAu8CBTyuPXpV4eSCJ5fePEPnxN8NqBaPKQ9fHR",
		"AsTRADtvb6tTmrsqULQ9Wji9PigDMjhfEMza6zkynEvV",
		"AsTRAEoyMofR3vUPpf9k68Gsfb6ymTZttEtsAbv8Bk4d",
		"AStrAJv2RN2hKCHxwUMtqmSxgdcNZbihCwc1mCSnG83W",
		"Astran35aiQUF57XZsmkWMtNCtXGLzs8upfiqXxth2bz",
		"AStRAnpi6kFrKypragExgeRoJ1QnKH7pbSjLAKQVWUum",
		"ASTRaoF93eYt73TYvwtsv6fMWHWbGmMUZfVZPo3CRU9C",
	},
	"Stellium": { // unverified
		"ste11JV3MLMM7x7EJUM2sXcJC1H7F4jBLnP9a9PG8PH",
		"ste11MWPjXCRfQryCshzi86SGhuXjF4Lv6xMXD2AoSt",
		"ste11p5x8tJ53H1NbNQsRBg1YNRd4GcVpxtDw8PBpmb",
		"ste11p7e2KLYou5bwtt35H7BM6uMdo4pvioGjJXKFcN",
		"ste11TMV68LMi1BguM4RQujtbNCZvf1sjsASpqgAvSX",
	},
	"Lightspeed": { // unverified
		"53PhM3UTdMQWu5t81wcd35AHGc5xpmHoRjem7GQPvXjA",
		"9tYF5yPDC1NP8s6diiB3kAX6ZZnva9DM3iDwJkBRarBB",
	},
	"Soyas": { // unverified
		"soyas4s6L8KWZ8rsSk1mF3d1mQScoTGGAgjk98bF8nP",
		"soyascXFW5wEEYiwfEmHy2pNwomqzvggJosGVD6TJdY",
		"soyasDBdKjADwPz3xk82U3TNPRDKEWJj7wWLajNHZ1L",
		"soyasE2abjBAynmHbGWgEwk4ctBy7JMTUCNrMbjcnyH",
		"soyasi59njacMUPvo3TM5paHjeK8pYSdovXgFi32gRt",
		"soyasQYhJxv8uZgWDxhg72td6piAf7XTkoyWHtSATEz",
		"soyastP66xyYC8XADXZjdMM5BAVGD2YRvz8dwtLsqb8",
		"soyasvdgUJWYcUCzDxpmjUnNjH7KamXLXTzLwFvdVPE",
		"soyasvxAunisNxaoRxkKGjNir7KmbwYnr37JmefkX9G",
		"soyas5doVFUwH8s5zK8gEvCL5KR5ogDmf52LsrJEZ9h",
	},
	"Speedlanding": { // unverified
		"SpEEdz8S1KorkMZqjMUxfxrmWwofmp6ReNP2Nx6CUmq",
		"SpeeDy3GJM4wcrQmk1itRFWgidvxX4rwjTLMv78wwjE",
		"SPeEdva37vW8vRtqgYjprQs1g3965icfVN5Rt7SMAyh",
		"speEdrSEpox5GUfHWcBc7tQjRuSfUin2yvB7qoYvvJh",
		"SPeEDmkHkN3A2roSZf6aZyEMsmrGqTHKqwP51y2Y4rV",
		"SpeedLdTJXh2RKpXEaP8JCxkWoUVXhtdPQ1EnxBJMxc",
		"SpEediGKLbbXndSYTzwmz6Z3NDgHQLDcTDEvGFkSMH9",
		"speede8xCcUq2Tiv1efXeTuE3k9TDNq8TnGKaKSc6J4",
	},
	"Solami": { // unverified
		"15qWd4huAkoxvhDsHMfpUn27TW1YBYMMJJ2jkAkbeam",
		"9XuGciSwr5wb7dLTQm91JhuBTvj3GG8WjuRDc3obeam",
		"kiQioJNyFG7pU36ELLsRKXkeT48kFbk3b6rSgrWbeam",
		"kjmVhW1UzJrW2sU5bY5NtZ79jpvjSStsj37Pzmabeam",
		"kREnjPWFpt4AHeY5pijPmyXaCrMnbatUQJo7d3Xbeam",
		"praRZG6N6MdbsT4EFpKgZJWReZGXQhAMFcH68oCbeam",
		"SqoKQKU5uwBxovq3R7yEBxFwptc4z7vwoghU3M9beam",
		"sV72TY66T1RfmDSeHPPbwX6wwJ3bBv5hd4ehJ8tbeam",
		"swf8MyEeLo7gtRUo27UuJj6naCASUrypU7dbteSbeam",
		"uiuaQsxA47JybQAVN4FTfYuoEDkMiXV1r591Aewbeam",
	},
	"LunarLander": { // unverified
		"moon17L6BgxXRX5uHKudAmqVF96xia9h8ygcmG2sL3F",
		"moon26Sek222Md7ZydcAGxoKG832DK36CkLrS3PQY4c",
		"moon7fwyajcVstMoBnVy7UBcTx87SBtNoGGAaH2Cb8V",
		"moonBtH9HvLHjLqi9ivyrMVKgFUsSfrz9BwQ9khhn1u",
		"moonCJg8476LNFLptX1qrK8PdRsA1HD1R6XWyu9MB93",
		"moonF2sz7qwAtdETnrgxNbjonnhGGjd6r4W4UC9284s",
		"moonKfftMiGSak3cezvhEqvkPSzwrmQxQHXuspC96yj",
		"moonQBUKBpkifLcTd78bfxxt4PYLwmJ5admLW6cBBs8",
		"moonXwpKwoVkMegt5Bc776cSW793X1irL5hHV1vJ3JA",
		"moonZ6u9E2fgk6eWd82621eLPHt9zuJuYECXAYjMY1C",
	},
	"Glaive": { // unverified
		"GLaiv4GMRYQmthatDS98uQT4HoucgxWT8NeJz6oSwxeU",
		"GLaivL5uPrDpvd1wTtvat38KGqb5WLhEdqQfnmNd3oNr",
		"GLaivinAWh21NaJMhtExtD5G2gZs1xnvaYVZmwqobWZL",
		"GLaivJSUL71FcocYa8tks5vpVyYzvaDMHtyrzfQF2ABr",
		"GLaivRU6eDKrta3p3psFAWPEFLzCjeMHGpPUuQqTjtyv",
		"GLaivq5dU8qHayz9Qf13LjPfVy3SmUhbmickfGiZdmfh",
	},
}

// tipAccountMap is a reverse lookup from tip account to provider
var tipAccountMap map[string]string

func init() {
	tipAccountMap = make(map[string]string)
	for provider, accounts := range TIP_ACCOUNTS {
		for _, account := range accounts {
			tipAccountMap[account] = provider
		}
	}
}

// IsTipAccount checks if an account is a known tip account
func IsTipAccount(account string) bool {
	_, ok := tipAccountMap[account]
	return ok
}

// GetTipProvider returns the provider of a tip account, or an empty string
func GetTipProvider(account string) string {
	return tipAccountMap[account]
}
