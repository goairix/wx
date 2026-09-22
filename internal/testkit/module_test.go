package testkit

import (
	"os"
	"strings"
	"testing"
)

func TestModulePathIsV2(t *testing.T) {
	data, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	if !strings.Contains(string(data), "module github.com/goairix/wx/v2") {
		t.Fatalf("go.mod does not declare module github.com/goairix/wx/v2")
	}
}
