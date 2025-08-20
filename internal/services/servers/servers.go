package servers

import (
	"errors"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/vpn"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("server not found")
)

type ServersService struct {
	ursSrvRepo *repo.UserServerRepository
	srvRepo    *repo.ServerRepository
	ursRepo    *repo.UserRepository
	VpnService vpn.VpnService
	logger     *zap.Logger
}

func NewServersService(ursSrvRepo *repo.UserServerRepository, srvRepo *repo.ServerRepository, ursRepo *repo.UserRepository, vpnService vpn.VpnService, logger *zap.Logger) *ServersService {
	return &ServersService{
		ursSrvRepo,
		srvRepo,
		ursRepo,
		vpnService,
		logger,
	}
}

func (s *ServersService) GetAllServers() ([]*models.Server, error) {
	s.logger.Debug("getting all servers")
	servers, err := s.srvRepo.FindAll()
	if err != nil {
		s.logger.Error("failed to get all servers", zap.Error(err))
		return nil, err
	}
	s.logger.Debug("all servers fetched", zap.Int("count", len(servers)))
	return servers, nil
}

func (s *ServersService) GetServersByRegion(region string) ([]*models.Server, error) {
	s.logger.Debug("getting servers by region", zap.String("region", region))
	servers, err := s.srvRepo.FindByRegion(region)
	if err != nil {
		s.logger.Error("failed to get servers by region", zap.String("region", region), zap.Error(err))
		return nil, err
	}
	s.logger.Debug("servers by region fetched", zap.String("region", region), zap.Int("count", len(servers)))
	return servers, nil
}

func (s *ServersService) GetServerByID(id int64) (*models.Server, error) {
	s.logger.Debug("getting server by id", zap.Int64("server_id", id))
	server, err := s.srvRepo.FindByID(id)
	if err != nil {
		s.logger.Error("failed to get server by id", zap.Int64("server_id", id), zap.Error(err))
		return nil, err
	}
	s.logger.Debug("server fetched", zap.Int64("server_id", id))
	return server, nil
}

func (s *ServersService) GetConfig(userID int64, serverID int64) (string, error) {
	s.logger.Debug("getting config", zap.Int64("user_id", userID), zap.Int64("server_id", serverID))
	us, err := s.ursSrvRepo.FindByUserIDServerID(userID, serverID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("failed to find user-server config", zap.Int64("user_id", userID), zap.Int64("server_id", serverID), zap.Error(err))
		return "", err
	}
	if us != nil {
		s.logger.Debug("config found for user-server", zap.Int64("user_id", userID), zap.Int64("server_id", serverID))
		return us.Config, nil
	}

	srv, err := s.srvRepo.FindByID(serverID)
	if err != nil {
		s.logger.Error("failed to find server by id", zap.Int64("server_id", serverID), zap.Error(err))
		return "", err
	}

	usr, err := s.ursRepo.FindByID(userID)
	if err != nil {
		s.logger.Error("failed to find user by id", zap.Int64("user_id", userID), zap.Error(err))
		return "", err
	}

	s.logger.Debug("creating VPN user", zap.String("host", srv.Host), zap.String("username", usr.Username))
	config, err := s.VpnService.CreateUser(srv.Host, usr.Username)
	if err != nil {
		s.logger.Error("failed to create VPN user", zap.String("host", srv.Host), zap.String("username", usr.Username), zap.Error(err))
		return "", err
	}

	if err := s.ursSrvRepo.CreateOrUpdate(userID, serverID, config); err != nil {
		s.logger.Error("failed to save user-server config", zap.Int64("user_id", userID), zap.Int64("server_id", serverID), zap.Error(err))
		return "", err
	}

	s.logger.Debug("vpn config created and saved", zap.Int64("user_id", userID), zap.Int64("server_id", serverID))
	return config, nil
}

func (s *ServersService) GetRegions() ([]string, error) {
	s.logger.Debug("getting all regions")
	regions, err := s.srvRepo.FindAllRegions()
	if err != nil {
		s.logger.Error("failed to get all regions", zap.Error(err))
		return nil, err
	}
	s.logger.Debug("all regions fetched", zap.Int("count", len(regions)))
	return regions, nil
}
