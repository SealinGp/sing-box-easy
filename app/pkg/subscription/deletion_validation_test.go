package subscription

import (
	"os"
	"strings"
	"testing"
)

func TestRejectedDeletionKeepsSubscriptionActive(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh request", true: "retry after earlier failure"}[pending], func(t *testing.T) {
			manager, path, restarter := deletionManager(t)
			raw := `{"route":{"rules":[{"outbound":"media"}]},"outbounds":[{"type":"direct","tag":"direct"},{"type":"selector","tag":"media","outbounds":["node | sub_1"]},{"type":"socks","tag":"node | sub_1","server":"example.com","server_port":1080}]}`
			if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			store := manager.repository.(deletionRepository)
			if pending {
				if err := store.MarkDeleting("sub_1"); err != nil {
					t.Fatal(err)
				}
			}
			err := manager.Delete("sub_1")
			if err == nil || !strings.Contains(err.Error(), "media") {
				t.Fatalf("error = %v", err)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != raw || restarter.calls != 0 {
				t.Fatal("rejected delete changed config or restarted service")
			}
			got, err := store.IsDeleting("sub_1")
			if err != nil {
				t.Fatal(err)
			}
			if got != pending {
				t.Fatalf("pending = %v, want %v", got, pending)
			}
			if _, err := manager.Get("sub_1"); err != nil {
				t.Fatal(err)
			}
			if !pending {
				if err := manager.checkActive("sub_1"); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
