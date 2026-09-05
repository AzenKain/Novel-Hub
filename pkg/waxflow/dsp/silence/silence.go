// Package silence maps the near-silent spans of a stream: the detector behind the analyze job's silence half, which a library manager reads to trim leading and trailing pauses or to propose track boundaries.
package silence

import (
	"fmt"
	"math"
	"time"

	"novelhub/pkg/waxflow/waxerr"
)

// Version identifies the detector algorithm revision (ADR-0004 style).
const Version = "silence-1"

// Span is one detected silence, in frames on the analyzed stream's own timeline (ADR-0006).
type Span struct {
	From int64
	To   int64
}

// Len returns the span's length in frames.
func (s Span) Len() int64 { return s.To - s.From }

// Detector finds the silence spans of one stream.
type Detector struct {
	channels int
	thresh   float32
	minLen   int64

	pos     int64
	inRun   bool
	runFrom int64

	spans        []Span
	dropped      int
	droppedFrame int64
	total        int64
	flushed      bool
}

// New returns a detector for one stream.
func New(rate, channels int, thresholdDB float64, minDuration time.Duration) (*Detector, error) {
	if rate <= 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("silence: detector rate %d must be positive", rate))
	}
	if channels <= 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("silence: detector channel count %d must be positive", channels))
	}
	if !(thresholdDB < 0 && thresholdDB > math.Inf(-1)) {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("silence: threshold %g dBFS must be negative and finite", thresholdDB))
	}
	if minDuration <= 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("silence: minimum duration %v must be positive", minDuration))
	}
	return &Detector{
		channels: channels,
		thresh:   float32(math.Pow(10, thresholdDB/20)),
		minLen:   minFrames(minDuration, rate),
	}, nil
}

func minFrames(minDuration time.Duration, rate int) int64 {
	whole := int64(minDuration / time.Second)
	rem := int64(minDuration % time.Second)
	return whole*int64(rate) + (rem*int64(rate)+int64(time.Second)-1)/int64(time.Second)
}

// Process consumes one chunk of planar float32 PCM: chans[c][i] is sample i of channel c.
func (d *Detector) Process(chans [][]float32) error {
	if d.flushed {
		return waxerr.New(waxerr.CodeInvalidRequest, "silence: Process after Flush")
	}
	if len(chans) != d.channels {
		return waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("silence: chunk has %d channels, detector expects %d", len(chans), d.channels))
	}
	n := len(chans[0])
	for _, ch := range chans[1:] {
		if len(ch) != n {
			return waxerr.New(waxerr.CodeInvalidRequest, "silence: channel slices differ in length")
		}
	}
	for i := 0; i < n; i++ {
		var peak float32
		for c := range chans {
			v := chans[c][i]
			if v < 0 {
				v = -v
			}
			if v > peak {
				peak = v
			}
		}
		if peak < d.thresh {
			if !d.inRun {
				d.inRun = true
				d.runFrom = d.pos + int64(i)
			}
		} else if d.inRun {
			d.close(d.pos + int64(i))
		}
	}
	d.pos += int64(n)
	return nil
}

func (d *Detector) close(to int64) {
	d.inRun = false
	if to-d.runFrom < d.minLen {
		d.dropped++
		d.droppedFrame += to - d.runFrom
		return
	}
	d.spans = append(d.spans, Span{From: d.runFrom, To: to})
	d.total += to - d.runFrom
}

// Flush ends the stream, closing a span that runs to the last sample (the trailing-silence case, which silencedetect reports the same way).
func (d *Detector) Flush() {
	if d.flushed {
		return
	}
	d.flushed = true
	if d.inRun {
		d.close(d.pos)
	}
}

// Spans returns the detected silences in stream order.
func (d *Detector) Spans() []Span { return d.spans }

// Dropped counts the runs discarded for falling short of the minimum duration.
func (d *Detector) Dropped() int { return d.dropped }

// DroppedSamples is the summed length of the dropped runs, in frames, and it is the diagnostic a caller cannot otherwise compute.
func (d *Detector) DroppedSamples() int64 { return d.droppedFrame }

// TotalSamples is the summed length of the kept spans, in frames.
func (d *Detector) TotalSamples() int64 { return d.total }

// Samples is the number of frames the detector has consumed.
func (d *Detector) Samples() int64 { return d.pos }
