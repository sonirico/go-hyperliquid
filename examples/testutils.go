package examples

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
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
			os.Unsetenv(key)
		}
	}
}
