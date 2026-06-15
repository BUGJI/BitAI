package payments

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"bitapi/backend/internal/models"
	bcrypto "bitapi/backend/internal/pkg/crypto"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrRedeemInvalid = errors.New("兑换码无效")
	ErrRedeemUsed    = errors.New("兑换码已使用")
	ErrOrderSettled  = errors.New("订单已处理，不能重复操作")
)

type Service struct {
	db *gorm.DB
}

type CreatedRedeemCode struct {
	Code string            `json:"code"`
	Item models.RedeemCode `json:"item"`
}

func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateOrder(userID uint, amountMicros int64, provider string) (models.PaymentOrder, error) {
	if provider == "" {
		provider = "manual"
	}
	order := models.PaymentOrder{
		UserID:       userID,
		OrderNo:      "bo_" + uuid.NewString(),
		AmountMicros: amountMicros,
		Status:       models.OrderStatusPending,
		Provider:     provider,
	}
	err := s.db.Create(&order).Error
	return order, err
}

func (s *Service) MarkOrderPaid(orderID uint) (models.PaymentOrder, error) {
	var order models.PaymentOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, orderID).Error; err != nil {
			return err
		}
		if order.Status != models.OrderStatusPending {
			return ErrOrderSettled
		}
		now := time.Now()
		order.Status = models.OrderStatusPaid
		order.PaidAt = &now
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("id = ?", order.UserID).Updates(map[string]any{
			"balance_micros":         gorm.Expr("balance_micros + ?", order.AmountMicros),
			"total_recharged_micros": gorm.Expr("total_recharged_micros + ?", order.AmountMicros),
		}).Error
	})
	return order, err
}

func (s *Service) RejectOrder(orderID uint) (models.PaymentOrder, error) {
	var order models.PaymentOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, orderID).Error; err != nil {
			return err
		}
		if order.Status != models.OrderStatusPending {
			return ErrOrderSettled
		}
		order.Status = models.OrderStatusRejected
		order.PaidAt = nil
		return tx.Save(&order).Error
	})
	return order, err
}

func (s *Service) CreateRedeemCode(createdBy uint, amountMicros int64, maxUses int, expiresAt *time.Time) (CreatedRedeemCode, error) {
	if maxUses <= 0 {
		maxUses = 1
	}
	code, err := randomRedeemCode(16)
	if err != nil {
		return CreatedRedeemCode{}, err
	}
	item := models.RedeemCode{
		Code:         code,
		CodeHash:     bcrypto.SHA256Hex(code),
		CodePrefix:   code,
		AmountMicros: amountMicros,
		Status:       models.RedeemStatusActive,
		MaxUses:      maxUses,
		ExpiresAt:    expiresAt,
		CreatedByID:  createdBy,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return CreatedRedeemCode{}, err
	}
	return CreatedRedeemCode{Code: code, Item: item}, nil
}

func (s *Service) DisableRedeemCode(id uint) (models.RedeemCode, error) {
	var item models.RedeemCode
	if err := s.db.First(&item, id).Error; err != nil {
		return item, err
	}
	item.Status = models.RedeemStatusDisabled
	return item, s.db.Save(&item).Error
}

func (s *Service) EnableRedeemCode(id uint) (models.RedeemCode, error) {
	var item models.RedeemCode
	if err := s.db.First(&item, id).Error; err != nil {
		return item, err
	}
	item.Status = models.RedeemStatusActive
	return item, s.db.Save(&item).Error
}

func (s *Service) DeleteRedeemCode(id uint) error {
	return s.db.Delete(&models.RedeemCode{}, id).Error
}

func (s *Service) Redeem(userID uint, code string) (models.RedeemCodeUsage, error) {
	codes := redeemCodeCandidates(code)
	if len(codes) == 0 {
		return models.RedeemCodeUsage{}, ErrRedeemInvalid
	}
	hashes := make([]string, 0, len(codes))
	for _, value := range codes {
		hashes = append(hashes, bcrypto.SHA256Hex(value))
	}
	var usage models.RedeemCodeUsage
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var item models.RedeemCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code_hash IN ?", hashes).First(&item).Error; err != nil {
			return ErrRedeemInvalid
		}
		if item.Status != models.RedeemStatusActive || item.UsedCount >= item.MaxUses {
			return ErrRedeemInvalid
		}
		if item.ExpiresAt != nil && time.Now().After(*item.ExpiresAt) {
			return ErrRedeemInvalid
		}
		var existing models.RedeemCodeUsage
		if err := tx.Where("redeem_code_id = ? AND user_id = ?", item.ID, userID).First(&existing).Error; err == nil {
			return ErrRedeemUsed
		}
		usage = models.RedeemCodeUsage{
			RedeemCodeID: item.ID,
			UserID:       userID,
			AmountMicros: item.AmountMicros,
			CreatedAt:    time.Now(),
		}
		if err := tx.Create(&usage).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.RedeemCode{}).Where("id = ?", item.ID).UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error; err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
			"balance_micros":         gorm.Expr("balance_micros + ?", item.AmountMicros),
			"total_recharged_micros": gorm.Expr("total_recharged_micros + ?", item.AmountMicros),
		}).Error
	})
	return usage, err
}

func randomRedeemCode(length int) (string, error) {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var builder strings.Builder
	builder.Grow(length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		builder.WriteByte(alphabet[n.Int64()])
	}
	return builder.String(), nil
}

func redeemCodeCandidates(code string) []string {
	raw := strings.TrimSpace(code)
	if raw == "" {
		return nil
	}
	upper := strings.ToUpper(raw)
	if upper == raw {
		return []string{raw}
	}
	return []string{raw, upper}
}
