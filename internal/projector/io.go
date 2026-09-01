package projector

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(data), nil
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func physicalLineCount(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	count := strings.Count(string(data), "\n")
	if data[len(data)-1] != '\n' {
		count++
	}
	return count
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func resolvePath(root, value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(root, value))
}

func ensureOutputDir(path string) error {
	if path == "" || !filepath.IsAbs(path) {
		return fmt.Errorf("output directory must be an absolute caller-owned path")
	}
	if err := ensureOutsideRepository(path); err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	return nil
}

func EnsureCallerOutputDir(path string) error {
	return ensureOutputDir(path)
}

func ensureCallerFile(path string, data []byte) error {
	if path == "" || !filepath.IsAbs(path) {
		return fmt.Errorf("generated output must be an absolute caller-owned path")
	}
	if err := ensureOutsideRepository(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ensureOutsideRepository(path string) error {
	root := findRepositoryRoot()
	if root == "" {
		return nil
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return err
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return fmt.Errorf("caller-owned output must be outside repository")
	}
	return nil
}

func findRepositoryRoot() string {
	current, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		info, statErr := os.Stat(filepath.Join(current, ".git"))
		if statErr == nil && info.IsDir() {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func BuildInventory(root string) (Inventory, error) {
	inventory := Inventory{RootREADMEExcluded: true}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if entry.IsDir() && excludedDirectory(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			inventory.DescendantDirs++
			return nil
		}
		if relative == "README.md" || !entry.Type().IsRegular() {
			return nil
		}
		inventory.RegularFiles++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := physicalLineCount(data)
		switch filepath.Ext(entry.Name()) {
		case ".go":
			inventory.GoFiles++
			inventory.GoPhysicalLines += lines
		case ".gooo":
			inventory.GoooFiles++
			inventory.GoooPhysicalLines += lines
		}
		return nil
	})
	return inventory, err
}

func excludedDirectory(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".cache", "cache", "vendor", "toolchain", "toolchains", "node_modules", "dist", "bin":
		return true
	default:
		return strings.Contains(strings.ToLower(name), "toolchain")
	}
}
