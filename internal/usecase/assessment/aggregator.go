package assessment

import (
	"context"
	"time"

	"github.com/Caritas-Team/reviewer/internal/logger"
)

type PairResult struct {
	RequestID string
	Total     int
	Diff      AssessmentDiff
}

type PairError struct {
	RequestID string
	StudentID string
	Err       error
}

// ResultAggregator собирает результаты и ошибки
func ResultAggregator(
	ctx context.Context,
	resultCh <-chan PairResult,
	errorCh <-chan PairError,
	storage *ResultStorage,
	log *logger.Logger,
	ttl time.Duration,
) {
	for {
		select {
		case <-ctx.Done():
			return

		case r, ok := <-resultCh:
			if !ok {
				return
			}
			aggregateOK(ctx, storage, log, ttl, r)

		case e, ok := <-errorCh:
			if !ok {
				return
			}
			aggregateErr(ctx, storage, log, ttl, e)
		}
	}
}

func aggregateOK(ctx context.Context, storage *ResultStorage, log *logger.Logger, ttl time.Duration, r PairResult) {
	cur, err := storage.Get(ctx, r.RequestID)
	if err != nil {
		// если результата еще нет — стартуем с нуля
		cur = &ProcessingResult{
			Status:            "processing",
			ProgressPercent:   0,
			ProcessedStudents: 0,
			TotalStudents:     r.Total,
			Results:           nil,
		}
	}

	if cur.TotalStudents == 0 {
		cur.TotalStudents = r.Total
	}
	if r.Total > 0 && cur.TotalStudents == 0 {
		cur.TotalStudents = r.Total
	}

	cur.Results = append(cur.Results, r.Diff)

	cur.ProcessedStudents = len(cur.Results)
	if cur.TotalStudents > 0 {
		cur.ProgressPercent = cur.ProcessedStudents * 100 / cur.TotalStudents
	}

	if cur.TotalStudents > 0 && cur.ProcessedStudents >= cur.TotalStudents {
		cur.Status = "completed"
		cur.ProgressPercent = 100

		if err := storage.Set(ctx, r.RequestID, cur, ttl); err != nil {
			log.Error("failed to store completed result", "request_id", r.RequestID, "err", err)
		} else {
			log.Info("assessment completed", "request_id", r.RequestID, "total_students", cur.TotalStudents)
		}
		return
	}

	cur.Status = "processing"
	if err := storage.Set(ctx, r.RequestID, cur, ttl); err != nil {
		log.Error("failed to store processing state", "request_id", r.RequestID, "err", err)
	}
}

func aggregateErr(ctx context.Context, storage *ResultStorage, log *logger.Logger, ttl time.Duration, e PairError) {
	// при ошибке — failed (по ТЗ failed можно не кэшировать "навсегда", но TTL пусть будет)
	payload := map[string]any{
		"student_id": e.StudentID,
		"message":    e.Err.Error(),
	}

	cur, err := storage.Get(ctx, e.RequestID)
	if err != nil {
		cur = &ProcessingResult{}
	}

	cur.Status = "failed"
	cur.Error = payload

	if err := storage.Set(ctx, e.RequestID, cur, ttl); err != nil {
		log.Error("failed to store failed result", "request_id", e.RequestID, "err", err)
	} else {
		log.Error("assessment failed", "request_id", e.RequestID, "student_id", e.StudentID, "err", e.Err)
	}
}
