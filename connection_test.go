package socketioclient

import (
	"context"
	"log"
	"testing"
)

func TestConnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := log.Default()

	sio := New(logger)
	if err := sio.Connect(ctx, `wss://realtime.streamelements.com/socket.io/`); err != nil {
		t.Fatalf("Failed: %v\n", err)
	}
	defer sio.Close()
}
