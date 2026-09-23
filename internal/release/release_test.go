package release

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestV2ReleaseLayout(t *testing.T) {
	root := repositoryRoot(t)

	module := readFile(t, filepath.Join(root, "go.mod"))
	if !strings.Contains(module, "module github.com/goairix/wx/v2\n") {
		t.Fatalf("go.mod does not declare the v2 module: %q", module)
	}
	if !strings.Contains(module, "\ngo 1.17\n") {
		t.Fatalf("go.mod does not preserve the Go 1.17 baseline: %q", module)
	}

	required := []string{
		"README.md",
		"MIGRATION.md",
		"official/README.md",
		"miniapp/README.md",
		"work/README.md",
		"openplatform/README.md",
		"mobileapp/README.md",
		"healthcard/README.md",
		"core/transport",
		"core/cache",
		"core/errors",
	}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("required release path %q: %v", name, err)
		}
	}

	removed := []string{
		"app",
		"base",
		"health_card",
		"kernel",
		"mini_program",
		"open_platform",
		"open_work",
		"support",
		"work/account_id",
		"work/mini_program",
	}
	for _, name := range removed {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("legacy path %q still exists", name)
		}
	}
}

func TestMigrationGuideCoversBreakingPathChanges(t *testing.T) {
	root := repositoryRoot(t)
	guide := readFile(t, filepath.Join(root, "MIGRATION.md"))

	requiredMappings := []string{
		"mini_program",
		"miniapp",
		"app",
		"mobileapp",
		"health_card",
		"healthcard",
		"open_platform",
		"openplatform",
		"account_id",
		"accountid",
		"support/http",
		"core/transport",
		"context.Context",
		"NewClient",
	}
	for _, mapping := range requiredMappings {
		if !strings.Contains(guide, mapping) {
			t.Errorf("MIGRATION.md does not cover %q", mapping)
		}
	}
}

func TestReleaseHasNoRetiredErrorDependency(t *testing.T) {
	root := repositoryRoot(t)
	for _, name := range []string{"go.mod", "go.sum"} {
		contents := readFile(t, filepath.Join(root, name))
		retiredImport := "github.com/pkg/" + "errors"
		if strings.Contains(contents, retiredImport) {
			t.Errorf("%s still contains retired dependency %q", name, retiredImport)
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate release test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "../.."))
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	contents, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(contents)
}
