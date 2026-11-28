package appconfig

import (
	"fmt"
	"os"
)

// ReadBootstrapPassword keeps installer credentials out of process arguments.
// Read verbatim: spaces, quotes and backslashes are valid password characters.
func ReadBootstrapPassword(password, path string) (string, error) {
	if path == "" {
		return password, nil
	}
	if password != "" {
		return "", fmt.Errorf("admin_pass and admin_pass_file cannot be used together")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read password file: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("administrator password file is empty")
	}
	return string(data), nil
}
