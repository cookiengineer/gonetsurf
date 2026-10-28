package network

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"gonetsurf/ipc"
	"gonetsurf/storages/cookies"
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

func TestBrokerCookiePartitions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Cookie") == "" {
			http.SetCookie(writer, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
			writer.Write([]byte("set"))
			return
		}
		writer.Write([]byte("cookie=" + request.Header.Get("Cookie")))
	}))
	defer server.Close()

	token, _ := ipc.NewCapability()
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()
	broker := &Server{
		Conn:    ipc.NewConn(serverSide),
		Token:   token,
		Fetcher: HTTPFetcher{Jar: cookies.NewJar()},
	}
	go broker.Serve(context.Background())
	client := &Client{Conn: ipc.NewConn(clientSide), Token: token}
	if _, err := client.Fetch(context.Background(), Request{URL: server.URL, Partition: "site-a"}); err != nil {
		t.Fatal(err)
	}
	response, err := client.Fetch(context.Background(), Request{URL: server.URL, Partition: "site-a"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(response.Body), "session=abc") {
		t.Errorf("cookie response = %q", response.Body)
	}
	response, err = client.Fetch(context.Background(), Request{URL: server.URL, Partition: "site-b"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(response.Body), "session=abc") {
		t.Errorf("cross-partition cookie leaked: %q", response.Body)
	}
}

func TestHTTPFetcherCache(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&requests, 1)
		writer.Header().Set("Cache-Control", "max-age=60")
		writer.Write([]byte("cached"))
	}))
	defer server.Close()
	fetcher := HTTPFetcher{Cache: NewCache()}
	first, err := fetcher.Fetch(context.Background(), Request{URL: server.URL, Partition: "site"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := fetcher.Fetch(context.Background(), Request{URL: server.URL, Partition: "site"})
	if err != nil {
		t.Fatal(err)
	}
	if string(first.Body) != "cached" || string(second.Body) != "cached" {
		t.Errorf("bodies = %q %q", first.Body, second.Body)
	}
	if atomic.LoadInt32(&requests) != 1 {
		t.Errorf("requests = %d, want 1 (cache miss)", requests)
	}
	if _, err := fetcher.Fetch(context.Background(), Request{URL: server.URL, Partition: "site", Reload: true}); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&requests) != 2 {
		t.Errorf("requests after reload = %d, want 2", requests)
	}
	if _, err := fetcher.Fetch(context.Background(), Request{URL: server.URL, Partition: "other"}); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&requests) != 3 {
		t.Errorf("requests after partition change = %d, want 3", requests)
	}
}
