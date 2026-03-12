package realtime

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxReconnectDelay  = 30 * time.Second
	baseReconnectDelay = 1 * time.Second
	pingInterval       = 30 * time.Second
	pongWait           = 10 * time.Second
)

// Client manages a WebSocket connection to Appwrite's realtime endpoint.
type Client struct {
	config    SubscriptionConfig
	eventCh   chan BoardEvent
	errCh     chan error
	done      chan struct{}
	closeOnce sync.Once
	mu        sync.Mutex
	conn      *websocket.Conn
}

// NewClient creates a new realtime Client but does not connect yet.
func NewClient(config SubscriptionConfig) *Client {
	return &Client{
		config:  config,
		eventCh: make(chan BoardEvent, 64),
		errCh:   make(chan error, 8),
		done:    make(chan struct{}),
	}
}

// Events returns the channel that receives parsed board events.
func (c *Client) Events() <-chan BoardEvent {
	return c.eventCh
}

// Errors returns the channel that receives connection errors.
func (c *Client) Errors() <-chan error {
	return c.errCh
}

// Connect establishes the WebSocket connection and starts the read loop.
// This is a blocking call — run it in a goroutine.
func (c *Client) Connect() {
	attempt := 0
	for {
		select {
		case <-c.done:
			return
		default:
		}

		err := c.connectOnce()
		if err != nil {
			select {
			case c.errCh <- fmt.Errorf("realtime connection failed: %w", err):
			default:
			}
		}

		select {
		case <-c.done:
			return
		default:
		}

		attempt++
		delay := backoffDelay(attempt)
		select {
		case <-c.done:
			return
		case <-time.After(delay):
		}
	}
}

func (c *Client) connectOnce() error {
	wsURL := c.buildURL()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		conn.Close()
	}()

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pingInterval + pongWait))
	})

	pingDone := make(chan struct{})
	go c.pingLoop(conn, pingDone)
	defer close(pingDone)

	for {
		select {
		case <-c.done:
			return nil
		default:
		}

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return nil
			}
			return fmt.Errorf("read failed: %w", err)
		}

		c.handleMessage(msgBytes)
	}
}

func (c *Client) buildURL() string {
	base := websocketURL(c.config.AppwriteEndpoint)

	channels := []string{
		fmt.Sprintf("databases.%s.collections.items.documents", c.config.DatabaseID),
		fmt.Sprintf("databases.%s.collections.item_history.documents", c.config.DatabaseID),
	}

	params := url.Values{}
	params.Set("project", c.config.ProjectID)
	for _, ch := range channels {
		params.Add("channels[]", ch)
	}

	return base + "?" + params.Encode()
}

func (c *Client) handleMessage(msgBytes []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return
	}

	if errMsg, ok := msg["type"].(string); ok && errMsg == "error" {
		c.handleErrorMessage(msg)
		return
	}

	dataRaw, ok := msg["data"]
	if !ok {
		return
	}
	data, ok := dataRaw.(map[string]interface{})
	if !ok {
		return
	}

	event := parseRealtimeMessage(data, c.config.BoardID)
	if event == nil {
		return
	}

	select {
	case c.eventCh <- *event:
	default:
	}
}

func (c *Client) handleErrorMessage(msg map[string]interface{}) {
	errData, _ := msg["data"].(map[string]interface{})
	errMessage, _ := errData["message"].(string)
	if errMessage == "" {
		return
	}
	select {
	case c.errCh <- fmt.Errorf("appwrite realtime error: %s", errMessage):
	default:
	}
}

func (c *Client) pingLoop(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.Lock()
			if c.conn == conn {
				err := conn.WriteMessage(websocket.PingMessage, nil)
				c.mu.Unlock()
				if err != nil {
					return
				}
			} else {
				c.mu.Unlock()
				return
			}
		}
	}
}

// Close shuts down the client.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn != nil {
			conn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			)
			conn.Close()
		}
	})
}

func backoffDelay(attempt int) time.Duration {
	delay := baseReconnectDelay * time.Duration(math.Pow(2, float64(attempt-1)))
	if delay > maxReconnectDelay {
		delay = maxReconnectDelay
	}
	return delay
}

func channelFromEvent(eventStr string) string {
	parts := strings.Split(eventStr, ".")
	if len(parts) >= 4 {
		return parts[3]
	}
	return eventStr
}
