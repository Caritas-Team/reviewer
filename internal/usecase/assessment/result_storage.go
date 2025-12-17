package assessment

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Caritas-Team/reviewer/internal/memcached"
	"github.com/bradfitz/gomemcache/memcache"
)

var ErrNotFound = errors.New("result not found")

type AssessmentDiff struct {
	StudentID string      `json:"student_id"`
	Diffs     []FieldDiff `json:"diffs"`
}

type FieldDiff struct {
	Field    string `json:"field"`
	Expected any    `json:"expected"`
	Actual   any    `json:"actual"`
}

type ProcessingResult struct {
	Status            string           `json:"status"`
	ProgressPercent   int              `json:"progress_percent,omitempty"`
	ProcessedStudents int              `json:"processed_students,omitempty"`
	TotalStudents     int              `json:"total_students,omitempty"`
	Results           []AssessmentDiff `json:"results,omitempty"`
	Error             any              `json:"error,omitempty"`
}

type ResultStorage struct {
	cache memcached.CacheInterface
}

func NewResultStorage(cache memcached.CacheInterface) *ResultStorage {
	return &ResultStorage{cache: cache}
}

func (s *ResultStorage) Set(ctx context.Context, requestID string, res *ProcessingResult, ttl time.Duration) error {
	if res == nil {
		return errors.New("nil result")
	}
	if res.Status == "" {
		return errors.New("empty status")
	}

	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return s.cache.Set(ctx, requestID, b, ttl)
}

func (s *ResultStorage) Get(ctx context.Context, requestID string) (*ProcessingResult, error) {
	b, err := s.cache.Get(ctx, requestID)
	if err != nil {
		if errors.Is(err, memcached.ErrCacheMiss) || errors.Is(err, memcache.ErrCacheMiss) ||
			strings.Contains(strings.ToLower(err.Error()), "cache miss") {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var res ProcessingResult
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}
	if res.Status == "" {
		return nil, errors.New("invalid cached result: empty status")
	}
	return &res, nil
}

func (s *ResultStorage) GetAndDelete(ctx context.Context, requestID string, keepInCache bool) (*ProcessingResult, error) {
	b, err := s.cache.Get(ctx, requestID)
	if err != nil {
		if errors.Is(err, memcached.ErrCacheMiss) || errors.Is(err, memcache.ErrCacheMiss) ||
			strings.Contains(strings.ToLower(err.Error()), "cache miss") {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var res ProcessingResult
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}
	if res.Status == "" {
		return nil, errors.New("invalid cached result: empty status")
	}

	switch res.Status {
	case "completed":
		if !keepInCache {
			_ = s.cache.Delete(ctx, requestID)
		}
	case "failed":
		_ = s.cache.Delete(ctx, requestID)
	}

	return &res, nil
}
