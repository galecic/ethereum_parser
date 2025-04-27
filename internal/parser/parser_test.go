package parser

import (
	"context"
	"sync"
	"testing"

	"github.com/galecic/ethereum_parser/internal/models"
	"github.com/stretchr/testify/assert"
)

type MockClient struct {
	blockNumber int
	txs         map[int][]models.Transaction
}

func (m *MockClient) GetBlockNumber(ctx context.Context) (int, error) {
	return m.blockNumber, nil
}

func (m *MockClient) GetTxsFromBlock(ctx context.Context, blockNumber int) ([]models.Transaction, error) {
	return m.txs[blockNumber], nil
}

type MockDataStore struct {
	sync.Mutex
	currentBlock         int
	lastProcessedTxIndex int
	subscribedAddresses  map[models.Address]bool
	transactions         map[models.Address][]models.Transaction
}

func (m *MockDataStore) GetCurrentBlock() int {
	m.Lock()
	defer m.Unlock()
	return m.currentBlock
}

func (m *MockDataStore) SetCurrentBlock(block int) {
	m.Lock()
	defer m.Unlock()
	m.currentBlock = block
}

func (m *MockDataStore) GetLastProcessedTxIndex() int {
	m.Lock()
	defer m.Unlock()
	return m.lastProcessedTxIndex
}

func (m *MockDataStore) SetLastProcessedTxIndex(index int) {
	m.Lock()
	defer m.Unlock()
	m.lastProcessedTxIndex = index
}

func (m *MockDataStore) AddressExists(address models.Address) bool {
	m.Lock()
	defer m.Unlock()
	return m.subscribedAddresses[address]
}

func (m *MockDataStore) AddSubscriber(address models.Address) {
	m.Lock()
	defer m.Unlock()
	if m.subscribedAddresses == nil {
		m.subscribedAddresses = make(map[models.Address]bool)
	}
	m.subscribedAddresses[address] = true
}

func (m *MockDataStore) AddTx(address models.Address, tx models.Transaction) {
	m.Lock()
	defer m.Unlock()
	if m.transactions == nil {
		m.transactions = make(map[models.Address][]models.Transaction)
	}
	m.transactions[address] = append(m.transactions[address], tx)
}

func (m *MockDataStore) GetTransactions(address models.Address) []models.Transaction {
	m.Lock()
	defer m.Unlock()
	return m.transactions[address]
}

func TestParserRuntime_parseTxs(t *testing.T) {
	mockDataStore := &MockDataStore{}
	mockDataStore.AddSubscriber("0xdef")
	mockDataStore.AddSubscriber("0xghi")

	txs := []models.Transaction{
		{Hash: "0x123", From: "0xabc", To: "0xdef", TransactionIndex: "0x1", BlockNumber: "0xa"},
		{Hash: "0x456", From: "0xghi", To: "0xjkl", TransactionIndex: "0x2", BlockNumber: "0xa"},
	}

	cfg := ParserConfig{Workers: 2}
	ctx := context.Background()
	parser := NewParserRuntime(ctx, nil, mockDataStore, cfg)

	err := parser.parseTxs(ctx, &txs)
	assert.NoError(t, err)

	// Verify transactions were added to the subscribed addresses
	assert.Len(t, mockDataStore.GetTransactions("0xdef"), 1)
	assert.Equal(t, "0x123", mockDataStore.GetTransactions("0xdef")[0].Hash)

	assert.Len(t, mockDataStore.GetTransactions("0xghi"), 1)
	assert.Equal(t, "0x456", mockDataStore.GetTransactions("0xghi")[0].Hash)

	// Verify the current block and last processed transaction index were updated
	assert.Equal(t, 10, mockDataStore.GetCurrentBlock())        // 0xa in decimal
	assert.Equal(t, 2, mockDataStore.GetLastProcessedTxIndex()) // 0x2 in decimal
}

func TestParserRuntime_matchTx(t *testing.T) {
	mockDataStore := &MockDataStore{}
	mockDataStore.AddSubscriber("0xdef")

	txStream := make(chan models.Transaction, 2)
	txStream <- models.Transaction{Hash: "0x123", From: "0xabc", To: "0xdef", TransactionIndex: "0x1", BlockNumber: "0xa"}
	txStream <- models.Transaction{Hash: "0x456", From: "0xghi", To: "0xjkl", TransactionIndex: "0x2", BlockNumber: "0xa"}
	close(txStream)

	cfg := ParserConfig{}
	ctx := context.Background()
	parser := NewParserRuntime(ctx, nil, mockDataStore, cfg)

	parser.matchTx(ctx, txStream)

	// Verify the transaction was added to the subscribed address
	assert.Len(t, mockDataStore.GetTransactions("0xdef"), 1)
	assert.Equal(t, "0x123", mockDataStore.GetTransactions("0xdef")[0].Hash)
}
func TestParserRuntime_getNewTxs(t *testing.T) {
	mockDataStore := &MockDataStore{
		currentBlock: 10,
	}
	mockClient := &MockClient{
		blockNumber: 12,
		txs: map[int][]models.Transaction{
			10: {
				{Hash: "0x123", From: "0xabc", To: "0xdef"},
			},
			11: {
				{Hash: "0x456", From: "0xghi", To: "0xjkl"},
			},
			12: {
				{Hash: "0x789", From: "0xabc", To: "0xdef"},
			},
		},
	}

	cfg := ParserConfig{}
	ctx := context.Background()
	parser := NewParserRuntime(ctx, mockClient, mockDataStore, cfg)

	txs, err := parser.getNewTxs(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, txs)
	assert.Len(t, *txs, 3) // Transactions from blocks 10, 11, and 12
	assert.Equal(t, "0x123", (*txs)[0].Hash)
	assert.Equal(t, "0x456", (*txs)[1].Hash)
	assert.Equal(t, "0x789", (*txs)[2].Hash)
}
