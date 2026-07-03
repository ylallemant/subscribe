// Package reading scores how comfortably a subtitle can be read in the time it
// is on screen, using either characters-per-second (CPS, the subtitle-industry
// standard) or words-per-second (WPS).
//
// Severity is a continuous value in [0,1]: 0 means comfortable (render with no
// colour) and 1 means well over budget (full red). The web UI mirrors this same
// formula so colouring updates live as the user types; this implementation is
// used for server-side reporting and export validation.
package reading

import (
	"strings"
	"unicode/utf8"
)

// Metric selects the reading-speed unit.
type Metric string

const (
	MetricCPS Metric = "cps"
	MetricWPS Metric = "wps"
)

// Config holds the thresholds. WarnFraction/FullFraction are expressed as a
// fraction of Max: colour starts at WarnFraction*Max and reaches full red at
// FullFraction*Max.
type Config struct {
	Metric       Metric
	CPSMax       float64
	WPSMax       float64
	WarnFraction float64
	FullFraction float64
}

// Default returns sensible defaults (CPS, 17 chars/s, 3 words/s).
func Default() Config {
	return Config{
		Metric:       MetricCPS,
		CPSMax:       17,
		WPSMax:       3,
		WarnFraction: 0.85,
		FullFraction: 1.25,
	}
}

// Result is the outcome of scoring one block.
type Result struct {
	CharsPerSecond float64 `json:"cps"`
	WordsPerSecond float64 `json:"wps"`
	Severity       float64 `json:"severity"` // 0 = fine, 1 = full red
}

// Evaluate scores text displayed for durationSeconds.
func (c Config) Evaluate(text string, durationSeconds float64) Result {
	chars := charCount(text)
	words := wordCount(text)

	var cps, wps float64
	if durationSeconds > 0 {
		cps = float64(chars) / durationSeconds
		wps = float64(words) / durationSeconds
	}

	var actual, max float64
	switch c.Metric {
	case MetricWPS:
		actual, max = wps, c.WPSMax
	default:
		actual, max = cps, c.CPSMax
	}

	return Result{
		CharsPerSecond: cps,
		WordsPerSecond: wps,
		Severity:       severity(actual, max, c.WarnFraction, c.FullFraction),
	}
}

// severity ramps linearly from 0 (at warn*max) to 1 (at full*max).
func severity(actual, max, warn, full float64) float64 {
	if max <= 0 || full <= warn {
		return 0
	}
	lo := warn * max
	hi := full * max
	if actual <= lo {
		return 0
	}
	if actual >= hi {
		return 1
	}
	return (actual - lo) / (hi - lo)
}

func charCount(text string) int {
	// Count visible characters; ignore newlines that only join wrapped lines.
	return utf8.RuneCountInString(strings.ReplaceAll(text, "\n", " ")) -
		strings.Count(text, "\n")
}

func wordCount(text string) int {
	return len(strings.Fields(text))
}
