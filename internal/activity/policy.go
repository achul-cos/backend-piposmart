package activity

import "database/sql"

type remarkPolicyResult struct {
	Stage  string
	Status string
	Score  sql.NullInt64
}

func applyRemarkPolicy(current LeadState, remarkScore int64) (remarkPolicyResult, error) {
	if remarkScore < 0 || remarkScore > 3 {
		return remarkPolicyResult{}, ErrInvalidScore
	}

	result := remarkPolicyResult{
		Stage:  current.Stage,
		Status: current.Status,
		Score:  current.CurrentScore,
	}
	if result.Stage == "" {
		result.Stage = StageNew
	}
	if result.Status == "" {
		result.Status = StatusOpen
	}

	switch remarkScore {
	case 0:
		result.Stage = StageInvalid
		result.Status = StatusInvalid
		result.Score = sql.NullInt64{Int64: 0, Valid: true}
	case 1:
		if isStickyPotential(current) {
			return result, nil
		}
		result.Stage = StagePossible
		result.Status = StatusOpen
		result.Score = sql.NullInt64{Int64: 1, Valid: true}
	case 2:
		result.Stage = StagePotential
		result.Status = StatusOpen
		result.Score = sql.NullInt64{Int64: 2, Valid: true}
	case 3:
		result.Stage = StageClosing
		result.Status = StatusOpen
		result.Score = sql.NullInt64{Int64: 3, Valid: true}
	}
	return result, nil
}

func isStickyPotential(current LeadState) bool {
	if current.Stage == StagePotential || current.Stage == StageClosing {
		return true
	}
	return current.CurrentScore.Valid && current.CurrentScore.Int64 >= 2
}

func leadStateChanged(current LeadState, next remarkPolicyResult) bool {
	if current.Stage != next.Stage || current.Status != next.Status {
		return true
	}
	if current.CurrentScore.Valid != next.Score.Valid {
		return true
	}
	return current.CurrentScore.Valid && current.CurrentScore.Int64 != next.Score.Int64
}
