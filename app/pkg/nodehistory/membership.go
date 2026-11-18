package nodehistory

import (
	"encoding/json"

	"xorm.io/xorm"
)

// Membership is the current set of nodes owned by a subscription. Empty Tags
// is a durable deletion fence: an already-running manual test cannot recreate
// history after the node was removed by an update.
type Membership struct {
	SubID string `xorm:"'sub_id' pk"`
	Tags  string `xorm:"TEXT"`
}

func (Membership) TableName() string { return "node_probe_membership" }

func readMembership(tx *xorm.Session, subID string) (map[string]bool, error) {
	if subID == "" {
		return nil, nil
	}
	var row Membership
	found, err := tx.ID(subID).Get(&row)
	if err != nil || !found {
		return nil, err
	}
	var tags []string
	if err := json.Unmarshal([]byte(row.Tags), &tags); err != nil {
		return nil, err
	}
	allowed := make(map[string]bool, len(tags))
	for _, tag := range tags {
		allowed[tag] = true
	}
	return allowed, nil
}

// SyncSubscription prunes removed nodes and publishes the new admission set in
// the same transaction. Unchanged nodes keep every retained measurement.
func (s *Store) SyncSubscription(subID string, tags []string) error {
	raw, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	allowed := make(map[string]bool, len(tags))
	for _, tag := range tags {
		allowed[tag] = true
	}
	tx := s.e.NewSession()
	defer tx.Close()
	if err := tx.Begin(); err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO node_probe_membership(sub_id,tags) VALUES(?,?)
ON CONFLICT(sub_id) DO UPDATE SET tags=excluded.tags`, subID, string(raw)); err != nil {
		return err
	}
	var targets []Target
	if err := tx.Where("sub_id = ?", subID).Find(&targets); err != nil {
		return err
	}
	var removed []int64
	for _, target := range targets {
		if !allowed[target.Tag] {
			removed = append(removed, target.ID)
		}
	}
	if len(removed) > 0 {
		if _, err := tx.In("target_id", removed).Delete(new(Sample)); err != nil {
			return err
		}
		if _, err := tx.In("id", removed).Delete(new(Target)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
