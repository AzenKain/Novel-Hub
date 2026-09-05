package services

import (
	"context"
	"strings"
	"unicode"

	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/response"
	"novelhub/pkg/apperrors"
)

// PotentialDuplicateBooks scans the whole library for same-work pairs by title similarity (bigram Dice + Jaro-Winkler, max) plus author match.
func (s *bookService) PotentialDuplicateBooks(ctx context.Context) ([]*response.PotentialDuplicateResponse, error) {
	rows, err := s.bookRepo.ListAllTitleAuthor(ctx)
	if err != nil {
		return nil, err
	}

	normTitles := make([]string, len(rows))
	for i, r := range rows {
		normTitles[i] = normalizeTitle(r.Title)
	}

	out := make([]*response.PotentialDuplicateResponse, 0)
	for i := 0; i < len(rows); i++ {
		titleA := normTitles[i]
		if titleA == "" {
			continue
		}
		for j := i + 1; j < len(rows); j++ {
			titleB := normTitles[j]
			if titleB == "" {
				continue
			}
			if !titleMatches(titleA, titleB) {
				continue
			}
			if !sameAuthor(rows[i].AuthorName, rows[j].AuthorName) {
				continue
			}
			out = append(out, &response.PotentialDuplicateResponse{
				SourceID:    rows[i].ID,
				SourceTitle: rows[i].Title,
				TargetID:    rows[j].ID,
				TargetTitle: rows[j].Title,
				AuthorName:  rows[i].AuthorName,
				Similarity:  similarityMax(titleA, titleB),
			})
		}
	}
	return out, nil
}

// MergeBooks folds every row that references sourceID into targetID, deletes the source book, then moves its surviving files into the target's directory.
func (s *bookService) MergeBooks(ctx context.Context, sourceID string, targetID string) error {
	if sourceID == targetID {
		return apperrors.New(apperrors.ErrBadRequest, "cannot merge a book into itself")
	}
	if _, err := s.GetBook(ctx, sourceID); err != nil {
		return err
	}
	if _, err := s.GetBook(ctx, targetID); err != nil {
		return err
	}

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	txRepo := s.bookRepo.WithTx(tx)
	if err := txRepo.MergeBookData(ctx, sourceID, targetID); err != nil {
		return err
	}
	if err := txRepo.DeleteBook(ctx, sourceID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	txRepo.FlushCache(ctx)

	if err := s.fileRepo.MoveBookFiles(ctx, sourceID, targetID); err != nil {
		log.Warn().Err(err).Str("source_id", sourceID).Str("target_id", targetID).Msg("failed to move book files during merge")
	}
	if err := s.fileRepo.RemoveBookDir(ctx, sourceID); err != nil {
		log.Warn().Err(err).Str("source_id", sourceID).Msg("failed to remove merged book directory")
	}
	return nil
}

func normalizeTitle(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func titleMatches(normA, normB string) bool {
	shorter, longer := len(normA), len(normB)
	if shorter > longer {
		shorter, longer = longer, shorter
	}
	if shorter*2 < longer {
		return false
	}
	return similarityMax(normA, normB) >= 0.85
}

func similarityMax(a, b string) float64 {
	d := bigramDice(a, b)
	if j := jaroWinkler(a, b); j > d {
		return j
	}
	return d
}

func sameAuthor(a, b string) bool {
	if a == "" && b == "" {
		return true
	}
	normA := normalizeTitle(a)
	normB := normalizeTitle(b)
	if normA == normB {
		return true
	}
	return jaroWinkler(normA, normB) >= 0.9
}

func bigramDice(a, b string) float64 {
	if a == b {
		return 1
	}
	if len(a) < 2 || len(b) < 2 {
		return 0
	}
	remain := make(map[string]int, len(a)-1)
	for i := 0; i < len(a)-1; i++ {
		remain[a[i:i+2]]++
	}
	matches := 0
	for i := 0; i < len(b)-1; i++ {
		bg := b[i : i+2]
		if remain[bg] > 0 {
			remain[bg]--
			matches++
		}
	}
	return 2 * float64(matches) / float64(len(a)-1+len(b)-1)
}

func jaroWinkler(a, b string) float64 {
	if a == b {
		return 1
	}
	if a == "" || b == "" {
		return 0
	}
	if len(a) < len(b) {
		a, b = b, a
	}
	matchDist := len(a)/2 - 1
	if matchDist < 0 {
		matchDist = 0
	}
	aMatches := make([]bool, len(a))
	bMatches := make([]bool, len(b))
	matches := 0
	for i := 0; i < len(a); i++ {
		lo := i - matchDist
		if lo < 0 {
			lo = 0
		}
		hi := i + matchDist + 1
		if hi > len(b) {
			hi = len(b)
		}
		for j := lo; j < hi; j++ {
			if bMatches[j] || a[i] != b[j] {
				continue
			}
			aMatches[i] = true
			bMatches[j] = true
			matches++
			break
		}
	}
	if matches == 0 {
		return 0
	}
	transpositions := 0
	k := 0
	for i := 0; i < len(a); i++ {
		if !aMatches[i] {
			continue
		}
		for !bMatches[k] {
			k++
		}
		if a[i] != b[k] {
			transpositions++
		}
		k++
	}
	transpositions /= 2
	jaro := (float64(matches)/float64(len(a)) +
		float64(matches)/float64(len(b)) +
		float64(matches-transpositions)/float64(matches)) / 3
	prefix := 0
	maxPrefix := 4
	if len(a) < maxPrefix {
		maxPrefix = len(a)
	}
	if len(b) < maxPrefix {
		maxPrefix = len(b)
	}
	for i := 0; i < maxPrefix && a[i] == b[i]; i++ {
		prefix++
	}
	return jaro + float64(prefix)*0.1*(1-jaro)
}
