package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/v3rsi/tgbot-versionx/internal/security"
)

var envKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// parseEnvLine parses a single KEY=VALUE line from a .env file.
func parseEnvLine(line string) (key, value string, ok bool, err error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false, nil
	}
	if len(line) > maxEnvLineBytes {
		return "", "", false, fmt.Errorf("line exceeds %d bytes", maxEnvLineBytes)
	}

	line = strings.TrimPrefix(line, "export ")
	key, value, found := strings.Cut(line, "=")
	if !found {
		return "", "", false, errors.New("expected KEY=VALUE")
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false, errors.New("environment variable name is empty")
	}
	if !envKeyPattern.MatchString(key) {
		return "", "", false, fmt.Errorf("invalid environment variable name %q", key)
	}

	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value, err = strconv.Unquote(value)
		if err != nil {
			return "", "", false, fmt.Errorf("invalid quoted value: %w", err)
		}
	} else if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		value = value[1 : len(value)-1]
	}

	return key, value, true, nil
}

// readEnvFile parses a .env file into a map without mutating the process environment.
func readEnvFile(path string, sanitizer security.Sanitizer) (map[string]string, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}
	if info.Size() > maxEnvFileBytes {
		return nil, fmt.Errorf("%s exceeds maximum size of %d bytes", path, maxEnvFileBytes)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), maxEnvLineBytes)

	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		if lineNumber > maxEnvFileLines {
			return nil, fmt.Errorf("%s exceeds maximum of %d lines", path, maxEnvFileLines)
		}
		key, value, ok, err := parseEnvLine(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("parse %s line %d: %w", path, lineNumber, err)
		}
		if !ok {
			continue
		}
		if sanitizer != nil {
			if err := sanitizer.PreventConfigInjection(key, value); err != nil {
				return nil, fmt.Errorf("parse %s line %d: %w", path, lineNumber, err)
			}
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return values, nil
}

// applyEnvMapToProcess loads values into the process environment for keys that
// are not already set. Existing environment variables always win.
func applyEnvMapToProcess(values map[string]string) error {
	for key, value := range values {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}
	return nil
}
