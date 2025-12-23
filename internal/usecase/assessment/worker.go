package assessment

import (
	"context"
	"time"

	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/model"
)

type PairProcessor interface {
	Process(ctx context.Context, pair model.StudentPair) (AssessmentDiff, error)
}

// Worker читает из канала пачки StudentPair одного request_id и обновляет статус в ResultStorage.
// Handler должен быть "тупым": только читает из storage и отдает ответ; все смены статусов происходят здесь.
type Worker struct {
	log       *logger.Logger
	storage   *ResultStorage
	processor PairProcessor
	ttl       time.Duration
}

func NewWorker(log *logger.Logger, storage *ResultStorage, processor PairProcessor, ttl time.Duration) *Worker {
	return &Worker{log: log, storage: storage, processor: processor, ttl: ttl}
}

func (w *Worker) Run(ctx context.Context, input <-chan []model.StudentPair) {
	for {
		select {
		case <-ctx.Done():
			return
		case pairs := <-input:
			if len(pairs) == 0 {
				continue
			}
			w.handle(ctx, pairs)
		}
	}
}

func (w *Worker) handle(ctx context.Context, pairs []model.StudentPair) {
	if len(pairs) == 0 {
		return
	}

	requestID := pairs[0].RequestID
	total := len(pairs)

	if w.processor == nil {
		_ = w.storage.Set(ctx, requestID, &ProcessingResult{
			Status:        "failed",
			TotalStudents: total,
			Error:         "processor is nil",
		}, w.ttl)
		return
	}

	_ = w.storage.Set(ctx, requestID, &ProcessingResult{
		Status:            "processing",
		ProgressPercent:   0,
		ProcessedStudents: 0,
		TotalStudents:     total,
	}, w.ttl)

	results := make([]AssessmentDiff, 0, total)

	for i := 0; i < total; i++ {
		if ctx.Err() != nil {
			return
		}

		diff, err := w.processor.Process(ctx, pairs[i])
		if err != nil {
			_ = w.storage.Set(ctx, requestID, &ProcessingResult{
				Status:            "failed",
				ProgressPercent:   (i * 100) / total,
				ProcessedStudents: i,
				TotalStudents:     total,
				Error:             err.Error(),
			}, w.ttl)
			return
		}

		results = append(results, diff)

		processed := i + 1
		progress := processed * 100 / total

		_ = w.storage.Set(ctx, requestID, &ProcessingResult{
			Status:            "processing",
			ProgressPercent:   progress,
			ProcessedStudents: processed,
			TotalStudents:     total,
		}, w.ttl)
	}

	_ = w.storage.Set(ctx, requestID, &ProcessingResult{
		Status:            "completed",
		ProgressPercent:   100,
		ProcessedStudents: total,
		TotalStudents:     total,
		Results:           results,
	}, w.ttl)
}
