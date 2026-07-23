package activity

import (
	"database/sql"
	"testing"
)

func TestRemarkOneDoesNotDowngradePotential(t *testing.T) {
	current := LeadState{
		Stage:        StagePotential,
		Status:       StatusOpen,
		CurrentScore: sql.NullInt64{Int64: 2, Valid: true},
	}

	next, err := applyRemarkPolicy(current, 1)
	if err != nil {
		t.Fatalf("apply remark policy: %v", err)
	}
	if next.Stage != StagePotential {
		t.Fatalf("stage turun menjadi %s, ingin tetap %s", next.Stage, StagePotential)
	}
	if !next.Score.Valid || next.Score.Int64 != 2 {
		t.Fatalf("score turun menjadi %+v, ingin tetap 2", next.Score)
	}
	if leadStateChanged(current, next) {
		t.Fatal("remark 1 pada potential tidak boleh membentuk perubahan stage")
	}
}

func TestRemarkZeroInvalidatesLead(t *testing.T) {
	current := LeadState{
		Stage:        StagePossible,
		Status:       StatusOpen,
		CurrentScore: sql.NullInt64{Int64: 1, Valid: true},
	}

	next, err := applyRemarkPolicy(current, 0)
	if err != nil {
		t.Fatalf("apply remark policy: %v", err)
	}
	if next.Stage != StageInvalid || next.Status != StatusInvalid {
		t.Fatalf("remark 0 menghasilkan stage/status %s/%s", next.Stage, next.Status)
	}
	if !next.Score.Valid || next.Score.Int64 != 0 {
		t.Fatalf("remark 0 menghasilkan score %+v", next.Score)
	}
}
