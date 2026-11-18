package nodehistory

import (
	"strings"
	"time"
)

type storedPoint struct {
	Tag, Source string
	TestURL     string `xorm:"'test_url'"`
	At          int64
	Delay       int
	Success     bool
}

// Recent makes one bounded query for all visible nodes, not one query per card.
func (s *Store) Recent(tags []string, limit int) (map[string][]Point, error) {
	result := make(map[string][]Point)
	if len(tags) == 0 {
		return result, nil
	}
	age, _ := s.bounds()
	args := []any{time.Now().Add(-age).UnixMilli()}
	for _, tag := range tags {
		args = append(args, tag)
	}
	args = append(args, limit)
	query := `SELECT tag,test_url,at,delay,success,source FROM (
SELECT t.tag,t.test_url,s.at,s.delay,s.success,s.source,s.id,
ROW_NUMBER() OVER (PARTITION BY t.tag ORDER BY s.at DESC,s.id DESC) AS position
FROM node_probe_samples s JOIN node_probe_targets t ON t.id=s.target_id
WHERE s.at>=? AND t.tag IN (` + strings.TrimSuffix(strings.Repeat("?,", len(tags)), ",") + `))
WHERE position<=? ORDER BY at ASC,id ASC`
	var rows []storedPoint
	if err := s.e.SQL(query, args...).Find(&rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.Tag] = append(result[row.Tag], Point{Time: time.UnixMilli(row.At), Delay: row.Delay, Success: row.Success, Source: row.Source, TestURL: row.TestURL})
	}
	return result, nil
}

func (s *Store) Series(tag string) ([]Point, error) {
	_, points := s.bounds()
	rows, err := s.Recent([]string{tag}, points)
	if err != nil {
		return nil, err
	}
	if rows[tag] == nil {
		return []Point{}, nil
	}
	return rows[tag], nil
}
