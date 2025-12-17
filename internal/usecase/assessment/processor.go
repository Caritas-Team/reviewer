package assessment

import (
	"context"

	"github.com/Caritas-Team/reviewer/internal/model"
)

type StubProcessor struct{}

func (StubProcessor) Process(ctx context.Context, pair model.StudentPair) (AssessmentDiff, error) {
	return AssessmentDiff{
		StudentID: pair.StudentID,
		Diffs:     nil,
	}, nil
}
