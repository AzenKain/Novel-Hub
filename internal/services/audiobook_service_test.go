package services

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"novelhub/internal/models"
	"novelhub/pkg/waxflow"
	"novelhub/pkg/waxflow/container"
)

func writeWav(t *testing.T, path string, rate int, seconds float64, freq float64) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	n := int(float64(rate) * seconds)
	dataLen := n * 2
	// RIFF header + PCM fmt chunk + data chunk
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+dataLen))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(header[22:24], 1) // mono
	binary.LittleEndian.PutUint32(header[24:28], uint32(rate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(rate*2))
	binary.LittleEndian.PutUint16(header[32:34], 2)
	binary.LittleEndian.PutUint16(header[34:36], 16)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataLen))
	if _, err := f.Write(header); err != nil {
		t.Fatal(err)
	}

	samples := make([]byte, dataLen)
	for i := 0; i < n; i++ {
		v := int16(math.Sin(2*math.Pi*freq*float64(i)/float64(rate)) * 10000)
		binary.LittleEndian.PutUint16(samples[i*2:], uint16(v))
	}
	if _, err := f.Write(samples); err != nil {
		t.Fatal(err)
	}
}

func TestMergeTracks(t *testing.T) {
	dir := t.TempDir()
	wav1 := filepath.Join(dir, "part1.wav")
	wav2 := filepath.Join(dir, "part2.wav")
	writeWav(t, wav1, 44100, 1.0, 440) // 1.0s @ 44.1k
	writeWav(t, wav2, 22050, 0.5, 880) // 0.5s @ 22.05k — exercises auto-resample

	out := filepath.Join(dir, "merged.m4b")
	of, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}

	files := []*models.BookFileEntity{
		{Path: wav1, Format: "wav"},
		{Path: wav2, Format: "wav"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := mergeTracks(ctx, files, of); err != nil {
		t.Fatalf("mergeTracks: %v", err)
	}
	_ = of.Close()

	probe, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	src, err := container.FileSource(probe)
	if err != nil {
		t.Fatal(err)
	}
	e := waxflow.New()
	info, err := e.Probe(src, "", nil)
	if err != nil {
		t.Fatalf("probe merged output: %v", err)
	}
	defaultTrack := info.Default()
	if defaultTrack.Samples <= 0 {
		t.Fatalf("merged output has no samples")
	}
	gotSecs := float64(defaultTrack.Samples) / float64(defaultTrack.Fmt.Rate)
	if gotSecs < 1.45 || gotSecs > 1.55 {
		t.Fatalf("merged duration = %.2fs, want ~1.5s", gotSecs)
	}

	if len(info.Chapters) != 2 {
		t.Fatalf("got %d chapters, want 2", len(info.Chapters))
	}
	first := info.Chapters[0]
	if first.Start != 0 || first.Title != "part1" {
		t.Fatalf("chapter 0 = %+v, want start 0 title part1", first)
	}
	if second := info.Chapters[1]; second.Start < 900*time.Millisecond || second.Start > 1100*time.Millisecond {
		t.Fatalf("chapter 1 start = %v, want ~1s", second.Start)
	}
}

func TestMergeAudioRejectsFewerThanTwo(t *testing.T) {
	s := NewAudiobookService(nil, nil, nil, nil)
	if _, err := s.MergeAudio(context.Background(), "book-1", "Merged", []string{"file-1"}); err == nil {
		t.Fatal("expected error for a single file")
	}
}

// Real container fixtures vendored from waxflow's own test corpus — every format
// the merger claims to accept, plus one mixed-format case (the auto-resample path).
// ponytail: no OGG/Opus fixtures exist in the vendored corpus; those decode via
// the same container sniffing path, add fixtures if a real OGG regression appears.
// Raw ADTS .aac has no container duration, so openMergeMember measures it with a
// full decode pass before the timeline is planned.
func probeDuration(t *testing.T, path string) time.Duration {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	src, err := container.FileSource(f)
	if err != nil {
		t.Fatalf("sniff %s: %v", path, err)
	}
	e := waxflow.New()
	info, err := e.Probe(src, "", nil)
	if err != nil {
		t.Fatalf("probe %s: %v", path, err)
	}
	tr := info.Default()
	samples := tr.Samples
	if samples < 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		res, err := e.Analyze(ctx, src, "", waxflow.AnalyzeOptions{})
		if err != nil {
			t.Fatalf("measure %s: %v", path, err)
		}
		samples = res.Samples
	}
	return time.Duration(float64(samples) / float64(tr.Fmt.Rate) * float64(time.Second))
}

func TestMergeTracksAllFormats(t *testing.T) {
	base := "../../pkg/waxflow/container"
	cases := []struct {
		name string
		a, b string
	}{
		{"mp3", base + "/mpa/testdata/golden-44k-mono-128-seek.mp3", base + "/mpa/testdata/golden-44k-stereo-128.mp3"},
		{"flac", base + "/flacn/testdata/golden-sine-s16-l5.flac", base + "/flacn/testdata/golden-noise-s16-l0.flac"},
		{"wav", base + "/riff/testdata/golden-f32.wav", base + "/riff/testdata/golden-rf64.wav"},
		{"aiff", base + "/aiff/testdata/golden-s16.aiff", base + "/aiff/testdata/golden-s24.aiff"},
		{"m4a-alac", base + "/mp4/testdata/alac-stereo.m4a", base + "/mp4/testdata/alac-mono-tail.m4a"},
		{"aac-adts", base + "/adts/testdata/stereo.aac", base + "/adts/testdata/mono.aac"},
		{"mixed-mp3-flac", base + "/mpa/testdata/golden-44k-mono-128-seek.mp3", base + "/flacn/testdata/golden-sine-s16-l5.flac"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.Join(dir, "merged.m4b")
			of, err := os.Create(out)
			if err != nil {
				t.Fatal(err)
			}
			files := []*models.BookFileEntity{
				{Path: tc.a, Format: "x"},
				{Path: tc.b, Format: "x"},
			}
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			if err := mergeTracks(ctx, files, of); err != nil {
				t.Fatalf("mergeTracks: %v", err)
			}
			_ = of.Close()

			probe, err := os.Open(out)
			if err != nil {
				t.Fatal(err)
			}
			defer probe.Close()
			src, err := container.FileSource(probe)
			if err != nil {
				t.Fatal(err)
			}
			info, err := waxflow.New().Probe(src, "", nil)
			if err != nil {
				t.Fatalf("probe merged output: %v", err)
			}
			tr := info.Default()
			if tr.Samples <= 0 {
				t.Fatalf("merged output has no samples")
			}
			got := time.Duration(float64(tr.Samples) / float64(tr.Fmt.Rate) * float64(time.Second))
			want := probeDuration(t, tc.a) + probeDuration(t, tc.b)
			lo, hi := want*9/10, want*11/10
			if got < lo || got > hi {
				t.Fatalf("merged duration = %v, want ~%v", got, want)
			}
			if len(info.Chapters) != 2 {
				t.Fatalf("got %d chapters, want 2", len(info.Chapters))
			}
			if info.Chapters[0].Start != 0 {
				t.Fatalf("chapter 0 start = %v, want 0", info.Chapters[0].Start)
			}
		})
	}
}