package repo

import "fmt"

// PendingDeletion survives failures between database, config and engine changes.
type PendingDeletion struct {
	ID string `xorm:"'id' pk"`
}

func (PendingDeletion) TableName() string { return "subscription_pending_deletions" }
func (s *Store) MarkDeleting(id string) error {
	_, err := s.e.Exec("INSERT OR IGNORE INTO subscription_pending_deletions (id) VALUES (?)", id)
	return err
}
func (s *Store) PendingDeletions() ([]string, error) {
	var rows []PendingDeletion
	if err := s.e.Find(&rows); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids, nil
}
func (s *Store) IsDeleting(id string) (bool, error) { return s.e.ID(id).Exist(&PendingDeletion{}) }

// FinishDeletion removes both records in the same SQLite transaction.
func (s *Store) FinishDeletion(id string) error {
	tx := s.e.NewSession()
	defer tx.Close()
	if err := tx.Begin(); err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM subscription_probe_samples WHERE sub_id = ?", id); err != nil {
		return err
	}
	n, err := tx.ID(id).Delete(&Subscription{})
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("subscription not found")
	}
	if _, err := tx.ID(id).Delete(&PendingDeletion{}); err != nil {
		return err
	}
	return tx.Commit()
}
