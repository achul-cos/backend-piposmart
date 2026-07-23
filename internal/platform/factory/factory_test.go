package factory

import (
	"testing"
	"time"
)

func TestFactoryBuildsDeterministicOwners(t *testing.T) {
	asOf := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	first := New(20260723, asOf).BuildOwner(1)
	second := New(20260723, asOf).BuildOwner(1)

	if first != second {
		t.Fatalf("owner factory tidak deterministik\nfirst=%+v\nsecond=%+v", first, second)
	}
}

func TestFactoryBuildsDeterministicPasswordHash(t *testing.T) {
	asOf := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	first := New(20260723, asOf).BuildUser("SALES", 1)
	second := New(20260723, asOf).BuildUser("SALES", 1)

	if first.PasswordHash != second.PasswordHash {
		t.Fatal("hash dummy seed harus deterministik")
	}
	if first.PasswordHash == DummyPassword {
		t.Fatal("password dummy tidak boleh disimpan plain text")
	}
}
