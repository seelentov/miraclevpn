// Package servers provides server management services for the application.
package servers

import (
	"errors"
	"log"
	"math"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/vpn"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("server not found")
)

type ServerWithStatus struct {
	Server    *models.Server
	Online    int
	Available bool
}

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

func (s *ServersService) GetAllServersWithStatus() ([]*ServerWithStatus, error) {
	srvs, err := s.GetAllServers()
	if err != nil {
		return nil, err
	}

	result := make([]*ServerWithStatus, len(srvs))
	var wg sync.WaitGroup

	for i, srv := range srvs {
		wg.Add(1)
		go func(idx int, server *models.Server) {
			defer wg.Done()

			var online int
			var available bool
			var inner sync.WaitGroup
			inner.Add(2)

			go func() {
				defer inner.Done()
				status, err := s.vpnService.GetStatus(server.Host)
				if err == nil {
					online = len(status.Clients)
				}
			}()

			go func() {
				defer inner.Done()
				avail, err := s.vpnService.CheckAvailable(server.Host)
				if err == nil {
					available = avail
				}
			}()

			inner.Wait()
			result[idx] = &ServerWithStatus{
				Server:    server,
				Online:    online,
				Available: available,
			}
		}(i, srv)
	}

	wg.Wait()
	return result, nil
}

// GetBestAvailableServer returns the available server with the most free slots.
// Unlimited servers (MaxUsers=0) are scored higher than capped ones.
func (s *ServersService) GetBestAvailableServer() (*ServerWithStatus, error) {
	srvList, err := s.GetAllServersWithStatus()
	if err != nil {
		return nil, err
	}

	score := func(sw *ServerWithStatus) int {
		if sw.Server.MaxUsers == 0 {
			return math.MaxInt / 2
		}
		return (sw.Server.MaxUsers - sw.Online) * 1000
	}

	var best *ServerWithStatus
	for _, sw := range srvList {
		if !sw.Available {
			continue
		}
		if sw.Server.MaxUsers > 0 && sw.Online >= sw.Server.MaxUsers {
			continue
		}
		if best == nil || score(sw) > score(best) {
			best = sw
		}
	}

	if best == nil {
		return nil, ErrNotFound
	}
	return best, nil
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

func (s *ServersService) GetBest() ([]*models.Server, error) {
	return s.srvRepo.FindBest()
}

func (s *ServersService) GetOnlyBest() (*models.Server, error) {
	return s.srvRepo.FindSuperBest()
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
	if us != nil && us.Config != "" {
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

	if err := s.ursSrvRepo.CreateOrUpdate(userID, serverID, config, username, nil); err != nil {
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
		s.logger.Error("failed to get status", zap.Int64("server_id", serverID), zap.Error(err))
		return nil, 0, err
	}

	return srv, len(stat.Clients), nil
}

func (s *ServersService) UpdateOnline() error {
	servers, err := s.GetAllServers()
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	for _, ser := range servers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			log.Println(ser)

			status, err := s.vpnService.GetStatus(ser.Host)

			if err != nil {
				s.logger.Error("failed to get status", zap.Int64("server_id", ser.ID), zap.Error(err))
				return
			}

			for _, client := range status.Clients {
				if err := s.ursSrvRepo.UpdateExpirationByConfigFile(client.CommonName, time.Now()); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					s.logger.Error("failed to update user_server date", zap.String("config", client.CommonName), zap.Error(err))
					return
				}
			}
		}()
	}
	wg.Wait()

	return nil
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

		if us.ConfigFileExpired != nil {
			if err := s.vpnService.DeleteUser(srv.Host, *(us.ConfigFileExpired)); err != nil {
				s.logger.Error("failed to delete VPN user",
					zap.String("host", srv.Host),
					zap.String("userID", usr.ID),
					zap.Error(err))
				return err
			}
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

		if err := s.ursSrvRepo.CreateOrUpdate(usr.ID, srv.ID, config, fileName, &(us.ConfigFile)); err != nil {
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
	expired, err := s.ursSrvRepo.FindExpiredByUser()
	if err != nil {
		return err
	}

	if err := s.ursSrvRepo.Delete(expired); err != nil {
		return err
	}

	for _, e := range expired {
		ser, err := s.srvRepo.FindByID(e.ServerID)
		if err != nil {
			return err
		}

		if err := s.deleteVPNuser(ser.Host, e.ConfigFile); err != nil {
			return err
		}

		if e.ConfigFileExpired != nil {
			if err := s.deleteVPNuser(ser.Host, *(e.ConfigFileExpired)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *ServersService) deleteVPNuser(host, user string) error {
	if err := s.vpnService.DeleteUser(host, user); err != nil {
		return err
	}

	if err := s.vpnService.KickUser(host, user); err != nil {
		return err
	}

	return nil
}

func (s *ServersService) FindPreview() ([]*models.Server, error) {
	return s.srvRepo.FindPreview()
}

func (s *ServersService) SendRequest(region string, userID string) error {
	return s.srvRepo.SendRequest(region, userID)
}

func (s *ServersService) GetRegionStatus(region string) (servers []*models.Server, currentUsersCount int, err error) {
	servers, err = s.srvRepo.FindByRegion(region)
	if err != nil {
		return nil, 0, err
	}

	currentUsersCount = 0

	wg := sync.WaitGroup{}

	for _, sr := range servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stat, err := s.vpnService.GetStatus(sr.Host)
			if err != nil {
				s.logger.Error("failed to get status", zap.Int64("server_id", sr.ID), zap.Error(err))
			}

			currentUsersCount += len(stat.Clients)
		}()

	}

	wg.Wait()

	return servers, currentUsersCount, nil

}
