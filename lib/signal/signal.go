package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// Context returns a cancellable context will be complete when the SIGTERM or SIGINT are observed.
func Context() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

		<-ch
		cancel()

		// in case we get a second request to kill, don't gracefully shut down
		<-ch
		os.Exit(1)
	}()

	return ctx
}
