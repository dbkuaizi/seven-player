package player

import (
	"encoding/json"
	"fmt"
	"net"
	"testing"
)

func TestMPVIPCTimePositionMS(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	client := &MPVIPCClient{
		conn:    clientConn,
		encoder: json.NewEncoder(clientConn),
		decoder: json.NewDecoder(clientConn),
	}
	defer client.Close()

	done := make(chan error, 1)
	go func() {
		decoder := json.NewDecoder(serverConn)
		encoder := json.NewEncoder(serverConn)

		var request struct {
			Command   []string `json:"command"`
			RequestID int64    `json:"request_id"`
		}
		if err := decoder.Decode(&request); err != nil {
			done <- err
			return
		}
		if len(request.Command) != 2 || request.Command[0] != "get_property" || request.Command[1] != "time-pos" {
			done <- fmt.Errorf("unexpected command: %#v", request.Command)
			return
		}

		if err := encoder.Encode(map[string]any{
			"event": "idle",
		}); err != nil {
			done <- err
			return
		}
		if err := encoder.Encode(map[string]any{
			"request_id": request.RequestID,
			"error":      "success",
			"data":       12.345,
		}); err != nil {
			done <- err
			return
		}

		done <- nil
	}()

	got, err := client.TimePositionMS()
	if err != nil {
		t.Fatalf("TimePositionMS() error = %v", err)
	}
	if got != 12345 {
		t.Fatalf("TimePositionMS() = %d, want 12345", got)
	}
	if err := <-done; err != nil {
		t.Fatalf("server error = %v", err)
	}
}
