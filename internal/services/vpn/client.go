// Package vpn provides VPN management services for the application.
package vpn

import "time"

type VpnClient struct {
	CommonName     string
	RealAddress    string
	VirtualAddress string
	BytesReceived  int64
	BytesSent      int64
	Rate           int64
	ConnectedSince time.Time
}

type Status struct {
	Online  bool
	Clients []*VpnClient
}

type TraficStatus struct {
	Rates                 []*TraficRate
	TotalSendRate         int64 //Total send rate last 10 sec
	TotalReceiveRate      int64 //Total receive rate last 10 sec
	PeakSendRate          int64
	PeakReceiveRate       int64
	PeakRate              int64 // Total
	CumulativeSendRate    int64
	CumulativeReceiveRate int64
	CumulativeRate        int64 // Total
}

type TraficRate struct {
	VirtualAddress string // Хост вида 10.8.0.n
	Rate           int64  // last 10 sec
}

type VpnService interface {
	GetStatus(host string) (*Status, error)
	CreateUser(host string) (config string, filename string, err error)
	DeleteUser(host string, username string) error
}
