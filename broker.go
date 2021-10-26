package socketioclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type Message struct {
	Name    string
	Payload json.RawMessage
	Nonce   json.RawMessage
}
type broker struct {
	sio         *SocketIO
	messages    map[string]map[chan *Message]struct{}
	mtxMessages *sync.Mutex
}

func NewBroker(sio *SocketIO) *broker {
	return &broker{sio, make(map[string]map[chan *Message]struct{}), new(sync.Mutex)}
}

func (b *broker) Subscribe(name string, handler chan *Message) {
	b.mtxMessages.Lock()
	defer b.mtxMessages.Unlock()

	handlers, ok := b.messages[name]
	if !ok {
		handlers = make(map[chan *Message]struct{})
		b.messages[name] = handlers
	}
	handlers[handler] = struct{}{}
}

func (b *broker) Unsubscribe(name string, handler chan *Message) {
	b.mtxMessages.Lock()
	defer b.mtxMessages.Unlock()

	if handlers, ok := b.messages[name]; ok {
		delete(handlers, handler)
	}
}

func (b *broker) publish(name string, msg *Message) {
	b.mtxMessages.Lock()
	defer b.mtxMessages.Unlock()

	handlers, ok := b.messages[name]
	if ok {
		for h := range handlers {
			h <- msg
		}
	}
}

func (b *broker) Listen(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		p, err := b.sio.Receive(ctx)
		if err != nil {
			return err
		}
		switch p.Type {
		case PacketMESSAGE:
			if p.Action != ActionEVENT {
				continue
			}
			// allocate the buffer, or the payload will be overriden
			cpy := make([]byte, len(p.Payload))
			copy(cpy, p.Payload)

			msg, err := parseEventMessage(cpy)
			if err != nil {
				b.sio.logger.Printf("could not parse message. %v", err)
				continue
			}
			b.publish(msg.Name, msg)
		}
	}
}

// parseEventMessage expected format
//  msg = ["name", { ... }, { ...}]
// where
// 	msg[0] = event name
// 	msg[1] = payload
// 	msg[2] = nonce (optional)
func parseEventMessage(body []byte) (*Message, error) {
	/*
		expecting "["EVENTNAME", { }]" as payload
	*/
	var data []json.RawMessage

	// begin of event "["
	dec := json.NewDecoder(bytes.NewReader(body))

	if err := dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("unable to decode message. %v", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty event message")
	}
	var name string
	if err := json.Unmarshal(data[0], &name); err != nil {
		return nil, fmt.Errorf("could not decode event name")
	}

	payload := []byte{}
	nonce := payload
	if len(data) > 1 {
		payload = data[1]
	}
	if len(data) > 2 {
		nonce = data[2]
	}
	return &Message{Name: name, Payload: payload, Nonce: nonce}, nil
}
