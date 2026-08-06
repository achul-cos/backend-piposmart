package seeder

import "testing"

func TestNormalizeSalesName(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"CS 2 - Lidya", "Lidya"},
		{"CS 4 RISKY", "Risky"},
		{"CS 4 Risky", "Risky"},
		{"CS 2 CINDY", "Cindy"},
		{"CS 1- Septi", "Septi"},
		{"CS 1 MAULI", "Mauli"},
		{"CS 6 YULI", "Yuli"},
		{"CS 3 KRISTINA", "Kristina"},
		{"Magdalena ( CS 5 )", "Magdalena"},
		{"Kristina ( CS 3 )", "Kristina"},
		{"Meta", "Meta"},
		{"Perusahaan", "Perusahaan"},
		{"Akun Testing", "Akun Testing"},
		{"Akun Testing Clipper", "Akun Testing Clipper"},
		{"Akun Hilang", "Akun Hilang"},
		{"Akun Karyawan", "Akun Karyawan"},
		{"CS 2", "Cs 2"},
		{"TRAINEE CS 1", "Trainee Cs 1"},
		{"TRAINEE CS 4", "Trainee Cs 4"},
		{"  CS 2   -   Lidya  ", "Lidya"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := normalizeSalesName(tc.raw); got != tc.want {
			t.Errorf("normalizeSalesName(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestNormalizeSalesName_SamePersonDifferentSlotsCollide(t *testing.T) {
	// "CS 4 RISKY" and a hypothetical later re-format of the same person under a different
	// dash convention should collapse to the same identity key.
	a := salesIdentityKey("CS 4 - Risky")
	b := salesIdentityKey("CS 4 RISKY")
	if a != b {
		t.Fatalf("expected same identity key, got %q vs %q", a, b)
	}
}

func TestNormalizeSalesName_DifferentPeopleSameSlotStayDistinct(t *testing.T) {
	// Real turnover pattern: slot "CS 4" held by "Wati" in the PIC column but "Risky" in a
	// later Share column — these must NOT collapse into one identity.
	wati := salesIdentityKey("CS 4 - Wati")
	risky := salesIdentityKey("CS 4 RISKY")
	if wati == risky {
		t.Fatalf("expected distinct identity keys for different people, both got %q", wati)
	}
}
