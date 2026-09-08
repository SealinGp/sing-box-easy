package settings

import "github.com/SealinGp/sing-box-easy/app/pkg/settings/repo"

// SetMany commits a settings form atomically after its business validation.
func (m *ManagerXORM) SetMany(values map[string]string) error {
	tx := m.e.NewSession()
	defer tx.Close()
	if err := tx.Begin(); err != nil {
		return err
	}
	defer tx.Rollback()
	for key, value := range values {
		n, err := tx.ID(key).Cols("value").Update(&repo.Setting{Value: value})
		if err != nil {
			return err
		}
		if n == 0 {
			if _, err := tx.Insert(&repo.Setting{Key: key, Value: value}); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
