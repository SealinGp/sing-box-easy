package noderules

// FilterTemplate is a ready-made Filter the UI can offer as a one-click starting
// point, so users do not have to hand-build common buckets. Templates are pure
// data; applying one just calls CreateFilter.
type FilterTemplate struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	OutboundType string    `json:"outbound_type"`
	Matchers     []Matcher `json:"matchers"`
}

// filterTemplates is the built-in template catalog. Geographic regions use
// `code` matchers (covering CN/EN/code/emoji spellings); the streaming template
// shows the non-geographic use the user asked for.
var filterTemplates = []FilterTemplate{
	{
		ID:           "tpl_asia",
		Name:         "Asia",
		Description:  "Hong Kong, Taiwan, Japan, Korea, Singapore",
		OutboundType: OutboundTypeURLTest,
		Matchers: []Matcher{
			{Type: MatcherCode, Value: "HK"},
			{Type: MatcherCode, Value: "TW"},
			{Type: MatcherCode, Value: "JP"},
			{Type: MatcherCode, Value: "KR"},
			{Type: MatcherCode, Value: "SG"},
		},
	},
	{
		ID:           "tpl_americas",
		Name:         "Americas",
		Description:  "United States, Canada",
		OutboundType: OutboundTypeURLTest,
		Matchers: []Matcher{
			{Type: MatcherCode, Value: "US"},
			{Type: MatcherCode, Value: "CA"},
		},
	},
	{
		ID:           "tpl_europe",
		Name:         "Europe",
		Description:  "UK, Germany, France, Netherlands",
		OutboundType: OutboundTypeURLTest,
		Matchers: []Matcher{
			{Type: MatcherCode, Value: "UK"},
			{Type: MatcherCode, Value: "DE"},
			{Type: MatcherCode, Value: "FR"},
			{Type: MatcherCode, Value: "NL"},
		},
	},
	{
		ID:           "tpl_streaming",
		Name:         "Streaming",
		Description:  "Nodes tagged for streaming / unlocking (e.g. Netflix, YouTube, IPLC)",
		OutboundType: OutboundTypeSelector,
		Matchers: []Matcher{
			{Type: MatcherKeyword, Value: "Streaming"},
			{Type: MatcherKeyword, Value: "Netflix"},
			{Type: MatcherKeyword, Value: "YouTube"},
			{Type: MatcherKeyword, Value: "流媒体"},
			{Type: MatcherKeyword, Value: "解锁"},
		},
	},
}

// Templates returns the built-in filter templates (for the API / UI gallery).
func Templates() []FilterTemplate {
	out := make([]FilterTemplate, len(filterTemplates))
	copy(out, filterTemplates)
	return out
}

// TemplateByID returns a template by its catalog ID.
func TemplateByID(id string) (FilterTemplate, bool) {
	for _, t := range filterTemplates {
		if t.ID == id {
			return t, true
		}
	}
	return FilterTemplate{}, false
}
