package hyperliquid

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// loadEnvClean loads environment variables from the specified .env file(s)
// after first clearing all HL_* prefixed environment variables.
//
// This prevents issues where previously set HL_* variables (from shell or
// previous test runs) persist even when commented out or removed from .env files.
//
// In CI environments (detected by CI=true), skips clearing to preserve CI-injected credentials.
//
// Usage:
//
//	loadEnvClean(".env.testnet")              // Load single file
//	loadEnvClean(".env.testnet", ".env.local") // Load multiple files
//	loadEnvClean()                             // Load default .env
func loadEnvClean(filenames ...string) error {
	// Skip clearing in CI - credentials come from CI environment, not .env files
	// Check multiple CI indicators: GitHub Actions, GitLab CI, CircleCI, Travis, Jenkins, etc.
	if !isCI() {
		clearHyperliquidEnv()
	}

	// Load new environment from files (will be no-op in CI if files don't exist)
	return godotenv.Overload(filenames...)
}

// isCI detects if running in a CI environment
func isCI() bool {
	// GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" || os.Getenv("CI") == "true" {
		return true
	}
	// GitLab CI
	if os.Getenv("GITLAB_CI") == "true" {
		return true
	}
	// CircleCI
	if os.Getenv("CIRCLECI") == "true" {
		return true
	}
	// Travis CI
	if os.Getenv("TRAVIS") == "true" {
		return true
	}
	// Jenkins
	if os.Getenv("JENKINS_URL") != "" {
		return true
	}
	return false
}

// clearHyperliquidEnv removes all HL_* prefixed environment variables.
// This ensures a clean slate before loading new environment configuration.
func clearHyperliquidEnv() {
	for _, env := range os.Environ() {
		// env format is "KEY=VALUE"
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}

		key := pair[0]
		if strings.HasPrefix(key, "HL_") {
			_ = os.Unsetenv(key)
		}
	}
}

// assertKeysCovered unmarshals raw into v, marshals v back and fails listing
// every JSON key path present in raw and absent after the round trip.
// Array elements collapse to "[]" in the paths.
func assertKeysCovered(t *testing.T, raw []byte, v any) {
	t.Helper()

	missing, err := uncoveredKeyPaths(raw, v)
	require.NoError(t, err)
	require.Empty(t, missing, "keys present in the payload and lost by the struct")
}

func uncoveredKeyPaths(raw []byte, v any) ([]string, error) {
	if err := json.Unmarshal(raw, v); err != nil {
		return nil, err
	}
	roundTrip, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var want, got any
	if err := json.Unmarshal(raw, &want); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(roundTrip, &got); err != nil {
		return nil, err
	}

	gotPaths := make(map[string]struct{})
	collectKeyPaths(got, "", gotPaths)
	wantPaths := make(map[string]struct{})
	collectKeyPaths(want, "", wantPaths)

	var missing []string
	for path := range wantPaths {
		if _, ok := gotPaths[path]; !ok {
			missing = append(missing, path)
		}
	}
	sort.Strings(missing)
	return missing, nil
}

func collectKeyPaths(node any, prefix string, paths map[string]struct{}) {
	switch n := node.(type) {
	case map[string]any:
		for key, child := range n {
			path := prefix + "." + key
			paths[path] = struct{}{}
			collectKeyPaths(child, path, paths)
		}
	case []any:
		for _, child := range n {
			collectKeyPaths(child, prefix+"[]", paths)
		}
	}
}
