// Package dotenv loads .env files into environment variables.
// It is intended for local development only — inject gitignored secrets
// without modifying system environment or deployment configs.
package dotenv

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
)

// SetEnv matches the signature of os.Setenv so callers can pass it directly,
// or pass t.Setenv in tests for automatic cleanup.
type SetEnv = func(key string, value string) error

// TryLoad applies the .env file at the given path, silently doing nothing if
// the file does not exist. Any other error (parse, permission, setenv failure)
// causes a panic. Successful load is logged as a warning.
//
// Typical usage:
//
//	dotenv.TryLoad("/path/to/.env.local", os.Setenv)
//
// In tests (auto-resets after each test):
//
//	dotenv.TryLoad("/path/to/.env.test.local", func(k, v string) error { t.Setenv(k, v); return nil })
func TryLoad(absolutePathToEnvFile string, setEnvFn SetEnv) {
	err := Load(absolutePathToEnvFile, setEnvFn)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Error(".env parse error!", "envFile", absolutePathToEnvFile, "err", err)
			panic(".env parse error!")
		}
	} else {
		slog.Warn(".env overrides ENV variables!", "envFile", absolutePathToEnvFile)
	}
}

// Load applies all key=value pairs from the given .env file using setEnvFn.
// Returns an error if the file cannot be opened, parsed, or applied.
func Load(absolutePathToEnvFile string, setEnvFn SetEnv) error {
	file, err := os.Open(absolutePathToEnvFile)
	if err != nil {
		return err
	}
	defer file.Close()

	envs, err := parse(file)
	if err != nil {
		return err
	}

	for key, value := range envs {
		if err := setEnvFn(key, value); err != nil {
			return err
		}
	}

	return nil
}

// parse reads key=value pairs from r.
//
// Rules:
//   - Blank lines and lines starting with # are skipped
//   - Optional "export " prefix is stripped (e.g. export KEY=value)
//   - Inline comments are stripped (KEY=value # comment → value)
//   - Leading/trailing whitespace around keys and values is trimmed
//   - Surrounding single or double quotes on the value are stripped
func parse(r io.Reader) (map[string]string, error) {
	envs := map[string]string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// strip optional "export " prefix
		line = strings.TrimPrefix(line, "export ")

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = stripInlineComment(strings.TrimSpace(value))
		value = stripQuotes(value)

		envs[key] = value
	}
	return envs, scanner.Err()
}

// stripInlineComment removes a trailing # comment from an unquoted value.
// Quoted values (starting with " or ') are left untouched so that a #
// inside quotes is preserved.
func stripInlineComment(s string) string {
	if len(s) == 0 {
		return s
	}
	// quoted values: don't touch
	if s[0] == '"' || s[0] == '\'' {
		return s
	}
	if idx := strings.Index(s, " #"); idx != -1 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// stripQuotes removes matching surrounding single or double quotes.
func stripQuotes(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}
