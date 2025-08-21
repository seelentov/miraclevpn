package vpn

import "time"

type VpnClient struct {
	CommonName     string
	RealAddress    string
	BytesReceived  int64
	BytesSent      int64
	ConnectedSince time.Time
}

type Status struct {
	Online  bool
	Clients []*VpnClient
}

type VpnService interface {
	GetStatus(host string) (*Status, error)
	CreateUser(host string, username string) (string, error)
	DeleteUser(host string, username string) error
}
