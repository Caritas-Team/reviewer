package assessment

import (
	"context"
	"fmt"
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

	var (
		sumControlBefore, sumControlAfter     float64
		sumObtainingBefore, sumObtainingAfter float64
		sumSocialBefore, sumSocialAfter       float64
		sumInfoBefore, sumInfoAfter           float64
	)

	results := make([]AssessmentDiff, 0, total)

	for i := 0; i < total; i++ {
		if ctx.Err() != nil {
			return
		}

		pair := pairs[i]

		if pair.Before == nil || pair.After == nil {
			err := fmt.Errorf("student %s: Before or After is nil", pair.StudentID)
			_ = w.storage.Set(ctx, requestID, &ProcessingResult{
				Status:            "failed",
				ProgressPercent:   (i * 100) / total,
				ProcessedStudents: i,
				TotalStudents:     total,
				Error:             err.Error(),
			}, w.ttl)
			return
		}

		diff, err := w.processor.Process(ctx, pair)
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

		before := pair.Before.CommunicativeFuncs
		after := pair.After.CommunicativeFuncs

		sumControlBefore += before.Control
		sumControlAfter += after.Control
		sumObtainingBefore += before.ObtainingDesired
		sumObtainingAfter += after.ObtainingDesired
		sumSocialBefore += before.SocialInteraction
		sumSocialAfter += after.SocialInteraction
		sumInfoBefore += before.InformationExchange
		sumInfoAfter += after.InformationExchange

		results = append(results, diff)

		// Обновление прогресса
		processed := i + 1
		progress := processed * 100 / total
		_ = w.storage.Set(ctx, requestID, &ProcessingResult{
			Status:            "processing",
			ProgressPercent:   progress,
			ProcessedStudents: processed,
			TotalStudents:     total,
		}, w.ttl)
	}

	//ВЫЧИСЛЯЕМ ГРУППОВУЮ ДЕЛЬТУ
	n := float64(total)
	groupComm := &GroupCommFuncsMetrics{
		ControlDelta:             (sumControlAfter/n - sumControlBefore/n),
		ObtainingDesiredDelta:    (sumObtainingAfter/n - sumObtainingBefore/n),
		SocialInteractionDelta:   (sumSocialAfter/n - sumSocialBefore/n),
		InformationExchangeDelta: (sumInfoAfter/n - sumInfoBefore/n),
	}

	_ = w.storage.Set(ctx, requestID, &ProcessingResult{
		Status:                "completed",
		ProgressPercent:       100,
		ProcessedStudents:     total,
		TotalStudents:         total,
		Results:               results,
		GroupCommFuncsMetrics: groupComm,
	}, w.ttl)
}
