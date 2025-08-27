package repo

import (
	"errors"
	"miraclevpn/internal/models"

	"gorm.io/gorm"
)

type KeyValueRepository struct {
	db *gorm.DB
}

func NewKeyValueRepository(db *gorm.DB) *KeyValueRepository {
	return &KeyValueRepository{
		db: db,
	}
}

func (r *KeyValueRepository) Get(key string) (string, error) {
	var kv models.KeyValue
	if err := r.db.Where("key = ?", key).First(&kv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return kv.Value, nil
}

func (r *KeyValueRepository) GetLike(keyLike string) (map[string]string, error) {
	var kvs []models.KeyValue
	result := make(map[string]string)

	if err := r.db.Where("key LIKE ?", keyLike).Find(&kvs).Error; err != nil {
		return nil, err
	}

	for _, kv := range kvs {
		result[kv.Key] = kv.Value
	}

	return result, nil
}
