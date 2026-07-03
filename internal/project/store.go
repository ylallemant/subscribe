// Package project is the on-disk store for translation projects.
//
// Layout:
//
//	<dataDir>/project/<slug>/
//	  settings.yaml                # format, fps, cps/wps, reference language
//	  translations/
//	    <slug>.<ISO639-3>.<ext>    # one file per language (incl. the original)
//
// There is no separate "original" file: the reference is simply the translation
// file whose language equals settings.referenceLang. Every uploaded file is
// converted to the project's format and stored in translations/.
package project

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ylallemant/subscribe/internal/export"
	"github.com/ylallemant/subscribe/internal/parser"
)

// Store manages projects rooted at a data directory.
type Store struct {
	root string
}

// Summary is the light-weight listing entry for a project.
type Summary struct {
	Slug         string   `json:"slug"`
	DisplayName  string   `json:"displayName"`
	Languages    []string `json:"languages"`
	HasReference bool     `json:"hasReference"`
}

// Project is the full detail of a project.
type Project struct {
	Slug      string   `json:"slug"`
	Settings  Settings `json:"settings"`
	Languages []string `json:"languages"` // ISO codes that have a file
}

var ErrNotFound = errors.New("project not found")

// NewStore ensures <dataDir>/project exists and returns a Store.
func NewStore(dataDir string) (*Store, error) {
	s := &Store{root: dataDir}
	if err := os.MkdirAll(s.projectsDir(), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return s, nil
}

func (s *Store) projectsDir() string             { return filepath.Join(s.root, "project") }
func (s *Store) dir(slug string) string          { return filepath.Join(s.projectsDir(), slug) }
func (s *Store) transDir(slug string) string     { return filepath.Join(s.dir(slug), "translations") }
func (s *Store) settingsPath(slug string) string { return filepath.Join(s.dir(slug), "settings.yaml") }

func (s *Store) exists(slug string) bool {
	info, err := os.Stat(s.dir(slug))
	return err == nil && info.IsDir()
}

// List returns all projects, sorted by display name.
func (s *Store) List() ([]Summary, error) {
	entries, err := os.ReadDir(s.projectsDir())
	if err != nil {
		return nil, err
	}
	out := []Summary{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		slug := e.Name()
		set := s.readSettings(slug)
		out = append(out, Summary{
			Slug:         slug,
			DisplayName:  orElse(set.DisplayName, slug),
			Languages:    s.languages(slug),
			HasReference: s.hasReference(slug, set),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].DisplayName) < strings.ToLower(out[j].DisplayName)
	})
	return out, nil
}

// Create makes a new project from a display name, choosing a unique slug.
func (s *Store) Create(displayName string) (*Project, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, errors.New("display name is required")
	}
	base := Slugify(displayName)
	if base == "" {
		base = "project"
	}
	slug := base
	for i := 2; s.exists(slug); i++ {
		slug = fmt.Sprintf("%s_%d", base, i)
	}
	if err := os.MkdirAll(s.transDir(slug), 0o755); err != nil {
		return nil, err
	}
	if err := s.writeSettings(slug, DefaultSettings(displayName)); err != nil {
		return nil, err
	}
	return s.Get(slug)
}

// Get returns full project detail.
func (s *Store) Get(slug string) (*Project, error) {
	if !s.exists(slug) {
		return nil, ErrNotFound
	}
	return &Project{
		Slug:      slug,
		Settings:  s.readSettings(slug),
		Languages: s.languages(slug),
	}, nil
}

// Delete removes a project and all its files.
func (s *Store) Delete(slug string) error {
	if !s.exists(slug) {
		return ErrNotFound
	}
	return os.RemoveAll(s.dir(slug))
}

// ImportFile stores an uploaded subtitle file as the given language, converting
// it to the project's format. srcExt is the uploaded file's extension so we can
// parse SRT/VTT as well as the plain format.
func (s *Store) ImportFile(slug, lang string, content []byte, srcExt string) error {
	if !s.exists(slug) {
		return ErrNotFound
	}
	set := s.readSettings(slug)
	blocks, err := parser.ParseFormat(bytes.NewReader(content), parser.FormatFromExt(srcExt), fpsOf(set))
	if err != nil {
		return fmt.Errorf("parse uploaded file: %w", err)
	}
	out, err := export.Render(exportFormat(set.Format), blocks, nil, fpsOf(set))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.transDir(slug), 0o755); err != nil {
		return err
	}
	return atomicWrite(s.translationPath(slug, lang, set), []byte(out))
}

// SaveSettings persists settings; if the format changed, existing translation
// files are re-converted to the new format.
func (s *Store) SaveSettings(slug string, set Settings) error {
	if !s.exists(slug) {
		return ErrNotFound
	}
	old := s.readSettings(slug)
	if set.DisplayName == "" {
		set.DisplayName = old.DisplayName
	}
	if err := s.writeSettings(slug, set); err != nil {
		return err
	}
	if old.Format != set.Format {
		return s.reconvert(slug, old.Format, set.Format, fpsOf(set))
	}
	return nil
}

// reconvert rewrites every translation file from oldFmt to newFmt.
func (s *Store) reconvert(slug, oldFmt, newFmt string, fps float64) error {
	oldExt := (Settings{Format: oldFmt}).Ext()
	entries, err := os.ReadDir(s.transDir(slug))
	if err != nil {
		return nil
	}
	suffix := "." + oldExt
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), suffix) {
			continue
		}
		oldPath := filepath.Join(s.transDir(slug), e.Name())
		f, err := os.Open(oldPath)
		if err != nil {
			continue
		}
		blocks, err := parser.ParseFormat(f, parserFormat(oldFmt), fps)
		f.Close()
		if err != nil {
			continue
		}
		out, err := export.Render(exportFormat(newFmt), blocks, nil, fps)
		if err != nil {
			continue
		}
		lang := langFromFilename(slug, e.Name())
		newPath := s.translationPath(slug, lang, Settings{Format: newFmt})
		if err := atomicWrite(newPath, []byte(out)); err != nil {
			return err
		}
		if newPath != oldPath {
			os.Remove(oldPath)
		}
	}
	return nil
}

// Blocks parses the reference-language file into blocks.
func (s *Store) Blocks(slug string) ([]parser.Block, error) {
	if !s.exists(slug) {
		return nil, ErrNotFound
	}
	set := s.readSettings(slug)
	path, ok := s.referenceFile(slug, set)
	if !ok {
		return nil, fmt.Errorf("no reference file for this project yet — add one and set the original language")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open reference: %w", err)
	}
	defer f.Close()
	return parser.ParseFormat(f, parserFormat(set.Format), fpsOf(set))
}

// referenceFile returns the path of the reference-language file and whether it
// exists.
func (s *Store) referenceFile(slug string, set Settings) (string, bool) {
	if set.ReferenceLang == "" {
		return "", false
	}
	p := s.translationPath(slug, set.ReferenceLang, set)
	return p, fileExists(p)
}

func (s *Store) hasReference(slug string, set Settings) bool {
	_, ok := s.referenceFile(slug, set)
	return ok
}

// languages lists ISO codes that have a translation file (sorted, never nil).
func (s *Store) languages(slug string) []string {
	entries, err := os.ReadDir(s.transDir(slug))
	if err != nil {
		return []string{}
	}
	seen := map[string]bool{}
	out := []string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if lang := langFromFilename(slug, e.Name()); lang != "" && !seen[lang] {
			seen[lang] = true
			out = append(out, lang)
		}
	}
	sort.Strings(out)
	return out
}

// langFromFilename extracts "<lang>" from "<slug>.<lang>.<ext>".
func langFromFilename(slug, name string) string {
	prefix := slug + "."
	if !strings.HasPrefix(name, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(name, prefix) // "<lang>.<ext>"
	if lang, _, ok := strings.Cut(rest, "."); ok {
		return lang
	}
	return ""
}

func (s *Store) translationPath(slug, lang string, set Settings) string {
	return filepath.Join(s.transDir(slug), fmt.Sprintf("%s.%s.%s", slug, lang, set.Ext()))
}

// LoadTranslation returns the translation for lang aligned to the reference
// blocks. A missing file yields an all-empty slice (a brand-new translation).
func (s *Store) LoadTranslation(slug, lang string) ([]string, error) {
	blocks, err := s.Blocks(slug)
	if err != nil {
		return nil, err
	}
	set := s.readSettings(slug)
	empty := make([]string, len(blocks))

	f, err := os.Open(s.translationPath(slug, lang, set))
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	trBlocks, err := parser.ParseFormat(f, parserFormat(set.Format), fpsOf(set))
	if err != nil {
		return empty, nil // unreadable/partial → start blank rather than fail
	}
	return align(blocks, trBlocks), nil
}

// SaveTranslation renders the translations to the project format and writes the
// <slug>.<lang>.<ext> file atomically.
func (s *Store) SaveTranslation(slug, lang string, translations []string) error {
	blocks, err := s.Blocks(slug)
	if err != nil {
		return err
	}
	set := s.readSettings(slug)
	out, err := export.Render(exportFormat(set.Format), blocks, translations, fpsOf(set))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.transDir(slug), 0o755); err != nil {
		return err
	}
	return atomicWrite(s.translationPath(slug, lang, set), []byte(out))
}

// ReferenceSource returns the reference rendered as the plain time-boxed format
// (used by the client to build exports); ok is false if no reference is set.
func (s *Store) ReferenceSource(slug string) (string, bool) {
	blocks, err := s.Blocks(slug)
	if err != nil {
		return "", false
	}
	out, err := export.Render(export.Plain, blocks, nil, fpsOf(s.readSettings(slug)))
	if err != nil {
		return "", false
	}
	return out, true
}

// --- settings io ------------------------------------------------------------

func (s *Store) readSettings(slug string) Settings {
	b, err := os.ReadFile(s.settingsPath(slug))
	if err != nil {
		return DefaultSettings(slug)
	}
	set := ParseSettings(string(b))
	if set.FPS == 0 {
		set.FPS = 25
	}
	if set.Format == "" {
		set.Format = "srt"
	}
	if set.Metric == "" {
		set.Metric = "cps"
	}
	return set
}

func (s *Store) writeSettings(slug string, set Settings) error {
	return atomicWrite(s.settingsPath(slug), []byte(set.MarshalYAML()))
}

// --- helpers ----------------------------------------------------------------

func fpsOf(set Settings) float64 {
	if set.FPS > 0 {
		return set.FPS
	}
	return 25
}

func parserFormat(f string) parser.Format {
	switch f {
	case "srt":
		return parser.SRT
	case "vtt":
		return parser.VTT
	default:
		return parser.Plain
	}
}

func exportFormat(f string) export.Format {
	switch f {
	case "srt":
		return export.SRT
	case "vtt":
		return export.VTT
	default:
		return export.Plain
	}
}

// align maps translated blocks onto reference blocks: exact time box, then start
// timecode, then position when counts match.
func align(ref, tr []parser.Block) []string {
	byKey := make(map[string]string, len(tr))
	byStart := make(map[string]string, len(tr))
	for _, b := range tr {
		byKey[b.Start.String()+"|"+b.End.String()] = b.Text()
		byStart[b.Start.String()] = b.Text()
	}
	sameCount := len(ref) == len(tr)
	out := make([]string, len(ref))
	for i, b := range ref {
		if v, ok := byKey[b.Start.String()+"|"+b.End.String()]; ok {
			out[i] = v
		} else if v, ok := byStart[b.Start.String()]; ok {
			out[i] = v
		} else if sameCount {
			out[i] = tr[i].Text()
		}
	}
	return out
}

// atomicWrite writes via a temp file + rename so readers never see a partial
// file (last-write-wins under concurrent saves).
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func orElse(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
