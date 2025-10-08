// Package vpn provides VPN management services for the application.
package vpn

import (
	"time"
)

type VpnClient struct {
	CommonName     string    `json:"common_name"`
	RealAddress    string    `json:"real_address"`
	VirtualAddress string    `json:"virtual_address"`
	BytesReceived  int64     `json:"bytes_received"`
	BytesSent      int64     `json:"bytes_sent"`
	ConnectedSince time.Time `json:"connected_since"`
}

type Status struct {
	Online  bool
	Clients []*VpnClient
}

type TraficStatus struct {
	ClientName    string
	BytesSend     int64
	BytesReceived int64
}

type VpnService interface {
	GetStatus(host string) (*Status, error)
	CreateUser(host string) (config string, filename string, err error)
	DeleteUser(host string, username string) error
	GetRate(host string, address string, sec int) (int64, int64, error)
	KickUser(host string, username string) error
	GetAllRate(host string, sec int) ([]*TraficStatus, error)
}
