package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/ylallemant/subscribe/internal/timecode"
)

// Format identifies a subtitle file format for parsing.
type Format string

const (
	Plain Format = "plain" // the time-boxed HH:MM:SS:FF format (.txt/.plain)
	SRT   Format = "srt"
	VTT   Format = "vtt"
)

// FormatFromExt guesses the format from a file extension (".srt", ".vtt", …).
func FormatFromExt(ext string) Format {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "srt":
		return SRT
	case "vtt":
		return VTT
	default:
		return Plain
	}
}

// ParseFormat parses content in the given format into blocks. SRT/VTT are
// millisecond-based, so fps is used to convert their timestamps back to the
// frame-based model.
func ParseFormat(r io.Reader, f Format, fps float64) ([]Block, error) {
	switch f {
	case SRT, VTT:
		return parseTimestamped(r, fps)
	default:
		return Parse(r)
	}
}

// "HH:MM:SS,mmm --> HH:MM:SS.mmm" — accepts comma (SRT) or dot (VTT), and cue
// settings trailing the end timestamp are ignored.
var cueLine = regexp.MustCompile(
	`(\d{1,2}):(\d{2}):(\d{2})[.,](\d{1,3})\s*-->\s*(\d{1,2}):(\d{2}):(\d{2})[.,](\d{1,3})`)

// parseTimestamped handles both SRT and WebVTT: any block is introduced by a
// line containing "-->"; the lines that follow (until blank) are its text. Index
// lines, "WEBVTT" headers, "NOTE" blocks and cue identifiers are ignored.
func parseTimestamped(r io.Reader, fps float64) ([]Block, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		blocks  []Block
		current *Block
	)
	flush := func() {
		if current != nil {
			for len(current.Lines) > 0 && strings.TrimSpace(current.Lines[len(current.Lines)-1]) == "" {
				current.Lines = current.Lines[:len(current.Lines)-1]
			}
			blocks = append(blocks, *current)
			current = nil
		}
	}

	for sc.Scan() {
		line := sc.Text()
		if m := cueLine.FindStringSubmatch(line); m != nil {
			flush()
			start := timecode.FromSeconds(toSeconds(m[1], m[2], m[3], m[4]), fps)
			end := timecode.FromSeconds(toSeconds(m[5], m[6], m[7], m[8]), fps)
			current = &Block{Index: len(blocks), Start: start, End: end}
			continue
		}
		if current == nil {
			continue // preamble / headers before the first cue
		}
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		current.Lines = append(current.Lines, line)
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no subtitle cues found")
	}
	return blocks, nil
}

func toSeconds(h, m, s, frac string) float64 {
	hh, _ := strconv.Atoi(h)
	mm, _ := strconv.Atoi(m)
	ss, _ := strconv.Atoi(s)
	// Normalise the fractional part to milliseconds (pad/truncate to 3 digits).
	for len(frac) < 3 {
		frac += "0"
	}
	ms, _ := strconv.Atoi(frac[:3])
	return float64(hh)*3600 + float64(mm)*60 + float64(ss) + float64(ms)/1000
}
