package customer

import "testing"

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"0812-3456-7890":  "6281234567890",
		"+62 812 345 678": "62812345678",
		"81234567890":     "6281234567890",
		"":                "",
	}

	for input, want := range cases {
		got, err := NormalizePhone(input)
		if err != nil {
			t.Fatalf("NormalizePhone(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizePhone(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizePhoneRejectsInvalidPhone(t *testing.T) {
	if _, err := NormalizePhone("abc"); err == nil {
		t.Fatal("phone invalid seharusnya ditolak")
	}
}
