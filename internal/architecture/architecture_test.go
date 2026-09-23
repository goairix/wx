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
	"health_card",
	"mini_program",
	"open_platform",
	"open_work",
	"support",
}

var legacyPackages = []string{
	modulePath + "/official/qr_code",
	modulePath + "/work/account_id",
	modulePath + "/work/mini_program",
}

var legacyDirectories = []string{
	"official/qr_code",
	"work/account_id",
	"work/mini_program",
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
	for _, legacy := range legacyDirectories {
		if _, err := os.Stat(filepath.Join(root, legacy)); !os.IsNotExist(err) {
			violations = append(
				violations,
				"legacy directory still exists: "+legacy,
			)
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
			if platform, domain := platformDomain(current.ImportPath); platform != "" &&
				domain != "" &&
				imported == modulePath+"/core/transport" {
				violations = append(
					violations,
					fmt.Sprintf(
						"domain package %s depends on concrete transport",
						current.ImportPath,
					),
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

func TestCrossesDomainBoundary(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		imported string
		want     bool
	}{
		{
			name:     "same platform sibling domain",
			current:  modulePath + "/official/menu",
			imported: modulePath + "/official/user",
			want:     true,
		},
		{
			name:     "cross platform domain",
			current:  modulePath + "/official/menu",
			imported: modulePath + "/miniapp/user",
			want:     true,
		},
		{
			name:     "domain importing platform root",
			current:  modulePath + "/official/menu",
			imported: modulePath + "/official",
			want:     true,
		},
		{
			name:     "same domain child",
			current:  modulePath + "/official/menu/internal",
			imported: modulePath + "/official/menu/model",
			want:     false,
		},
		{
			name:     "shared platform internal",
			current:  modulePath + "/official/menu",
			imported: modulePath + "/official/internal/api",
			want:     false,
		},
		{
			name:     "shared package cannot import concrete domain",
			current:  modulePath + "/official/internal/api",
			imported: modulePath + "/official/menu",
			want:     true,
		},
		{
			name:     "shared package cannot import platform root",
			current:  modulePath + "/official/internal/api",
			imported: modulePath + "/official",
			want:     true,
		},
		{
			name:     "shared package cannot import another platform",
			current:  modulePath + "/official/internal/api",
			imported: modulePath + "/miniapp/internal/api",
			want:     true,
		},
		{
			name:     "shared package may import same platform shared layer",
			current:  modulePath + "/healthcard/contracts",
			imported: modulePath + "/healthcard/model",
			want:     false,
		},
		{
			name:     "openplatform composes official",
			current:  modulePath + "/openplatform",
			imported: modulePath + "/official",
			want:     false,
		},
		{
			name:     "openplatform composes miniapp",
			current:  modulePath + "/openplatform",
			imported: modulePath + "/miniapp",
			want:     false,
		},
		{
			name:     "openplatform composes work authorizer",
			current:  modulePath + "/openplatform",
			imported: modulePath + "/work/authorizer",
			want:     false,
		},
		{
			name:     "other platform root cannot compose cross platform",
			current:  modulePath + "/official",
			imported: modulePath + "/miniapp",
			want:     true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := crossesDomainBoundary(test.current, test.imported)
			if got != test.want {
				t.Fatalf("crossesDomainBoundary(%q, %q) = %v, want %v",
					test.current,
					test.imported,
					got,
					test.want,
				)
			}
		})
	}
}

func TestLegacyPackageDetection(t *testing.T) {
	for _, importPath := range []string{
		modulePath + "/work/account_id",
		modulePath + "/work/account_id/internal",
		modulePath + "/work/mini_program",
		modulePath + "/official/qr_code",
	} {
		if legacyRoot(importPath) == "" {
			t.Fatalf("legacy package was not detected: %s", importPath)
		}
	}
	if legacyRoot(modulePath+"/work/accountid") != "" {
		t.Fatal("current work/accountid package was marked legacy")
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
	for _, legacy := range legacyPackages {
		if importPath == legacy || strings.HasPrefix(importPath, legacy+"/") {
			return strings.TrimPrefix(legacy, modulePath+"/")
		}
	}
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
	platform, domain := platformDomain(packagePath)
	importedPlatform, importedDomain := platformDomain(importedPath)
	if platform == "" || importedPlatform == "" {
		return false
	}
	if domain == "" {
		if platform == importedPlatform {
			return false
		}
		return !allowedPlatformComposition(packagePath, importedPath)
	}
	if sharedPlatformPackage(domain) {
		if importedPlatform != platform || importedDomain == "" {
			return true
		}
		return !sharedPlatformPackage(importedDomain)
	}
	if importedPlatform != platform {
		return true
	}
	if importedDomain == "" {
		return true
	}
	if domain == importedDomain || sharedPlatformPackage(importedDomain) {
		return false
	}
	return true
}

func platformDomain(importPath string) (string, string) {
	platform := platformRoot(importPath)
	if platform == "" {
		return "", ""
	}
	parts := strings.Split(strings.TrimPrefix(importPath, modulePath+"/"), "/")
	if len(parts) < 2 {
		return platform, ""
	}
	return platform, parts[1]
}

func allowedPlatformComposition(packagePath, importedPath string) bool {
	if packagePath != modulePath+"/openplatform" {
		return false
	}
	switch importedPath {
	case modulePath + "/official",
		modulePath + "/miniapp",
		modulePath + "/work/authorizer":
		return true
	default:
		return false
	}
}

func sharedPlatformPackage(name string) bool {
	switch name {
	case "contracts", "internal", "model":
		return true
	default:
		return false
	}
}
