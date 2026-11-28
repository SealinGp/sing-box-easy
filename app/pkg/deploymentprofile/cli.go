package deploymentprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func WriteFromRequest(ctx context.Context, input io.Reader, output string) error {
	if output == "" {
		return fmt.Errorf("profile_output is required")
	}
	var request Request
	if err := json.NewDecoder(io.LimitReader(input, 1<<20)).Decode(&request); err != nil {
		return fmt.Errorf("read profile request: %w", err)
	}
	result, err := Generate(ctx, request)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(result.Config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, append(raw, '\n'), 0600)
}
