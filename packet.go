package socketioclient

import "fmt"

type PacketType int
type ActionType int

const (
	PacketOPEN PacketType = iota
	PacketCLOSE
	PacketPING
	PacketPONG
	PacketMESSAGE
	PacketUPGRADE
	PacketNOOP
)

const (
	ActionCONNECT ActionType = iota
	ActionDISCONNECT
	ActionEVENT
	ActionACK
	ActionERROR
	ActionBINARY_EVENT
	ActionBINARY_ACK
)

type Packet struct {
	Type    PacketType
	Action  ActionType
	Payload []byte
}

func atoi(b byte) (int, error) {
	if b >= '0' && b <= '9' {
		return int(b - '0'), nil
	}
	return 0, fmt.Errorf("not a number %v", b)
}

func parsePacket(data []byte) (*Packet, error) {
	pt, err := atoi(data[0])
	if err != nil {
		return nil, err
	}
	p := Packet{
		Type: PacketType(pt),
	}
	data = data[1:]

	if p.Type == PacketMESSAGE {
		if len(data) < 1 {
			return nil, fmt.Errorf("expected action type, but is empty")
		}
		pa, err := atoi(data[0])
		if err != nil {
			return nil, err
		}
		p.Action = ActionType(pa)
		data = data[1:]
	}

	if len(data) != 0 {
		p.Payload = data
	} else {
		p.Payload = []byte(`{}`)
	}

	return &p, nil
}
