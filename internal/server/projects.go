package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"

	"github.com/ylallemant/subscribe/internal/langs"
	"github.com/ylallemant/subscribe/internal/project"
)

// storeReady guards project endpoints when no data dir is configured.
func (s *srv) storeReady(w http.ResponseWriter) bool {
	if s.opts.Store == nil {
		http.Error(w, "project storage is not configured", http.StatusServiceUnavailable)
		return false
	}
	return true
}

func (s *srv) handleLanguages(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, langs.All)
}

func (s *srv) handleListProjects(w http.ResponseWriter, r *http.Request) {
	if !s.storeReady(w) {
		return
	}
	list, err := s.opts.Store.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []project.Summary{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *srv) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	if !s.storeReady(w) {
		return
	}
	var req struct {
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	p, err := s.opts.Store.Create(req.DisplayName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *srv) handleGetProject(w http.ResponseWriter, r *http.Request) {
	if !s.storeReady(w) {
		return
	}
	p, err := s.opts.Store.Get(r.PathValue("slug"))
	if s.writeStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *srv) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	if !s.storeReady(w) {
		return
	}
	if s.writeStoreErr(w, s.opts.Store.Delete(r.PathValue("slug"))) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *srv) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	if !s.storeReady(w) {
		return
	}
	var set project.Settings
	if err := json.NewDecoder(r.Body).Decode(&set); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	switch set.Format {
	case "txt", "srt", "vtt":
	default:
		http.Error(w, "format must be txt, srt or vtt", http.StatusBadRequest)
		return
	}
	if set.Metric != "wps" {
		set.Metric = "cps"
	}
	if set.ReferenceLang != "" && !langs.Valid(set.ReferenceLang) {
		http.Error(w, "unknown reference language", http.StatusBadRequest)
		return
	}
	if err := s.opts.Store.SaveSettings(r.PathValue("slug"), set); s.writeStoreErr(w, err) {
		return
	}
	p, _ := s.opts.Store.Get(r.PathValue("slug"))
	writeJSON(w, http.StatusOK, p)
}

// handleImportFile accepts a multipart upload (fields "file" and "lang") and
// stores it as the project's translation file for that language.
func (s *srv) handleImportFile(w http.ResponseWriter, r *http.Request) {
	if !s.storeReady(w) {
		return
	}
	slug := r.PathValue("slug")
	content, name, err := readUploadNamed(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lang := r.FormValue("lang")
	if !langs.Valid(lang) {
		http.Error(w, "choose a valid language for the file", http.StatusBadRequest)
		return
	}
	ext := filepath.Ext(name)
	if err := s.opts.Store.ImportFile(slug, lang, []byte(content), ext); s.writeStoreErr(w, err) {
		return
	}
	p, _ := s.opts.Store.Get(slug)
	writeJSON(w, http.StatusOK, p)
}

// originalResponse carries the parsed reference plus the project's reading
// config and the plain source (so the client can request exports).
type originalResponse struct {
	Config clientConfig `json:"config"`
	Blocks []blockDTO   `json:"blocks"`
	Source string       `json:"source"`
}

func (s *srv) handleProjectOriginal(w http.ResponseWriter, r *http.Request) {
	if !s.storeReady(w) {
		return
	}
	slug := r.PathValue("slug")
	blocks, err := s.opts.Store.Blocks(slug)
	if s.writeStoreErr(w, err) {
		return
	}
	p, _ := s.opts.Store.Get(slug)
	cfg := s.settingsConfig(p.Settings)
	src, _ := s.opts.Store.ReferenceSource(slug)
	writeJSON(w, http.StatusOK, originalResponse{
		Config: cfg,
		Blocks: blocksDTO(blocks, cfg.FPS),
		Source: src,
	})
}

func (s *srv) handleLoadTranslation(w http.ResponseWriter, r *http.Request) {
	if !s.storeReady(w) {
		return
	}
	lang := r.PathValue("lang")
	if !langs.Valid(lang) {
		http.Error(w, "unknown language", http.StatusBadRequest)
		return
	}
	tr, err := s.opts.Store.LoadTranslation(r.PathValue("slug"), lang)
	if s.writeStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"translations": tr})
}

func (s *srv) handleSaveTranslation(w http.ResponseWriter, r *http.Request) {
	if !s.storeReady(w) {
		return
	}
	lang := r.PathValue("lang")
	if !langs.Valid(lang) {
		http.Error(w, "unknown language", http.StatusBadRequest)
		return
	}
	var req struct {
		Translations []string `json:"translations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := s.opts.Store.SaveTranslation(r.PathValue("slug"), lang, req.Translations); s.writeStoreErr(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// settingsConfig maps project settings to the browser reading-speed config.
func (s *srv) settingsConfig(set project.Settings) clientConfig {
	cfg := clientConfig{
		FPS:          set.FPS,
		Metric:       set.Metric,
		CPSMax:       set.CPSMax,
		WPSMax:       set.WPSMax,
		WarnFraction: s.opts.Reading.WarnFraction,
		FullFraction: s.opts.Reading.FullFraction,
	}
	if cfg.FPS <= 0 {
		cfg.FPS = 25
	}
	return cfg
}

// writeStoreErr writes an appropriate status for a store error; returns true if
// it handled (wrote) an error.
func (s *srv) writeStoreErr(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, project.ErrNotFound):
		http.Error(w, "project not found", http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	return true
}
