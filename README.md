# Ethereum parser

Web applications consist of eth client, parser and data store. \
They are defined by their APIs and easily swapable.

## How to run
### Build:
go build ./cmd/web
### Run:
./web

## Functionality 
### Get the current block
curl -X GET http://localhost:8000/current-block

### Subscribe to an Address
curl -X POST http://localhost:8000/subscribe \
     -H "Content-Type: application/json" \
     -d '{"address": "0xb0bc44ca9ef6eb6f4eaac6807c9f6307f8136497"}'

### Get transactions of a subscribed address
curl -X GET http://localhost:8000/transactions/0xb0bc44ca9ef6eb6f4eaac6807c9f6307f8136497
