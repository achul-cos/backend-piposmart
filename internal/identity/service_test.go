package identity

import "testing"

func TestValidatePasswordRejectsWeakPassword(t *testing.T) {
	if err := validatePassword("short"); err == nil {
		t.Fatal("password pendek seharusnya ditolak")
	}
}

func TestHasPermission(t *testing.T) {
	user := User{Permissions: []string{"users.read", "users.manage_sales"}}
	if !hasPermission(user, "users.manage_sales") {
		t.Fatal("permission seharusnya ditemukan")
	}
	if hasPermission(user, "users.manage_all") {
		t.Fatal("permission yang tidak dimiliki seharusnya false")
	}
}
