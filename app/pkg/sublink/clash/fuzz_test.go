package clash

import "testing"

// FuzzParse guards the raw-map accessors. Every value in a proxy entry comes
// from an untrusted feed and is type-asserted, so a mapping where a scalar was
// expected (or vice versa) must return an error, never panic — a panic here
// takes down the panel on a subscription refresh.
//
// Seeds cover the shapes a hand-written or hostile profile actually produces;
// run longer with: go test ./app/pkg/sublink/clash -fuzz FuzzParse
func FuzzParse(f *testing.F) {
	f.Add("proxies:\n  - {name: A, type: trojan, server: a.com, port: 443, password: p}\n")
	f.Add("proxies:\n  - {name: A, type: ss, server: a.com, port: notaport}\n")
	f.Add("proxies:\n  - [1,2,3]\n")
	f.Add("proxies:\n  - {name: {a: b}, type: [x], server: 1, port: {}, ws-opts: hello}\n")
	f.Add("proxies:\n  - {type: vmess, server: a.com, port: 1, uuid: 1, network: ws, ws-opts: {headers: notamap}}\n")
	f.Add("proxies: notalist\n")
	f.Add("proxies:\n  - null\n")
	f.Fuzz(func(t *testing.T, body string) {
		Detect([]byte(body))
		_, _, _ = Parse([]byte(body)) // must never panic
	})
}
