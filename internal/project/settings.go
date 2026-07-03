package project

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// Settings is the per-project configuration persisted as settings.yaml. It is a
// flat set of scalars, so it is read/written with a tiny hand-rolled YAML
// encoder rather than a third-party dependency.
type Settings struct {
	DisplayName   string  `json:"displayName"`
	Format        string  `json:"format"`        // txt | srt | vtt
	FPS           float64 `json:"fps"`           // frames per second
	CPSMax        float64 `json:"cpsMax"`        // max chars/second
	WPSMax        float64 `json:"wpsMax"`        // max words/second
	Metric        string  `json:"metric"`        // cps | wps
	ReferenceLang string  `json:"referenceLang"` // ISO 639-3 of the original
}

// DefaultSettings returns settings for a new project.
func DefaultSettings(displayName string) Settings {
	return Settings{
		DisplayName: displayName,
		Format:      "srt",
		FPS:         25,
		CPSMax:      17,
		WPSMax:      3,
		Metric:      "cps",
	}
}

// Ext returns the file extension (no dot) for the project's format.
func (s Settings) Ext() string {
	switch s.Format {
	case "srt":
		return "srt"
	case "vtt":
		return "vtt"
	default:
		return "txt"
	}
}

// MarshalYAML renders the settings as a flat YAML document.
func (s Settings) MarshalYAML() string {
	var b strings.Builder
	b.WriteString("# subscribe project settings\n")
	writeYAMLString(&b, "displayName", s.DisplayName)
	writeYAMLString(&b, "format", s.Format)
	writeYAMLNumber(&b, "fps", s.FPS)
	writeYAMLNumber(&b, "cpsMax", s.CPSMax)
	writeYAMLNumber(&b, "wpsMax", s.WPSMax)
	writeYAMLString(&b, "metric", s.Metric)
	writeYAMLString(&b, "referenceLang", s.ReferenceLang)
	return b.String()
}

// ParseSettings reads the flat YAML produced by MarshalYAML. Unknown keys are
// ignored; missing numeric fields keep their zero value.
func ParseSettings(content string) Settings {
	s := Settings{}
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = unquoteYAML(strings.TrimSpace(val))
		switch key {
		case "displayName":
			s.DisplayName = val
		case "format":
			s.Format = val
		case "fps":
			s.FPS = atof(val)
		case "cpsMax":
			s.CPSMax = atof(val)
		case "wpsMax":
			s.WPSMax = atof(val)
		case "metric":
			s.Metric = val
		case "referenceLang":
			s.ReferenceLang = val
		}
	}
	return s
}

func writeYAMLString(b *strings.Builder, key, val string) {
	fmt.Fprintf(b, "%s: %q\n", key, val)
}
func writeYAMLNumber(b *strings.Builder, key string, val float64) {
	fmt.Fprintf(b, "%s: %s\n", key, strconv.FormatFloat(val, 'g', -1, 64))
}
func unquoteYAML(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
	}
	return s
}
func atof(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
