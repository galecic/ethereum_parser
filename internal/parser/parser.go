package parser

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/galecic/ethereum_parser/internal/client"
	"github.com/galecic/ethereum_parser/internal/data_store"
	"github.com/galecic/ethereum_parser/internal/helpers"
	"github.com/galecic/ethereum_parser/internal/models"
)

type Parser interface {
	// GetCurrentBlock - last parsed block
	GetCurrentBlock() int
	// Subscribe - add address to observer
	Subscribe(ctx context.Context, address models.Address)
	// GetTransactions -  list of inbound or outbound transactions for an address
	GetTransactions(ctx context.Context, address models.Address) []models.Transaction
}

type ParserRuntime struct {
	ctx       context.Context
	cfg       ParserConfig
	client    client.Client
	dataStore data_store.DataStore
}

type ParserConfig struct {
	TxFetchInterval time.Duration
	Workers         int
}

func NewParserRuntime(ctx context.Context, client client.Client, data data_store.DataStore, cfg ParserConfig) *ParserRuntime {
	return &ParserRuntime{
		ctx:       ctx,
		client:    client,
		dataStore: data,
		cfg:       cfg,
	}
}

func (p *ParserRuntime) Parse() {
	ticker := time.NewTicker(p.cfg.TxFetchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			err := p.processNewTxs(p.ctx)
			if err != nil {
				log.Println("processTx error", err)
			}
		}
	}
}

func (p *ParserRuntime) processNewTxs(ctx context.Context) error {

	txs, err := p.getNewTxs(ctx)
	if err != nil {
		return err
	}

	if err = p.parseTxs(ctx, txs); err != nil {
		return err
	}

	return nil
}

func (p *ParserRuntime) getNewTxs(ctx context.Context) (*[]models.Transaction, error) {
	remoteBlockNumber, err := p.client.GetBlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	localBlockNumber := p.GetCurrentBlock()

	txs := make([]models.Transaction, 0)

	if localBlockNumber != 0 {
		for localBlockNumber < remoteBlockNumber {
			txsFromCurrentBlock, err := p.client.GetTxsFromBlock(ctx, localBlockNumber)
			if err != nil {
				return nil, err
			}
			txs = append(txs, txsFromCurrentBlock...)

			localBlockNumber++
		}
		txsFromCurrentBlock, err := p.client.GetTxsFromBlock(ctx, localBlockNumber)
		if err != nil {
			return nil, err
		}
		txs = append(txs, txsFromCurrentBlock...)

	} else {
		localBlockNumber = remoteBlockNumber
		p.dataStore.SetCurrentBlock(localBlockNumber)

		txsFromCurrentBlock, err := p.client.GetTxsFromBlock(ctx, localBlockNumber)

		if err != nil {
			return nil, err
		}
		txs = append(txs, txsFromCurrentBlock...)
	}
	return &txs, nil
}

func (p *ParserRuntime) parseTxs(
	ctx context.Context,
	txs *[]models.Transaction,
) error {

	txChan := make(chan models.Transaction)

	wg := sync.WaitGroup{}
	for i := 0; i < p.cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			p.matchTx(ctx, txChan)
			wg.Done()
		}()
	}
	go func() {
		for _, tx := range *txs {
			txChan <- tx
		}
		close(txChan)
	}()

	wg.Wait()

	lastProcessedTxIndex, err := helpers.ParseHexInt((*txs)[len(*txs)-1].TransactionIndex)
	if err != nil {
		return err
	}
	currentBlock, err := helpers.ParseHexInt((*txs)[len(*txs)-1].BlockNumber)
	if err != nil {
		return err
	}

	p.dataStore.SetCurrentBlock(currentBlock)
	p.dataStore.SetLastProcessedTxIndex(lastProcessedTxIndex)

	return err
}

func (p *ParserRuntime) matchTx(
	ctx context.Context,
	txStream chan models.Transaction,
) {
	for tx := range txStream {
		txIdx, err := helpers.ParseHexInt(tx.TransactionIndex)
		if err != nil {
			log.Println("ParseHexInt", err)
			continue
		}

		blockNumber, err := helpers.ParseHexInt(tx.BlockNumber)
		if err != nil {
			log.Println("ParseHexInt", err)
			continue
		}

		if blockNumber == p.dataStore.GetCurrentBlock() {
			if txIdx <= p.dataStore.GetLastProcessedTxIndex() {
				continue
			}
		}

		for _, addr := range []models.Address{tx.From, tx.To} {
			if p.dataStore.AddressExists(addr) {
				p.dataStore.AddTx(addr, tx)
				log.Println("Match found: address", addr, "tx", tx.Hash)
				break
			}
		}
	}
}

func (p *ParserRuntime) GetCurrentBlock() int {
	return p.dataStore.GetCurrentBlock()
}

func (p *ParserRuntime) Subscribe(ctx context.Context, address models.Address) {
	if !address.Valid() {
		log.Println("invalid address")
		return
	}
	p.dataStore.AddSubscriber(address)
}

func (p *ParserRuntime) GetTransactions(ctx context.Context, address models.Address) []models.Transaction {
	if !address.Valid() {
		log.Println("invalid address")
		return nil
	}

	if p.dataStore.AddressExists(address) {
		return p.dataStore.GetTransactions(address)
	} else {
		log.Println("address not subscribed")
		return nil
	}
}
