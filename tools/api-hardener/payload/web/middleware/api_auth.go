//go:build toolsignore
// +build toolsignore

package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/logger"
	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/mhsanaei/3x-ui/v2/web/session"
)

const (
	apiUserContextKey      = "api_user"
	apiVirtualUserIDOffset = 1_000_000
)

type apiRateLimiterStore struct {
	mu       sync.Mutex
	limiters map[int]*rate.Limiter
	limits   map[int]int
}

func newAPIRateLimiterStore() *apiRateLimiterStore {
	return &apiRateLimiterStore{
		limiters: make(map[int]*rate.Limiter),
		limits:   make(map[int]int),
	}
}

func (s *apiRateLimiterStore) allow(userID int, perMinute int) bool {
	if perMinute <= 0 {
		return true
	}

	limiter := s.getLimiter(userID, perMinute)
	if limiter == nil {
		return true
	}
	return limiter.Allow()
}

func (s *apiRateLimiterStore) getLimiter(userID int, perMinute int) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.limiters[userID]
	currentLimit := s.limits[userID]
	if ok && currentLimit == perMinute {
		return current
	}

	// Recreate limiter when the configured limit changes
	interval := time.Minute / time.Duration(perMinute)
	s.limiters[userID] = rate.NewLimiter(rate.Every(interval), perMinute)
	s.limits[userID] = perMinute
	return s.limiters[userID]
}

// NewAPIAuthMiddleware enforces API token authentication and per-user rate limits.
// It optionally allows existing session-based access if apiTokenOnly is disabled.
func NewAPIAuthMiddleware(apiUserService *service.APIUserService, settingService *service.SettingService) gin.HandlerFunc {
	limiterStore := newAPIRateLimiterStore()

	return func(c *gin.Context) {
		tokenOnly, err := settingService.GetAPITokenOnly()
		if err != nil {
			logger.Warning("read apiTokenOnly failed:", err)
		}

		token := extractAPIToken(c)
		if token == "" && !tokenOnly && session.IsLogin(c) {
			c.Next()
			return
		}

		if token == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		apiUser, err := apiUserService.VerifyToken(token)
		if err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		effectiveLimit := apiUserService.EffectiveRateLimit(apiUser)
		if !limiterStore.allow(apiUser.Id, effectiveLimit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		c.Set(apiUserContextKey, apiUser)
		session.SetContextUser(c, &model.User{
			Id:       apiVirtualUserIDOffset + apiUser.Id,
			Username: "api:" + apiUser.Name,
		})

		c.Next()
	}
}

// GetAPIUserFromContext returns the authenticated API user (if any) set by the middleware.
func GetAPIUserFromContext(c *gin.Context) *model.APIUser {
	if c == nil {
		return nil
	}
	user, ok := c.Get(apiUserContextKey)
	if !ok {
		return nil
	}
	apiUser, ok := user.(*model.APIUser)
	if !ok {
		return nil
	}
	return apiUser
}

func extractAPIToken(c *gin.Context) string {
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	headerToken := strings.TrimSpace(c.GetHeader("X-API-Token"))
	if headerToken != "" {
		return headerToken
	}
	queryToken := strings.TrimSpace(c.Query("api_token"))
	return queryToken
}
