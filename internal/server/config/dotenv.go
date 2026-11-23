package config

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

const dotenvFile = ".env.local"

// dotenv parses .env.local in the working directory. A missing file yields an empty map. The
// format is KEY=value per line, with # comment lines, blank lines and one pair of matching quotes
// around the value removed; there is no export prefix, interpolation or inline comment.
func dotenv() (map[string]string, error) {
	values := map[string]string{}
	f, err := os.Open(dotenvFile)
	if errors.Is(err, fs.ErrNotExist) {
		return values, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for n := 1; scanner.Scan(); n++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			return nil, fmt.Errorf("%s line %d: want KEY=value", dotenvFile, n)
		}
		values[key] = unquote(strings.TrimSpace(value))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", dotenvFile, err)
	}
	return values, nil
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}
