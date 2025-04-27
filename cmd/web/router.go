package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/galecic/ethereum_parser/internal/models"
	"github.com/galecic/ethereum_parser/internal/parser"
)

type Router struct {
	parser parser.Parser
	http.Handler
}

const (
	addressParam = "address"
)

func NewRouter(parser parser.Parser) *Router {
	r := &Router{
		parser: parser,
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /current-block", r.GetCurrentBlock)
	mux.HandleFunc("POST /subscribe", r.Subscribe)
	mux.HandleFunc(fmt.Sprintf("GET /transactions/{%s}", addressParam), r.GetTransactions)

	r.Handler = mux

	return r
}

func (h *Router) GetCurrentBlock(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, CurrentBlock{
		CurrentBlockHeight: h.parser.GetCurrentBlock(),
	})
}

type SubscribeRequest struct {
	Address string `json:"address"`
}

func (h *Router) Subscribe(w http.ResponseWriter, r *http.Request) {
	var request SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		handleError(w, err)
		log.Println("error decoding request", err)
		return
	}
	addr := models.Address(request.Address)
	h.parser.Subscribe(r.Context(), addr)

	w.WriteHeader(http.StatusOK)
}

func (h *Router) GetTransactions(w http.ResponseWriter, r *http.Request) {
	addr := models.Address(r.PathValue(addressParam))
	txs := h.parser.GetTransactions(r.Context(), addr)
	writeJSON(w, http.StatusOK, txs)
}

type errorStatus = struct {
	err        error
	statusCode int
	msg        string
}

var errorsList = []errorStatus{
	{
		err:        errors.New("addres not subscribed"),
		statusCode: http.StatusNotFound,
		msg:        "not found subscriber",
	},
	{
		err:        errors.New("no transactions found"),
		statusCode: http.StatusNotFound,
		msg:        "not found transactions",
	},
	{
		err:        errors.New("address already subscribed"),
		statusCode: http.StatusConflict,
		msg:        "address already subscribed",
	},
	{
		err:        errors.New("invalid address"),
		statusCode: http.StatusBadRequest,
		msg:        "invalid address",
	},
}

type ErrorResponse struct {
	Err string `json:"error"`
	Msg string `json:"msg"`
}

func handleError(w http.ResponseWriter, err error) {

	var errStatus = errorStatus{
		statusCode: http.StatusBadRequest,
		msg:        "UNKNOWN ERROR",
	}
	for _, e := range errorsList {
		if errors.Is(err, e.err) {
			errStatus = e

			break
		}
	}
	w.WriteHeader(errStatus.statusCode)
	err = json.NewEncoder(w).Encode(ErrorResponse{
		Msg: errStatus.msg,
		Err: err.Error(),
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
}

type CurrentBlock struct {
	CurrentBlockHeight int `json:"currentBlockHeight"`
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}
}
