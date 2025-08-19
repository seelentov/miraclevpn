package servers

import (
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/vpn"
)

type ServersService struct {
	ursSrvRepo *repo.UserServerRepository
	srvRepo    *repo.ServerRepository
	ursRepo    *repo.UserRepository
	VpnService vpn.VpnService
}

func NewServersService(ursSrvRepo *repo.UserServerRepository, srvRepo *repo.ServerRepository, ursRepo *repo.UserRepository, vpnService vpn.VpnService) *ServersService {
	return &ServersService{
		ursSrvRepo,
		srvRepo,
		ursRepo,
		vpnService,
	}
}

func (s *ServersService) GetAllServers() ([]*models.Server, error) {
	return s.srvRepo.FindAll()
}

func (s *ServersService) GetServersByRegion(region string) ([]*models.Server, error) {
	return s.srvRepo.FindByRegion(region)
}

func (s *ServersService) GetServerByID(id int64) (*models.Server, error) {
	return s.srvRepo.FindByID(id)
}

func (s *ServersService) GetConfig(userID int64, serverID int64) (string, error) {
	us, err := s.ursSrvRepo.FindByUserIDServerID(userID, serverID)
	if err != nil {
		return "", err
	}
	if us != nil {
		return us.Config, nil
	}

	srv, err := s.srvRepo.FindByID(serverID)
	if err != nil {
		return "", err
	}

	usr, err := s.ursRepo.FindByID(userID)
	if err != nil {
		return "", err
	}

	config, err := s.VpnService.CreateUser(srv.Host, usr.Phone)
	if err != nil {
		return "", err
	}

	if err := s.ursSrvRepo.CreateOrUpdate(userID, serverID, config); err != nil {
		return "", err
	}

	return config, nil
}
