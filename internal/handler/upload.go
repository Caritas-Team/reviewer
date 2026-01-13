package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/memcached"
	"github.com/Caritas-Team/reviewer/internal/model"
	"github.com/Caritas-Team/reviewer/internal/usecase/assessment"
	"github.com/Caritas-Team/reviewer/internal/usecase/file"
	"github.com/google/uuid"
)

type UploadTask struct {
	RequestID     string              `json:"request_id"`
	Status        string              `json:"status"`
	StudentPairs  []model.StudentPair `json:"student_pairs"`
	TotalFiles    int                 `json:"total_files"`
	TotalStudents int                 `json:"total_students"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	EstimatedSec  int                 `json:"estimated_completion_sec"`
}

type UploadRequestMeta struct {
	Organization string `json:"organization,omitempty"`
	Specialist   string `json:"specialist,omitempty"`
}

type UploadHandler struct {
	cfg          config.Config
	log          *logger.Logger
	cache        memcached.CacheInterface
	results      *assessment.ResultStorage
	inputChan    chan<- []model.StudentPair
	maxTotalSize int64
	parser       *file.DocumentParser
}

func NewUploadHandler(cfg config.Config, log *logger.Logger, cache memcached.CacheInterface, results *assessment.ResultStorage, inputChan chan<- []model.StudentPair) *UploadHandler {
	return &UploadHandler{
		cfg:          cfg,
		log:          log,
		cache:        cache,
		results:      results,
		inputChan:    inputChan,
		maxTotalSize: cfg.Files.MaxFileSize * int64(cfg.Files.MaxFilesPerRequest),
		parser:       &file.DocumentParser{},
	}
}

func (h *UploadHandler) UploadAssessmentsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	defer func() {
		if r.Body != nil {
			_ = r.Body.Close()
		}
		if err := recover(); err != nil {
			h.log.Error("Panic recovered in upload handler",
				"error", err,
				"path", r.URL.Path,
				"method", r.Method,
			)
			h.sendError(w, http.StatusInternalServerError, "internal_error",
				"Произошла внутренняя ошибка сервера",
				map[string]any{"recovered": fmt.Sprintf("%v", err)})
		}
	}()

	requestID := r.Header.Get("X-Request-Id")
	if requestID == "" {
		h.sendError(w, http.StatusBadRequest, "validation_error",
			"Отсутствует заголовок X-Request-Id",
			map[string]any{"field": "X-Request-Id", "constraint": "required"})
		return
	}

	if _, err := uuid.Parse(requestID); err != nil {
		h.sendError(w, http.StatusBadRequest, "validation_error",
			"Неверный формат X-Request-Id, должен быть UUID",
			map[string]any{"field": "X-Request-Id", "format": "uuid", "value": requestID})
		return
	}

	h.log.WithContext(ctx).Info("Upload request started",
		"request_id", requestID,
		"method", r.Method,
		"path", r.URL.Path,
	)

	if err := h.checkIdempotency(ctx, requestID, w); err != nil {
		return
	}

	if err := r.ParseMultipartForm(h.maxTotalSize); err != nil {
		if errors.Is(err, http.ErrMissingBoundary) {
			h.sendError(w, http.StatusBadRequest, "validation_error",
				"Отсутствует разделитель multipart данных",
				map[string]any{"field": "Content-Type", "constraint": "multipart/form-data"})
		} else if strings.Contains(err.Error(), "request body too large") {
			h.sendError(w, http.StatusRequestEntityTooLarge, "payload_too_large",
				"Общий размер файлов превышает ограничение в 50MB",
				map[string]any{
					"max_size_mb": 50,
					"constraint":  "max_total_size",
				})
		} else {
			h.log.WithContext(ctx).Error("Failed to parse multipart form", "error", err)
			h.sendError(w, http.StatusInternalServerError, "internal_error",
				"Не удалось разобрать данные формы",
				map[string]any{"error": err.Error()})
		}
		return
	}

	var meta UploadRequestMeta
	if metaStr := r.FormValue("meta"); metaStr != "" {
		if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
			h.log.WithContext(ctx).Warn("Failed to parse meta field", "error", err)
		}
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		h.sendError(w, http.StatusBadRequest, "validation_error",
			"Не указаны файлы в поле 'files'",
			map[string]any{"field": "files", "constraint": "required"})
		return
	}

	if len(files) < 2 || len(files) > 20 {
		h.sendError(w, http.StatusBadRequest, "validation_error",
			fmt.Sprintf("Количество файлов должно быть от 2 до 20 (получено %d)", len(files)),
			map[string]any{
				"field":      "files",
				"min_items":  2,
				"max_items":  20,
				"got_items":  len(files),
				"constraint": "item_count",
			})
		return
	}
	if len(files)%2 != 0 {
		h.sendError(w, http.StatusBadRequest, "validation_error",
			fmt.Sprintf("Количество файлов должно быть четным (получено %d)", len(files)),
			map[string]any{
				"field":      "files",
				"got_items":  len(files),
				"constraint": "even_count",
			})
		return
	}

	var totalSize int64
	for _, fileHeader := range files {
		totalSize += fileHeader.Size
	}
	if totalSize > h.maxTotalSize {
		receivedSizeMB := float64(totalSize) / (1024 * 1024)
		h.sendError(w, http.StatusRequestEntityTooLarge, "payload_too_large",
			fmt.Sprintf("Общий размер файлов превышает 50MB (получено %.2fMB)", receivedSizeMB),
			map[string]any{
				"max_size_mb":      50,
				"received_size_mb": fmt.Sprintf("%.2f", receivedSizeMB),
				"constraint":       "max_total_size",
			})
		return
	}

	// Парсим и валидируем файлы
	documents, err := h.parseAndValidateFiles(ctx, files)
	if err != nil {
		details := make(map[string]any)
		if strings.Contains(err.Error(), "Идентификатор студента не найден") {
			details["field"] = "por02"
			details["constraint"] = "required"
		} else if strings.Contains(err.Error(), "Дата не найдена") {
			details["field"] = "por01"
			details["constraint"] = "required"
		} else if strings.Contains(err.Error(), "Неверный формат даты") {
			details["field"] = "por01"
			details["format"] = "YYYY-MM-DD"
			details["constraint"] = "date_format"
		} else if strings.Contains(err.Error(), "Некорректный формат JSON") {
			details["constraint"] = "valid_json"
		}

		h.sendError(w, http.StatusBadRequest, "parse_error", err.Error(), details)
		return
	}

	// Группируем документы по студентам и создаем пары
	studentPairs, err := h.createStudentPairs(requestID, documents)
	if err != nil {
		details := make(map[string]any)
		if strings.Contains(err.Error(), "has") && strings.Contains(err.Error(), "documents") {
			parts := strings.Split(err.Error(), "'")
			if len(parts) >= 2 {
				details["student_id"] = parts[1]
			}
			if strings.Contains(err.Error(), "expected exactly 2") {
				details["constraint"] = "exactly_two_per_student"
			} else if strings.Contains(err.Error(), "same date") {
				details["constraint"] = "different_dates"
			}
		}

		h.sendError(w, http.StatusBadRequest, "validation_error", err.Error(), details)
		return
	}

	_ = h.results.Set(ctx, requestID, &assessment.ProcessingResult{
		Status:            "processing",
		ProgressPercent:   0,
		ProcessedStudents: 0,
		TotalStudents:     len(studentPairs),
	}, time.Hour)

	if meta.Organization != "" || meta.Specialist != "" {
		h.log.WithContext(ctx).Info("Upload request with metadata",
			"request_id", requestID,
			"organization", meta.Organization,
			"specialist", meta.Specialist,
		)
	}

	// Создаем задачу
	task := &UploadTask{
		RequestID:     requestID,
		Status:        "processing",
		StudentPairs:  studentPairs,
		TotalFiles:    len(files),
		TotalStudents: len(studentPairs),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		EstimatedSec:  15,
	}

	if err := h.saveTask(ctx, task); err != nil {
		h.log.WithContext(ctx).Error("Failed to save task", "error", err)
		h.sendError(w, http.StatusInternalServerError, "internal_error",
			"Не удалось зарегистрировать задачу",
			map[string]any{"error": err.Error()})
		return
	}

	_ = h.results.Set(ctx, requestID, &assessment.ProcessingResult{
		Status:            "processing",
		ProgressPercent:   0,
		ProcessedStudents: 0,
		TotalStudents:     len(studentPairs),
	}, time.Hour)

	// Отправляем в канал обработки
	h.sendToProcessingChannel(ctx, studentPairs)

	h.log.WithContext(ctx).Info("Upload validation completed",
		"request_id", requestID,
		"files_count", task.TotalFiles,
		"students_count", task.TotalStudents,
		"student_ids", extractStudentIDsFromPairs(studentPairs),
		"organization", meta.Organization,
		"specialist", meta.Specialist,
	)

	h.sendSuccessResponse(w, task)
}

func (h *UploadHandler) parseAndValidateFiles(ctx context.Context, files []*multipart.FileHeader) ([]model.AssessmentDocument, error) {
	documents := make([]model.AssessmentDocument, 0, len(files))

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("не удалось открыть файл '%s': %w", fileHeader.Filename, err)
		}
		defer func() { _ = file.Close() }()

		doc, err := h.parser.Parse(file, fileHeader.Filename)
		if err != nil {
			return nil, fmt.Errorf("файл '%s': %w", fileHeader.Filename, err)
		}

		documents = append(documents, *doc)

		h.log.WithContext(ctx).Debug("File parsed successfully",
			"filename", fileHeader.Filename,
			"student_id", doc.Metadata.StudentID,
			"date", doc.Metadata.Date.Format("2006-01-02"),
		)
	}

	return documents, nil
}

// createStudentPairs создает пары документов для каждого студента
func (h *UploadHandler) createStudentPairs(requestID string, documents []model.AssessmentDocument) ([]model.StudentPair, error) {
	groups := make(map[string][]model.AssessmentDocument)

	for _, doc := range documents {
		groups[doc.Metadata.StudentID] = append(groups[doc.Metadata.StudentID], doc)
	}

	pairs := make([]model.StudentPair, 0, len(groups))

	for studentID, docs := range groups {
		if len(docs) != 2 {
			return nil, fmt.Errorf("студент '%s' имеет %d документов, требуется ровно 2", studentID, len(docs))
		}

		// Сортируем по дате (от более ранней к более поздней)
		if docs[0].Metadata.Date.After(docs[1].Metadata.Date) {
			docs[0], docs[1] = docs[1], docs[0]
		}

		docs[0].Metadata.AssessmentType = "before"
		docs[1].Metadata.AssessmentType = "after"

		if docs[0].Metadata.Date.Equal(docs[1].Metadata.Date) {
			return nil, fmt.Errorf("студент '%s' имеет документы с одинаковой датой", studentID)
		}

		pair := model.StudentPair{
			RequestID: requestID,
			StudentID: studentID,
			Before:    &docs[0],
			After:     &docs[1],
		}

		pairs = append(pairs, pair)
	}

	return pairs, nil
}

// sendToProcessingChannel отправляет пары в канал обработки
func (h *UploadHandler) sendToProcessingChannel(ctx context.Context, pairs []model.StudentPair) {
	select {
	case h.inputChan <- pairs:
		h.log.WithContext(ctx).Debug("Sent student pairs to processing channel",
			"student_count", len(pairs),
		)
	case <-ctx.Done():
		h.log.WithContext(ctx).Warn("Context cancelled while sending to processing channel")
	}
}

func extractStudentIDsFromPairs(pairs []model.StudentPair) []string {
	ids := make([]string, len(pairs))
	for i, pair := range pairs {
		ids[i] = pair.StudentID
	}
	return ids
}

func (h *UploadHandler) checkIdempotency(ctx context.Context, requestID string, w http.ResponseWriter) error {
	data, err := h.cache.Get(ctx, fmt.Sprintf("task:%s", requestID))
	if err != nil {
		if err == memcached.ErrCacheMiss {
			return nil
		}
		h.log.WithContext(ctx).Error("Failed to check cache for idempotency", "error", err)
		h.sendError(w, http.StatusInternalServerError, "internal_error",
			"Не удалось проверить статус запроса",
			map[string]any{"error": err.Error()})
		return err
	}

	var existingTask UploadTask
	if err := json.Unmarshal(data, &existingTask); err != nil {
		h.log.WithContext(ctx).Error("Failed to unmarshal existing task", "error", err)

		if err := h.cache.Delete(ctx, fmt.Sprintf("task:%s", requestID)); err != nil {
			h.log.WithContext(ctx).Warn("Failed to delete corrupted cache entry", "error", err)
		}
		return nil
	}

	switch existingTask.Status {
	case "processing":
		h.sendError(w, http.StatusConflict, "conflict",
			fmt.Sprintf("Запрос с ID '%s' уже обрабатывается", requestID),
			map[string]any{
				"request_id":     requestID,
				"current_status": "processing",
			})
		return errors.New("запрос уже обрабатывается")
	case "completed":
		h.sendExistingResult(w, &existingTask)
		return errors.New("запрос уже завершен")
	case "failed":
		h.log.WithContext(ctx).Info("Повторная обработка неудачного запроса", "request_id", requestID)
		return nil
	default:
		return nil
	}
}

func (h *UploadHandler) saveTask(ctx context.Context, task *UploadTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("что-то пошло не так: %w", err)
	}

	err = h.cache.Set(ctx, fmt.Sprintf("task:%s", task.RequestID), data, time.Hour)
	if err != nil {
		return fmt.Errorf("не удалось сохранить задачу в кэше: %w", err)
	}

	return nil
}

func (h *UploadHandler) sendSuccessResponse(w http.ResponseWriter, task *UploadTask) {
	response := map[string]any{
		"request_id":               task.RequestID,
		"status":                   task.Status,
		"accepted_files":           task.TotalFiles,
		"students_count":           task.TotalStudents,
		"estimated_completion_sec": task.EstimatedSec,
		"created_at":               task.CreatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-Id", task.RequestID)
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Не удалось сформировать ответ об успехе", "error", err)
	}
}

func (h *UploadHandler) sendExistingResult(w http.ResponseWriter, task *UploadTask) {
	response := map[string]any{
		"request_id": task.RequestID,
		"status":     task.Status,
		"message":    "Запрос уже завершен",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Не удалось сформировать ответ с существующим результатом", "error", err)
	}
}

func (h *UploadHandler) sendError(w http.ResponseWriter, status int, errorType, message string, details map[string]any) {
	response := map[string]any{
		"error":   errorType,
		"message": message,
	}

	if details != nil {
		response["details"] = details
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Не удалось сформировать ответ с ошибкой", "error", err)
	}
}
