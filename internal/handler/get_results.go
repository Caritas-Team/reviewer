package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/usecase/assessment"
	"github.com/google/uuid"
)

const assessmentsPrefix = "/v1/assessments/"

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type AssessmentProcessingResponse struct {
	Status            string `json:"status"`
	ProgressPercent   int    `json:"progress_percent"`
	ProcessedStudents int    `json:"processed_students"`
	TotalStudents     int    `json:"total_students"`
}

type AssessmentCompletedResponse struct {
	Status        string                      `json:"status"`
	Results       []assessment.AssessmentDiff `json:"results"`
	GroupAverages []assessment.GroupAverage   `json:"group_averages,omitempty"`
	GroupProgress []assessment.GroupProgress  `json:"group_progress,omitempty"`
	GroupDiff     *assessment.GroupDiff       `json:"group_diff,omitempty"`
}

type AssessmentFailedResponse struct {
	Status string `json:"status"`
	Error  any    `json:"error"`
}

func extractRequestID(r *http.Request) string {
	if v := r.PathValue("request_id"); v != "" {
		return v
	}
	path := r.URL.Path
	if !strings.HasPrefix(path, assessmentsPrefix) {
		return ""
	}
	v := strings.TrimPrefix(path, assessmentsPrefix)
	v = strings.Trim(v, "/")
	if i := strings.IndexByte(v, '/'); i >= 0 {
		return ""
	}
	return v
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	var resp ErrorResponse
	resp.Error.Message = msg
	writeJSON(w, code, resp)
}

func parseBoolQuery(r *http.Request, name string) (bool, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return false, nil
	}
	return strconv.ParseBool(raw)
}

func respond(w http.ResponseWriter, res *assessment.ProcessingResult) {
	switch res.Status {
	case "processing":
		writeJSON(w, http.StatusOK, AssessmentProcessingResponse{
			Status:            "processing",
			ProgressPercent:   res.ProgressPercent,
			ProcessedStudents: res.ProcessedStudents,
			TotalStudents:     res.TotalStudents,
		})
	case "completed":
		writeJSON(w, http.StatusOK, AssessmentCompletedResponse{
			Status:        "completed",
			Results:       res.Results,
			GroupAverages: res.GroupAverages,
			GroupProgress: res.GroupProgress,
			GroupDiff:     res.GroupDiff,
		})
	case "failed":
		writeJSON(w, http.StatusOK, AssessmentFailedResponse{
			Status: "failed",
			Error:  res.Error,
		})
	default:
		writeError(w, http.StatusInternalServerError, "Что-то пошло не так")
	}
}

func getOrWriteError(w http.ResponseWriter, log *logger.Logger, err error, requestID string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, assessment.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Результат не найден")
		return true
	}
	log.Error("assessment results get failed", "request_id", requestID, "err", err)
	writeError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
	return true
}

func GetAssessmentResultsHandler(storage *assessment.ResultStorage, log *logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		requestID := extractRequestID(r)
		if requestID == "" {
			writeError(w, http.StatusBadRequest, "Отсутствует request_id")
			return
		}
		if _, err := uuid.Parse(requestID); err != nil {
			writeError(w, http.StatusBadRequest, "Неверный формат request_id")
			return
		}

		keepInCache, err := parseBoolQuery(r, "keep_in_cache")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Неверный параметр keep_in_cache")
			return
		}

		ctx := r.Context()

		res, err := storage.Get(ctx, requestID)
		if getOrWriteError(w, log, err, requestID) {
			return
		}

		if res.Status == "completed" || res.Status == "failed" {
			final, err := storage.GetAndDelete(ctx, requestID, keepInCache)
			if getOrWriteError(w, log, err, requestID) {
				return
			}
			res = final
		}
		respond(w, res)
	}
}
