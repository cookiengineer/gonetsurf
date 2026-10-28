package storage

import (
	"context"
	"net"
	"path/filepath"
	"testing"

	"gonetsurf/ipc"
)

func brokerPair(t *testing.T) (*Server, *Client) {
	t.Helper()
	token, _ := ipc.NewCapability()
	serverSide, clientSide := net.Pipe()
	t.Cleanup(func() {
		serverSide.Close()
		clientSide.Close()
	})
	server := &Server{Conn: ipc.NewConn(serverSide), Token: token, Store: NewStore()}
	go server.Serve(context.Background())
	return server, &Client{Conn: ipc.NewConn(clientSide), Token: token}
}

func TestStoragePartitionsAndOrigins(t *testing.T) {
	_, client := brokerPair(t)
	ctx := context.Background()
	if err := client.Set(ctx, "site-a", "https://example.com", "token", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := client.Set(ctx, "site-b", "https://example.com", "token", []byte("two")); err != nil {
		t.Fatal(err)
	}
	value, found, err := client.Get(ctx, "site-a", "https://example.com", "token")
	if err != nil || !found || string(value) != "one" {
		t.Errorf("site-a = (%q,%v,%v)", value, found, err)
	}
	value, found, err = client.Get(ctx, "site-b", "https://example.com", "token")
	if err != nil || !found || string(value) != "two" {
		t.Errorf("site-b = (%q,%v,%v)", value, found, err)
	}
	_, found, err = client.Get(ctx, "site-a", "https://other.example", "token")
	if err != nil || found {
		t.Errorf("origin isolation = (%v,%v)", found, err)
	}
	if err := client.Delete(ctx, "site-a", "https://example.com", "token"); err != nil {
		t.Fatal(err)
	}
	_, found, err = client.Get(ctx, "site-a", "https://example.com", "token")
	if err != nil || found {
		t.Errorf("delete = (%v,%v)", found, err)
	}
}

func TestStorageRejectsCapability(t *testing.T) {
	token, _ := ipc.NewCapability()
	wrongToken, _ := ipc.NewCapability()
	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()
	server := &Server{Conn: ipc.NewConn(serverSide), Token: token, Store: NewStore()}
	go server.Serve(context.Background())
	client := &Client{Conn: ipc.NewConn(clientSide), Token: wrongToken}
	if _, _, err := client.Get(context.Background(), "p", "o", "k"); err == nil {
		t.Fatal("expected an authorization error")
	}
}

func TestFileStorePersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "storage.json")
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("site", "https://example.com", "key", []byte("value")); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	value, found := reloaded.Get("site", "https://example.com", "key")
	if !found || string(value) != "value" {
		t.Errorf("reloaded = (%q,%v)", value, found)
	}
	if err := reloaded.Delete("site", "https://example.com", "key"); err != nil {
		t.Fatal(err)
	}
	again, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := again.Get("site", "https://example.com", "key"); found {
		t.Error("deleted value persisted")
	}
}
