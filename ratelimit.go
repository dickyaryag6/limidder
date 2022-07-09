package ratelimit

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"net/http"
	"strconv"
	"time"
)

// add yaml and json tag so people can use config file
type Config struct {
	Limit    uint64 `yaml:"limit" json:"limit"`
	Duration int64  `yaml:"duration" json:"duration"`
}

type Request struct {
	Key      string
	Limit    uint64
	Duration time.Duration
}

type Result struct {
	State         State
	TotalRequests uint64
	ExpiresAt     time.Time
}

type RateLimiterConfig struct {
	Extractor                    Extractor
	StrategyName                 string
	strategy                     Strategy
	Config                       map[string]*Config
	ApplyConfigToAllPaths        bool
	ApplyUserRateLimitToAllPaths bool
}

func (r *RateLimiterConfig) SetStrategy(strategyName string, client *redis.Client) {
	switch strategyName {
	case SlidingWindow:
		r.strategy = NewSlidingWindowStrategy(client)
	default:
		r.strategy = nil
	}
}

type httpRateLimiterHandler struct {
	config *RateLimiterConfig
}

func InitRateLimiterMiddleware(config *RateLimiterConfig, client *redis.Client) *httpRateLimiterHandler {

	config.SetStrategy(config.StrategyName, client)

	return &httpRateLimiterHandler{
		config: config,
	}
}

func (h *httpRateLimiterHandler) writeRespone(writer http.ResponseWriter, status int, msg string, args ...interface{}) {
	writer.Header().Set("Content-Type", "text/plain")
	writer.WriteHeader(status)
	if _, err := writer.Write([]byte(fmt.Sprintf(msg, args...))); err != nil {
		fmt.Printf("failed to write body to HTTP request: %v", err)
	}
}

func (h *httpRateLimiterHandler) getLimitAndDuration(request *http.Request) (limit uint64, duration time.Duration, err error) {

	var configData *Config

	if h.config.ApplyConfigToAllPaths {
		configData = h.config.Config["all"]
		if configData == nil {
			return limit, duration, errRateLimitNotSetForAllPaths
		}
	} else {
		route := request.Method + " " + request.URL.String()
		configData = h.config.Config[route]
		if configData == nil {
			return limit, duration, errRateLimitNotSet
		}
	}

	limit = configData.Limit
	duration = time.Duration(configData.Duration * int64(time.Second))

	return
}

func (h *httpRateLimiterHandler) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		key, err := h.config.Extractor.ExtractKey(request, h.config.ApplyUserRateLimitToAllPaths)
		if err != nil {
			h.writeRespone(writer, http.StatusBadRequest, "failed to collect rate limiting key from request: %v", err)
			return
		}

		limit, duration, err := h.getLimitAndDuration(request)

		// not all paths need to be rate limited, so if the config is not set, just let it passes
		if errors.Is(err, errRateLimitNotSet) {
			next.ServeHTTP(writer, request)
			return
		}
		if err != nil {
			h.writeRespone(writer, http.StatusInternalServerError, "failed to run rate limiting for request: %v", err)
			return
		}

		result, err := h.config.strategy.Run(request.Context(), &Request{
			Key:      key,
			Limit:    limit,
			Duration: duration,
		})
		if err != nil {
			h.writeRespone(writer, http.StatusInternalServerError, "failed to run rate limiting for request: %v", err)
			return
		}

		// set the rate limiting headers both on allow or deny results so the client knows what is going on
		writer.Header().Set(rateLimitingTotalRequests, strconv.FormatUint(result.TotalRequests, 10))
		writer.Header().Set(rateLimitingState, stateStrings[result.State])
		writer.Header().Set(rateLimitingExpiresAt, result.ExpiresAt.Format(time.RFC3339))

		// when the state is Deny, just return a 429 response to the client and stop the request handling flow
		if result.State == Deny {
			h.writeRespone(writer, http.StatusTooManyRequests, "you have sent too many requests to this service, slow down please")
			return
		}

		next.ServeHTTP(writer, request)
	})
}
