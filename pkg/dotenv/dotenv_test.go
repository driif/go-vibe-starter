package dotenv_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/driif/go-vibe-starter/pkg/dotenv"
)

var pwd, _ = os.Getwd()

func TestTryLoadOverride(t *testing.T) {
	if os.Getenv("IS_THIS_A_TEST_ENV") != "" {
		t.Fatal("IS_THIS_A_TEST_ENV should be empty before test")
	}

	dotenv.TryLoad(
		filepath.Join(pwd, "testdata/.env1.local"),
		func(k, v string) error { t.Setenv(k, v); return nil })

	if got := os.Getenv("IS_THIS_A_TEST_ENV"); got != "yes" {
		t.Errorf("IS_THIS_A_TEST_ENV: got %q, want %q", got, "yes")
	}
	if got := os.Getenv("PSQL_USER"); got != "dotenv_override_psql_user" {
		t.Errorf("PSQL_USER: got %q, want %q", got, "dotenv_override_psql_user")
	}

	dotenv.TryLoad(
		filepath.Join(pwd, "testdata/.env2.local"),
		func(k, v string) error { t.Setenv(k, v); return nil })

	if got := os.Getenv("IS_THIS_A_TEST_ENV"); got != "yes still" {
		t.Errorf("IS_THIS_A_TEST_ENV: got %q, want %q", got, "yes still")
	}
	if got := os.Getenv("PSQL_USER"); got != "dotenv_override_psql_user_2" {
		t.Errorf("PSQL_USER: got %q, want %q", got, "dotenv_override_psql_user_2")
	}
}

func TestTryLoadSilentOnMissingFile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("should not panic on missing file, got: %v", r)
		}
	}()
	dotenv.TryLoad(
		filepath.Join(pwd, "testdata/.env.does.not.exist"),
		func(k, v string) error { t.Setenv(k, v); return nil })
}

func TestTryLoadEmptyFile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("should not panic on empty file, got: %v", r)
		}
	}()
	dotenv.TryLoad(
		filepath.Join(pwd, "testdata/.env.local.sample"),
		func(k, v string) error { t.Setenv(k, v); return nil })

	if got := os.Getenv("EMPTY_VARIABLE_INIT"); got != "" {
		t.Errorf("EMPTY_VARIABLE_INIT should be empty, got %q", got)
	}
}

func TestTryLoadLenientOnMalformedLines(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("should not panic on malformed lines, got: %v", r)
		}
	}()
	dotenv.TryLoad(
		filepath.Join(pwd, "testdata/.env.local.malformed"),
		func(k, v string) error { t.Setenv(k, v); return nil })
}

func TestParseFeatures(t *testing.T) {
	dotenv.TryLoad(
		filepath.Join(pwd, "testdata/.env.features"),
		func(k, v string) error { t.Setenv(k, v); return nil })

	cases := []struct {
		key  string
		want string
	}{
		{"EXPORT_KEY", "exported_value"},
		{"INLINE_COMMENT", "value"},
		{"DOUBLE_QUOTED", "hello world"},
		{"SINGLE_QUOTED", "hello world"},
	}
	for _, tc := range cases {
		if got := os.Getenv(tc.key); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.key, got, tc.want)
		}
	}
}
