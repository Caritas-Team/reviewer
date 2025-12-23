package assessment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/google/uuid"
)

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for condition")
}

func TestResultAggregator_Success(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Config{Logging: config.Logging{Level: "debug", Format: "json"}}
	log := logger.NewLogger(cfg)

	cache := newMemCacheMock()
	st := NewResultStorage(cache)

	resultCh := make(chan PairResult, 10)
	errorCh := make(chan PairError, 10)

	ttl := 5 * time.Minute
	go ResultAggregator(ctx, resultCh, errorCh, st, log, ttl)

	reqID := uuid.NewString()

	diff1 := AssessmentDiff{StudentID: "S1", PeriodStart: "2025-12-18", PeriodEnd: "2025-12-19"}
	diff2 := AssessmentDiff{StudentID: "S2", PeriodStart: "2025-12-18", PeriodEnd: "2025-12-19"}

	// 1) отправили первый результат -> processing + 50%
	resultCh <- PairResult{RequestID: reqID, Total: 2, Diff: diff1}

	waitFor(t, 2*time.Second, func() bool {
		res, err := st.Get(ctx, reqID)
		if err != nil {
			return false
		}
		return res.Status == "processing" &&
			res.TotalStudents == 2 &&
			res.ProcessedStudents == 1 &&
			res.ProgressPercent == 50 &&
			len(res.Results) == 1
	})

	// 2) отправили второй результат -> completed + 100%
	resultCh <- PairResult{RequestID: reqID, Total: 2, Diff: diff2}

	waitFor(t, 2*time.Second, func() bool {
		res, err := st.Get(ctx, reqID)
		if err != nil {
			return false
		}
		if res.Status != "completed" {
			return false
		}
		if res.TotalStudents != 2 || res.ProcessedStudents != 2 || res.ProgressPercent != 100 {
			return false
		}
		if len(res.Results) != 2 {
			return false
		}
		return res.Results[0].StudentID == "S1" && res.Results[1].StudentID == "S2"
	})
}

func TestResultAggregator_Failed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Config{Logging: config.Logging{Level: "debug", Format: "json"}}
	log := logger.NewLogger(cfg)

	cache := newMemCacheMock()
	st := NewResultStorage(cache)

	resultCh := make(chan PairResult, 10)
	errorCh := make(chan PairError, 10)

	ttl := 5 * time.Minute
	go ResultAggregator(ctx, resultCh, errorCh, st, log, ttl)

	reqID := uuid.NewString()
	errBoom := errors.New("boom")

	errorCh <- PairError{RequestID: reqID, StudentID: "S1", Err: errBoom}

	waitFor(t, 2*time.Second, func() bool {
		res, err := st.Get(ctx, reqID)
		if err != nil {
			return false
		}
		if res.Status != "failed" {
			return false
		}

		m, ok := res.Error.(map[string]any)
		if !ok {
			return false
		}
		return m["student_id"] == "S1" && m["message"] == "boom"
	})
}
