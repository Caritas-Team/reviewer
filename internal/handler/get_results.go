package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/usecase/assessment"
	"github.com/google/uuid"
)

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
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
		writeJSON(w, http.StatusOK, map[string]any{
			"status":             "processing",
			"progress_percent":   res.ProgressPercent,
			"processed_students": res.ProcessedStudents,
			"total_students":     res.TotalStudents,
		})
	case "completed":
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "completed",
			"results": res.Results,
		})
	case "failed":
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "failed",
			"error":  res.Error,
		})
	default:
		writeError(w, http.StatusInternalServerError, "Unknown status")
	}
}

func getOrWriteError(w http.ResponseWriter, log *logger.Logger, err error, requestID string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, assessment.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Result not found")
		return true
	}
	log.Error("assessment results get failed", "request_id", requestID, "err", err)
	writeError(w, http.StatusInternalServerError, "Internal server error")
	return true
}

func GetAssessmentResultsHandler(storage *assessment.ResultStorage, log *logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		const prefix = "/v1/assessments/"
		path := r.URL.Path
		if !strings.HasPrefix(path, prefix) {
			writeError(w, http.StatusBadRequest, "Invalid path")
			return
		}

		requestID := strings.Trim(strings.TrimPrefix(path, prefix), "/")
		if requestID == "" || strings.Contains(requestID, "/") {
			writeError(w, http.StatusBadRequest, "Missing or invalid request_id")
			return
		}

		if _, err := uuid.Parse(requestID); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request_id format")
			return
		}

		wait, err := parseBoolQuery(r, "wait")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid wait parameter")
			return
		}

		keepInCache, err := parseBoolQuery(r, "keep_in_cache")
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid keep_in_cache parameter")
			return
		}

		ctx := r.Context()

		res, err := storage.Get(ctx, requestID)
		if getOrWriteError(w, log, err, requestID) {
			return
		}

		if wait && res.Status == "processing" {
			lpCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			ticker := time.NewTicker(300 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-lpCtx.Done():
					respond(w, res)
					return
				case <-ticker.C:
					tmp, err := storage.Get(lpCtx, requestID)
					if errors.Is(err, assessment.ErrNotFound) {
						writeError(w, http.StatusNotFound, "Result not found")
						return
					}
					if err != nil {
						log.Error("assessment results long poll get failed", "request_id", requestID, "err", err)
						writeError(w, http.StatusInternalServerError, "Internal server error")
						return
					}

					res = tmp
					if res.Status != "processing" {
						final, err := storage.GetAndDelete(ctx, requestID, keepInCache)
						if getOrWriteError(w, log, err, requestID) {
							return
						}
						res = final
						respond(w, res)
						return
					}
				}
			}
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
