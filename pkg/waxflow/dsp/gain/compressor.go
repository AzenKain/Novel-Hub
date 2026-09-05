package gain

import (
	"fmt"
	"math"
	"time"

	"novelhub/pkg/waxflow/waxerr"
)

// CompressorVersion is the compressor node's revision for cache keys: bump on any change to a preset's curve, the detector, or the smoothing constants, all of which change decoded samples.
const CompressorVersion = "compressor-1"

// Preset names a dynamics curve.
type Preset string

const (
	PresetOff   Preset = ""
	PresetVoice Preset = "voice"
)

// Presets lists the dynamics curves this build implements, in table order.
func Presets() []Preset { return []Preset{PresetVoice} }

const settleTimeConstants = 40

const ln10Over20 = math.Ln10 / 20

type curve struct {
	thresholdDB float64
	ratio       float64
	kneeDB      float64
	attack      time.Duration
	release     time.Duration
	makeupDB    float64
}

var curves = map[Preset]curve{
	PresetVoice: {
		thresholdDB: -20,
		ratio:       2.5,
		kneeDB:      6,
		attack:      10 * time.Millisecond,
		release:     250 * time.Millisecond,
		makeupDB:    6,
	},
}

// Compressor is a feed-forward dynamics processor: a peak detector drives a soft-knee static curve, whose gain is smoothed by an attack and a release one-pole and applied to every channel alike, so the stereo image cannot shift.
type Compressor struct {
	channels int

	thresholdDB float64
	invRatio    float64
	kneeDB      float64
	halfKneeDB  float64
	kneeStart   float64
	makeup      float64
	release     time.Duration

	aAtk float64
	aRel float64

	g float64
}

// NewCompressor returns a compressor for one stream at the named preset, which must not be PresetOff (the chain inserts no node for that).
func NewCompressor(rate, channels int, p Preset) (*Compressor, error) {
	if rate <= 0 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("gain: compressor rate %d must be positive", rate))
	}
	if channels < 1 {
		return nil, waxerr.New(waxerr.CodeInvalidRequest,
			fmt.Sprintf("gain: compressor channel count %d must be positive", channels))
	}
	c, ok := curves[p]
	if !ok {
		return nil, waxerr.New(waxerr.CodeUnsupportedFormat,
			fmt.Sprintf("gain: unknown dynamics preset %q", p))
	}
	k := &Compressor{
		channels:    channels,
		thresholdDB: c.thresholdDB,
		invRatio:    1 / c.ratio,
		kneeDB:      c.kneeDB,
		halfKneeDB:  c.kneeDB / 2,
		kneeStart:   FromDB(c.thresholdDB - c.kneeDB/2),
		makeup:      FromDB(c.makeupDB),
		release:     c.release,
		aAtk:        1 - math.Exp(-1/(c.attack.Seconds()*float64(rate))),
		aRel:        1 - math.Exp(-1/(c.release.Seconds()*float64(rate))),
	}
	k.Reset()
	return k, nil
}

// Reset clears all state for a new stream segment.
func (c *Compressor) Reset() { c.g = 0 }

// Horizon reports the pre-roll a restarted run needs before its output rejoins a continuous run's bit-exactly (the dsp.Settler capability): 10 s for the voice preset, from its 250 ms release.
func (c *Compressor) Horizon() time.Duration {
	return time.Duration(settleTimeConstants * float64(c.release))
}

// Process consumes frames from src and produces compressed frames into dst, per channel, in lockstep.
func (c *Compressor) Process(dst, src [][]float32) (produced, consumed int) {
	if len(dst) != c.channels || len(src) != c.channels {
		panic("gain: compressor channel count mismatch")
	}
	n := min(checkFrames("compressor destination", dst), checkFrames("compressor source", src))
	for i := 0; i < n; i++ {
		var peak float32
		for ch := range src {
			v := src[ch][i]
			if v < 0 {
				v = -v
			}
			if v > peak {
				peak = v
			}
		}
		target := c.targetDB(float64(peak))
		if target < c.g {
			c.g += (target - c.g) * c.aAtk
		} else {
			c.g += (target - c.g) * c.aRel
		}
		g := float32(math.Exp(c.g*ln10Over20) * c.makeup)
		for ch := range dst {
			dst[ch][i] = src[ch][i] * g
		}
	}
	return n, n
}

// Drain flushes the delayed tail after the final Process call.
func (c *Compressor) Drain(dst [][]float32) (produced int) {
	if len(dst) != c.channels {
		panic("gain: compressor channel count mismatch")
	}
	return 0
}

const maxPeak = 2

func (c *Compressor) targetDB(peak float64) float64 {
	if peak <= c.kneeStart {
		return 0
	}
	if peak > maxPeak {
		peak = maxPeak
	}
	over := 20*math.Log10(peak) - c.thresholdDB
	if over >= c.halfKneeDB {
		return over * (c.invRatio - 1)
	}
	x := over + c.halfKneeDB
	return (c.invRatio - 1) * x * x / (2 * c.kneeDB)
}
