package containercheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

type composeFile struct {
	Services map[string]any `yaml:"services"`
}

func TestComposeFileIsValidAndContainsSprintOneServices(t *testing.T) {
	root := repositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "compose.yaml"))
	if err != nil {
		t.Fatalf("read compose.yaml: %v", err)
	}

	var compose composeFile
	if err := yaml.Unmarshal(content, &compose); err != nil {
		t.Fatalf("parse compose.yaml: %v", err)
	}

	for _, service := range []string{"mysql", "migrate", "api", "worker", "seed"} {
		if _, exists := compose.Services[service]; !exists {
			t.Errorf("compose service %q tidak ditemukan", service)
		}
	}
}

func TestDockerfileUsesMultiStageAndNonRootRuntime(t *testing.T) {
	root := repositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "Dockerfile"))
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}
	dockerfile := string(content)

	required := []string{
		"FROM golang:1.26.5-alpine AS builder",
		"FROM alpine:3.22",
		"USER crm",
		"ENTRYPOINT [\"/app/crm\"]",
		"HEALTHCHECK",
	}
	for _, item := range required {
		if !strings.Contains(dockerfile, item) {
			t.Errorf("Dockerfile tidak memiliki %q", item)
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	return filepath.Clean(filepath.Join(workingDirectory, "..", "..", ".."))
}
