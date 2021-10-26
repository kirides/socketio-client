package socketioclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
)

type SocketIO struct {
	bufRx    *bytes.Buffer
	bufTx    *bytes.Buffer
	ws       *websocket.Conn
	mtxTx    *sync.Mutex
	mtxRx    *sync.Mutex
	ctx      context.Context
	cancelFn context.CancelFunc
	logger   Logger
	LogLevel LogLevel
}

type Logger interface {
	Printf(format string, args ...interface{})
}
type noopLogger struct{}

func (noopLogger) Printf(format string, args ...interface{}) {
}

func New(logger Logger) *SocketIO {
	if logger == nil {
		logger = noopLogger{}
	}
	ctx, cancelFn := context.WithCancel(context.Background())
	sio := &SocketIO{
		bufTx:    bytes.NewBuffer(nil),
		bufRx:    bytes.NewBuffer(nil),
		ctx:      ctx,
		cancelFn: cancelFn,
		mtxTx:    new(sync.Mutex),
		mtxRx:    new(sync.Mutex),
		logger:   logger,
		LogLevel: LogWarn,
	}

	return sio
}

func (sio *SocketIO) Connect(ctx context.Context, endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("EIO", "3")
	q.Set("transport", "websocket")
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), http.Header{})
	if err != nil {
		return fmt.Errorf("failed to dial %v. %w", u, err)
	}
	sio.ws = conn

	pOpen, err := sio.Receive(ctx)
	if err != nil || pOpen.Type != PacketOPEN {
		return fmt.Errorf("failed to receive OPEN packet. %w", err)
	}
	type open struct {
		PingInterval *int `json:"pingInterval"`
	}
	var openData open
	if err := json.Unmarshal(pOpen.Payload, &openData); err != nil {
		return fmt.Errorf("could not unmarshal OPEN packet. %w", err)
	}

	pConn, err := sio.Receive(ctx)
	if err != nil || pConn.Type != PacketMESSAGE || pConn.Action != ActionCONNECT {
		return fmt.Errorf("failed to receive MESSAGE CONNECT packet. %w", err)
	}
	if openData.PingInterval != nil {
		pingInterval := *openData.PingInterval

		go func() {
			ticker := time.NewTicker(time.Millisecond * time.Duration(pingInterval))
			defer ticker.Stop()
			for {
				select {
				case <-sio.ctx.Done():
					return
				case <-ticker.C:
					if sio.LogLevel <= LogDebug {
						sio.logger.Printf("PING")
					}
					if err := sio.Ping(sio.ctx); err != nil {
						if sio.LogLevel <= LogError {
							sio.logger.Printf("failed to send ping. %v", err)
						}
						return
					}
				}
			}
		}()
	}

	return nil
}

func (sio *SocketIO) Ping(ctx context.Context) error {
	return sio.Send(ctx, PacketPING, nil, "", nil)
}

func (sio *SocketIO) SendMessage(ctx context.Context, a ActionType, eventName string, payload interface{}) error {
	return sio.Send(ctx, PacketMESSAGE, &a, eventName, payload)
}

func (sio *SocketIO) Send(ctx context.Context, t PacketType, a *ActionType, eventName string, payload interface{}) error {
	sio.mtxTx.Lock()
	defer sio.mtxTx.Unlock()
	sio.bufTx.Reset()
	sio.bufTx.Grow(2)

	if _, err := fmt.Fprintf(sio.bufTx, "%d", int64(t)); err != nil {
		return err
	}
	if a != nil {
		if _, err := fmt.Fprintf(sio.bufTx, "%d", int64(*a)); err != nil {
			return err
		}
	}
	if eventName != "" && payload != nil {
		sio.bufTx.WriteString(`[`)
		enc := json.NewEncoder(sio.bufTx)
		enc.Encode(eventName)
		sio.bufTx.WriteString(`,`)
		enc.Encode(payload)
		sio.bufTx.WriteString(`]`)
	}
	return sio.ws.WriteMessage(websocket.TextMessage, sio.bufTx.Bytes())
}

func (sio *SocketIO) Receive(ctx context.Context) (*Packet, error) {
	sio.mtxRx.Lock()
	defer sio.mtxRx.Unlock()

	sio.bufRx.Reset()

	t, rdr, err := sio.ws.NextReader()
	if err != nil {
		return nil, err
	}
	if t != websocket.TextMessage {
		return nil, fmt.Errorf("only Text is currently supported")
	}
	n, err := sio.bufRx.ReadFrom(rdr)
	if err != nil {
		return nil, fmt.Errorf("failed to read message. %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("empty packet")
	}
	p, err := parsePacket(sio.bufRx.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to parse packet. %w", err)
	}
	return p, nil
}

func (sio *SocketIO) Close() error {
	sio.cancelFn()
	if sio.ws != nil {
		sio.ws.Close()
		sio.ws = nil
	}
	return nil
}
