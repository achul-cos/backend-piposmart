package closing

import (
	"errors"
	"testing"
)

func TestValidateCreateClosingRequestAcceptsCallAndChatStatuses(t *testing.T) {
	req := CreateClosingRequest{
		PlanID:         1,
		DiscountAmount: "0",
		CallStatus:     "TERHUBUNG",
		ChatStatus:     "TERBALAS",
	}
	if err := validateCreateClosingRequest(req); err != nil {
		t.Fatalf("validateCreateClosingRequest: %v", err)
	}
}

func TestValidateCreateClosingRequestRejectsNoInteractionChannel(t *testing.T) {
	err := validateCreateClosingRequest(CreateClosingRequest{
		PlanID:         1,
		DiscountAmount: "0",
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
}
