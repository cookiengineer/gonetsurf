package network

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"gonetsurf/ipc"
)

func TestBrokerFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Test", "yes")
		writer.WriteHeader(201)
		writer.Write([]byte("broker body"))
	}))
	defer server.Close()

	token, _ := ipc.NewCapability()
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()
	broker := &Server{Conn: ipc.NewConn(serverSide), Token: token, Fetcher: HTTPFetcher{}}
	go broker.Serve(context.Background())
	client := &Client{Conn: ipc.NewConn(clientSide), Token: token}
	response, err := client.Fetch(context.Background(), Request{URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != 201 || string(response.Body) != "broker body" || response.Header["X-Test"][0] != "yes" {
		t.Errorf("response = %#v", response)
	}
}

func TestBrokerRejectsCapability(t *testing.T) {
	token, _ := ipc.NewCapability()
	wrongToken, _ := ipc.NewCapability()
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()
	broker := &Server{Conn: ipc.NewConn(serverSide), Token: token, Fetcher: HTTPFetcher{}}
	go broker.Serve(context.Background())
	client := &Client{Conn: ipc.NewConn(clientSide), Token: wrongToken}
	if _, err := client.Fetch(context.Background(), Request{URL: "https://example.com/"}); err == nil {
		t.Fatal("expected an authorization error")
	}
}
