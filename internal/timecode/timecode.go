// Package timecode parses and formats the HH:MM:SS:FF (hours:minutes:seconds:frames)
// timecodes used by the plain subtitle format, and converts them to durations
// using a configurable frame rate.
package timecode

import (
	"fmt"
	"time"
)

// Timecode is a frame-accurate position expressed as HH:MM:SS:FF.
type Timecode struct {
	Hours   int
	Minutes int
	Seconds int
	Frames  int
}

// Parse reads a "HH:MM:SS:FF" string.
func Parse(s string) (Timecode, error) {
	var tc Timecode
	n, err := fmt.Sscanf(s, "%d:%d:%d:%d", &tc.Hours, &tc.Minutes, &tc.Seconds, &tc.Frames)
	if err != nil || n != 4 {
		return Timecode{}, fmt.Errorf("invalid timecode %q: expected HH:MM:SS:FF", s)
	}
	return tc, nil
}

// String renders the timecode back to HH:MM:SS:FF.
func (t Timecode) String() string {
	return fmt.Sprintf("%02d:%02d:%02d:%02d", t.Hours, t.Minutes, t.Seconds, t.Frames)
}

// ToSeconds returns the absolute position in seconds, given the frame rate.
func (t Timecode) ToSeconds(fps float64) float64 {
	if fps <= 0 {
		fps = 25
	}
	return float64(t.Hours)*3600 +
		float64(t.Minutes)*60 +
		float64(t.Seconds) +
		float64(t.Frames)/fps
}

// Duration returns end-start as a time.Duration, given the frame rate.
func Duration(start, end Timecode, fps float64) time.Duration {
	d := end.ToSeconds(fps) - start.ToSeconds(fps)
	if d < 0 {
		d = 0
	}
	return time.Duration(d * float64(time.Second))
}

// FromSeconds builds a frame-accurate Timecode from an absolute number of
// seconds, given the frame rate. Used when importing SRT/VTT (which are
// millisecond-based) back into the frame-based model.
func FromSeconds(sec, fps float64) Timecode {
	if fps <= 0 {
		fps = 25
	}
	if sec < 0 {
		sec = 0
	}
	whole := int(sec)
	frames := int((sec-float64(whole))*fps + 0.5)
	if frames >= int(fps) {
		frames = int(fps) - 1
	}
	return Timecode{
		Hours:   whole / 3600,
		Minutes: (whole % 3600) / 60,
		Seconds: whole % 60,
		Frames:  frames,
	}
}

// SRT formats the timecode as SubRip "HH:MM:SS,mmm", converting frames to
// milliseconds via the frame rate.
func (t Timecode) SRT(fps float64) string {
	total := t.ToSeconds(fps)
	h := int(total) / 3600
	m := (int(total) % 3600) / 60
	s := int(total) % 60
	ms := int((total - float64(int(total))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

// VTT formats the timecode as WebVTT "HH:MM:SS.mmm".
func (t Timecode) VTT(fps float64) string {
	total := t.ToSeconds(fps)
	h := int(total) / 3600
	m := (int(total) % 3600) / 60
	s := int(total) % 60
	ms := int((total - float64(int(total))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}
