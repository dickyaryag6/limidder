package limidder

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Strategy interface {
	Run(ctx context.Context, r *Request) (*Result, error)
}

func NewSlidingWindowStrategy(client *redis.Client) Strategy {
	return &slidingWindow{
		client: client,
	}
}

type slidingWindow struct {
	client *redis.Client
	m      sync.Mutex
}

func (s *slidingWindow) Run(ctx context.Context, r *Request) (*Result, error) {

	s.m.Lock()
	defer s.m.Unlock()

	now := time.Now()
	// every request needs an UUID
	item := uuid.New()

	minimum := now.Add(-r.Duration)

	p := s.client.Pipeline()

	a := strconv.FormatInt(minimum.UnixNano(), 10)

	// we then remove all requests that have already expired on this set
	removeByScore := p.ZRemRangeByScore(r.Key, "0", a)

	// we add the current request
	add := p.ZAdd(r.Key, redis.Z{
		Score:  float64(now.UnixNano()), // to save the timestamp
		Member: item.String(),
	})

	// count how many non-expired requests we have on the sorted set
	count := p.ZCount(r.Key, "-inf", "+inf")

	if _, err := p.Exec(); err != nil {
		return nil, errors.Wrapf(err, "failed to execute sorted set pipeline for key: %v", r.Key)
	}

	if err := removeByScore.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to remove items from key %v", r.Key)
	}

	if err := add.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to add item to key %v", r.Key)
	}

	totalRequests, err := count.Result()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to count items for key %v", r.Key)
	}

	expiresAt := now.Add(r.Duration)
	requests := uint64(totalRequests)

	fmt.Println(requests)

	if requests > r.Limit {
		return &Result{
			State:         Deny,
			TotalRequests: requests,
			ExpiresAt:     expiresAt,
		}, nil
	}

	return &Result{
		State:         Allow,
		TotalRequests: requests,
		ExpiresAt:     expiresAt,
	}, nil
}
