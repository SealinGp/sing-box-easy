package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	MinimumCoreVersion = "1.12.12"
	MaximumTestedCore  = "1.14.x"
)

var coreVersionPattern = regexp.MustCompile(`(?m)^sing-box version\s+(v?([0-9]+)\.([0-9]+)\.([0-9]+)[^\s]*)\s*$`)

// CoreVersion is the installed sing-box version. Raw retains prerelease and
// vendor suffixes for diagnostics while the numeric fields drive capabilities.
type CoreVersion struct {
	Major int    `json:"major"`
	Minor int    `json:"minor"`
	Patch int    `json:"patch"`
	Raw   string `json:"raw"`
}

// ParseCoreVersion parses the stable first line emitted by `sing-box version`.
func ParseCoreVersion(output string) (CoreVersion, error) {
	match := coreVersionPattern.FindStringSubmatch(output)
	if len(match) != 5 {
		return CoreVersion{}, fmt.Errorf("could not parse sing-box version output")
	}
	major, err := strconv.Atoi(match[2])
	if err != nil {
		return CoreVersion{}, fmt.Errorf("invalid sing-box major version: %w", err)
	}
	minor, err := strconv.Atoi(match[3])
	if err != nil {
		return CoreVersion{}, fmt.Errorf("invalid sing-box minor version: %w", err)
	}
	patch, err := strconv.Atoi(match[4])
	if err != nil {
		return CoreVersion{}, fmt.Errorf("invalid sing-box patch version: %w", err)
	}
	return CoreVersion{Major: major, Minor: minor, Patch: patch, Raw: match[1]}, nil
}

// CoreCapabilities describes version-gated behavior exposed to API clients.
// Unknown future versions remain usable through raw configuration APIs but are
// marked unsupported until they enter the released-binary test matrix.
type CoreCapabilities struct {
	Version          string `json:"version"`
	Supported        bool   `json:"supported"`
	Minimum          string `json:"minimum"`
	MaximumTested    string `json:"maximum_tested"`
	DNSEvaluate      bool   `json:"dns_evaluate"`
	DNSRespond       bool   `json:"dns_respond"`
	DNSRace          bool   `json:"dns_race"`
	DNSMatchResponse bool   `json:"dns_match_response"`
	DNSOptimistic    bool   `json:"dns_optimistic"`
}

// CapabilitiesForCoreVersion maps the installed version to features the panel
// may safely offer in its structured editors.
func CapabilitiesForCoreVersion(version CoreVersion) CoreCapabilities {
	dnsActions := version.Major > 1 || version.Major == 1 && version.Minor >= 14
	atLeastMinimum := version.Major > 1 ||
		version.Major == 1 && (version.Minor > 12 || version.Minor == 12 && version.Patch >= 12)
	testedLine := version.Major == 1 && version.Minor >= 12 && version.Minor <= 14
	return CoreCapabilities{
		Version:          version.Raw,
		Supported:        atLeastMinimum && testedLine,
		Minimum:          MinimumCoreVersion,
		MaximumTested:    MaximumTestedCore,
		DNSEvaluate:      dnsActions,
		DNSRespond:       dnsActions,
		DNSRace:          dnsActions,
		DNSMatchResponse: dnsActions,
		DNSOptimistic:    dnsActions,
	}
}

// CoreAdapter is the narrow process boundary for the installed sing-box core.
type CoreAdapter interface {
	Version(ctx context.Context) (CoreVersion, error)
	Validate(ctx context.Context, configPath string) error
}

type binaryCoreAdapter struct {
	path string
}

// NewBinaryCoreAdapter uses the configured sing-box executable as the source
// of truth for both version detection and semantic config validation.
func NewBinaryCoreAdapter(path string) CoreAdapter {
	if strings.TrimSpace(path) == "" {
		path = "sing-box"
	}
	return &binaryCoreAdapter{path: path}
}

func (a *binaryCoreAdapter) Version(ctx context.Context) (CoreVersion, error) {
	output, err := exec.CommandContext(ctx, a.path, "version").CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return CoreVersion{}, fmt.Errorf("sing-box version failed: %s", detail)
	}
	return ParseCoreVersion(string(output))
}

func (a *binaryCoreAdapter) Validate(ctx context.Context, configPath string) error {
	cmd := exec.CommandContext(ctx, a.path, "check", "-c", configPath)
	cmd.Env = append(os.Environ(), "ENABLE_DEPRECATED_SPECIAL_OUTBOUNDS=true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return fmt.Errorf("config validation failed: %s", detail)
	}
	if bytes.Contains(output, []byte("ERROR")) || bytes.Contains(output, []byte("FATAL")) {
		return fmt.Errorf("config validation failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}
