package presets_test

import (
	"testing"

	"file-management-service/pkg/rbac"
	"file-management-service/pkg/rbac/presets"
)

// TestAllPresetsAreValid iterates every registered preset and ensures it
// passes Config.Validate() and can be used to construct an RBACChecker.
// If you add a new preset, register it in All() and this test covers it
// automatically — no per-preset test file needed.
func TestAllPresetsAreValid(t *testing.T) {
	all := presets.All()
	if len(all) == 0 {
		t.Fatal("presets.All() returned no presets — at least one must be registered")
	}

	for _, p := range all {
		t.Run(p.Name, func(t *testing.T) {
			cfg := p.Config()

			// Step 1: Config must pass structural validation.
			if err := cfg.Validate(); err != nil {
				t.Fatalf("preset %q has invalid config: %v", p.Name, err)
			}

			// Step 2: Config must produce a working RBACChecker.
			checker, err := rbac.New(cfg)
			if err != nil {
				t.Fatalf("preset %q failed rbac.New: %v", p.Name, err)
			}
			if checker == nil {
				t.Fatalf("preset %q: rbac.New returned nil checker without error", p.Name)
			}
		})
	}
}

// TestAllPresetsHaveUniqueNames ensures no two presets share the same name.
func TestAllPresetsHaveUniqueNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range presets.All() {
		if seen[p.Name] {
			t.Errorf("duplicate preset name: %q", p.Name)
		}
		seen[p.Name] = true
	}
}
