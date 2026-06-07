package player

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

type MPVIPCClient struct {
	conn    net.Conn
	encoder *json.Encoder
	decoder *json.Decoder
	mu      sync.Mutex
	nextID  int64
}

type mpvIPCResponse struct {
	Data      json.RawMessage `json:"data"`
	Error     string          `json:"error"`
	RequestID int64           `json:"request_id"`
}

func DialMPVIPC(server string, timeout time.Duration) (*MPVIPCClient, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return nil, errors.New("mpv IPC server is empty")
	}

	conn, err := dialMPVIPC(server, timeout)
	if err != nil {
		return nil, err
	}

	return &MPVIPCClient{
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(conn),
	}, nil
}

func (c *MPVIPCClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *MPVIPCClient) TimePositionMS() (int64, error) {
	data, err := c.command("get_property", "time-pos")
	if err != nil {
		return 0, err
	}
	if len(data) == 0 || string(data) == "null" {
		return 0, nil
	}

	var seconds float64
	if err := json.Unmarshal(data, &seconds); err != nil {
		return 0, err
	}
	if seconds <= 0 {
		return 0, nil
	}
	return int64(seconds * 1000), nil
}

func (c *MPVIPCClient) command(name string, args ...string) (json.RawMessage, error) {
	if c == nil || c.conn == nil {
		return nil, io.ErrClosedPipe
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	requestID := c.nextID
	command := make([]any, 0, len(args)+1)
	command = append(command, name)
	for _, arg := range args {
		command = append(command, arg)
	}

	_ = c.conn.SetDeadline(time.Now().Add(2 * time.Second))
	defer c.conn.SetDeadline(time.Time{})

	if err := c.encoder.Encode(map[string]any{
		"command":    command,
		"request_id": requestID,
	}); err != nil {
		return nil, err
	}

	for {
		var response mpvIPCResponse
		if err := c.decoder.Decode(&response); err != nil {
			return nil, err
		}
		if response.RequestID != requestID {
			continue
		}
		if response.Error != "" && response.Error != "success" {
			return nil, fmt.Errorf("mpv IPC %s failed: %s", name, response.Error)
		}
		return response.Data, nil
	}
}
