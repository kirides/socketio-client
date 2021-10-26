package socketioclient

import (
	"testing"
)

func TestParseEventMessage(t *testing.T) {
	msg, err := parseEventMessage([]byte(`["event", { "name": "a" }]`))
	if err != nil {
		t.Fatalf("failed. %v", err)
	}
	if msg.Name != "event" {
		t.Fatalf("expected %q got %q", "event", msg.Name)
	}
}

/*
func TestListenEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := log.Default()

	sio := New(logger)
	if err := sio.Connect(ctx, `wss://realtime.streamelements.com/socket.io/`); err != nil {
		t.Fatalf("Failed: %v\n", err)
	}
	defer sio.Close()

	type authRequest struct {
		Method string `json:"method"`
		Token  string `json:"token"`
	}
	authReq := authRequest{Method: "jwt", Token: ""}
	sio.SendMessage(ctx, ActionEVENT, "authenticate", authReq)

	broker := NewBroker(sio)

	handler := make(chan *Message, 5)
	broker.Subscribe("redemption", handler)
	defer broker.Unsubscribe("redemption", handler)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		broker.Listen(ctx)
	}()

	select {
	case <-ctx.Done():
		t.FailNow()
	case msg := <-handler:
		var resp struct {
			Abc string `json:"abc"`
		}
		if err := json.Unmarshal(msg.Payload, &resp); err != nil {
			t.FailNow()
		}

		t.Logf("%#v", msg)
		cancel()
	}
	wg.Wait()
}
*/
