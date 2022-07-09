package limidder

import "errors"

var (
	_            Extractor = &httpHeaderExtractor{}
	stateStrings           = map[State]string{
		Allow: "Allow",
		Deny:  "Deny",
	}

	SlidingWindow = "sliding_window"
)

type State int64

const (
	Deny  State = 0
	Allow State = 1
)

const (
	rateLimitingTotalRequests = "Rate-Limiting-Total-Requests"
	rateLimitingState         = "Rate-Limiting-State"
	rateLimitingExpiresAt     = "Rate-Limiting-Expires-At"
)

// error
var (
	errRateLimitNotSet = errors.New("rate limit is not set")
	errRateLimitNotSetForAllPaths = errors.New("rate limit for all paths is not set")
)
