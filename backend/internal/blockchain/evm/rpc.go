package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("evm rpc error %d: %s", e.Code, e.Message)
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type rpcResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

// call mengirim satu request JSON-RPC lewat HTTP dan mengisi out dengan hasilnya.
func (c *Client) call(ctx context.Context, method string, params any, out any) error {
	if c.rpcURL == "" {
		return fmt.Errorf("evm: rpc url kosong")
	}

	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.rpcURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("evm: %s: %w", method, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 && len(data) == 0 {
		return fmt.Errorf("evm: %s: http %d", method, resp.StatusCode)
	}

	var envelope rpcResponse
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("evm: decode %s: %w", method, err)
	}
	if envelope.Error != nil {
		return envelope.Error
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(envelope.Result, out)
}

// batch mengirim beberapa request dalam satu HTTP request.
func (c *Client) batch(ctx context.Context, reqs []rpcRequest) ([]json.RawMessage, error) {
	if len(reqs) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(reqs)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("evm: batch: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}

	var envelopes []rpcResponse
	if err := json.Unmarshal(data, &envelopes); err != nil {
		return nil, fmt.Errorf("evm: decode batch: %w", err)
	}

	out := make([]json.RawMessage, len(reqs))
	for _, env := range envelopes {
		if env.ID < 1 || env.ID > len(reqs) {
			continue
		}
		if env.Error != nil {
			continue
		}
		out[env.ID-1] = env.Result
	}
	return out, nil
}

func (c *Client) httpClient() *http.Client {
	if c.http != nil {
		return c.http
	}
	return &http.Client{Timeout: 20 * time.Second}
}
