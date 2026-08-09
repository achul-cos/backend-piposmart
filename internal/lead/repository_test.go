package lead

import (
	"strings"
	"testing"

	"backend_crm_piposmart/internal/identity"
)

func TestLeadWhereExcludesTestingAccounts(t *testing.T) {
	where, _ := leadWhere(identity.User{RoleCode: RoleAdmin}, ListParams{})
	if !strings.Contains(where, "o.is_testing_account = 0") {
		t.Fatalf("where = %q, want testing-account exclusion", where)
	}
}
