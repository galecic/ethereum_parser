package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/galecic/ethereum_parser/internal/helpers"
	"github.com/galecic/ethereum_parser/internal/models"
)

type Client interface {
	GetBlockNumber(ctx context.Context) (int, error)
	GetTxsFromBlock(ctx context.Context, blockNumber int) ([]models.Transaction, error)
}

const (
	jsonRpcVersion = "2.0"
)

type RPCRequest struct {
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      int    `json:"id"`
	JSONRPC string `json:"jsonrpc"`
}

func NewRequest(method string, params ...any) *RPCRequest {
	request := &RPCRequest{
		Method:  method,
		Params:  params,
		JSONRPC: jsonRpcVersion,
	}

	return request
}

type RPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
	ID      int       `json:"id"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return strconv.Itoa(e.Code) + ": " + e.Message
}

type HTTPError struct {
	Code int
	err  error
}

func (e *HTTPError) Error() string {
	return e.err.Error()
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type JsonRpcClient struct {
	endpoint           string
	httpClient         HTTPClient
	customHeaders      map[string]string
	allowUnknownFields bool
	defaultRequestID   int
}

type RPCRequests []*RPCRequest

func NewClient(endpoint string) Client {
	JsonRpcClient := &JsonRpcClient{
		endpoint:      endpoint,
		httpClient:    &http.Client{},
		customHeaders: make(map[string]string),
	}
	return JsonRpcClient
}

func (c *JsonRpcClient) GetBlockNumber(ctx context.Context) (int, error) {
	var numberHex string
	rpcResponse, err := c.Call(ctx, "eth_blockNumber", &numberHex)
	if err != nil {
		return 0, err
	}
	resultStr, ok := rpcResponse.Result.(string)
	if !ok {
		return 0, fmt.Errorf("failed converting rpcResponse to string")
	}
	number, err := helpers.ParseHexInt(resultStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse hex int: %w", err)
	}
	return number, nil
}

type numberAndFullTxFlag [2]any

func (c *JsonRpcClient) GetTxsFromBlock(ctx context.Context, number int) ([]models.Transaction, error) {
	var (
		params = numberAndFullTxFlag{
			helpers.FormatHexInt(number),
			true,
		}
		response struct {
			Txs []models.Transaction `json:"transactions"`
		}
	)
	rpcResponse, err := c.Call(ctx, "eth_getBlockByNumber", params[:]...)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(rpcResponse.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rpcResponse.Result: %w", err)
	}

	err = json.Unmarshal(resultBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal rpcResponse.Result into response: %w", err)
	}

	return response.Txs, nil
}

func (client *JsonRpcClient) Call(ctx context.Context, method string, params ...any) (*RPCResponse, error) {

	request := &RPCRequest{
		ID:      client.defaultRequestID,
		Method:  method,
		Params:  params,
		JSONRPC: jsonRpcVersion,
	}

	return client.doCall(ctx, request)
}

func (client *JsonRpcClient) doCall(ctx context.Context, RPCRequest *RPCRequest) (*RPCResponse, error) {

	httpRequest, err := client.newRequest(ctx, RPCRequest)
	if err != nil {
		return nil, fmt.Errorf("rpc call %v() on %v: %w", RPCRequest.Method, client.endpoint, err)
	}
	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("rpc call %v() on %v: %w", RPCRequest.Method, httpRequest.URL.Redacted(), err)
	}
	defer httpResponse.Body.Close()

	var rpcResponse *RPCResponse
	decoder := json.NewDecoder(httpResponse.Body)
	if !client.allowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	decoder.UseNumber()
	err = decoder.Decode(&rpcResponse)

	if err != nil {
		if httpResponse.StatusCode >= 400 {
			return nil, &HTTPError{
				Code: httpResponse.StatusCode,
				err:  fmt.Errorf("rpc call %v() on %v status code: %v. could not decode body to rpc response: %w", RPCRequest.Method, httpRequest.URL.Redacted(), httpResponse.StatusCode, err),
			}
		}
		return nil, fmt.Errorf("rpc call %v() on %v status code: %v. could not decode body to rpc response: %w", RPCRequest.Method, httpRequest.URL.Redacted(), httpResponse.StatusCode, err)
	}

	if rpcResponse == nil {
		if httpResponse.StatusCode >= 400 {
			return nil, &HTTPError{
				Code: httpResponse.StatusCode,
				err:  fmt.Errorf("rpc call %v() on %v status code: %v. rpc response missing", RPCRequest.Method, httpRequest.URL.Redacted(), httpResponse.StatusCode),
			}
		}
		return nil, fmt.Errorf("rpc call %v() on %v status code: %v. rpc response missing", RPCRequest.Method, httpRequest.URL.Redacted(), httpResponse.StatusCode)
	}

	if httpResponse.StatusCode >= 400 {
		if rpcResponse.Error != nil {
			return rpcResponse, &HTTPError{
				Code: httpResponse.StatusCode,
				err:  fmt.Errorf("rpc call %v() on %v status code: %v. rpc response error: %v", RPCRequest.Method, httpRequest.URL.Redacted(), httpResponse.StatusCode, rpcResponse.Error),
			}
		}
		return rpcResponse, &HTTPError{
			Code: httpResponse.StatusCode,
			err:  fmt.Errorf("rpc call %v() on %v status code: %v. no rpc error available", RPCRequest.Method, httpRequest.URL.Redacted(), httpResponse.StatusCode),
		}
	}

	return rpcResponse, nil
}

func (client *JsonRpcClient) newRequest(ctx context.Context, req interface{}) (*http.Request, error) {

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", client.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	for k, v := range client.customHeaders {
		if k == "Host" {
			request.Host = v
		} else {
			request.Header.Set(k, v)
		}
	}

	return request, nil
}
