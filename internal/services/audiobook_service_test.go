package services

import (
	"context"
	"database/sql"
	"encoding/binary"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/waxflow"
	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/format"
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
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+dataLen))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	binary.LittleEndian.PutUint16(header[22:24], 1)
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

func segForWav(path string, start, end, gain float64) mergeSegment {
	return mergeSegment{
		path:     path,
		name:     strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		startSec: start,
		endSec:   end,
		gain:     gain,
	}
}

func drainSamples(t *testing.T, med format.Media) []int32 {
	t.Helper()
	def := med.Info().Default()
	if def.Fmt.Channels != 1 || def.Fmt.Type != audio.Int {
		t.Fatalf("expected mono int media, got %s", def.Fmt.String())
	}
	buf := audio.Get(def.Fmt, audio.StandardChunk)
	defer audio.Put(buf)
	var out []int32
	for {
		err := med.ReadChunk(buf)
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, buf.ChanI(0)...)
	}
}

func maxAbs(s []int32) int32 {
	m := int32(0)
	for _, v := range s {
		if v < 0 {
			v = -v
		}
		if v > m {
			m = v
		}
	}
	return m
}

func TestMergeTracks(t *testing.T) {
	dir := t.TempDir()
	wav1 := filepath.Join(dir, "part1.wav")
	wav2 := filepath.Join(dir, "part2.wav")
	writeWav(t, wav1, 44100, 1.0, 440)
	writeWav(t, wav2, 22050, 0.5, 880)

	out := filepath.Join(dir, "merged.m4b")
	of, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}

	segs := []mergeSegment{
		segForWav(wav1, 0, 2, 1.0),
		segForWav(wav2, 0, 2, 1.0),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := mergeTracks(ctx, segs, of); err != nil {
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
	if _, err := s.MergeAudio(context.Background(), "book-1", "Merged", []request.MergeAudioSegment{{FileID: "file-1"}}); err == nil {
		t.Fatal("expected error for a single file")
	}
}

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
			segs := []mergeSegment{
				segForWav(tc.a, 0, 1e9, 1.0),
				segForWav(tc.b, 0, 1e9, 1.0),
			}
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			if err := mergeTracks(ctx, segs, of); err != nil {
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

// TestMergeTracksSplitAndGain exercises the slice + gain timeline path: two halves of one file, the second doubled, concatenated back into the original duration.
func TestMergeTracksSplitAndGain(t *testing.T) {
	dir := t.TempDir()
	wav := filepath.Join(dir, "tone.wav")
	writeWav(t, wav, 44100, 1.0, 440)

	e := waxflow.New()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	segs := []mergeSegment{
		segForWav(wav, 0, 0.5, 1.0),
		segForWav(wav, 0.5, 1.0, 2.0),
	}
	members := make([]waxflow.ConcatSource, len(segs))
	for i := range segs {
		m, err := openMergeSegment(ctx, e, segs[i].path, segs[i].startSec, segs[i].endSec, segs[i].gain)
		if err != nil {
			t.Fatalf("openMergeSegment %d: %v", i, err)
		}
		members[i] = m
	}
	med, err := waxflow.Concat(members, waxflow.ConcatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer med.Close()
	samples := drainSamples(t, med)

	if n := len(samples); n < 43000 || n > 45000 {
		t.Fatalf("merged length = %d samples, want ~44100", n)
	}
	if peak := maxAbs(samples[:22050]); peak < 9000 || peak > 11000 {
		t.Fatalf("first half peak = %d, want ~10000 (gain 1.0)", peak)
	}
	if peak := maxAbs(samples[22050:]); peak < 19000 || peak > 22000 {
		t.Fatalf("second half peak = %d, want ~20000 (gain 2.0)", peak)
	}
}

// TestGainMediaClampsToIntDomain proves the gain wrapper clamps to the int16 domain instead of wrapping: 4x a 10000-amplitude tone would overflow int16 (40000) if the clamp were missing.
func TestGainMediaClampsToIntDomain(t *testing.T) {
	dir := t.TempDir()
	wav := filepath.Join(dir, "loud.wav")
	writeWav(t, wav, 44100, 0.5, 440)

	e := waxflow.New()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	m, err := openMergeSegment(ctx, e, wav, 0, 0.5, 4.0)
	if err != nil {
		t.Fatal(err)
	}
	med, err := waxflow.Concat([]waxflow.ConcatSource{m}, waxflow.ConcatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer med.Close()
	if peak := maxAbs(drainSamples(t, med)); peak != 32767 && peak != 32768 {
		t.Fatalf("peak = %d, want an int16 domain extreme (32767/32768), not the unclamped 40000", peak)
	}
}
func seedAudnexusFixture(t *testing.T) (*servicesDBFixture, *audiobookService) {
	t.Helper()
	db := auditDB(t)
	if _, err := db.ExecContext(context.Background(),
		`INSERT INTO libraries (id, name) VALUES ('libad', 'L')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(),
		`INSERT INTO books (id, library_id, title, status, updated_at) VALUES ('bookad', 'libad', 'Audio Book', 'active', ?)`,
		time.Now().UTC().Format("2006-01-02 15:04:05")); err != nil {
		t.Fatal(err)
	}
	repo := repositories.NewAudiobookRepository(db, cache.NewRamCache())
	svc := &audiobookService{
		repo:            repo,
		httpClient:      &http.Client{Timeout: 5 * time.Second},
		audnexusBaseURL: "replaced-by-test",
	}
	return &servicesDBFixture{db: db}, svc
}

type servicesDBFixture struct{ db *sql.DB }

func TestLookupChaptersFromAudnexusRejectsNonAlphanumericASIN(t *testing.T) {
	_, svc := seedAudnexusFixture(t)
	for _, asin := range []string{"B002V0F37U?x", "B00/2893", "B002V0 37U"} {
		if _, err := svc.LookupChaptersFromAudnexus(context.Background(), "bookad", asin); err == nil {
			t.Fatalf("ASIN %q should be rejected", asin)
		}
	}
}

func TestLookupChaptersFromAudnexusMapsChapters(t *testing.T) {
	_, svc := seedAudnexusFixture(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/books/B002V0F37U/chapters" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"chapters":[
			{"title":"Opening","start":0,"end":90000,"index":0},
			{"title":"","start":90000,"end":120000,"index":1},
			{"title":"Chapter Two","start":120000,"end":240000,"index":2}
		]}`))
	}))
	defer srv.Close()
	svc.audnexusBaseURL = srv.URL

	chapters, err := svc.LookupChaptersFromAudnexus(context.Background(), "bookad", "B002V0F37U")
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters (empty title skipped), got %d", len(chapters))
	}
	if chapters[0].Title != "Opening" || chapters[0].StartSec != 0 || *chapters[0].EndSec != 90 {
		t.Fatalf("chapter 0 mismatch: %+v", chapters[0])
	}
	if chapters[1].StartSec != 120 || *chapters[1].EndSec != 240 {
		t.Fatalf("chapter 1 ms→s conversion mismatch: %+v", chapters[1])
	}
}

func TestLookupChaptersFromAudnexusPropagatesUpstreamError(t *testing.T) {
	_, svc := seedAudnexusFixture(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"asin not found"}`))
	}))
	defer srv.Close()
	svc.audnexusBaseURL = srv.URL

	if _, err := svc.LookupChaptersFromAudnexus(context.Background(), "bookad", "B002V0F37U"); err == nil {
		t.Fatal("expected error on upstream 404")
	}
}

func TestLookupChaptersFromAudnexusNoChaptersIsNotFound(t *testing.T) {
	_, svc := seedAudnexusFixture(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"chapters":[]}`))
	}))
	defer srv.Close()
	svc.audnexusBaseURL = srv.URL

	chapters, err := svc.LookupChaptersFromAudnexus(context.Background(), "bookad", "B002V0F37U")
	if err == nil || chapters != nil {
		t.Fatal("expected not-found error for empty chapter list")
	}
}
