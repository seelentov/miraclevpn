// Package vpn provides VPN management services for the application.
package vpn

import "time"

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
	TotalSendRate         int64 //Total send rate last 10 sec
	TotalReceiveRate      int64 //Total receive rate last 10 sec
	PeakSendRate          int64
	PeakReceiveRate       int64
	PeakRate              int64 // Total
	CumulativeSendRate    int64
	CumulativeReceiveRate int64
	CumulativeRate        int64 // Total
}

type VpnService interface {
	GetStatus(host string) (*Status, error)
	CreateUser(host string) (config string, filename string, err error)
	DeleteUser(host string, username string) error
}
