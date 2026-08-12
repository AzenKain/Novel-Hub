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

	"novelhub/internal/dtos/response"
	"novelhub/internal/gen/sqlc"
	"novelhub/internal/models"
	"novelhub/internal/repositories"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/netx"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/worker"
	"novelhub/pkg/waxflow"
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
	MergeAudio(ctx context.Context, bookID string, title string, fileIDs []string) (string, error)
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
	BookID  string   `json:"book_id"`
	Title   string   `json:"title"`
	FileIDs []string `json:"file_ids"`
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

func (s *audiobookService) MergeAudio(ctx context.Context, bookID string, title string, fileIDs []string) (string, error) {
	if len(fileIDs) < 2 {
		return "", apperrors.New(apperrors.ErrBadRequest, "merge requires at least 2 audio files")
	}

	payload, err := jsonx.MarshalString(mergeAudioPayload{BookID: bookID, Title: title, FileIDs: fileIDs})
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
	if len(payload.FileIDs) < 2 {
		return apperrors.New(apperrors.ErrBadRequest, "merge requires at least 2 audio files")
	}
	files, err := s.resolveMergeFiles(ctx, payload.FileIDs)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", "novelhub-merge-*.m4b")
	if err != nil {
		return apperrors.New(apperrors.ErrInternalError, "Failed to create temp output")
	}
	tmpPath := tmp.Name()
	if err := mergeTracks(ctx, files, tmp); err != nil {
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

func (s *audiobookService) resolveMergeFiles(ctx context.Context, fileIDs []string) ([]*models.BookFileEntity, error) {
	files := make([]*models.BookFileEntity, 0, len(fileIDs))
	for _, id := range fileIDs {
		f, err := s.bookRepo.GetBookFileById(ctx, id)
		if err != nil || f == nil {
			return nil, apperrors.New(apperrors.ErrNotFound, "source file not found")
		}
		if !slices.Contains(strings.Split(allowedMergeFormats, ","), strings.ToLower(f.Format)) {
			return nil, apperrors.New(apperrors.ErrBadRequest, "only m4a, m4b, mp3, flac, ogg, wav, aac files can be merged")
		}
		files = append(files, f)
	}
	return files, nil
}

func openMergeMember(ctx context.Context, e *waxflow.Engine, path string) (waxflow.ConcatSource, error) {
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

func mergeTracks(ctx context.Context, files []*models.BookFileEntity, out io.Writer) error {
	e := waxflow.New()
	members := make([]waxflow.ConcatSource, 0, len(files))
	for _, f := range files {
		member, err := openMergeMember(ctx, e, f.Path)
		if err != nil {
			return apperrors.New(apperrors.ErrInternalError, fmt.Sprintf("failed to open source %s: %v", f.Path, err))
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
		stem := strings.TrimSuffix(filepath.Base(files[i].Path), filepath.Ext(files[i].Path))
		chapters = append(chapters, container.Chapter{Start: time.Duration(startMs) * time.Millisecond, Title: stem})
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