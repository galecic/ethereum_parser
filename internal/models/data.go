package models

import (
	"strings"
)

type Address string

const (
	addrPrefix = "0x"
	addrLen    = len(addrPrefix) + 40
)

func (a Address) Valid() bool {
	return len(a) == addrLen && strings.Contains(string(a), addrPrefix)
}

type Transaction struct {
	BlockNumber      string  `json:"blockNumber"`
	From             Address `json:"from"`
	Hash             string  `json:"hash"`
	To               Address `json:"to"`
	TransactionIndex string  `json:"transactionIndex"`
}

func (tx Transaction) BelongsToAddr(addr Address) bool {
	return tx.From == addr || tx.To == addr
}
