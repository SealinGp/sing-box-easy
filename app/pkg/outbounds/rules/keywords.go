package noderules

import "strings"

// KeywordEntry describes one built-in region/country code and its synonyms,
// surfaced to the frontend so users can pick codes instead of hand-typing every
// spelling. Synonyms intentionally span Chinese, English, the bare code, and the
// emoji flag so a single `code` matcher catches all common tag spellings.
type KeywordEntry struct {
	Code     string   `json:"code"`     // canonical short code, e.g. "HK"
	Label    string   `json:"label"`    // human label, e.g. "Hong Kong"
	Synonyms []string `json:"synonyms"` // all spellings (incl. code + emoji)
}

// keywordCatalog is the built-in code → synonyms table. Lowercased text synonyms
// are matched as substrings; the emoji is matched verbatim. Order within a row
// does not matter (OR semantics). Curated to avoid obvious collisions (e.g. KR
// uses "韩国/한국/korea", not a bare 2-letter token that would over-match).
var keywordCatalog = []KeywordEntry{
	{Code: "HK", Label: "Hong Kong", Synonyms: []string{"香港", "港", "hong kong", "hongkong", "hk", "🇭🇰"}},
	{Code: "TW", Label: "Taiwan", Synonyms: []string{"台湾", "臺灣", "台灣", "taiwan", "tw", "🇹🇼"}},
	{Code: "JP", Label: "Japan", Synonyms: []string{"日本", "东京", "大阪", "japan", "tokyo", "osaka", "jp", "🇯🇵"}},
	{Code: "KR", Label: "South Korea", Synonyms: []string{"韩国", "韓國", "한국", "首尔", "korea", "seoul", "kr", "🇰🇷"}},
	{Code: "SG", Label: "Singapore", Synonyms: []string{"新加坡", "狮城", "singapore", "sg", "🇸🇬"}},
	{Code: "US", Label: "United States", Synonyms: []string{"美国", "美國", "united states", "usa", "us", "america", "🇺🇸"}},
	{Code: "CA", Label: "Canada", Synonyms: []string{"加拿大", "canada", "ca", "🇨🇦"}},
	{Code: "UK", Label: "United Kingdom", Synonyms: []string{"英国", "英國", "united kingdom", "britain", "england", "uk", "gb", "🇬🇧"}},
	{Code: "DE", Label: "Germany", Synonyms: []string{"德国", "德國", "germany", "frankfurt", "de", "🇩🇪"}},
	{Code: "FR", Label: "France", Synonyms: []string{"法国", "法國", "france", "paris", "fr", "🇫🇷"}},
	{Code: "NL", Label: "Netherlands", Synonyms: []string{"荷兰", "荷蘭", "netherlands", "nl", "🇳🇱"}},
	{Code: "RU", Label: "Russia", Synonyms: []string{"俄罗斯", "俄羅斯", "russia", "moscow", "ru", "🇷🇺"}},
	{Code: "AU", Label: "Australia", Synonyms: []string{"澳大利亚", "澳洲", "australia", "sydney", "au", "🇦🇺"}},
	{Code: "IN", Label: "India", Synonyms: []string{"印度", "india", "mumbai", "in", "🇮🇳"}},
}

// codeIndex maps an upper-cased code to its synonyms for O(1) expansion.
var codeIndex = func() map[string][]string {
	m := make(map[string][]string, len(keywordCatalog))
	for _, e := range keywordCatalog {
		m[strings.ToUpper(e.Code)] = e.Synonyms
	}
	return m
}()

// Catalog returns the built-in keyword catalog (for the API / UI picker).
func Catalog() []KeywordEntry {
	out := make([]KeywordEntry, len(keywordCatalog))
	copy(out, keywordCatalog)
	return out
}

// CodeSynonyms returns the synonym list for a code (case-insensitive). When the
// code is unknown it falls back to the code itself, so an arbitrary code still
// behaves like a literal keyword rather than matching nothing.
func CodeSynonyms(code string) []string {
	if syns, ok := codeIndex[strings.ToUpper(strings.TrimSpace(code))]; ok {
		return syns
	}
	return []string{strings.TrimSpace(code)}
}
