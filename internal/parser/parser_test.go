package parser

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	in := "00:00:20:06 - 00:00:22:08\nPendant cinq siècles,\n\n" +
		"00:00:22:08 – 00:00:24:09\navant la fin\nde la Seconde Guerre Mondiale\n"

	blocks, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[0].Start.String() != "00:00:20:06" || blocks[0].End.String() != "00:00:22:08" {
		t.Errorf("block 0 timecodes: %s - %s", blocks[0].Start, blocks[0].End)
	}
	if blocks[0].Text() != "Pendant cinq siècles," {
		t.Errorf("block 0 text: %q", blocks[0].Text())
	}
	// en-dash separator + multi-line text
	if len(blocks[1].Lines) != 2 {
		t.Errorf("block 1 lines: %v", blocks[1].Lines)
	}
}

func TestParseNoBlocks(t *testing.T) {
	if _, err := Parse(strings.NewReader("just some prose\n")); err == nil {
		t.Fatal("expected error for input with no timecode lines")
	}
}
