package password

import "testing"

func TestArgon2idHashCanBeVerified(t *testing.T) {
	hash, err := HashArgon2id("Password123!")
	if err != nil {
		t.Fatalf("HashArgon2id() error = %v", err)
	}

	ok, err := VerifyArgon2id("Password123!", hash)
	if err != nil {
		t.Fatalf("VerifyArgon2id() error = %v", err)
	}
	if !ok {
		t.Fatal("password seharusnya valid")
	}
}

func TestArgon2idRejectsWrongPassword(t *testing.T) {
	hash := HashArgon2idWithSalt("Password123!", []byte("1234567890abcdef"))

	ok, err := VerifyArgon2id("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyArgon2id() error = %v", err)
	}
	if ok {
		t.Fatal("password salah seharusnya ditolak")
	}
}
