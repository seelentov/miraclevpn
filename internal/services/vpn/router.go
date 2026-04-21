package vpn

import (
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
)

// VpnRouter implements VpnService and delegates to the appropriate client
// based on server type (ovpn or amneziawg). It uses FindByHost to look up
// server type, defaulting to the OVPN client for unknown or empty types.
type VpnRouter struct {
	ovpnClient VpnService
	awgClient  VpnService
	serverRepo *repo.ServerRepository
}

func NewVpnRouter(ovpnClient, awgClient VpnService, serverRepo *repo.ServerRepository) *VpnRouter {
	return &VpnRouter{
		ovpnClient: ovpnClient,
		awgClient:  awgClient,
		serverRepo: serverRepo,
	}
}

func (r *VpnRouter) clientForHost(host string) VpnService {
	srv, err := r.serverRepo.FindByHost(host)
	if err == nil && srv.Type == models.ServerTypeAmneziaWG {
		return r.awgClient
	}
	return r.ovpnClient
}

func (r *VpnRouter) GetStatus(host string) (*Status, error) {
	return r.clientForHost(host).GetStatus(host)
}

func (r *VpnRouter) CreateUser(host string) (string, string, error) {
	return r.clientForHost(host).CreateUser(host)
}

func (r *VpnRouter) DeleteUser(host string, username string) error {
	return r.clientForHost(host).DeleteUser(host, username)
}

func (r *VpnRouter) GetRate(host string, address string, sec int) (int64, int64, error) {
	return r.clientForHost(host).GetRate(host, address, sec)
}

func (r *VpnRouter) KickUser(host string, username string) error {
	return r.clientForHost(host).KickUser(host, username)
}

func (r *VpnRouter) GetAllRate(host string, sec int) ([]*TraficStatus, error) {
	return r.clientForHost(host).GetAllRate(host, sec)
}

func (r *VpnRouter) CheckAvailable(host string) (bool, error) {
	return r.clientForHost(host).CheckAvailable(host)
}
