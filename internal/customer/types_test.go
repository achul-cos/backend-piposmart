package customer

import "testing"

func TestOwnerOverviewSubscriptionLabel(t *testing.T) {
	cases := map[string]string{
		"SUBSCRIBE":      "BERLANGGANAN",
		"berlangganan":   "BERLANGGANAN",
		"ACTIVE":         "BERLANGGANAN",
		"TRIAL":          "NOT_SUBSCRIBE",
		"NOT_SUBSCRIBE":  "NOT_SUBSCRIBE",
		"":               "NOT_SUBSCRIBE",
	}

	for input, want := range cases {
		if got := ownerOverviewSubscriptionLabel(input); got != want {
			t.Fatalf("ownerOverviewSubscriptionLabel(%q) = %q, want %q", input, got, want)
		}
	}
}
