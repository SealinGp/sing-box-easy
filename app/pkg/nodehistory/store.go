// Package nodehistory owns durable observations shared by scheduled and manual probes.
package nodehistory

import (
	"fmt"
	"strings"
	"time"

	"xorm.io/xorm"
)

// Target keeps repeated tags and URLs out of every sample row.
type Target struct {
	ID      int64  `xorm:"pk autoincr 'id'"`
	Tag     string `xorm:"notnull unique(node_target) index"`
	TestURL string `xorm:"'test_url' notnull unique(node_target)"`
	SubID   string `xorm:"'sub_id' index"`
}

func (Target) TableName() string { return "node_probe_targets" }

type Sample struct {
	ID       int64 `xorm:"pk autoincr 'id'"`
	TargetID int64 `xorm:"'target_id' notnull index(target_at)"`
	At       int64 `xorm:"notnull index(target_at)"` // Unix milliseconds
	Delay    int
	Success  bool
	Source   string
}

func (Sample) TableName() string { return "node_probe_samples" }

type Measurement struct {
	Tag, SubID, TestURL, Source string
	At                          time.Time
	Delay                       int
	Success                     bool
}

type Point struct {
	Time    time.Time `json:"time"`
	Delay   int       `json:"delay"`
	Success bool      `json:"success"`
	Source  string    `json:"source"`
	TestURL string    `json:"test_url"`
}

type Store struct {
	e      *xorm.Engine
	bounds func() (time.Duration, int)
}

func NewStore(e *xorm.Engine, bounds func() (time.Duration, int)) *Store {
	return &Store{e: e, bounds: bounds}
}

func (s *Store) Init() error { return s.e.Sync2(new(Target), new(Sample), new(Membership)) }

// SubscriptionID reads the ownership suffix minted for imported nodes.
func SubscriptionID(tag string) string {
	position := strings.LastIndex(tag, " | sub_")
	if position < 0 || position+len(" | sub_") >= len(tag) {
		return ""
	}
	return tag[position+3:]
}

// RecordBatch commits a measured run and enforces age/count limits atomically.
// No caller records untestable nodes or a failed controller connection here.
func (s *Store) RecordBatch(values []Measurement) error {
	if len(values) == 0 {
		return nil
	}
	// Read settings before acquiring SQLite's single transaction connection.
	age, points := s.bounds()
	tx := s.e.NewSession()
	defer tx.Close()
	if err := tx.Begin(); err != nil {
		return err
	}
	defer tx.Rollback()
	tags := make(map[string]bool)
	memberships := make(map[string]map[string]bool)
	for _, value := range values {
		if value.Tag == "" || value.At.IsZero() || value.Delay < 0 {
			return fmt.Errorf("invalid node observation")
		}
		allowed, loaded := memberships[value.SubID]
		if !loaded {
			var err error
			allowed, err = readMembership(tx, value.SubID)
			if err != nil {
				return err
			}
			memberships[value.SubID] = allowed
		}
		if allowed != nil && !allowed[value.Tag] {
			continue
		}
		target := Target{Tag: value.Tag, TestURL: value.TestURL, SubID: value.SubID}
		found, err := tx.Where("tag = ? AND test_url = ?", value.Tag, value.TestURL).Get(&target)
		if err != nil {
			return err
		}
		if !found {
			if _, err := tx.Insert(&target); err != nil {
				return err
			}
		}
		row := Sample{TargetID: target.ID, At: value.At.UnixMilli(), Delay: value.Delay, Success: value.Success, Source: value.Source}
		if _, err := tx.Insert(&row); err != nil {
			return err
		}
		tags[value.Tag] = true
	}
	if err := s.trim(tx, tags, age, points); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) trim(tx *xorm.Session, tags map[string]bool, age time.Duration, points int) error {
	if points < 1 || age <= 0 {
		return fmt.Errorf("invalid node history retention")
	}
	if _, err := tx.Exec("DELETE FROM node_probe_samples WHERE at < ?", time.Now().Add(-age).UnixMilli()); err != nil {
		return err
	}
	for tag := range tags {
		_, err := tx.Exec(`DELETE FROM node_probe_samples WHERE id IN (
SELECT s.id FROM node_probe_samples s JOIN node_probe_targets t ON t.id=s.target_id
WHERE t.tag=? ORDER BY s.at DESC,s.id DESC LIMIT -1 OFFSET ?)`, tag, points)
		if err != nil {
			return err
		}
	}
	_, err := tx.Exec("DELETE FROM node_probe_targets WHERE id NOT IN (SELECT target_id FROM node_probe_samples)")
	return err
}

// TrimAll applies a tightened retention policy immediately, also to quiet nodes.
func (s *Store) TrimAll() error {
	age, points := s.bounds()
	var targets []Target
	if err := s.e.Find(&targets); err != nil {
		return err
	}
	tags := make(map[string]bool)
	for _, target := range targets {
		tags[target.Tag] = true
	}
	tx := s.e.NewSession()
	defer tx.Close()
	if err := tx.Begin(); err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.trim(tx, tags, age, points); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Count() (int64, error) { return s.e.Count(new(Sample)) }
