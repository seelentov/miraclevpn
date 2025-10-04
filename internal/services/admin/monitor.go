// Package admin provides services for administrating
package admin

import (
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/vpn"
	"miraclevpn/pkg/ovpn"
	"sort"
	"sync"
)

type MonitorService struct {
	vpnSrv     *ovpn.Client
	usRepo     *repo.UserServerRepository
	serverRepo *repo.ServerRepository
}

func NewMonitorService(
	vpnSrv *ovpn.Client,
	usRepo *repo.UserServerRepository,
	serverRepo *repo.ServerRepository,
) *MonitorService {
	return &MonitorService{
		vpnSrv,
		usRepo,
		serverRepo,
	}
}

type ClientData struct {
	Client *vpn.VpnClient
	UserID string
}

func (s *MonitorService) GetStatus(host string, getClients bool) (clients []*ClientData, count int, bytesReceived int64, bytesSent int64, rate int64, err error) {
	status, err := s.vpnSrv.GetStatus(host)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}

	if getClients {
		clients = make([]*ClientData, 0)
	}

	count = 0
	bytesReceived = 0
	bytesSent = 0
	rate = 0

	for _, client := range status.Clients {
		count++

		us, _ := s.usRepo.FindByConfigFile(client.CommonName, true)

		if getClients {
			client := &ClientData{
				Client: client,
			}

			if us != nil {
				client.UserID = us.UserID
			}

			clients = append(clients, client)
		}
		bytesReceived += client.BytesReceived
		bytesSent += client.BytesSent
		rate += client.Rate
	}

	if getClients {
		sort.Slice(clients, func(i, j int) bool {
			return clients[i].Client.BytesReceived > clients[j].Client.BytesReceived
		})
	}

	return clients, count, bytesReceived, bytesSent, rate, nil
}

type HostData struct {
	Host          string
	Count         int
	BytesReceived int64
	BytesSent     int64
	Rate          int64
}

func (s *MonitorService) GetHosts() ([]*HostData, error) {
	srvs, err := s.serverRepo.FindAll()
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	m := sync.Mutex{}
	res := make([]*HostData, 0)

	addRes := func(data *HostData) {
		m.Lock()
		defer m.Unlock()
		res = append(res, data)
	}

	for _, srv := range srvs {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()

			_, count, bytesReceived, bytesSent, rate, _ := s.GetStatus(host, false)

			data := &HostData{
				Host:          host,
				Count:         count,
				BytesReceived: bytesReceived,
				BytesSent:     bytesSent,
				Rate:          rate,
			}

			addRes(data)
		}(srv.Host)
	}
	wg.Wait()

	return res, nil
}
