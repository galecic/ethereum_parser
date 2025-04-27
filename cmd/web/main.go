package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/galecic/ethereum_parser/internal/client"
	"github.com/galecic/ethereum_parser/internal/data_store"
	"github.com/galecic/ethereum_parser/internal/parser"
)

func main() {
	serverAddr := flag.String("serverAddr", "localhost:8000", "server address")
	publicNode := flag.String("eth_publich_node", "https://ethereum-rpc.publicnode.com", "public node address")
	fetchTxsPeriod := flag.Duration("period", 5*time.Second, "fetch transactions period")
	workers := flag.Int("threads", 10, "number of coroutines for parsing transactions")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	client := client.NewClient(*publicNode)
	db := data_store.NewDataStore()

	cfg := parser.ParserConfig{
		TxFetchInterval: *fetchTxsPeriod,
		Workers:         *workers,
	}

	parser := parser.NewParserRuntime(ctx, client, db, cfg)

	router := NewRouter(parser)

	httpServer := &http.Server{
		Addr:    *serverAddr,
		Handler: router,
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		parser.Parse()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("ListenAndServe error", err)
			cancel()
		}
	}()
	log.Println("Server started", *serverAddr)

	<-ctx.Done()
	ctx = context.Background()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Println("Server shutdown")
	}

	wg.Wait()
	log.Println("Server stopped")
}
