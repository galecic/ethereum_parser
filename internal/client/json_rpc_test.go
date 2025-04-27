package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	ethAddr = "https://ethereum-rpc.publicnode.com"
)

func TestGetBlockNumber(t *testing.T) {
	ctx := context.Background()
	client := NewClient(ethAddr)

	number, err := client.GetBlockNumber(ctx)
	require.NoError(t, err)

	require.NotZero(t, number)
	t.Log(number)
}

func TestGetTxsFromBlock(t *testing.T) {
	ctx := context.Background()
	client := NewClient(ethAddr)

	number, err := client.GetBlockNumber(ctx)
	require.NoError(t, err)
	txs, err := client.GetTxsFromBlock(ctx, number)
	require.NoError(t, err)

	require.NotNil(t, txs)
	t.Log(len(txs))
}
