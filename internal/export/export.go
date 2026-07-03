// Package export renders translated blocks to the supported subtitle formats.
package export

import (
	"fmt"
	"strings"

	"github.com/ylallemant/subscribe/internal/parser"
)

// Format identifies an output format.
type Format string

const (
	Plain Format = "plain"
	SRT   Format = "srt"
	VTT   Format = "vtt"
)

// Extension returns the file extension (without dot) for the format.
func (f Format) Extension() string {
	switch f {
	case SRT:
		return "srt"
	case VTT:
		return "vtt"
	default:
		return "txt"
	}
}

// ContentType returns the MIME type for the format.
func (f Format) ContentType() string {
	switch f {
	case VTT:
		return "text/vtt; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}

// Render produces the output. translations[i] is the translated text for
// blocks[i]; empty entries fall back to the reference text so the file stays
// well-formed even when a block is not yet translated.
func Render(f Format, blocks []parser.Block, translations []string, fps float64) (string, error) {
	text := func(i int) string {
		if i < len(translations) && strings.TrimSpace(translations[i]) != "" {
			return translations[i]
		}
		return blocks[i].Text()
	}

	switch f {
	case SRT:
		return renderSRT(blocks, text, fps), nil
	case VTT:
		return renderVTT(blocks, text, fps), nil
	case Plain:
		return renderPlain(blocks, text), nil
	default:
		return "", fmt.Errorf("unknown export format %q", f)
	}
}

func renderPlain(blocks []parser.Block, text func(int) string) string {
	var b strings.Builder
	for i, blk := range blocks {
		fmt.Fprintf(&b, "%s - %s\n%s\n", blk.Start, blk.End, text(i))
		if i < len(blocks)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func renderSRT(blocks []parser.Block, text func(int) string, fps float64) string {
	var b strings.Builder
	for i, blk := range blocks {
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n",
			i+1, blk.Start.SRT(fps), blk.End.SRT(fps), text(i))
	}
	return b.String()
}

func renderVTT(blocks []parser.Block, text func(int) string, fps float64) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n\n")
	for i, blk := range blocks {
		fmt.Fprintf(&b, "%s --> %s\n%s\n\n",
			blk.Start.VTT(fps), blk.End.VTT(fps), text(i))
	}
	return b.String()
}
