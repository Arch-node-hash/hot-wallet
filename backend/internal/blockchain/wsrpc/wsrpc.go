// Package wsrpc adalah client JSON-RPC di atas WebSocket yang dipakai untuk
// berlangganan notifikasi sekaligus meminta data (batch) lewat koneksi yang
// sama. Client ini tidak terikat ke EVM maupun Solana.
package wsrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// Request adalah satu panggilan JSON-RPC di dalam batch.
type Request struct {
	Method string
	Params any
}

// Client membungkus satu koneksi WebSocket JSON-RPC.
type Client struct {
	conn    *websocket.Conn
	writeMu sync.Mutex

	mu      sync.Mutex
	nextID  uint64
	pending map[uint64]chan envelope
	subs    map[string]func(json.RawMessage)
}

type envelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *uint64         `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  *notification   `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

type notification struct {
	Subscription string          `json:"subscription"`
	Result       json.RawMessage `json:"result"`
}

// Error adalah error JSON-RPC dari node.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// Dial membuka koneksi WebSocket ke url.
func Dial(ctx context.Context, url string, timeout time.Duration) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("wsrpc: url kosong")
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, _, err := websocket.Dial(dialCtx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("wsrpc: dial %s: %w", url, err)
	}
	conn.SetReadLimit(4 << 20)

	return &Client{
		conn:    conn,
		pending: make(map[uint64]chan envelope),
		subs:    make(map[string]func(json.RawMessage)),
	}, nil
}

// Close menutup koneksi dengan handshake normal.
func (c *Client) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

// CloseNow menutup koneksi tanpa handshake.
func (c *Client) CloseNow() {
	c.conn.CloseNow()
}

// ReadLoop membaca pesan dari socket sampai terjadi error/context selesai.
// Wajib dijalankan sebagai satu-satunya reader.
func (c *Client) ReadLoop(ctx context.Context) error {
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return err
		}
		if len(data) == 0 {
			continue
		}

		if data[0] == '[' {
			var batch []envelope
			if err := json.Unmarshal(data, &batch); err != nil {
				continue
			}
			for _, env := range batch {
				c.dispatch(env)
			}
			continue
		}

		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		c.dispatch(env)
	}
}

func (c *Client) dispatch(env envelope) {
	// Notifikasi langganan: eth_subscription (EVM) atau accountNotification (Solana).
	if env.Params != nil && env.Params.Subscription != "" {
		c.mu.Lock()
		handler := c.subs[env.Params.Subscription]
		c.mu.Unlock()
		if handler != nil {
			go handler(env.Params.Result)
		}
		return
	}

	if env.ID == nil {
		return
	}

	c.mu.Lock()
	ch := c.pending[*env.ID]
	c.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- env:
	default:
	}
}

// Call mengirim satu request dan menunggu hasilnya.
func (c *Client) Call(ctx context.Context, method string, params any, out any) error {
	id := atomic.AddUint64(&c.nextID, 1)
	ch := make(chan envelope, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	defer c.forget(id)

	if err := c.write(ctx, map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case env := <-ch:
		if env.Error != nil {
			return env.Error
		}
		if out == nil {
			return nil
		}
		return json.Unmarshal(env.Result, out)
	}
}

// Batch mengirim beberapa request sekaligus dalam satu pesan WebSocket.
// Error per-item tidak menggagalkan seluruh batch; item tersebut bernilai nil.
func (c *Client) Batch(ctx context.Context, reqs []Request) ([]json.RawMessage, error) {
	if len(reqs) == 0 {
		return nil, nil
	}

	msgs := make([]map[string]any, len(reqs))
	chans := make([]chan envelope, len(reqs))
	ids := make([]uint64, len(reqs))

	c.mu.Lock()
	for i, req := range reqs {
		id := atomic.AddUint64(&c.nextID, 1)
		ids[i] = id
		chans[i] = make(chan envelope, 1)
		c.pending[id] = chans[i]
		msgs[i] = map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  req.Method,
			"params":  req.Params,
		}
	}
	c.mu.Unlock()
	defer func() {
		for _, id := range ids {
			c.forget(id)
		}
	}()

	if err := c.write(ctx, msgs); err != nil {
		return nil, err
	}

	out := make([]json.RawMessage, len(reqs))
	for i := range reqs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case env := <-chans[i]:
			if env.Error != nil {
				continue
			}
			out[i] = env.Result
		}
	}
	return out, nil
}

// Subscribe mendaftarkan langganan. method contoh: "eth_subscribe" atau
// "accountSubscribe". ID langganan dikembalikan sebagai string.
func (c *Client) Subscribe(ctx context.Context, method string, params any, handler func(json.RawMessage)) (string, error) {
	var raw json.RawMessage
	if err := c.Call(ctx, method, params, &raw); err != nil {
		return "", err
	}

	id := normalizeID(raw)
	if id == "" {
		return "", fmt.Errorf("wsrpc: id langganan kosong")
	}

	c.mu.Lock()
	c.subs[id] = handler
	c.mu.Unlock()
	return id, nil
}

// Unsubscribe melepas langganan (best effort).
func (c *Client) Unsubscribe(ctx context.Context, id string) {
	var raw json.RawMessage
	_ = c.Call(ctx, "eth_unsubscribe", []any{id}, &raw)
	c.mu.Lock()
	delete(c.subs, id)
	c.mu.Unlock()
}

func (c *Client) write(ctx context.Context, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *Client) forget(id uint64) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func normalizeID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}
