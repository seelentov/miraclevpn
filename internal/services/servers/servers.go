// Package servers provides server management services for the application.
package servers

import (
	"errors"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/vpn"
	"time"

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
	vpnService vpn.VpnService
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

func (s *ServersService) GetConfig(userID string, serverID int64) (string, error) {
	s.logger.Debug("getting config", zap.String("user_id", userID), zap.Int64("server_id", serverID))
	us, err := s.ursSrvRepo.FindByUserIDServerID(userID, serverID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("failed to find user-server config", zap.String("user_id", userID), zap.Int64("server_id", serverID), zap.Error(err))
		return "", err
	}
	if us != nil {
		s.logger.Debug("config found for user-server", zap.String("user_id", userID), zap.Int64("server_id", serverID))
		return us.Config, nil
	}

	srv, err := s.srvRepo.FindByID(serverID)
	if err != nil {
		s.logger.Error("failed to find server by id", zap.Int64("server_id", serverID), zap.Error(err))
		return "", err
	}

	usr, err := s.ursRepo.FindByID(userID)
	if err != nil {
		s.logger.Error("failed to find user by id", zap.String("user_id", userID), zap.Error(err))
		return "", err
	}

	s.logger.Debug("creating VPN user", zap.String("host", srv.Host), zap.String("UID", usr.ID))
	config, username, err := s.vpnService.CreateUser(srv.Host)
	if err != nil {
		s.logger.Error("failed to create VPN user", zap.String("host", srv.Host), zap.String("UID", usr.ID), zap.Error(err))
		return "", err
	}

	if err := s.ursSrvRepo.CreateOrUpdate(userID, serverID, config, username); err != nil {
		s.logger.Error("failed to save user-server config", zap.String("user_id", userID), zap.Int64("server_id", serverID), zap.Error(err))
		return "", err
	}

	s.logger.Debug("vpn config created and saved", zap.String("user_id", userID), zap.Int64("server_id", serverID))
	return config, nil
}

func (s *ServersService) GetRegions() ([]*models.Region, error) {
	s.logger.Debug("getting all regions")
	regions, err := s.srvRepo.FindAllRegions()
	if err != nil {
		s.logger.Error("failed to get all regions", zap.Error(err))
		return nil, err
	}
	s.logger.Debug("all regions fetched", zap.Int("count", len(regions)))
	return regions, nil
}

func (s *ServersService) GetServerStatus(serverID int64) (server *models.Server, currentUsersCount int, err error) {
	srv, err := s.srvRepo.FindByID(serverID)
	if err != nil {
		s.logger.Error("failed to find server by id", zap.Int64("server_id", serverID), zap.Error(err))
		return nil, 0, err
	}

	stat, err := s.vpnService.GetStatus(srv.Host)
	if err != nil {
		return nil, 0, err
	}

	return srv, len(stat.Clients), nil
}

func (s *ServersService) UpdateExpired(expiration time.Duration) error {
	s.logger.Info("starting update of expired servers", zap.Duration("expiration", expiration))

	uss, err := s.ursSrvRepo.FindExpired(expiration)
	if err != nil {
		s.logger.Error("failed to find expired servers", zap.Duration("expiration", expiration), zap.Error(err))
		return err
	}

	s.logger.Info("found expired server-user associations", zap.Int("count", len(uss)))

	for i, us := range uss {
		s.logger.Debug("processing association",
			zap.Int("index", i),
			zap.Int64("serverID", us.ServerID),
			zap.String("userID", us.UserID))

		srv, err := s.srvRepo.FindByID(us.ServerID)
		if err != nil {
			s.logger.Error("failed to find server by ID",
				zap.Int64("serverID", us.ServerID),
				zap.Error(err))
			return err
		}

		usr, err := s.ursRepo.FindByID(us.UserID)
		if err != nil {
			s.logger.Error("failed to find user by ID",
				zap.String("userID", us.UserID),
				zap.Error(err))
			return err
		}

		s.logger.Info("deleting expired VPN user",
			zap.String("host", srv.Host),
			zap.String("userID", usr.ID))

		if err := s.vpnService.DeleteUser(srv.Host, us.ConfigFile); err != nil {
			s.logger.Error("failed to delete VPN user",
				zap.String("host", srv.Host),
				zap.String("userID", usr.ID),
				zap.Error(err))
			return err
		}

		s.logger.Info("creating new VPN user",
			zap.String("host", srv.Host),
			zap.String("userID", usr.ID))

		config, fileName, err := s.vpnService.CreateUser(srv.Host)
		if err != nil {
			s.logger.Error("failed to create VPN user",
				zap.String("host", srv.Host),
				zap.String("userID", usr.ID),
				zap.Error(err))
			return err
		}

		s.logger.Info("updating user-server association",
			zap.String("userID", usr.ID),
			zap.Int64("serverID", srv.ID))

		if err := s.ursSrvRepo.CreateOrUpdate(usr.ID, srv.ID, config, fileName); err != nil {
			s.logger.Error("failed to update user-server association",
				zap.String("userID", usr.ID),
				zap.Int64("serverID", srv.ID),
				zap.Error(err))
			return err
		}

		s.logger.Info("successfully updated expired association",
			zap.String("userID", usr.ID),
			zap.Int64("serverID", srv.ID))
	}

	s.logger.Info("completed update of expired servers",
		zap.Int("processed_count", len(uss)))
	return nil
}

func (s *ServersService) RemoveExpiredByUser() error {
	return s.ursSrvRepo.RemoveExpiredByUser()
}

func (s *ServersService) FindPreview() ([]*models.Server, error) {
	return s.srvRepo.FindPreview()
}

func (s *ServersService) SendRequest(region string, userID string) error {
	return s.srvRepo.SendRequest(region, userID)
}
