package assessment

import (
	"context"
	"fmt"

	"github.com/Caritas-Team/reviewer/internal/model"
)

type Processor struct {
	calc *DiffCalculator
}

func NewProcessor(calc *DiffCalculator) *Processor {
	return &Processor{calc: calc}
}

func (p Processor) Process(ctx context.Context, pair model.StudentPair) (AssessmentDiff, error) {
	_ = ctx

	if pair.Before == nil || pair.After == nil {
		return AssessmentDiff{}, fmt.Errorf("student %s: missing before/after document", pair.StudentID)
	}

	diff, err := p.calc.Calculate(pair.Before, pair.After)
	if err != nil {
		return AssessmentDiff{}, fmt.Errorf("student %s: %w", pair.StudentID, err)
	}

	return diff, nil
}
