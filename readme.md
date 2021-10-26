# Introduction

This is a very barebones socket.io client library.

it only supports a very small subset of socket.io.
- WebSocket transport
- hardcoded Engine.IO 3 (used by realtime.streamelements.com)
- no long-polling
- no namespaces
- very barebones `Send(Message)` implementation

## Example Low-Level consumption

```go

func main() {
    // cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    logger := log.Default()
    sio := New(logger)

    if err := sio.Connect(ctx, `wss://realtime.streamelements.com/socket.io/`); err != nil {
        panic(err)
    }
    defer sio.Close()

    for {
        /*
        type Packet struct {
            Type    PacketType
            Action  ActionType
            Payload []byte
        }
        */
        p, _ := sio.Receive(ctx)

        if p.Type == TypeMESSAGE {
            type demoPayload struct {
                Type string `json:"type"`
                // ...
            }
            var demo demoPayload
            // in case Payload is just json
            _ = json.Unmarshal(p.Payload, &demo)
        }
    }
}
```

## Example Message Event consumption using Broker

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := log.Default()

	sio := New(logger)
	_ = sio.Connect(ctx, `wss://realtime.streamelements.com/socket.io/`)
	defer sio.Close()

	broker := NewBroker(sio)

	onMyEvent := make(chan *Message, 5)
	broker.Subscribe("my:event", onMyEvent)
	// defer broker.Unsubscribe("my:event", onMyEvent)

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		broker.Listen(ctx)
	}()

	select {
	case <-ctx.Done():
		// done
	case msg := <-onMyEvent:
        // handle msg
        /*       
        type Message struct {
            Name    string
            Payload json.RawMessage
            Nonce   json.RawMessage
        }
        */
	}
	wg.Wait()
}
```
