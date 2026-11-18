package subscription

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/nodehistory"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/node"
	"xorm.io/xorm"
)

type historyFeed struct{}

func (historyFeed) Resolve(context.Context, []string, feed.FetchOptions) ([]*node.SubNode, *feed.FetchMeta, error) {
	return []*node.SubNode{sn("replacement", "new.example", 443)}, &feed.FetchMeta{}, nil
}

func TestSubscriptionRefreshAndDeletionPruneNodeHistory(t *testing.T) {
	manager, _, _ := deletionManager(t)
	e, err := xorm.NewEngine("sqlite", filepath.Join(t.TempDir(), "nodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	history := nodehistory.NewStore(e, func() (time.Duration, int) { return 24 * time.Hour, 10 })
	if err := history.Init(); err != nil {
		t.Fatal(err)
	}
	manager.nodeHistory = history
	manager.sublinkManager = historyFeed{}
	values := []nodehistory.Measurement{
		{Tag: "node | sub_1", SubID: "sub_1", At: time.Now(), Delay: 100, Success: true},
		{Tag: "node | sub_10", SubID: "sub_10", At: time.Now(), Delay: 200, Success: true},
	}
	if err := history.RecordBatch(values); err != nil {
		t.Fatal(err)
	}
	result, err := manager.Refresh(context.Background(), "sub_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.DeletedKeys) != 1 || len(result.AddedTags) != 1 {
		t.Fatalf("diff=%+v", result)
	}
	if got, _ := history.Series(values[0].Tag); len(got) != 0 {
		t.Fatal("removed node history survived subscription refresh")
	}
	if got, _ := history.Series(values[1].Tag); len(got) != 1 {
		t.Fatal("another subscription was pruned")
	}
	if err := history.RecordBatch(values[:1]); err != nil {
		t.Fatal(err)
	}
	if got, _ := history.Series(values[0].Tag); len(got) != 0 {
		t.Fatal("late probe resurrected removed history")
	}
	added := nodehistory.Measurement{Tag: result.AddedTags[0], SubID: "sub_1", At: time.Now(), Delay: 90, Success: true}
	if err := history.RecordBatch([]nodehistory.Measurement{added}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Refresh(context.Background(), "sub_1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := history.Series(added.Tag); len(got) != 1 {
		t.Fatal("unchanged node history lost on refresh")
	}
	if err := manager.Delete("sub_1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := history.Series(added.Tag); len(got) != 0 {
		t.Fatal("deleted subscription history survived")
	}
}
