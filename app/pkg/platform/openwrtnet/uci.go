package openwrtnet

import (
	"fmt"
	"strings"
)

// findAnonSection locates an anonymous UCI section by one of its options, e.g.
// the zone whose name is "sing-box-easy":
//
//	firewall.@zone[2].name='sing-box-easy'  ->  "firewall.@zone[2]"
//
// Anonymous sections have no stable name, and their indices shift whenever a
// neighbouring section is removed. Resolving the index at use time — rather
// than remembering the one we created — is what keeps Revert from deleting
// somebody else's zone after an unrelated firewall edit.
func findAnonSection(uciShowOutput, pkg, sectionType, key, value string) (string, bool) {
	prefix := fmt.Sprintf("%s.@%s[", pkg, sectionType)
	suffix := fmt.Sprintf(".%s='%s'", key, value)

	for _, line := range strings.Split(uciShowOutput, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
			continue
		}
		return strings.TrimSuffix(line, suffix), true
	}
	return "", false
}

// parseUCIList splits the output of `uci -q get` into values.
//
// A UCI list option prints space-separated on a single line, while a plain
// option prints one value. Both shapes reduce to a slice here so callers need
// not care which the operator had configured.
func parseUCIList(out string) []string {
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return nil
	}
	return fields
}
