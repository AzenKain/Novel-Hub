package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"novelhub/internal/dtos/request"
	"novelhub/internal/models"
	"novelhub/pkg/jsonx"
	"novelhub/pkg/netx"
)

var (
	aniListURL     = "https://graphql.anilist.co"
	openLibraryURL = "https://openlibrary.org/search.json"
	googleBooksURL = "https://www.googleapis.com/books/v1/volumes"
)

type OnlineEnrichResult struct {
	Title         string
	Creator       string
	Publisher     string
	Language      string
	Description   string
	CoverURL      string
	Subjects      []string
	GoogleBooksID string
	AnilistID     string
	OpenLibraryID string
}

func cleanEnrichQuery(title string) string {
	// Remove content in parenthesis, brackets, braces
	reParens := regexp.MustCompile(`\s*[\(\[\{].*?[\)\]\}]\s*`)
	cleaned := reParens.ReplaceAllString(title, "")

	// Remove volume/chapter numbers (case insensitive)
	reVol := regexp.MustCompile(`(?i)\s*(?:tập|vol(?:ume)?|quyển|chương|chuong)\b.*`)
	cleaned = reVol.ReplaceAllString(cleaned, "")

	// Remove punctuation/symbols, keeping alphanumeric characters and spaces
	rePunct := regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	cleaned = rePunct.ReplaceAllString(cleaned, "")

	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return strings.TrimSpace(title)
	}
	return cleaned
}

func searchAniList(ctx context.Context, client *http.Client, query string) (*OnlineEnrichResult, error) {
	const aniListGraphQL = `query ($search: String) {
  Page(page: 1, perPage: 1) {
    media(search: $search, type: MANGA) {
      id
      title {
        romaji
        english
        native
      }
      description
      coverImage {
        large
      }
      countryOfOrigin
      genres
      staff {
        edges {
          role
          node {
            name {
              full
            }
          }
        }
      }
    }
  }
}`

	variables := map[string]any{"search": query}
	payload := map[string]any{
		"query":     aniListGraphQL,
		"variables": variables,
	}

	jsonBytes, err := jsonx.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", aniListURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anilist status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed aniListResponse
	if err := jsonx.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, err
	}

	mediaList := parsed.Data.Page.Media
	if len(mediaList) == 0 {
		return nil, nil
	}

	media := mediaList[0]

	// Determine best title
	title := media.Title.English
	if title == "" {
		title = media.Title.Romaji
	}
	if title == "" {
		title = media.Title.Native
	}

	// Format description: clean HTML tags
	desc := media.Description
	reHTML := regexp.MustCompile(`<[^>]*>`)
	desc = reHTML.ReplaceAllString(desc, "")
	desc = strings.TrimSpace(desc)

	// Extract authors
	var authors []string
	for _, edge := range media.Staff.Edges {
		role := strings.ToLower(edge.Role)
		if strings.Contains(role, "story") || strings.Contains(role, "author") || strings.Contains(role, "writer") || strings.Contains(role, "original") {
			if edge.Node.Name.Full != "" {
				authors = append(authors, edge.Node.Name.Full)
			}
		}
	}
	creator := ""
	if len(authors) > 0 {
		creator = authors[0]
	}

	// Map countryOfOrigin to language if needed
	lang := ""
	switch strings.ToUpper(media.CountryOfOrigin) {
	case "JP":
		lang = "ja"
	case "KR":
		lang = "ko"
	case "CN":
		lang = "zh"
	}

	return &OnlineEnrichResult{
		Title:         title,
		Creator:       creator,
		Language:      lang,
		Description:   desc,
		CoverURL:      media.CoverImage.Large,
		Subjects:      media.Genres,
		AnilistID:     fmt.Sprintf("%d", media.ID),
	}, nil
}

func searchOpenLibrary(ctx context.Context, client *http.Client, query string) (*OnlineEnrichResult, error) {
	url := fmt.Sprintf("%s?q=%s&limit=1", openLibraryURL, strings.ReplaceAll(query, " ", "+"))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openlibrary status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed openLibraryResponse
	if err := jsonx.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, err
	}

	if len(parsed.Docs) == 0 {
		return nil, nil
	}

	doc := parsed.Docs[0]
	creator := ""
	if len(doc.AuthorName) > 0 {
		creator = doc.AuthorName[0]
	}
	publisher := ""
	if len(doc.Publisher) > 0 {
		publisher = doc.Publisher[0]
	}
	lang := ""
	if len(doc.Language) > 0 {
		lang = doc.Language[0]
	}
	coverURL := ""
	if doc.CoverI > 0 {
		coverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", doc.CoverI)
	}

	return &OnlineEnrichResult{
		Title:         doc.Title,
		Creator:       creator,
		Publisher:     publisher,
		Language:      lang,
		CoverURL:      coverURL,
		Subjects:      doc.Subject,
		OpenLibraryID: doc.Key,
	}, nil
}

func searchGoogleBooks(ctx context.Context, client *http.Client, query string) (*OnlineEnrichResult, error) {
	url := fmt.Sprintf("%s?q=%s&maxResults=1", googleBooksURL, strings.ReplaceAll(query, " ", "+"))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google books status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed googleBooksResponse
	if err := jsonx.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, err
	}

	if len(parsed.Items) == 0 {
		return nil, nil
	}

	item := parsed.Items[0]
	creator := ""
	if len(item.VolumeInfo.Authors) > 0 {
		creator = item.VolumeInfo.Authors[0]
	}
	coverURL := item.VolumeInfo.ImageLinks.Thumbnail
	if coverURL == "" {
		coverURL = item.VolumeInfo.ImageLinks.SmallThumbnail
	}
	coverURL = strings.Replace(coverURL, "http://", "https://", 1)

	return &OnlineEnrichResult{
		Title:         item.VolumeInfo.Title,
		Creator:       creator,
		Publisher:     item.VolumeInfo.Publisher,
		Language:      item.VolumeInfo.Language,
		Description:   item.VolumeInfo.Description,
		CoverURL:      coverURL,
		Subjects:      item.VolumeInfo.Categories,
		GoogleBooksID: item.ID,
	}, nil
}

func (s *bookService) AutoEnrichBook(ctx context.Context, bookID string) error {
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		return err
	}

	// If book already has external IDs, skip it
	if (book.GoogleBooksID != nil && *book.GoogleBooksID != "") ||
		(book.AnilistID != nil && *book.AnilistID != "") ||
		(book.OpenLibraryID != nil && *book.OpenLibraryID != "") {
		return nil
	}

	query := cleanEnrichQuery(book.Title)
	client := netx.NewSafeHTTPClient(15 * time.Second)

	var result *OnlineEnrichResult

	// Try AniList (GraphQL)
	if res, err := searchAniList(ctx, client, query); err == nil && res != nil {
		result = res
	}

	// Try OpenLibrary
	if result == nil {
		if res, err := searchOpenLibrary(ctx, client, query); err == nil && res != nil {
			result = res
		}
	}

	// Try Google Books (using original title)
	if result == nil {
		if res, err := searchGoogleBooks(ctx, client, book.Title); err == nil && res != nil {
			result = res
		}
	}

	if result == nil {
		return fmt.Errorf("no metadata found for book: %s", book.Title)
	}

	// Fill in missing details
	if (book.Description == nil || *book.Description == "") && result.Description != "" {
		book.Description = &result.Description
	}

	if result.GoogleBooksID != "" {
		book.GoogleBooksID = &result.GoogleBooksID
	}
	if result.AnilistID != "" {
		book.AnilistID = &result.AnilistID
	}
	if result.OpenLibraryID != "" {
		book.OpenLibraryID = &result.OpenLibraryID
	}

	tx, err := s.txManager.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txRepo := s.bookRepo.WithTx(tx)

	// Enrich Author if missing
	if book.AuthorID == nil || *book.AuthorID == "" {
		authorName := result.Creator
		if authorName != "" {
			authorID, err := ensureAuthor(ctx, txRepo, authorName)
			if err == nil && authorID != "" {
				book.AuthorID = &authorID
			}
		}
	}

	// Enrich Tags (Subjects)
	for _, tagName := range result.Subjects {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}
		tag, err := txRepo.GetTagByName(ctx, tagName)
		if err != nil {
			tag = &models.TagEntity{ID: uuid.Must(uuid.NewV7()).String(), Name: tagName}
			if err := txRepo.CreateTag(ctx, tag); err != nil {
				return err
			}
		}
		if tag != nil && tag.ID != "" {
			_ = txRepo.AddBookTag(ctx, book.ID, tag.ID)
		}
	}

	if err := txRepo.UpdateBook(ctx, book); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	txRepo.FlushCache(ctx)

	// Enrich cover if missing
	if (book.CoverURL == nil || *book.CoverURL == "") && result.CoverURL != "" {
		dto := request.UpdateCoverDto{
			CoverURL: result.CoverURL,
		}
		if _, err := s.UpdateCover(ctx, book.ID, dto); err != nil {
			log.Warn().Err(err).Str("book_id", book.ID).Msg("failed to enrich cover image")
		}
	}

	return nil
}

func (s *bookService) BatchEnrichBooks(ctx context.Context) error {
	var cursor *time.Time
	cursorID := ""
	limit := int64(100)

	for {
		ids, err := s.bookRepo.ListBookIDs(ctx, cursor, cursorID, limit)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}

		books, err := s.bookRepo.GetBooksByIDs(ctx, ids)
		if err != nil {
			return err
		}

		for _, book := range books {
			if book == nil {
				continue
			}

			if book.Status == "active" &&
				(book.GoogleBooksID == nil || *book.GoogleBooksID == "") &&
				(book.AnilistID == nil || *book.AnilistID == "") &&
				(book.OpenLibraryID == nil || *book.OpenLibraryID == "") {

				if err := s.AutoEnrichBook(ctx, book.ID); err != nil {
					log.Warn().Err(err).Str("book_id", book.ID).Str("title", book.Title).Msg("batch enrichment failed for book")
				}
				time.Sleep(1 * time.Second)
			}
		}

		if len(ids) < int(limit) {
			break
		}

		lastBook := books[len(books)-1]
		cursor = &lastBook.CreatedAt
		cursorID = lastBook.ID
	}

	return nil
}

type aniListResponse struct {
	Data struct {
		Page struct {
			Media []struct {
				ID    int `json:"id"`
				Title struct {
					Romaji  string `json:"romaji"`
					English string `json:"english"`
					Native  string `json:"native"`
				} `json:"title"`
				Description string   `json:"description"`
				CoverImage  struct {
					Large string `json:"large"`
				} `json:"coverImage"`
				CountryOfOrigin string   `json:"countryOfOrigin"`
				Genres          []string `json:"genres"`
				Staff           struct {
					Edges []struct {
						Role string `json:"role"`
						Node struct {
							Name struct {
								Full string `json:"full"`
							} `json:"name"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"staff"`
			} `json:"media"`
		} `json:"Page"`
	} `json:"data"`
}

type openLibraryResponse struct {
	Docs []struct {
		Key        string   `json:"key"`
		Title      string   `json:"title"`
		AuthorName []string `json:"author_name"`
		Publisher  []string `json:"publisher"`
		Language   []string `json:"language"`
		CoverI     int      `json:"cover_i"`
		Subject    []string `json:"subject"`
	} `json:"docs"`
}

type googleBooksResponse struct {
	Items []struct {
		ID         string `json:"id"`
		VolumeInfo struct {
			Title       string   `json:"title"`
			Authors     []string `json:"authors"`
			Publisher   string   `json:"publisher"`
			Description string   `json:"description"`
			Categories  []string `json:"categories"`
			ImageLinks  struct {
				Thumbnail      string `json:"thumbnail"`
				SmallThumbnail string `json:"smallThumbnail"`
			} `json:"imageLinks"`
			Language string `json:"language"`
		} `json:"volumeInfo"`
	} `json:"items"`
}
