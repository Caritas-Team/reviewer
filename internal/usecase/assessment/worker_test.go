package assessment

import (
	"context"
	"testing"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/Caritas-Team/reviewer/internal/logger"
	"github.com/Caritas-Team/reviewer/internal/model"
	"github.com/google/uuid"
)

// stub processor
type stubProc struct{}

func (stubProc) Process(ctx context.Context, pair model.StudentPair) (AssessmentDiff, error) {
	return AssessmentDiff{
		StudentID:   pair.StudentID,
		PeriodStart: "2025-12-18",
		PeriodEnd:   "2025-12-19",
	}, nil
}

func TestWorker_Run_Success(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Config{Logging: config.Logging{Level: "debug", Format: "json"}}
	log := logger.NewLogger(cfg)

	// newMemCacheMock() должен быть объявлен в cache_mock_test.go
	cache := newMemCacheMock()
	st := NewResultStorage(cache)

	input := make(chan []model.StudentPair, 1)

	ttl := 5 * time.Minute
	w := NewWorker(log, st, stubProc{}, ttl)
	go w.Run(ctx, input)

	reqID := uuid.NewString()

	input <- []model.StudentPair{
		{
			RequestID: reqID,
			StudentID: "S1",
			Before: &model.AssessmentDocument{
				Metadata: model.AssessmentMetadata{
					StudentID:      "S1",
					Date:           time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC),
					AssessmentType: "before",
				},
				CommunicativeFuncs: model.CommunicativeFunctions{
					Control:             40.0,
					ObtainingDesired:    50.0,
					SocialInteraction:   60.0,
					InformationExchange: 70.0,
				},
			},
			After: &model.AssessmentDocument{
				Metadata: model.AssessmentMetadata{
					StudentID:      "S1",
					Date:           time.Date(2025, 12, 19, 0, 0, 0, 0, time.UTC),
					AssessmentType: "after",
				},
				CommunicativeFuncs: model.CommunicativeFunctions{
					Control:             45.0,
					ObtainingDesired:    55.0,
					SocialInteraction:   65.0,
					InformationExchange: 75.0,
				},
			},
		},
		{
			RequestID: reqID,
			StudentID: "S2",
			Before: &model.AssessmentDocument{
				Metadata: model.AssessmentMetadata{
					StudentID:      "S2",
					Date:           time.Date(2025, 12, 18, 0, 0, 0, 0, time.UTC),
					AssessmentType: "before",
				},
				CommunicativeFuncs: model.CommunicativeFunctions{
					Control:             30.0,
					ObtainingDesired:    40.0,
					SocialInteraction:   50.0,
					InformationExchange: 60.0,
				},
			},
			After: &model.AssessmentDocument{
				Metadata: model.AssessmentMetadata{
					StudentID:      "S2",
					Date:           time.Date(2025, 12, 19, 0, 0, 0, 0, time.UTC),
					AssessmentType: "after",
				},
				CommunicativeFuncs: model.CommunicativeFunctions{
					Control:             35.0,
					ObtainingDesired:    45.0,
					SocialInteraction:   55.0,
					InformationExchange: 65.0,
				},
			},
		},
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		res, err := st.Get(ctx, reqID)
		if err == nil && res.Status == "completed" {
			if res.TotalStudents != 2 {
				t.Fatalf("TotalStudents=%d, want 2", res.TotalStudents)
			}
			if res.ProcessedStudents != 2 {
				t.Fatalf("ProcessedStudents=%d, want 2", res.ProcessedStudents)
			}
			if res.ProgressPercent != 100 {
				t.Fatalf("ProgressPercent=%d, want 100", res.ProgressPercent)
			}
			if len(res.Results) != 2 {
				t.Fatalf("len(Results)=%d, want 2", len(res.Results))
			}
			if res.Results[0].StudentID != "S1" || res.Results[1].StudentID != "S2" {
				t.Fatalf("unexpected results order: %+v", res.Results)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timeout waiting for completed")
}
