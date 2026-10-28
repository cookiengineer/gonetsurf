// Package storage implements the capability-scoped storage broker for
// partition- and origin-isolated renderer storage.
package storage

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"gonetsurf/ipc"
)

// Request is one storage operation.
type Request struct {
	Action    string `json:"action"`
	Partition string `json:"partition"`
	Origin    string `json:"origin"`
	Key       string `json:"key"`
	Value     []byte `json:"value,omitempty"`
}

// Response is one storage result.
type Response struct {
	Found bool   `json:"found"`
	Value []byte `json:"value,omitempty"`
}

// Store is an in-memory partitioned key-value store. A persistent backend can
// replace it behind the same broker interface.
type Store struct {
	mutex  sync.RWMutex
	values map[string][]byte
}

// NewStore returns an empty store.
func NewStore() *Store {
	return &Store{values: make(map[string][]byte)}
}

func storageKey(partition, origin, key string) string {
	return partition + "\x00" + origin + "\x00" + key
}

// Get returns a value.
func (store *Store) Get(partition, origin, key string) ([]byte, bool) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	value, ok := store.values[storageKey(partition, origin, key)]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), value...), true
}

// Set stores a value.
func (store *Store) Set(partition, origin, key string, value []byte) error {
	if len(value) > 8<<20 {
		return fmt.Errorf("storage broker: value is too large")
	}
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.values[storageKey(partition, origin, key)] = append([]byte(nil), value...)
	return nil
}

// Delete removes a value.
func (store *Store) Delete(partition, origin, key string) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	delete(store.values, storageKey(partition, origin, key))
}

// Server serves storage requests over one IPC connection.
type Server struct {
	Conn  *ipc.Conn
	Token ipc.Capability
	Store *Store
}

// Serve processes requests until the connection closes or ctx ends.
func (server *Server) Serve(ctx context.Context) error {
	store := server.Store
	if store == nil {
		store = NewStore()
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		message, err := server.Conn.Receive()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if message.Kind != ipc.KindRequest {
			continue
		}
		if !tokensEqual(server.Token, message.Capability) {
			server.sendError(message.RequestID, "unauthorized capability")
			continue
		}
		var request Request
		if err := json.Unmarshal(message.Payload, &request); err != nil {
			server.sendError(message.RequestID, "invalid request")
			continue
		}
		if request.Partition == "" || request.Origin == "" || request.Key == "" {
			server.sendError(message.RequestID, "partition, origin, and key are required")
			continue
		}
		var response Response
		switch request.Action {
		case "get":
			value, found := store.Get(request.Partition, request.Origin, request.Key)
			response = Response{Found: found, Value: value}
		case "set":
			if err := store.Set(request.Partition, request.Origin, request.Key, request.Value); err != nil {
				server.sendError(message.RequestID, err.Error())
				continue
			}
			response = Response{Found: true}
		case "delete":
			store.Delete(request.Partition, request.Origin, request.Key)
			response = Response{Found: true}
		default:
			server.sendError(message.RequestID, "unknown action")
			continue
		}
		payload, _ := json.Marshal(response)
		if err := server.Conn.Send(ipc.Message{Kind: ipc.KindResponse, RequestID: message.RequestID, Capability: server.Token, Payload: payload}); err != nil {
			return err
		}
	}
}

func (server *Server) sendError(requestID uint64, text string) {
	payload, _ := json.Marshal(text)
	_ = server.Conn.Send(ipc.Message{Kind: ipc.KindError, RequestID: requestID, Capability: server.Token, Payload: payload})
}

// Client issues storage requests over one IPC connection.
type Client struct {
	Conn  *ipc.Conn
	Token ipc.Capability

	requestMutex sync.Mutex
	nextID       atomic.Uint64
}

// Get reads a value.
func (client *Client) Get(ctx context.Context, partition, origin, key string) ([]byte, bool, error) {
	response, err := client.call(ctx, Request{Action: "get", Partition: partition, Origin: origin, Key: key})
	return response.Value, response.Found, err
}

// Set writes a value.
func (client *Client) Set(ctx context.Context, partition, origin, key string, value []byte) error {
	_, err := client.call(ctx, Request{Action: "set", Partition: partition, Origin: origin, Key: key, Value: value})
	return err
}

// Delete removes a value.
func (client *Client) Delete(ctx context.Context, partition, origin, key string) error {
	_, err := client.call(ctx, Request{Action: "delete", Partition: partition, Origin: origin, Key: key})
	return err
}

func (client *Client) call(ctx context.Context, request Request) (Response, error) {
	client.requestMutex.Lock()
	defer client.requestMutex.Unlock()
	payload, err := json.Marshal(request)
	if err != nil {
		return Response{}, err
	}
	requestID := client.nextID.Add(1)
	deadline := time.Now().Add(30 * time.Second)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	_ = client.Conn.SetDeadline(deadline)
	defer client.Conn.SetDeadline(time.Time{})
	if err := client.Conn.Send(ipc.Message{Kind: ipc.KindRequest, RequestID: requestID, Capability: client.Token, Payload: payload}); err != nil {
		return Response{}, err
	}
	for {
		if err := ctx.Err(); err != nil {
			return Response{}, err
		}
		message, err := client.Conn.Receive()
		if err != nil {
			return Response{}, err
		}
		if message.RequestID != requestID {
			continue
		}
		switch message.Kind {
		case ipc.KindResponse:
			var response Response
			if err := json.Unmarshal(message.Payload, &response); err != nil {
				return Response{}, err
			}
			return response, nil
		case ipc.KindError:
			var text string
			_ = json.Unmarshal(message.Payload, &text)
			return Response{}, fmt.Errorf("storage broker: %s", text)
		}
	}
}

func tokensEqual(first, second ipc.Capability) bool {
	return subtle.ConstantTimeCompare(first[:], second[:]) == 1
}
