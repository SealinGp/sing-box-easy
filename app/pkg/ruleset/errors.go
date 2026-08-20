package ruleset

import (
	"errors"
	"fmt"
)

func errUnknownRuleType(ruleType string) error {
	return errors.New("unknown rule type: " + ruleType)
}

func errBadPortRange(raw string) error {
	return errors.New("bad port range: " + raw)
}

// sprintf is fmt.Sprintf, wrapped so the test helpers can format assertion
// messages without every file importing fmt.
func sprintf(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
