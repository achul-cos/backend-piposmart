package migration

import "testing"

func TestIsSupportedCommand(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"up":      true,
		"down":    true,
		"reset":   true,
		"clear":   true,
		"status":  true,
		"version": true,
		"redo":    false,
		"":        false,
	}

	for command, expected := range cases {
		command := command
		expected := expected

		t.Run(command, func(t *testing.T) {
			t.Parallel()
			if actual := isSupportedCommand(command); actual != expected {
				t.Fatalf("isSupportedCommand(%q) = %v, want %v", command, actual, expected)
			}
		})
	}
}

func TestIsClearPreservedTable(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"goose_db_version":      true,
		"goose_lock":            true,
		"seed_runs":             false,
		"subscription_packages": false,
		"users":                 false,
	}

	for table, expected := range cases {
		table := table
		expected := expected

		t.Run(table, func(t *testing.T) {
			t.Parallel()
			if actual := isClearPreservedTable(table); actual != expected {
				t.Fatalf("isClearPreservedTable(%q) = %v, want %v", table, actual, expected)
			}
		})
	}
}
