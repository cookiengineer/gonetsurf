// Command netsurf is the interactive browser shell.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"gonetsurf/runtime/sandbox"
	"gonetsurf/shells"
	"gonetsurf/urls"
	"gonetsurf/windowing/x11"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--broker" {
		if err := runBroker(); err != nil {
			fmt.Fprintln(os.Stderr, "netsurf broker:", err)
			os.Exit(1)
		}
		return
	}
	var initial *urls.URL
	if len(os.Args) > 1 {
		parsed, err := urls.Parse(os.Args[1])
		if err != nil {
			if candidate, candidateErr := urls.Parse("https://" + os.Args[1]); candidateErr == nil {
				parsed = candidate
			} else {
				fmt.Fprintln(os.Stderr, "netsurf:", err)
				os.Exit(2)
			}
		}
		initial = parsed
	}

	broker, err := startBroker()
	if err != nil {
		fmt.Fprintln(os.Stderr, "netsurf:", err)
		os.Exit(1)
	}
	defer broker.stop()

	const width = 1024
	const height = 768
	shell := shells.New(width, height, nil, shells.BrokerClients{Network: broker.network, Storage: broker.storage})
	if initial == nil {
		_ = shell.LoadSession(context.Background())
	}
	if initial != nil {
		shell.NavigateInitial(initial)
	}
	display := x11.New(x11.Options{Width: width, Height: height, Title: "NetSurf"})
	opened, err := display.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "netsurf:", err)
		os.Exit(1)
	}
	defer display.Close()
	if err := sandbox.ApplyUnprivilegedBaseline(); err != nil {
		fmt.Fprintln(os.Stderr, "netsurf: sandbox:", err)
		os.Exit(1)
	}
	if err := sandbox.InstallNetworkSocketGuard(); err != nil {
		fmt.Fprintln(os.Stderr, "netsurf: network sandbox:", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loopDone := make(chan error, 1)
	go func() {
		loopErr := display.EventLoop(ctx)
		if loopErr != nil {
			cancel()
		}
		loopDone <- loopErr
	}()

	runErr := shell.Run(ctx, opened)
	_ = shell.SaveSession(context.Background())
	cancel()
	select {
	case loopErr := <-loopDone:
		if runErr == nil && loopErr != nil && !errors.Is(loopErr, context.Canceled) {
			runErr = loopErr
		}
	default:
	}
	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		fmt.Fprintln(os.Stderr, "netsurf:", runErr)
		os.Exit(1)
	}
}
