package importing

import (
	"errors"
	"testing"
)

func TestValidateCommitStatus(t *testing.T) {
	t.Parallel()

	t.Run("validated is allowed", func(t *testing.T) {
		t.Parallel()
		if err := validateCommitStatus(BatchStatusValidated); err != nil {
			t.Fatalf("validateCommitStatus(validated) error = %v", err)
		}
	})

	t.Run("committing is idempotently allowed", func(t *testing.T) {
		t.Parallel()
		if err := validateCommitStatus(BatchStatusCommitting); err != nil {
			t.Fatalf("validateCommitStatus(committing) error = %v", err)
		}
	})

	t.Run("uploaded returns detailed invalid batch status", func(t *testing.T) {
		t.Parallel()
		err := validateCommitStatus(BatchStatusUploaded)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrInvalidBatchStatus) {
			t.Fatalf("errors.Is(err, ErrInvalidBatchStatus) = false, err = %v", err)
		}
		var statusErr *BatchStatusActionError
		if !errors.As(err, &statusErr) {
			t.Fatalf("errors.As(err, *BatchStatusActionError) = false, err = %v", err)
		}
		if statusErr.Action != "commit" {
			t.Fatalf("action = %q", statusErr.Action)
		}
		if statusErr.CurrentStatus != BatchStatusUploaded {
			t.Fatalf("current_status = %q", statusErr.CurrentStatus)
		}
		if len(statusErr.AllowedStatuses) != 3 {
			t.Fatalf("allowed_statuses len = %d", len(statusErr.AllowedStatuses))
		}
	})
}

func TestCanReuseCommitResult(t *testing.T) {
	t.Parallel()
	if !canReuseCommitResult(BatchStatusCommitting) {
		t.Fatal("expected COMMITTING to be reusable")
	}
	if !canReuseCommitResult(BatchStatusCommitted) {
		t.Fatal("expected COMMITTED to be reusable")
	}
	if canReuseCommitResult(BatchStatusValidated) {
		t.Fatal("did not expect VALIDATED to be reusable")
	}
}
