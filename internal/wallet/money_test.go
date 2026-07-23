package wallet

import "testing"

func TestParsePositiveMoneyToCents(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{name: "whole number", input: "1000000", want: 100000000},
		{name: "two decimals", input: "1250000.50", want: 125000050},
		{name: "one decimal", input: "10.5", want: 1050},
		{name: "zero rejected", input: "0", wantErr: true},
		{name: "three decimals rejected", input: "10.123", wantErr: true},
		{name: "negative rejected", input: "-1.00", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePositiveMoneyToCents(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestApplyBalanceRejectsNegativeDebit(t *testing.T) {
	_, err := applyBalance(100_000, 150_000, DirectionDebit)
	if err == nil {
		t.Fatalf("expected insufficient balance error")
	}
	if err != ErrInsufficientBalance {
		t.Fatalf("got %v, want %v", err, ErrInsufficientBalance)
	}
}

func TestTopupIdempotencyKeyRequiresKeyOrExternalReference(t *testing.T) {
	if _, err := topupIdempotencyKey(CreateTopupRequest{}); err != ErrIdempotencyRequired {
		t.Fatalf("got %v, want %v", err, ErrIdempotencyRequired)
	}

	key, err := topupIdempotencyKey(CreateTopupRequest{IdempotencyKey: "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "topup:abc" {
		t.Fatalf("got %q", key)
	}

	key, err = topupIdempotencyKey(CreateTopupRequest{ExternalReference: "EXT-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "topup:external:EXT-001" {
		t.Fatalf("got %q", key)
	}
}

func TestTransactionIdempotencyKeyScopedBySourceType(t *testing.T) {
	req := CreateWalletTransactionRequest{IdempotencyKey: "same-key"}
	debit, err := transactionIdempotencyKey(SourceManualDebit, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	refund, err := transactionIdempotencyKey(SourceRefund, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if debit == refund {
		t.Fatalf("expected source scoped idempotency keys to differ")
	}
}
