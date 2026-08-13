package services

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"novelhub/internal/dtos/request"
	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/netx"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/worker"
	"novelhub/pkg/waxflow"
	"novelhub/pkg/waxflow/audio"
	"novelhub/pkg/waxflow/container"
	"novelhub/pkg/waxflow/format"
)

const (
	audnexusChaptersURL = "https://api.audnex.us/books/%s/chapters"
	mergeAudioJobType   = "merge_audio"
	allowedMergeFormats = "m4a,m4b,mp3,flac,ogg,wav,aac"
)

type AudiobookService interface {
	ListChapters(ctx context.Context, bookID string) ([]*response.AudiobookChapterResponse, error)
	UpsertChapter(ctx context.Context, bookID string, dto UpsertAudiobookChapterInput) (*response.AudiobookChapterResponse, error)
	DeleteChapter(ctx context.Context, bookID string, id string) error
	DeleteChaptersForBook(ctx context.Context, bookID string) error
	LookupChaptersFromAudnexus(ctx context.Context, bookID string, asin string) ([]*response.AudiobookChapterResponse, error)
	MergeAudio(ctx context.Context, bookID string, title string, segments []request.MergeAudioSegment) (string, error)
	ExecuteMergeAudioJob(ctx context.Context, payloadJSON string) error
}

type UpsertAudiobookChapterInput struct {
	ID           string
	FileID       *string
	ChapterIndex int64
	Title        string
	StartSec     float64
	EndSec       *float64
}

type mergeAudioPayload struct {
	BookID   string                      `json:"book_id"`
	Title    string                      `json:"title"`
	Segments []request.MergeAudioSegment `json:"segments"`
}

// mergeSegment is a resolved timeline member: the file path plus the edit the
// timeline applies to it (sample window + linear gain).
type mergeSegment struct {
	path     string
	name     string // basename without extension, for the output chapter title
	startSec float64
	endSec   float64
	gain     float64
}

type audiobookService struct {
	repo       repositories.AudiobookRepository
	bookRepo   repositories.BookDBRepository
	fileRepo   repositories.BookFileRepository
	jobQueue   *worker.Queue
	httpClient *http.Client
}

func NewAudiobookService(repo repositories.AudiobookRepository, bookRepo repositories.BookDBRepository, fileRepo repositories.BookFileRepository, jobQueue *worker.Queue) AudiobookService {
	return &audiobookService{
		repo:       repo,
		bookRepo:   bookRepo,
		fileRepo:   fileRepo,
		jobQueue:   jobQueue,
		httpClient: netx.NewSafeHTTPClient(10 * time.Second),
	}
}

func (s *audiobookService) ListChapters(ctx context.Context, bookID string) ([]*response.AudiobookChapterResponse, error) {
	entities, err := s.repo.ListChapters(ctx, bookID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.AudiobookChapterResponse, len(entities))
	for i, e := range entities {
		out[i] = e.ToResponse()
	}
	return out, nil
}

func (s *audiobookService) UpsertChapter(ctx context.Context, bookID string, dto UpsertAudiobookChapterInput) (*response.AudiobookChapterResponse, error) {
	id := dto.ID
	if id == "" {
		id = uuid.NewString()
	}
	entity, err := s.repo.UpsertChapter(ctx, id, bookID, dto.FileID, dto.ChapterIndex, dto.Title, dto.StartSec, dto.EndSec)
	if err != nil {
		return nil, err
	}
	return entity.ToResponse(), nil
}

func (s *audiobookService) DeleteChapter(ctx context.Context, bookID string, id string) error {
	return s.repo.DeleteChapter(ctx, id)
}

func (s *audiobookService) DeleteChaptersForBook(ctx context.Context, bookID string) error {
	return s.repo.DeleteChaptersForBook(ctx, bookID)
}

type audnexusChaptersResponse struct {
	Chapters []struct {
		Title string `json:"title"`
		Start int64  `json:"start"`
		End   int64  `json:"end"`
		Index int64  `json:"index"`
	} `json:"chapters"`
}

func (s *audiobookService) LookupChaptersFromAudnexus(ctx context.Context, bookID string, asin string) ([]*response.AudiobookChapterResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(audnexusChaptersURL, asin), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "NovelHub/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("Audnexus lookup failed: %s", string(bodyBytes)))
	}

	var data audnexusChaptersResponse
	if err := jsonx.Unmarshal(bodyBytes, &data); err != nil {
		return nil, err
	}

	var out []*response.AudiobookChapterResponse
	for _, ch := range data.Chapters {
		if ch.Title == "" {
			continue
		}
		endSec := float64(ch.End) / 1000
		entity, err := s.repo.UpsertChapter(ctx, uuid.NewString(), bookID, nil, ch.Index, ch.Title, float64(ch.Start)/1000, &endSec)
		if err != nil {
			return nil, err
		}
		out = append(out, entity.ToResponse())
	}
	if len(out) == 0 {
		return nil, apperrors.New(apperrors.ErrNotFound, "no chapters found for this ASIN")
	}
	return out, nil
}

func (s *audiobookService) MergeAudio(ctx context.Context, bookID string, title string, segments []request.MergeAudioSegment) (string, error) {
	if len(segments) < 2 {
		return "", apperrors.New(apperrors.ErrBadRequest, "merge requires at least 2 audio segments")
	}

	payload, err := jsonx.MarshalString(mergeAudioPayload{BookID: bookID, Title: title, Segments: segments})
	if err != nil {
		return "", apperrors.New(apperrors.ErrInternalError, "Failed to marshal merge payload")
	}
	jobID := uuid.Must(uuid.NewV7()).String()
	if s.jobQueue != nil {
		if err := s.jobQueue.Enqueue(ctx, worker.Job{ID: jobID, Type: mergeAudioJobType, Payload: payload}); err != nil {
			return "", apperrors.New(apperrors.ErrInternalError, "Failed to enqueue merge job")
		}
		return jobID, nil
	}
	return jobID, s.ExecuteMergeAudioJob(ctx, payload)
}

func (s *audiobookService) ExecuteMergeAudioJob(ctx context.Context, payloadJSON string) error {
	var payload mergeAudioPayload
	if err := jsonx.UnmarshalString(payloadJSON, &payload); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, "Invalid merge payload")
	}
	if len(payload.Segments) < 2 {
		return apperrors.New(apperrors.ErrBadRequest, "merge requires at least 2 audio segments")
	}
	segs, err := s.resolveMergeSegments(ctx, payload.Segments)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", "novelhub-merge-*.m4b")
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to create temp output")
	}
	tmpPath := tmp.Name()
	if err := mergeTracks(ctx, segs, tmp); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	src, err := os.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	saved, err := s.fileRepo.SaveBook(ctx, payload.BookID, payload.Title+".m4b", src)
	_ = src.Close()
	_ = os.Remove(tmpPath)
	if err != nil {
		return err
	}

	return s.bookRepo.UpsertBookFile(ctx, sqlc.UpsertBookFileParams{
		ID:        uuid.Must(uuid.NewV7()).String(),
		BookID:    payload.BookID,
		Path:      saved.Path,
		Format:    "m4b",
		SizeBytes: saved.SizeBytes,
		ModTime:   saved.ModTime,
		Hash:      sql.NullString{},
		State:     sql.NullString{},
	})
}

func (s *audiobookService) resolveMergeSegments(ctx context.Context, segs []request.MergeAudioSegment) ([]mergeSegment, error) {
	out := make([]mergeSegment, 0, len(segs))
	for _, seg := range segs {
		f, err := s.bookRepo.GetBookFileById(ctx, seg.FileID)
		if err != nil || f == nil {
			return nil, apperrors.New(apperrors.ErrNotFound, "source file not found")
		}
		if !slices.Contains(strings.Split(allowedMergeFormats, ","), strings.ToLower(f.Format)) {
			return nil, apperrors.New(apperrors.ErrBadRequest, "only m4a, m4b, mp3, flac, ogg, wav, aac files can be merged")
		}
		out = append(out, mergeSegment{
			path:     f.Path,
			name:     strings.TrimSuffix(filepath.Base(f.Path), filepath.Ext(f.Path)),
			startSec: seg.StartSec,
			endSec:   seg.EndSec,
			gain:     seg.Gain,
		})
	}
	return out, nil
}

// openMergeSegment opens one timeline member: the source file, sliced to
// [startSec, endSec) and scaled by a linear gain. A whole file at unity gain
// is handed over unwrapped; anything else is a waxflow.Slice (streaming,
// sample-exact, bounded memory) optionally behind a gainMedia. No temp files:
// the timeline feeds the transcode directly.
func openMergeSegment(ctx context.Context, e *waxflow.Engine, path string, startSec, endSec float64, gain float64) (waxflow.ConcatSource, error) {
	f, err := os.Open(path)
	if err != nil {
		return waxflow.ConcatSource{}, err
	}
	src, err := container.FileSource(f)
	if err != nil {
		_ = f.Close()
		return waxflow.ConcatSource{}, err
	}
	info, err := e.Probe(src, "", nil)
	_ = f.Close()
	if err != nil {
		return waxflow.ConcatSource{}, err
	}
	track := info.Default()
	// Stream formats with no container (ADTS .aac) declare no length; the
	// timeline needs it, so measure with a full decode pass.
	if track.Samples < 0 {
		f2, err := os.Open(path)
		if err != nil {
			return waxflow.ConcatSource{}, err
		}
		src2, err := container.FileSource(f2)
		if err != nil {
			_ = f2.Close()
			return waxflow.ConcatSource{}, err
		}
		res, err := e.Analyze(ctx, src2, "", waxflow.AnalyzeOptions{})
		_ = f2.Close()
		if err != nil {
			return waxflow.ConcatSource{}, err
		}
		track.Samples = res.Samples
		track.SamplesExact = true
	}

	start := int64(startSec * float64(track.Fmt.Rate))
	end := int64(endSec * float64(track.Fmt.Rate))
	// The FE measures duration in the browser, which can round a few samples
	// past the server's count; clamp both ends so a full-file span stays full
	// and a split never lands past the file.
	if track.Samples >= 0 {
		start = min(start, track.Samples)
		end = min(end, track.Samples)
	}
	if start >= end {
		return waxflow.ConcatSource{}, fmt.Errorf("segment [%.2fs, %.2fs) is empty or past the end of the file", startSec, endSec)
	}

	// Whole file at unity gain: hand the source over unwrapped.
	if start == 0 && end >= track.Samples && gain == 1.0 {
		open := func() (format.Media, error) {
			f2, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			src2, err := container.FileSource(f2)
			if err != nil {
				_ = f2.Close()
				return nil, err
			}
			return e.OpenStream(src2, "")
		}
		return waxflow.ConcatSource{Track: track, Open: open}, nil
	}

	spanned, err := waxflow.SpanTrack(track, start, end)
	if err != nil {
		return waxflow.ConcatSource{}, err
	}
	open := func() (format.Media, error) {
		f2, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		src2, err := container.FileSource(f2)
		if err != nil {
			_ = f2.Close()
			return nil, err
		}
		med, err := e.OpenStream(src2, "")
		if err != nil {
			_ = f2.Close()
			return nil, err
		}
		sliced, err := waxflow.Slice(med, start, end)
		if err != nil {
			_ = med.Close()
			_ = f2.Close()
			return nil, err
		}
		if gain == 1.0 {
			return sliced, nil
		}
		return &gainMedia{Media: sliced, gain: gain, fmt: spanned.Fmt}, nil
	}
	return waxflow.ConcatSource{Track: spanned, Open: open}, nil
}

// gainMedia scales a Media's samples by a fixed linear factor, clamping to
// the domain's range. It is the per-segment gain of the merge timeline
// (1.0 = unchanged), matching the browser editor's gain strip.
type gainMedia struct {
	format.Media
	gain float64
	fmt  audio.Format
}

func (g *gainMedia) ReadChunk(dst *audio.Buffer) error {
	if err := g.Media.ReadChunk(dst); err != nil {
		return err
	}
	if g.fmt.Type == audio.Float {
		for c := 0; c < g.fmt.Channels; c++ {
			ch := dst.ChanF(c)
			for i, v := range ch {
				s := v * float32(g.gain)
				if s > 1 {
					s = 1
				} else if s < -1 {
					s = -1
				}
				ch[i] = s
			}
		}
		return nil
	}
	hi := int64((int64(1) << (g.fmt.BitDepth - 1)) - 1)
	lo := -(hi + 1)
	for c := 0; c < g.fmt.Channels; c++ {
		ch := dst.ChanI(c)
		for i, v := range ch {
			s := int64(float64(v) * g.gain)
			if s > hi {
				s = hi
			} else if s < lo {
				s = lo
			}
			ch[i] = int32(s)
		}
	}
	return nil
}

func mergeTracks(ctx context.Context, segs []mergeSegment, out io.Writer) error {
	e := waxflow.New()
	members := make([]waxflow.ConcatSource, 0, len(segs))
	for _, seg := range segs {
		member, err := openMergeSegment(ctx, e, seg.path, seg.startSec, seg.endSec, seg.gain)
		if err != nil {
			return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("failed to open source %s: %v", seg.path, err))
		}
		members = append(members, member)
	}

	tracks := make([]container.Track, len(members))
	for i := range members {
		tracks[i] = members[i].Track
	}
	bounds, envelope, err := waxflow.ConcatBoundaries(tracks, waxflow.ConcatOptions{})
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("audio formats are not compatible: %v", err))
	}

	chapters := make([]container.Chapter, 0, len(tracks))
	for i := range tracks {
		startMs := int64(0)
		if envelope.Rate > 0 {
			startMs = bounds[i].OffsetSamples * 1000 / int64(envelope.Rate)
		}
		chapters = append(chapters, container.Chapter{Start: time.Duration(startMs) * time.Millisecond, Title: segs[i].name})
	}

	med, err := waxflow.Concat(members, waxflow.ConcatOptions{})
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("failed to concatenate audio: %v", err))
	}
	defer med.Close()

	if _, err := e.TranscodeMedia(ctx, med, out, waxflow.TranscodeOptions{Format: "aac", Chapters: chapters}); err != nil {
		return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("failed to encode M4B: %v", err))
	}
	return nil
}