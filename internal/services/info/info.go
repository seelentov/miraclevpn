// Package info provides information retrieval services.
package info

import (
	"errors"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"

	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("not found")
)

type InfoService struct {
	newsRepo *repo.NewsRepository
	infoRepo *repo.InfoRepository
	KeyValue *repo.KeyValueRepository
}

func NewInfoService(newsRepo *repo.NewsRepository, infoRepo *repo.InfoRepository, keyValue *repo.KeyValueRepository) *InfoService {
	return &InfoService{
		newsRepo: newsRepo,
		infoRepo: infoRepo,
		KeyValue: keyValue,
	}
}

func (r *InfoService) GetNews(userID string) ([]*models.News, error) {
	return r.newsRepo.FindUnread(userID)
}

func (r *InfoService) GetInfo(slug string) (*models.Info, error) {
	news, err := r.infoRepo.FindBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return news, nil
}

func (r *InfoService) GetTechWork() (bool, string, error) {
	techWork, err := r.KeyValue.Get("tech_work")
	if err != nil {
		return false, "", err
	}

	if techWork != "true" {
		return false, "", nil
	}

	techWorkText, err := r.KeyValue.Get("tech_work_text")
	if err != nil {
		return false, "", err
	}

	return true, techWorkText, nil
}

func (r *InfoService) GetInfos() ([]*models.Info, error) {
	return r.infoRepo.FindAll()
}

func (r *InfoService) GetSupport() (map[string]string, error) {
	return r.KeyValue.GetLike("%\\_support")
}
