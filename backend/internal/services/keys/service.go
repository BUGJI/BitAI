package keys

import (
	"errors"
	"time"

	"bitapi/backend/internal/models"
	bcrypto "bitapi/backend/internal/pkg/crypto"
	"gorm.io/gorm"
)

var ErrKeyNotFound = errors.New("调用密钥不存在")

type Service struct {
	db *gorm.DB
}

type CreatedKey struct {
	Key    string        `json:"key"`
	APIKey models.APIKey `json:"api_key"`
}

func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(userID uint, name string, groupID *uint, quota int64, expiresAt *time.Time) (CreatedKey, error) {
	plain, err := bcrypto.RandomToken("bak", 36)
	if err != nil {
		return CreatedKey{}, err
	}
	apiKey := models.APIKey{
		UserID:           userID,
		GroupID:          groupID,
		Name:             name,
		Key:              plain,
		KeyHash:          bcrypto.SHA256Hex(plain),
		KeyPrefix:        bcrypto.KeyPrefix(plain),
		Status:           models.StatusActive,
		QuotaLimitMicros: quota,
		ExpiresAt:        expiresAt,
	}
	if err := s.db.Create(&apiKey).Error; err != nil {
		return CreatedKey{}, err
	}
	return CreatedKey{Key: plain, APIKey: apiKey}, nil
}

func (s *Service) List(userID uint) ([]models.APIKey, error) {
	var keys []models.APIKey
	err := s.db.Preload("Group").Where("user_id = ?", userID).Order("id desc").Find(&keys).Error
	return keys, err
}

func (s *Service) Delete(userID, id uint) error {
	result := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.APIKey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrKeyNotFound
	}
	return nil
}
