package scope

import (
	"os"
	"os/signal"
	"syscall"
)

// AwaitSignal blocks until a certain signal is received from the
// operating system and returns the received signal. If no signals
// are provided, it waits for SIGINT and SIGTERM.
func AwaitSignal(sigs ...os.Signal) os.Signal {
	if len(sigs) == 0 {
		sigs = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sigs...)
	defer signal.Stop(ch)
	return <-ch
}
