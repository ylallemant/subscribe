// Package server exposes the HTTP API and serves the embedded web UI.
//
// Two modes share the same server: a stateless "quick mode" (parse/export on
// demand, progress kept in the browser) and project mode, backed by the on-disk
// project store.
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ylallemant/subscribe/internal/export"
	"github.com/ylallemant/subscribe/internal/parser"
	"github.com/ylallemant/subscribe/internal/project"
	"github.com/ylallemant/subscribe/internal/reading"
	"github.com/ylallemant/subscribe/web"
)

// Options configures the server's reading-speed behaviour and project store.
type Options struct {
	FPS           float64
	Reading       reading.Config
	Store         *project.Store // on-disk project store
	DisableDelete bool           // when true, project deletion is forbidden
}

// New returns an http.Handler with the API and static assets mounted.
func New(opts Options) http.Handler {
	s := &srv{opts: opts}
	mux := http.NewServeMux()
	// Quick mode (stateless upload/review).
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/parse", s.handleParse)
	mux.HandleFunc("/api/export", s.handleExport)
	// Projects.
	mux.HandleFunc("GET /api/capabilities", s.handleCapabilities)
	mux.HandleFunc("GET /api/languages", s.handleLanguages)
	mux.HandleFunc("GET /api/projects", s.handleListProjects)
	mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	mux.HandleFunc("GET /api/projects/{slug}", s.handleGetProject)
	mux.HandleFunc("DELETE /api/projects/{slug}", s.handleDeleteProject)
	mux.HandleFunc("PUT /api/projects/{slug}/settings", s.handleSaveSettings)
	mux.HandleFunc("POST /api/projects/{slug}/files", s.handleImportFile)
	mux.HandleFunc("GET /api/projects/{slug}/original", s.handleProjectOriginal)
	mux.HandleFunc("GET /api/projects/{slug}/translations/{lang}", s.handleLoadTranslation)
	mux.HandleFunc("PUT /api/projects/{slug}/translations/{lang}", s.handleSaveTranslation)
	// Static assets. Served with no-cache so a rebuild's updated embedded
	// assets are always picked up (ES modules cache aggressively otherwise).
	mux.Handle("/", noCache(http.FileServer(http.FS(web.FS()))))
	return mux
}

// noCache tells the browser to always revalidate static assets, avoiding stale
// cached JS modules after the binary is rebuilt.
func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		h.ServeHTTP(w, r)
	})
}

type srv struct{ opts Options }

// clientConfig is the subset of config the browser needs to mirror the
// severity formula and label the UI.
type clientConfig struct {
	FPS          float64 `json:"fps"`
	Metric       string  `json:"metric"`
	CPSMax       float64 `json:"cpsMax"`
	WPSMax       float64 `json:"wpsMax"`
	WarnFraction float64 `json:"warnFraction"`
	FullFraction float64 `json:"fullFraction"`
}

func (s *srv) clientConfig() clientConfig {
	return clientConfig{
		FPS:          s.opts.FPS,
		Metric:       string(s.opts.Reading.Metric),
		CPSMax:       s.opts.Reading.CPSMax,
		WPSMax:       s.opts.Reading.WPSMax,
		WarnFraction: s.opts.Reading.WarnFraction,
		FullFraction: s.opts.Reading.FullFraction,
	}
}

func (s *srv) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.clientConfig())
}

// blockDTO is a parsed block enriched with everything the UI needs.
type blockDTO struct {
	Index    int      `json:"index"`
	Start    string   `json:"start"`
	End      string   `json:"end"`
	Lines    []string `json:"lines"`
	Text     string   `json:"text"`
	Duration float64  `json:"duration"` // seconds on screen
}

type parseResponse struct {
	Config clientConfig `json:"config"`
	Blocks []blockDTO   `json:"blocks"`
}

func (s *srv) handleParse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	content, err := readUpload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	blocks, err := parser.Parse(newStringReader(content))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := parseResponse{
		Config: s.clientConfig(),
		Blocks: blocksDTO(blocks, s.opts.FPS),
	}
	writeJSON(w, http.StatusOK, resp)
}

// blocksDTO converts parsed blocks to the wire form, computing on-screen
// duration from the given frame rate.
func blocksDTO(blocks []parser.Block, fps float64) []blockDTO {
	out := make([]blockDTO, 0, len(blocks))
	for _, b := range blocks {
		dur := b.End.ToSeconds(fps) - b.Start.ToSeconds(fps)
		if dur < 0 {
			dur = 0
		}
		out = append(out, blockDTO{
			Index:    b.Index,
			Start:    b.Start.String(),
			End:      b.End.String(),
			Lines:    b.Lines,
			Text:     b.Text(),
			Duration: dur,
		})
	}
	return out
}

type exportRequest struct {
	Format       string   `json:"format"`
	Source       string   `json:"source"`       // raw reference file content
	Translations []string `json:"translations"` // indexed by block
}

func (s *srv) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req exportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	blocks, err := parser.Parse(newStringReader(req.Source))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	format := export.Format(req.Format)
	out, err := export.Render(format, blocks, req.Translations, s.opts.FPS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", format.ContentType())
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="translation.%s"`, format.Extension()))
	_, _ = io.WriteString(w, out)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
