package closing

import (
	"errors"
	"testing"
)

func TestValidateCreateClosingRequest(t *testing.T) {
	req := CreateClosingRequest{
		PlanID:             1,
		DiscountAmount:     "10000.00",
		UniqueTransferCode: intPtr(321),
		InteractionType:    "CALL",
	}
	if err := validateCreateClosingRequest(req); err != nil {
		t.Fatalf("validate request: %v", err)
	}
}

func TestValidateCreateClosingRequestRejectsBadPromotionID(t *testing.T) {
	badPromotionID := int64(0)
	err := validateCreateClosingRequest(CreateClosingRequest{PlanID: 1, PromotionID: &badPromotionID})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
}

func TestValidateCreateClosingRequestRejectsRemarkInteractionType(t *testing.T) {
	err := validateCreateClosingRequest(CreateClosingRequest{PlanID: 1, InteractionType: "VISIT"})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
}

func intPtr(value int) *int {
	return &value
}
