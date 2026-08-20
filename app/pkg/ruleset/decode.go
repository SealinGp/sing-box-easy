package ruleset

import (
	"bytes"
	"strconv"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

// unsupportedVersionMarker is the prefix sing-box's srs reader uses when the
// file was written by a newer sing-box than this binary links against
// (common/srs/binary.go:62). Matched on text because the error is built with
// E.New and carries no sentinel to test with errors.Is.
const unsupportedVersionMarker = "unsupported version"

// magicBytes marks a binary .srs file: the ASCII letters "SRS"
// (common/srs/binary.go:20).
var magicBytes = [3]byte{0x53, 0x52, 0x53}

// Decode parses rule-set content into option rules.
//
// `format` comes from the config and is normally already resolved — sing-box's
// own option.RuleSet.UnmarshalJSON fills it in from the URL or path extension
// (option/rule_set.go:84-91). It is treated as a hint rather than as truth
// because this panel also builds configs programmatically, where nothing has
// run that decoder; the magic bytes are authoritative.
func Decode(format string, content []byte) ([]option.HeadlessRule, error) {
	var (
		compat option.PlainRuleSetCompat
		err    error
	)

	switch {
	case looksBinary(content):
		// recover=false: a rule this build cannot represent must surface as an
		// error, not be silently dropped. A set that quietly loses half its
		// rules would produce a confident, wrong "does not match".
		compat, err = srs.Read(bytes.NewReader(content), false)
	case format == C.RuleSetFormatBinary:
		// Claims binary but has no magic bytes — report the real problem
		// rather than trying to parse it as JSON and complaining about that.
		compat, err = srs.Read(bytes.NewReader(content), false)
	default:
		compat, err = json.UnmarshalExtended[option.PlainRuleSetCompat](content)
	}
	if err != nil {
		return nil, err
	}

	plain, err := compat.Upgrade()
	if err != nil {
		return nil, err
	}
	return plain.Rules, nil
}

func looksBinary(content []byte) bool {
	return len(content) >= len(magicBytes) &&
		content[0] == magicBytes[0] && content[1] == magicBytes[1] && content[2] == magicBytes[2]
}

func itoa(v int) string { return strconv.Itoa(v) }
