package data_store

import (
	"slices"
	"testing"

	"github.com/galecic/ethereum_parser/internal/models"
	"github.com/stretchr/testify/require"
)

func TestAddressExists(t *testing.T) {
	db := NewDataStore()

	addr := models.Address("0xb0bc44ca9ef6eb6f4eaac6807c9f6307f8136497")
	db.AddSubscriber(addr)

	require.True(t, db.AddressExists(addr))

}

func TestGetTransactions(t *testing.T) {
	db := NewDataStore()

	addr := models.Address("0xb0bc44ca9ef6eb6f4eaac6807c9f6307f8136497")
	db.AddTx(addr, models.Transaction{
		From: addr,
	})

	txs := db.GetTransactions(addr)

	require.True(t, slices.ContainsFunc(txs, func(transaction models.Transaction) bool {
		return transaction.From == addr
	}))
}
