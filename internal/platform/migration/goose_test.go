package migration

import "testing"

func TestIsSupportedCommand(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"up":      true,
		"down":    true,
		"reset":   true,
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
