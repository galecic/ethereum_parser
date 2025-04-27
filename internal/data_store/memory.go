package data_store

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/galecic/ethereum_parser/internal/models"
)

type DataStore interface {
	GetCurrentBlock() int
	SetCurrentBlock(currBlock int)
	GetLastProcessedTxIndex() int
	SetLastProcessedTxIndex(idx int)
	AddSubscriber(addr models.Address)
	AddTx(addr models.Address, tx models.Transaction)
	AddressExists(addr models.Address) bool
	GetTransactions(addr models.Address) []models.Transaction
}

type DB struct {
	mu                    sync.RWMutex
	txMap                 map[models.Address][]models.Transaction
	lastProcessedBlock    atomic.Int64
	lastProcessedTxsIndex atomic.Int64
}

func NewDataStore() DataStore {
	return &DB{
		txMap: make(map[models.Address][]models.Transaction),
	}
}

func (db *DB) AddSubscriber(addr models.Address) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if _, ok := db.txMap[addr]; ok {
		return
	}
	db.txMap[addr] = make([]models.Transaction, 0)
	log.Println("Added Subscriber", addr)
}

func (ds *DB) AddressExists(addr models.Address) bool {
	ds.mu.RLock()
	_, ok := ds.txMap[addr]
	ds.mu.RUnlock()

	return ok
}

func (ds *DB) AddTx(addr models.Address, tx models.Transaction) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.txMap[addr] = append(ds.txMap[addr], tx)
}

func (ds *DB) GetTransactions(addr models.Address) []models.Transaction {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	txs, ok := ds.txMap[addr]
	if !ok {
		return nil
	}

	return txs
}

func (ds *DB) GetCurrentBlock() int {
	return int(ds.lastProcessedBlock.Load())
}
func (ds *DB) GetLastProcessedTxIndex() int {
	return int(ds.lastProcessedTxsIndex.Load())
}
func (ds *DB) SetCurrentBlock(currBlock int) {
	ds.lastProcessedBlock.Store(int64(currBlock))
}
func (ds *DB) SetLastProcessedTxIndex(currIndex int) {
	ds.lastProcessedTxsIndex.Store(int64(currIndex))
}
