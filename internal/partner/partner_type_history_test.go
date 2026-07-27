package partner

import (
	"testing"

	"backend_crm_piposmart/internal/identity"
)

func TestCanManagePartnerType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		actor identity.User
		want  bool
	}{
		{
			name:  "admin can manage partner type commission master",
			actor: identity.User{RoleCode: identity.RoleAdmin},
			want:  true,
		},
		{
			name:  "supervisor cannot manage partner type commission master",
			actor: identity.User{RoleCode: identity.RoleSupervisor},
			want:  false,
		},
		{
			name:  "sales cannot manage partner type commission master",
			actor: identity.User{RoleCode: identity.RoleSales},
			want:  false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := canManagePartnerType(tc.actor); got != tc.want {
				t.Fatalf("canManagePartnerType() = %v, want %v", got, tc.want)
			}
		})
	}
}
