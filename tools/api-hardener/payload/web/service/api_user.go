//go:build toolsignore
// +build toolsignore

package service

import (
	"errors"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v2/database"
	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/logger"
	"github.com/mhsanaei/3x-ui/v2/util/crypto"
	"github.com/mhsanaei/3x-ui/v2/util/random"

	"gorm.io/gorm"
)

const (
	apiTokenLength       = 48
	apiTokenPrefixLength = 8
)

// ErrInvalidAPIToken is returned when a token cannot be matched to an enabled API user.
var ErrInvalidAPIToken = errors.New("invalid api token")

// APIUserService manages API-only users, their tokens, and rate limits.
type APIUserService struct {
	settingService SettingService
}

// CreateUser provisions a new API user with a freshly generated token.
// The plaintext token is returned only once to the caller.
func (s *APIUserService) CreateUser(name string, rateLimitPerMinute int) (*model.APIUser, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", errors.New("name can not be empty")
	}
	if rateLimitPerMinute < 0 {
		rateLimitPerMinute = 0
	}

	token, prefix, hash, err := s.generateToken()
	if err != nil {
		return nil, "", err
	}

	db := database.GetDB()
	apiUser := &model.APIUser{
		Name:               name,
		TokenPrefix:        prefix,
		TokenHash:          hash,
		RateLimitPerMinute: rateLimitPerMinute,
		Enabled:            true,
	}
	if err := db.Create(apiUser).Error; err != nil {
		return nil, "", err
	}
	return apiUser, token, nil
}

// ListUsers returns all non-deleted API users ordered by creation time.
func (s *APIUserService) ListUsers() ([]model.APIUser, error) {
	db := database.GetDB()
	var apiUsers []model.APIUser
	err := db.Model(&model.APIUser{}).
		Order("id asc").
		Find(&apiUsers).
		Error
	return apiUsers, err
}

// GetUser fetches a single API user by ID.
func (s *APIUserService) GetUser(id int) (*model.APIUser, error) {
	db := database.GetDB()
	apiUser := &model.APIUser{}
	err := db.Model(&model.APIUser{}).
		First(apiUser, id).
		Error
	if err != nil {
		return nil, err
	}
	return apiUser, nil
}

// SetEnabled toggles an API user's enabled state.
func (s *APIUserService) SetEnabled(id int, enabled bool) error {
	db := database.GetDB()
	return db.Model(&model.APIUser{}).
		Where("id = ?", id).
		Update("enabled", enabled).
		Error
}

// DeleteUser permanently removes an API user and its token.
func (s *APIUserService) DeleteUser(id int) error {
	db := database.GetDB()
	return db.Delete(&model.APIUser{}, id).Error
}

// UpdateRateLimit sets a per-minute rate limit for the given API user.
func (s *APIUserService) UpdateRateLimit(id int, rateLimitPerMinute int) error {
	if rateLimitPerMinute < 0 {
		rateLimitPerMinute = 0
	}
	db := database.GetDB()
	return db.Model(&model.APIUser{}).
		Where("id = ?", id).
		Update("rate_limit_per_minute", rateLimitPerMinute).
		Error
}

// RotateToken replaces the current token with a new secret and returns the plaintext token.
func (s *APIUserService) RotateToken(id int) (string, error) {
	token, prefix, hash, err := s.generateToken()
	if err != nil {
		return "", err
	}

	db := database.GetDB()
	err = db.Model(&model.APIUser{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"token_prefix": prefix,
			"token_hash":   hash,
		}).Error
	if err != nil {
		return "", err
	}
	return token, nil
}

// VerifyToken validates a presented token and returns the matching enabled API user.
func (s *APIUserService) VerifyToken(token string) (*model.APIUser, error) {
	token = strings.TrimSpace(token)
	if len(token) < apiTokenPrefixLength {
		return nil, ErrInvalidAPIToken
	}
	prefix := token[:apiTokenPrefixLength]

	db := database.GetDB()
	apiUser := &model.APIUser{}
	err := db.Model(&model.APIUser{}).
		Where("token_prefix = ? AND enabled = ?", prefix, true).
		First(apiUser).
		Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warning("api token lookup failed:", err)
		}
		return nil, ErrInvalidAPIToken
	}

	if !crypto.CheckPasswordHash(apiUser.TokenHash, token) {
		return nil, ErrInvalidAPIToken
	}

	now := time.Now()
	_ = db.Model(&model.APIUser{}).
		Where("id = ?", apiUser.Id).
		Update("last_used_at", now).
		Error

	return apiUser, nil
}

// EffectiveRateLimit returns the concrete per-minute limit using system defaults when unset.
func (s *APIUserService) EffectiveRateLimit(apiUser *model.APIUser) int {
	if apiUser == nil {
		return 0
	}
	if apiUser.RateLimitPerMinute > 0 {
		return apiUser.RateLimitPerMinute
	}
	limit, err := s.settingService.GetAPIDefaultRateLimit()
	if err != nil || limit < 0 {
		return 0
	}
	return limit
}

// Count returns total API users (excluding soft-deleted rows).
func (s *APIUserService) Count() (int64, error) {
	db := database.GetDB()
	var count int64
	err := db.Model(&model.APIUser{}).Count(&count).Error
	return count, err
}

func (s *APIUserService) generateToken() (token string, prefix string, hash string, err error) {
	token = random.Seq(apiTokenLength)
	prefix = token[:apiTokenPrefixLength]
	hash, err = crypto.HashPasswordAsBcrypt(token)
	return
}
