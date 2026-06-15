package billing

import (
	"testing"

	"bitapi/backend/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestChargeWritesUsageAndUpdatesAPIKeyWindows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.APIKey{}, &models.UsageLog{}, &models.BillingDedup{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	user := models.User{
		Email:         "user@example.test",
		PasswordHash:  "hash",
		DisplayName:   "User",
		Role:          models.RoleUser,
		Status:        models.StatusActive,
		BalanceMicros: 10_000,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	key := models.APIKey{
		UserID:    user.ID,
		Name:      "test",
		KeyHash:   "hash",
		KeyPrefix: "bak_test",
		Status:    models.StatusActive,
	}
	if err := db.Create(&key).Error; err != nil {
		t.Fatalf("create api key: %v", err)
	}

	result, err := New(db).Charge(ChargeInput{
		RequestID:        "req-1",
		UserID:           user.ID,
		APIKeyID:         key.ID,
		Platform:         models.PlatformOpenAI,
		PromptTokens:     6,
		CompletionTokens: 4,
		StatusCode:       200,
	})
	if err != nil {
		t.Fatalf("charge: %v", err)
	}
	if result.Usage.ID == 0 {
		t.Fatal("expected usage log to be created")
	}

	var updated models.APIKey
	if err := db.First(&updated, key.ID).Error; err != nil {
		t.Fatalf("load api key: %v", err)
	}
	if updated.QuotaUsedMicros != 100 || updated.Usage5hMicros != 100 || updated.Usage1dMicros != 100 || updated.Usage7dMicros != 100 {
		t.Fatalf("unexpected usage counters: quota=%d 5h=%d 1d=%d 7d=%d", updated.QuotaUsedMicros, updated.Usage5hMicros, updated.Usage1dMicros, updated.Usage7dMicros)
	}
}
