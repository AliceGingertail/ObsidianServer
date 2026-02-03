package models

type Protocol string

const (
	ProtocolWireGuard Protocol = "wireguard"
	ProtocolOpenVPN   Protocol = "openvpn"
)

func (p Protocol) IsValid() bool {
	switch p {
	case ProtocolWireGuard, ProtocolOpenVPN:
		return true
	default:
		return false
	}
}

func (p Protocol) String() string {
	return string(p)
}
