package target

import "testing"

func TestVisibilityWhere(t *testing.T) {
	cases := []struct {
		name     string
		role     string
		wantArgs int
	}{
		{"admin sees all", RoleAdmin, 0},
		{"supervisor sees all", RoleSupervisor, 0},
		{"sales scoped to self", RoleSales, 1},
		{"unknown role denied", "UNKNOWN", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clause, args := visibilityWhere(1, tc.role)
			if clause == "" {
				t.Fatal("expected non-empty clause")
			}
			if len(args) != tc.wantArgs {
				t.Fatalf("expected %d args, got %d", tc.wantArgs, len(args))
			}
			if tc.role == "UNKNOWN" && clause != "1 = 0" {
				t.Fatalf("expected deny-all clause for unknown role, got %q", clause)
			}
		})
	}
}
