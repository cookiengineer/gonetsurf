package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"gonetsurf/brokers/network"
	"gonetsurf/brokers/storage"
	"gonetsurf/ipc"
	"gonetsurf/storages/cookies"
)

const (
	envNetworkSocket = "GONETSURF_NETWORK_SOCKET"
	envStorageSocket = "GONETSURF_STORAGE_SOCKET"
	envBrokerToken   = "GONETSURF_BROKER_TOKEN"
	envCookiePath    = "GONETSURF_COOKIE_PATH"
	envStoragePath   = "GONETSURF_STORAGE_PATH"
)

type brokerProcess struct {
	command *exec.Cmd
	dir     string
	network *network.Client
	storage *storage.Client
}

func startBroker() (*brokerProcess, error) {
	directory, err := os.MkdirTemp("", "gonetsurf-broker-")
	if err != nil {
		return nil, err
	}
	stateDirectory, err := os.UserConfigDir()
	if err != nil {
		os.RemoveAll(directory)
		return nil, err
	}
	stateDirectory = filepath.Join(stateDirectory, "gonetsurf")
	if err := os.MkdirAll(stateDirectory, 0o700); err != nil {
		os.RemoveAll(directory)
		return nil, err
	}
	networkPath := filepath.Join(directory, "network.sock")
	storagePath := filepath.Join(directory, "storage.sock")
	token, err := ipc.NewCapability()
	if err != nil {
		os.RemoveAll(directory)
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		os.RemoveAll(directory)
		return nil, err
	}
	command := exec.Command(executable, "--broker")
	command.Env = append(os.Environ(),
		envNetworkSocket+"="+networkPath,
		envStorageSocket+"="+storagePath,
		envBrokerToken+"="+token.String(),
		envCookiePath+"="+filepath.Join(stateDirectory, "cookies.json"),
		envStoragePath+"="+filepath.Join(stateDirectory, "storage.json"),
	)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		os.RemoveAll(directory)
		return nil, err
	}
	broker := &brokerProcess{command: command, dir: directory}
	deadline := time.Now().Add(3 * time.Second)
	var networkConnection net.Conn
	var storageConnection net.Conn
	for time.Now().Before(deadline) {
		networkConnection, err = net.DialTimeout("unix", networkPath, 100*time.Millisecond)
		if err == nil {
			storageConnection, err = net.DialTimeout("unix", storagePath, 100*time.Millisecond)
			if err == nil {
				break
			}
			networkConnection.Close()
		}
		time.Sleep(20 * time.Millisecond)
	}
	if networkConnection == nil || storageConnection == nil || err != nil {
		broker.stop()
		return nil, fmt.Errorf("netsurf: broker did not start: %w", err)
	}
	broker.network = &network.Client{Conn: ipc.NewConn(networkConnection), Token: token}
	broker.storage = &storage.Client{Conn: ipc.NewConn(storageConnection), Token: token}
	return broker, nil
}

func (broker *brokerProcess) stop() {
	if broker == nil {
		return
	}
	if broker.command != nil && broker.command.Process != nil {
		_ = broker.command.Process.Signal(syscall.SIGTERM)
		_, _ = broker.command.Process.Wait()
	}
	if broker.dir != "" {
		os.RemoveAll(broker.dir)
	}
}

func runBroker() error {
	networkPath := os.Getenv(envNetworkSocket)
	storagePath := os.Getenv(envStorageSocket)
	token, err := ipc.ParseCapability(os.Getenv(envBrokerToken))
	if err != nil {
		return err
	}
	if networkPath == "" || storagePath == "" {
		return fmt.Errorf("netsurf: broker socket paths are not set")
	}
	os.Remove(networkPath)
	os.Remove(storagePath)
	networkListener, err := net.Listen("unix", networkPath)
	if err != nil {
		return err
	}
	defer networkListener.Close()
	storageListener, err := net.Listen("unix", storagePath)
	if err != nil {
		return err
	}
	defer storageListener.Close()

	jar := cookies.NewJar()
	if cookiePath := os.Getenv(envCookiePath); cookiePath != "" {
		if _, err := os.Stat(cookiePath); err == nil {
			_ = jar.Load(cookiePath)
		}
		defer jar.Save(cookiePath)
	}
	store := storage.NewStore()
	if storePath := os.Getenv(envStoragePath); storePath != "" {
		fileStore, err := storage.NewFileStore(storePath)
		if err != nil {
			return err
		}
		store = fileStore
	}

	networkConnection, err := networkListener.Accept()
	if err != nil {
		return err
	}
	defer networkConnection.Close()
	storageConnection, err := storageListener.Accept()
	if err != nil {
		return err
	}
	defer storageConnection.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	networkServer := &network.Server{
		Conn:    ipc.NewConn(networkConnection),
		Token:   token,
		Fetcher: network.HTTPFetcher{Jar: jar, Cache: network.NewCache(), Session: cookies.NewSessionManager(jar)},
	}
	storageServer := &storage.Server{
		Conn:  ipc.NewConn(storageConnection),
		Token: token,
		Store: store,
	}
	done := make(chan error, 2)
	go func() { done <- networkServer.Serve(ctx) }()
	go func() { done <- storageServer.Serve(ctx) }()
	for count := 0; count < 2; count++ {
		select {
		case err := <-done:
			if err != nil && ctx.Err() == nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}
