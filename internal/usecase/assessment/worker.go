package assessment

import (
	"context"
	"sort"
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
			Error:         "Процессор не инициализирован",
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

	// Рассчитываем средние значения по группе
	groupAverages := w.calculateGroupAverages(pairs)

	// Рассчитываем прогресс группы между датами
	groupProgress := w.calculateGroupProgress(groupAverages)

	groupDiff := w.calculateGroupVocabularyProgress(pairs)

	_ = w.storage.Set(ctx, requestID, &ProcessingResult{
		Status:            "completed",
		ProgressPercent:   100,
		ProcessedStudents: total,
		TotalStudents:     total,
		Results:           results,
		GroupAverages:     groupAverages,
		GroupProgress:     groupProgress,
		GroupDiff:         groupDiff,
	}, w.ttl)
}

// calculateGroupAverages группирует документы по датам и вычисляет средние
func (w *Worker) calculateGroupAverages(pairs []model.StudentPair) []GroupAverage {
	// Собираем все документы (и Before, и After)
	documentsByDate := make(map[string][]*model.AssessmentDocument)

	for _, pair := range pairs {
		if pair.Before != nil {
			dateKey := pair.Before.Metadata.Date.Format("2006-01-02")
			documentsByDate[dateKey] = append(documentsByDate[dateKey], pair.Before)
		}
		if pair.After != nil {
			dateKey := pair.After.Metadata.Date.Format("2006-01-02")
			documentsByDate[dateKey] = append(documentsByDate[dateKey], pair.After)
		}
	}

	// Вычисляем средние для каждой даты
	calc := &DiffCalculator{}
	averages := make([]GroupAverage, 0, len(documentsByDate))

	for dateKey, docs := range documentsByDate {
		avg, err := calc.CalculateGroupAverage(docs)
		if err != nil {
			// Логируем ошибку, но продолжаем обработку
			w.log.Error("failed to calculate group average", "date", dateKey, "error", err)
			continue
		}
		averages = append(averages, avg)
	}

	// Сортируем по дате
	sort.Slice(averages, func(i, j int) bool {
		return averages[i].Date < averages[j].Date
	})

	return averages
}

// calculateGroupProgress рассчитывает прогресс между средними значениями группы
func (w *Worker) calculateGroupProgress(averages []GroupAverage) []GroupProgress {
	if len(averages) < 2 {
		return nil
	}

	calc := &DiffCalculator{}
	progressList := make([]GroupProgress, 0)

	// Для каждой пары последовательных дат рассчитываем прогресс
	for i := 0; i < len(averages)-1; i++ {
		for j := i + 1; j < len(averages); j++ {
			progress, err := calc.CalculateGroupProgress(averages[i], averages[j])
			if err != nil {
				w.log.Error("failed to calculate group progress",
					"date1", averages[i].Date,
					"date2", averages[j].Date,
					"error", err)
				continue
			}
			progressList = append(progressList, progress)
		}
	}

	return progressList
}

// calculateGroupVocabularyProgress рассчитывает разницу по словарю для группы
func (w *Worker) calculateGroupVocabularyProgress(pairs []model.StudentPair) []GroupVocabularyProgress {
	if len(pairs) == 0 {
		return nil
	}

	// Собираем документы "до" и "после" из пар
	var beforeDocs []*model.AssessmentDocument
	var afterDocs []*model.AssessmentDocument

	for _, pair := range pairs {
		if pair.Before != nil {
			beforeDocs = append(beforeDocs, pair.Before)
		}
		if pair.After != nil {
			afterDocs = append(afterDocs, pair.After)
		}
	}

	// Проверяем, что есть данные для сравнения
	if len(beforeDocs) == 0 || len(afterDocs) == 0 {
		w.log.Warn("insufficient data for vocabulary progress calculation",
			"before_docs", len(beforeDocs),
			"after_docs", len(afterDocs))
		return nil
	}

	calc := &DiffCalculator{}
	vocabProgress, err := calc.CalculateGroupVocabularyProgress(beforeDocs, afterDocs)
	if err != nil {
		w.log.Error("failed to calculate group vocabulary progress", "error", err)
		return nil
	}

	return []GroupVocabularyProgress{vocabProgress}
}
