package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type LocalRuleFile struct {
	Tag       string          `json:"tag"`
	Path      string          `json:"path"`
	Directory string          `json:"directory"`
	Content   json.RawMessage `json:"content"`
	Revision  string          `json:"revision"`
}
type LocalRuleWrite struct {
	Tag      string          `json:"tag"`
	Existing bool            `json:"existing"`
	Revision string          `json:"revision"`
	Content  json.RawMessage `json:"content"`
}

func localRuleFilename(tag string) (string, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", errors.New("rule-set name is required")
	}
	name := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, tag)
	name = strings.TrimRight(name, ". ")
	if name == "" || len([]byte(name)) > 240 {
		return "", errors.New("invalid or too long rule-set filename")
	}
	return name + ".json", nil
}
func fileRevision(config, content []byte) string {
	sum := sha256.Sum256(append(append(append([]byte{}, config...), 0), content...))
	return hex.EncodeToString(sum[:])
}
func (m *Manager) localRuleFile(tag string) (LocalRuleFile, []byte, error) {
	raw, err := os.ReadFile(m.configPath)
	if err != nil {
		return LocalRuleFile{}, nil, err
	}
	dir, err := filepath.Abs(filepath.Join(filepath.Dir(m.configPath), "rule-sets", "user"))
	if err != nil {
		return LocalRuleFile{}, nil, err
	}
	result := LocalRuleFile{Tag: tag, Directory: dir, Content: json.RawMessage(`{"version":3,"rules":[]}`)}
	if tag != "" {
		var doc struct {
			Route struct {
				RuleSets []struct {
					Tag    string `json:"tag"`
					Type   string `json:"type"`
					Format string `json:"format"`
					Path   string `json:"path"`
				} `json:"rule_set"`
			} `json:"route"`
		}
		if err = json.Unmarshal(raw, &doc); err != nil {
			return result, nil, err
		}
		for _, r := range doc.Route.RuleSets {
			if r.Tag != tag {
				continue
			}
			if r.Type != "local" || (r.Format != "source" && !(r.Format == "" && strings.HasSuffix(r.Path, ".json"))) {
				return result, nil, errors.New("only local source JSON files can be edited")
			}
			result.Path = r.Path
			// Preserve the core's process-working-directory interpretation for relative paths.
			result.Path, err = filepath.Abs(result.Path)
			if err != nil {
				return result, nil, err
			}
			info, e := os.Lstat(result.Path)
			if e != nil {
				return result, nil, e
			}
			if !info.Mode().IsRegular() || info.Size() > 4<<20 {
				return result, nil, errors.New("rule-set must be a regular JSON file smaller than 4 MiB")
			}
			result.Content, err = os.ReadFile(result.Path)
			if err != nil {
				return result, nil, err
			}
			if !json.Valid(result.Content) {
				return result, nil, errors.New("existing rule-set is not valid JSON")
			}
			result.Revision = fileRevision(raw, result.Content)
			return result, raw, nil
		}
		return result, nil, errors.New("rule-set not found")
	}
	result.Revision = fileRevision(raw, nil)
	return result, raw, nil
}
func (m *Manager) ReadLocalRuleFile(tag string) (LocalRuleFile, error) {
	m.mutationMu.Lock()
	defer m.mutationMu.Unlock()
	file, _, err := m.localRuleFile(tag)
	return file, err
}
func (m *Manager) WriteLocalRuleFile(ctx context.Context, req LocalRuleWrite) (LocalRuleFile, error) {
	m.mutationMu.Lock()
	defer m.mutationMu.Unlock()
	req.Tag = strings.TrimSpace(req.Tag)
	lookup := ""
	if req.Existing {
		lookup = req.Tag
	}
	file, raw, err := m.localRuleFile(lookup)
	if err != nil {
		return file, err
	}
	if m.writeBarrier != nil {
		if err := m.writeBarrier(raw); err != nil {
			return file, err
		}
	}
	if req.Revision == "" || file.Revision != req.Revision {
		return file, errors.New("configuration or file changed; reload before saving")
	}
	name, err := localRuleFilename(req.Tag)
	if err != nil {
		return file, err
	}
	if !req.Existing {
		file.Path = filepath.Join(file.Directory, name)
	}
	if err = validateRuleSource(req.Content); err != nil {
		return file, err
	}
	var doc map[string]json.RawMessage
	if err = json.Unmarshal(raw, &doc); err != nil {
		return file, err
	}
	route := map[string]json.RawMessage{}
	if len(doc["route"]) > 0 {
		if err = json.Unmarshal(doc["route"], &route); err != nil {
			return file, err
		}
	}
	var sets []map[string]json.RawMessage
	if len(route["rule_set"]) > 0 {
		if err = json.Unmarshal(route["rule_set"], &sets); err != nil {
			return file, err
		}
	}
	index := -1
	for i, r := range sets {
		var tag string
		_ = json.Unmarshal(r["tag"], &tag)
		if tag == req.Tag {
			index = i
		}
	}
	if req.Existing && index < 0 {
		return file, errors.New("rule-set removed")
	}
	if !req.Existing && index >= 0 {
		return file, errors.New("rule-set name already exists")
	}
	if !req.Existing {
		sets = append(sets, map[string]json.RawMessage{"tag": jsonString(req.Tag), "type": jsonString("local"), "format": jsonString("source"), "path": jsonString(file.Path)})
		index = len(sets) - 1
	}
	if err = os.MkdirAll(filepath.Dir(file.Path), 0755); err != nil {
		return file, err
	}
	// Validate against a staging file before changing any active rule-set.
	tmp, err := writeRuleTemp(filepath.Dir(file.Path), req.Content)
	if err != nil {
		return file, err
	}
	defer os.Remove(tmp)
	oldPath := sets[index]["path"]
	oldFormat := sets[index]["format"]
	sets[index]["format"] = jsonString("source")
	sets[index]["path"] = jsonString(tmp)
	route["rule_set"], _ = json.Marshal(sets)
	doc["route"], _ = json.Marshal(route)
	staged, _ := json.Marshal(doc)
	if err = m.validateDocumentLocked(ctx, ConfigDocument{Raw: staged}); err != nil {
		return file, err
	}
	sets[index]["path"] = oldPath
	if oldFormat == nil {
		delete(sets[index], "format")
	} else {
		sets[index]["format"] = oldFormat
	}
	currentConfig, err := os.ReadFile(m.configPath)
	if err != nil {
		return file, err
	}
	if string(currentConfig) != string(raw) {
		return file, errors.New("configuration changed during validation")
	}
	if req.Existing {
		backup, err := writeRuleTemp(filepath.Dir(file.Path), file.Content)
		if err != nil {
			return file, err
		}
		defer os.Remove(backup)
		if err = os.Rename(backup, file.Path+".bak"); err != nil {
			return file, err
		}
		// Check for external edits made while core validation was running.
		currentConfig, err := os.ReadFile(m.configPath)
		if err != nil {
			return file, err
		}
		current, err := os.ReadFile(file.Path)
		if err != nil {
			return file, err
		}
		if fileRevision(currentConfig, current) != req.Revision {
			return file, errors.New("rule-set file changed during validation")
		}
		info, err := os.Stat(file.Path)
		if err != nil {
			return file, err
		}
		if err = os.Chmod(tmp, info.Mode().Perm()); err != nil {
			return file, err
		}
		if err = os.Rename(tmp, file.Path); err != nil {
			return file, err
		}
	} else {
		// Link is exclusive: a pre-existing file is never overwritten.
		if err = os.Link(tmp, file.Path); err != nil {
			return file, fmt.Errorf("create rule-set file: %w", err)
		}
		route["rule_set"], _ = json.Marshal(sets)
		doc["route"], _ = json.Marshal(route)
		updated, _ := json.MarshalIndent(doc, "", "  ")
		if err = m.saveDocumentLocked(ctx, ConfigDocument{Raw: updated}); err != nil {
			return file, errors.Join(err, os.Remove(file.Path))
		}
	}
	result, _, err := m.localRuleFile(req.Tag)
	return result, err
}
func jsonString(s string) json.RawMessage { data, _ := json.Marshal(s); return data }
func writeRuleTemp(dir string, content []byte) (string, error) {
	f, err := os.CreateTemp(dir, ".rules-*.tmp")
	if err != nil {
		return "", err
	}
	name := f.Name()
	// Rule-set files contain matching criteria and must be readable by the core's user.
	err = f.Chmod(0644)
	if err == nil {
		_, err = f.Write(content)
	}
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		os.Remove(name)
	}
	return name, err
}
func validateRuleSource(data []byte) error {
	if len(data) > 4<<20 {
		return errors.New("rule-set exceeds 4 MiB")
	}
	var source struct {
		Version int                          `json:"version"`
		Rules   []map[string]json.RawMessage `json:"rules"`
	}
	if err := json.Unmarshal(data, &source); err != nil {
		return err
	}
	if source.Version < 1 || len(source.Rules) == 0 {
		return errors.New("version and at least one non-empty rule are required")
	}
	for _, rule := range source.Rules {
		meaningful := false
		for key, value := range rule {
			if key != "invert" && key != "type" && key != "mode" && string(value) != "[]" && string(value) != "null" {
				meaningful = true
			}
		}
		if !meaningful {
			return errors.New("empty matching rule is not allowed")
		}
	}
	return nil
}
