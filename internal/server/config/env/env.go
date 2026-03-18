package env

import (
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	projectRootDir string
	dirOnce        sync.Once
)

func GetProjectRootDir() string {
	dirOnce.Do(func() {
		ex, err := os.Executable()
		if err != nil {
			slog.Error("Failed to get executable path while retrieving project root directory.", "err", err)
			panic("Failed to get executable path while retrieving project root directory.")
		}

		projectRootDir = GetEnv("PROJECT_ROOT_DIR", filepath.Dir(ex))
	})

	return projectRootDir
}

func GetEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func GetEnvAsInt(key string, defaultVal int) int {
	strVal := GetEnv(key, "")

	if val, err := strconv.Atoi(strVal); err == nil {
		return val
	}

	return defaultVal
}

func GetEnvAsUint32(key string, defaultVal uint32) uint32 {
	strVal := GetEnv(key, "")

	if val, err := strconv.ParseUint(strVal, 10, 32); err == nil {
		return uint32(val)
	}

	return defaultVal
}

func GetEnvAsUint8(key string, defaultVal uint8) uint8 {
	strVal := GetEnv(key, "")

	if val, err := strconv.ParseUint(strVal, 10, 8); err == nil {
		return uint8(val)
	}

	return defaultVal
}

func GetEnvAsBool(key string, defaultVal bool) bool {
	strVal := GetEnv(key, "")

	if val, err := strconv.ParseBool(strVal); err == nil {
		return val
	}

	return defaultVal
}

// GetEnvAsStringArr reads ENV and returns the values split by separator.
func GetEnvAsStringArr(key string, defaultVal []string, separator ...string) []string {
	strVal := GetEnv(key, "")

	if len(strVal) == 0 {
		return defaultVal
	}

	sep := ","
	if len(separator) >= 1 {
		sep = separator[0]
	}

	return strings.Split(strVal, sep)
}

// GetEnvAsStringArrTrimmed reads ENV and returns the whitespace trimmed values split by separator.
func GetEnvAsStringArrTrimmed(key string, defaultVal []string, separator ...string) []string {
	slc := GetEnvAsStringArr(key, defaultVal, separator...)

	for i := range slc {
		slc[i] = strings.TrimSpace(slc[i])
	}

	return slc
}

func GetEnvAsURL(key string, defaultVal string) *url.URL {
	strVal := GetEnv(key, "")

	if len(strVal) == 0 {
		u, err := url.Parse(defaultVal)
		if err != nil {
			slog.Error("Failed to parse default value for env variable as URL", "key", key, "defaultVal", defaultVal, "err", err)
			panic("Failed to parse default value for env variable as URL")
		}

		return u
	}

	u, err := url.Parse(strVal)
	if err != nil {
		slog.Error("Failed to parse env variable as URL", "key", key, "strVal", strVal, "err", err)
		panic("Failed to parse env variable as URL")
	}

	return u
}
