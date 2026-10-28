// Package network implements the capability-scoped network broker. Renderers
// send fetch requests over IPC; only the broker opens sockets.
package network

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
	"gonetsurf/protocols/http1"
	"gonetsurf/urls"
)

// Request is a broker fetch request.
type Request struct {
	URL    string              `json:"url"`
	Method string              `json:"method,omitempty"`
	Header map[string][]string `json:"header,omitempty"`
}

// Response is a broker fetch response.
type Response struct {
	Status int                 `json:"status"`
	Header map[string][]string `json:"header,omitempty"`
	Body   []byte              `json:"body,omitempty"`
}

// Fetcher performs a validated request.
type Fetcher interface {
	Fetch(ctx context.Context, request Request) (Response, error)
}

// HTTPFetcher fetches HTTP(S) requests with the project HTTP client.
type HTTPFetcher struct {
	Client *http1.Client
}

// Fetch implements Fetcher.
func (fetcher HTTPFetcher) Fetch(ctx context.Context, request Request) (Response, error) {
	target, err := urls.Parse(request.URL)
	if err != nil {
		return Response{}, err
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return Response{}, fmt.Errorf("network broker: unsupported scheme %q", target.Scheme)
	}
	method := request.Method
	if method == "" {
		method = "GET"
	}
	header := http1.Header{}
	for name, values := range request.Header {
		for _, value := range values {
			header = append(header, http1.Field{Name: name, Value: value})
		}
	}
	client := fetcher.Client
	if client == nil {
		client = &http1.Client{Transport: &http1.Transport{}}
	}
	response, err := client.Do(ctx, &http1.Request{Method: method, URL: target, Header: header})
	if err != nil {
		return Response{}, err
	}
	defer response.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 32<<20))
	if err != nil {
		return Response{}, err
	}
	headerMap := make(map[string][]string)
	for _, field := range response.Header {
		headerMap[field.Name] = append(headerMap[field.Name], field.Value)
	}
	return Response{Status: response.StatusCode, Header: headerMap, Body: body}, nil
}

// Server serves fetch requests over one IPC connection.
type Server struct {
	Conn    *ipc.Conn
	Token   ipc.Capability
	Fetcher Fetcher
}

// Serve processes requests until the connection closes or ctx ends.
func (server *Server) Serve(ctx context.Context) error {
	fetcher := server.Fetcher
	if fetcher == nil {
		fetcher = HTTPFetcher{}
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
		if message.Kind == ipc.KindCancel {
			continue
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
		response, err := fetcher.Fetch(ctx, request)
		if err != nil {
			server.sendError(message.RequestID, err.Error())
			continue
		}
		payload, err := json.Marshal(response)
		if err != nil {
			server.sendError(message.RequestID, err.Error())
			continue
		}
		if err := server.Conn.Send(ipc.Message{Kind: ipc.KindResponse, RequestID: message.RequestID, Capability: server.Token, Payload: payload}); err != nil {
			return err
		}
	}
}

func (server *Server) sendError(requestID uint64, text string) {
	payload, _ := json.Marshal(text)
	_ = server.Conn.Send(ipc.Message{Kind: ipc.KindError, RequestID: requestID, Capability: server.Token, Payload: payload})
}

// Client issues fetch requests over one IPC connection.
type Client struct {
	Conn  *ipc.Conn
	Token ipc.Capability

	requestMutex sync.Mutex
	nextID       atomic.Uint64
}

// Fetch performs one broker request.
func (client *Client) Fetch(ctx context.Context, request Request) (Response, error) {
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
			_ = client.Conn.Cancel(requestID)
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
			return Response{}, fmt.Errorf("network broker: %s", text)
		}
	}
}

func tokensEqual(first, second ipc.Capability) bool {
	return subtle.ConstantTimeCompare(first[:], second[:]) == 1
}
