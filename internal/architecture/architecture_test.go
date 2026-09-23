package architecture

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const modulePath = "github.com/goairix/wx/v2"

var legacyRoots = []string{
	"app",
	"base",
	"kernel",
	"mini_program",
	"open_platform",
	"open_work",
	"support",
}

var platforms = map[string]bool{
	"official":     true,
	"miniapp":      true,
	"work":         true,
	"openplatform": true,
	"mobileapp":    true,
	"healthcard":   true,
}

type listedPackage struct {
	ImportPath string
	Imports    []string
}

func TestPackageArchitecture(t *testing.T) {
	root := repositoryRoot(t)
	packages := listPackages(t, root)

	var violations []string
	for _, legacy := range legacyRoots {
		if _, err := os.Stat(filepath.Join(root, legacy)); !os.IsNotExist(err) {
			violations = append(violations, "legacy directory still exists: "+legacy)
		}
	}

	for _, current := range packages {
		if legacyRoot(current.ImportPath) != "" {
			violations = append(
				violations,
				"legacy package still listed: "+current.ImportPath,
			)
		}
		for _, imported := range current.Imports {
			if legacy := legacyRoot(imported); legacy != "" {
				violations = append(
					violations,
					fmt.Sprintf("%s imports legacy root %s", current.ImportPath, legacy),
				)
			}
			if strings.HasPrefix(current.ImportPath, modulePath+"/core/") &&
				platformRoot(imported) != "" {
				violations = append(
					violations,
					fmt.Sprintf("core package %s imports platform %s", current.ImportPath, imported),
				)
			}
			if crossesDomainBoundary(current.ImportPath, imported) {
				violations = append(
					violations,
					fmt.Sprintf("domain package %s imports sibling domain %s", current.ImportPath, imported),
				)
			}
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("architecture violations:\n- %s", strings.Join(violations, "\n- "))
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve architecture test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func listPackages(t *testing.T, root string) []listedPackage {
	t.Helper()
	command := exec.Command("go", "list", "-json", "./...")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("go list ./...: %v\n%s", err, exitErr.Stderr)
		}
		t.Fatalf("go list ./...: %v", err)
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))
	var packages []listedPackage
	for decoder.More() {
		var current listedPackage
		if err := decoder.Decode(&current); err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		packages = append(packages, current)
	}
	return packages
}

func legacyRoot(importPath string) string {
	for _, legacy := range legacyRoots {
		prefix := modulePath + "/" + legacy
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return legacy
		}
	}
	return ""
}

func platformRoot(importPath string) string {
	if !strings.HasPrefix(importPath, modulePath+"/") {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(importPath, modulePath+"/"), "/")
	if len(parts) > 0 && platforms[parts[0]] {
		return parts[0]
	}
	return ""
}

func crossesDomainBoundary(packagePath, importedPath string) bool {
	platform := platformRoot(packagePath)
	if platform == "" || platformRoot(importedPath) != platform {
		return false
	}

	packageParts := strings.Split(strings.TrimPrefix(packagePath, modulePath+"/"), "/")
	importParts := strings.Split(strings.TrimPrefix(importedPath, modulePath+"/"), "/")
	if len(packageParts) < 2 || len(importParts) < 2 {
		return false
	}
	if packageParts[1] == importParts[1] {
		return false
	}
	return !sharedPlatformPackage(importParts[1])
}

func sharedPlatformPackage(name string) bool {
	switch name {
	case "contracts", "internal", "model":
		return true
	default:
		return false
	}
}
