package customer

import (
	"strings"
	"testing"
)

func TestNormalizeListParams(t *testing.T) {
	params := normalizeListParams(ListParams{Page: -1, Limit: 999, Query: " demo "})
	if params.Page != 1 {
		t.Fatalf("Page = %d, want 1", params.Page)
	}
	if params.Limit != 100 {
		t.Fatalf("Limit = %d, want 100", params.Limit)
	}
	if params.Query != "demo" {
		t.Fatalf("Query = %q, want demo", params.Query)
	}
}

func TestNormalizeListParamsDefaultsToRegisteredOwners(t *testing.T) {
	params := normalizeListParams(ListParams{OwnerKind: ""})
	if params.OwnerKind != OwnerKindRegistered {
		t.Fatalf("OwnerKind = %q, want %q", params.OwnerKind, OwnerKindRegistered)
	}
}

func TestNormalizeListParamsAcceptsNonRegisterOwnerKind(t *testing.T) {
	params := normalizeListParams(ListParams{OwnerKind: "non-register"})
	if params.OwnerKind != OwnerKindNonRegister {
		t.Fatalf("OwnerKind = %q, want %q", params.OwnerKind, OwnerKindNonRegister)
	}
}

func TestOwnerWhereSeparatesRegisteredAndNonRegisterOwners(t *testing.T) {
	registeredWhere, _ := ownerWhere(Actor{RoleCode: RoleAdmin}, normalizeListParams(ListParams{}))
	if !strings.Contains(registeredWhere, "o.code NOT LIKE 'NONREG-%'") {
		t.Fatalf("registered where = %q, want NONREG exclusion", registeredWhere)
	}

	nonRegisterWhere, _ := ownerWhere(Actor{RoleCode: RoleAdmin}, normalizeListParams(ListParams{OwnerKind: OwnerKindNonRegister}))
	if !strings.Contains(nonRegisterWhere, "o.code LIKE 'NONREG-%'") {
		t.Fatalf("non-register where = %q, want NONREG inclusion", nonRegisterWhere)
	}
	if !strings.Contains(nonRegisterWhere, "registered_owner.code NOT LIKE 'NONREG-%'") {
		t.Fatalf("non-register where = %q, want registered phone duplicate exclusion", nonRegisterWhere)
	}
	if !strings.Contains(nonRegisterWhere, "registered_owner.phone = o.phone") {
		t.Fatalf("non-register where = %q, want phone match duplicate exclusion", nonRegisterWhere)
	}
	if !strings.Contains(nonRegisterWhere, "o.phone <> ''") {
		t.Fatalf("non-register where = %q, want empty phone exclusion", nonRegisterWhere)
	}
}

func TestNormalizeIDs(t *testing.T) {
	ids, err := normalizeIDs([]int64{3, 1, 3, 2})
	if err != nil {
		t.Fatalf("normalizeIDs error = %v", err)
	}
	want := []int64{3, 1, 2}
	if len(ids) != len(want) {
		t.Fatalf("len(ids) = %d, want %d", len(ids), len(want))
	}
	for index := range want {
		if ids[index] != want[index] {
			t.Fatalf("ids[%d] = %d, want %d", index, ids[index], want[index])
		}
	}
}

func TestNormalizeIDsRejectsInvalid(t *testing.T) {
	if _, err := normalizeIDs([]int64{1, 0}); err != ErrEmptyBulk {
		t.Fatalf("err = %v, want ErrEmptyBulk", err)
	}
}
