// Package parser reads and writes the time-boxed plain subtitle format:
//
//	00:00:30:09 - 00:00:31:24
//	ont déferlé depuis ses côtes
//	<blank line>
package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/ylallemant/subscribe/internal/timecode"
)

// Block is a single subtitle: a time box plus its text lines.
type Block struct {
	Index int               `json:"index"`
	Start timecode.Timecode `json:"-"`
	End   timecode.Timecode `json:"-"`
	Lines []string          `json:"lines"`
}

// Text joins the block's lines with newlines.
func (b Block) Text() string { return strings.Join(b.Lines, "\n") }

// timecodeLine matches "HH:MM:SS:FF - HH:MM:SS:FF" with hyphen, en-dash or
// em-dash separators (subtitle exporters vary).
var timecodeLine = regexp.MustCompile(
	`^\s*(\d{2}:\d{2}:\d{2}:\d{2})\s*[-\x{2013}\x{2014}]\s*(\d{2}:\d{2}:\d{2}:\d{2})\s*$`)

// Parse reads blocks from r. Blocks are separated by a timecode line; the text
// lines that follow (until the next timecode line or blank run) belong to it.
func Parse(r io.Reader) ([]Block, error) {
	var (
		blocks  []Block
		current *Block
		sc      = bufio.NewScanner(r)
	)
	// Allow long lines.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	flush := func() {
		if current != nil {
			// Trim trailing blank lines within the block.
			for len(current.Lines) > 0 && strings.TrimSpace(current.Lines[len(current.Lines)-1]) == "" {
				current.Lines = current.Lines[:len(current.Lines)-1]
			}
			blocks = append(blocks, *current)
			current = nil
		}
	}

	for sc.Scan() {
		line := sc.Text()
		if m := timecodeLine.FindStringSubmatch(line); m != nil {
			flush()
			start, err := timecode.Parse(m[1])
			if err != nil {
				return nil, err
			}
			end, err := timecode.Parse(m[2])
			if err != nil {
				return nil, err
			}
			current = &Block{Index: len(blocks), Start: start, End: end}
			continue
		}
		if current == nil {
			// Skip any preamble before the first timecode line.
			continue
		}
		if strings.TrimSpace(line) == "" && len(current.Lines) == 0 {
			continue // ignore blank right after the timecode line
		}
		current.Lines = append(current.Lines, line)
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no subtitle blocks found (expected lines like %q)", "00:00:30:09 - 00:00:31:24")
	}
	return blocks, nil
}
