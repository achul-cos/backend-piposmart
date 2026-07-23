package logging

import (
	"testing"

	"backend_crm_piposmart/internal/platform/config"
)

func TestNewRejectsInvalidLevel(t *testing.T) {
	_, err := New(config.LogConfig{Level: "verbose", Format: "json"})
	if err == nil {
		t.Fatal("New() seharusnya menolak level tidak valid")
	}
}

func TestNewRejectsInvalidFormat(t *testing.T) {
	_, err := New(config.LogConfig{Level: "info", Format: "xml"})
	if err == nil {
		t.Fatal("New() seharusnya menolak format tidak valid")
	}
}
