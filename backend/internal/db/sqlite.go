package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bitapi/backend/internal/config"
	"bitapi/backend/internal/models"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	if err := ensureDataDir(cfg.DatabaseDSN); err != nil {
		return nil, err
	}
	conn, err := gorm.Open(sqlite.Open(cfg.DatabaseDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := conn.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	if err := conn.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return nil, err
	}
	if err := conn.Exec("PRAGMA synchronous=NORMAL").Error; err != nil {
		return nil, err
	}
	if err := conn.Exec("PRAGMA temp_store=MEMORY").Error; err != nil {
		return nil, err
	}
	return conn, nil
}

func AutoMigrate(conn *gorm.DB) error {
	return conn.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
		&models.APIKey{},
		&models.Group{},
		&models.UpstreamAccount{},
		&models.GroupAccount{},
		&models.UsageLog{},
		&models.BillingDedup{},
		&models.Setting{},
		&models.VerificationChallenge{},
		&models.PaymentOrder{},
		&models.RedeemCode{},
		&models.RedeemCodeUsage{},
	)
}

func Seed(conn *gorm.DB, cfg config.Config) error {
	var count int64
	if err := conn.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(cfg.BootstrapPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user := models.User{
			Email:         cfg.BootstrapEmail,
			PasswordHash:  string(hash),
			DisplayName:   cfg.BootstrapName,
			Role:          models.RoleOwner,
			Status:        models.StatusActive,
			TokenVersion:  1,
			BalanceMicros: cfg.DefaultUserBalance,
		}
		if err := conn.Create(&user).Error; err != nil {
			return err
		}
	}

	if err := seedSetting(conn, "site.name", cfg.AppName, true); err != nil {
		return err
	}
	if err := seedSetting(conn, "signup.enabled", "true", true); err != nil {
		return err
	}
	smtpDefaults := []struct {
		key      string
		value    string
		isPublic bool
	}{
		{"smtp.enabled", "false", false},
		{"smtp.host", "", false},
		{"smtp.port", "587", false},
		{"smtp.username", "", false},
		{"smtp.password", "", false},
		{"smtp.from_email", "", false},
		{"smtp.from_name", "BitAPI", false},
		{"smtp.encryption", "starttls", false},
	}
	for _, item := range smtpDefaults {
		if err := seedSetting(conn, item.key, item.value, item.isPublic); err != nil {
			return err
		}
	}
	return seedDefaultGroup(conn)
}

func ensureDataDir(dsn string) error {
	if dsn == "" || dsn == ":memory:" {
		return nil
	}
	path := strings.TrimPrefix(dsn, "file:")
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	if path == "" || path == ":memory:" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}
	return nil
}

func seedSetting(conn *gorm.DB, key, value string, isPublic bool) error {
	var setting models.Setting
	err := conn.Where("key = ?", key).First(&setting).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return conn.Create(&models.Setting{Key: key, Value: value, IsPublic: isPublic}).Error
}

func seedDefaultGroup(conn *gorm.DB) error {
	var count int64
	if err := conn.Model(&models.Group{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return conn.Create(&models.Group{
		Name:              "默认兼容分组",
		Description:       "默认的兼容模型接口余额扣费分组。",
		Platform:          models.PlatformOpenAI,
		Mode:              models.GroupModeBalance,
		Status:            models.StatusActive,
		RateMultiplierPPM: 1000000,
		ModelMappingJSON:  "{}",
		ModelListJSON:     `["gpt-4o-mini","gpt-4.1-mini"]`,
		FeaturesJSON:      `{"chat":true,"stream":true}`,
		SortOrder:         1,
	}).Error
}
