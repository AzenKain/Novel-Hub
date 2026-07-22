package calibre

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"

	_ "modernc.org/sqlite"
)

type CalibreBookRecord struct {
	ID          int64             `json:"id"`
	Title       string            `json:"title"`
	Sort        string            `json:"sort,omitempty"`
	AuthorSort  string            `json:"authorSort,omitempty"`
	Path        string            `json:"path"`
	PubDate     *string           `json:"pubDate,omitempty"`
	Timestamp   *string           `json:"timestamp,omitempty"`
	HasCover    bool              `json:"hasCover"`
	UUID        string            `json:"uuid,omitempty"`
	ISBN        string            `json:"isbn,omitempty"`
	LCCN        string            `json:"lccn,omitempty"`
	Rating      *float64          `json:"rating,omitempty"` // 0.0 - 5.0
	Description string            `json:"description,omitempty"`
	Authors     []string          `json:"authors,omitempty"`
	Series      string            `json:"series,omitempty"`
	SeriesIndex *string           `json:"seriesIndex,omitempty"`
	Publishers  []string          `json:"publishers,omitempty"`
	Languages   []string          `json:"languages,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Identifiers map[string]string `json:"identifiers,omitempty"`
}

func ReadMetadataDB(ctx context.Context, dbPath string) ([]CalibreBookRecord, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("metadata.db file not found: %s", dbPath)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open calibre sqlite db: %w", err)
	}
	defer db.Close()

	commentsMap := fetchComments(ctx, db)
	authorsMap := fetchAuthors(ctx, db)
	seriesMap := fetchSeries(ctx, db)
	publishersMap := fetchPublishers(ctx, db)
	languagesMap := fetchLanguages(ctx, db)
	tagsMap := fetchTags(ctx, db)
	ratingsMap := fetchRatings(ctx, db)
	identifiersMap := fetchIdentifiers(ctx, db)

	rows, err := db.QueryContext(ctx, "SELECT id, title, sort, author_sort, path, pubdate, timestamp, has_cover, uuid, isbn, lccn, series_index FROM books")
	if err != nil {
		rows, err = db.QueryContext(ctx, "SELECT id, title, path FROM books")
		if err != nil {
			return nil, fmt.Errorf("query calibre books: %w", err)
		}
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	hasFullCols := len(cols) >= 12

	var records []CalibreBookRecord
	for rows.Next() {
		var r CalibreBookRecord
		var (
			sort, authorSort, pubdate, timestamp, uuidStr, isbn, lccn sql.NullString
			hasCover                                                  sql.NullInt64
			seriesIdx                                                 sql.NullFloat64
		)

		if hasFullCols {
			if err := rows.Scan(&r.ID, &r.Title, &sort, &authorSort, &r.Path, &pubdate, &timestamp, &hasCover, &uuidStr, &isbn, &lccn, &seriesIdx); err != nil {
				return nil, fmt.Errorf("scan calibre book full: %w", err)
			}
		} else {
			if err := rows.Scan(&r.ID, &r.Title, &r.Path); err != nil {
				return nil, fmt.Errorf("scan calibre book simple: %w", err)
			}
		}

		if sort.Valid {
			r.Sort = sort.String
		}
		if authorSort.Valid {
			r.AuthorSort = authorSort.String
		}
		if pubdate.Valid {
			r.PubDate = &pubdate.String
		}
		if timestamp.Valid {
			r.Timestamp = &timestamp.String
		}
		if hasCover.Valid {
			r.HasCover = hasCover.Int64 > 0
		}
		if uuidStr.Valid {
			r.UUID = uuidStr.String
		}
		if isbn.Valid {
			r.ISBN = isbn.String
		}
		if lccn.Valid {
			r.LCCN = lccn.String
		}
		if seriesIdx.Valid {
			idxStr := strconv.FormatFloat(seriesIdx.Float64, 'f', -1, 64)
			r.SeriesIndex = &idxStr
		}

		if desc, ok := commentsMap[r.ID]; ok {
			r.Description = desc
		}
		if authors, ok := authorsMap[r.ID]; ok {
			r.Authors = authors
		}
		if sName, ok := seriesMap[r.ID]; ok {
			r.Series = sName
		}
		if pubs, ok := publishersMap[r.ID]; ok {
			r.Publishers = pubs
		}
		if langs, ok := languagesMap[r.ID]; ok {
			r.Languages = langs
		}
		if tags, ok := tagsMap[r.ID]; ok {
			r.Tags = tags
		}
		if rating, ok := ratingsMap[r.ID]; ok {
			r.Rating = &rating
		}
		if ident, ok := identifiersMap[r.ID]; ok {
			r.Identifiers = ident
			if r.ISBN == "" && ident["isbn"] != "" {
				r.ISBN = ident["isbn"]
			}
		}

		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate calibre books: %w", err)
	}

	return records, nil
}

func fetchComments(ctx context.Context, db *sql.DB) map[int64]string {
	res := make(map[int64]string)
	rows, err := db.QueryContext(ctx, "SELECT book, text FROM comments")
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var bookID int64
		var text sql.NullString
		if err := rows.Scan(&bookID, &text); err == nil && text.Valid {
			res[bookID] = text.String
		}
	}
	_ = rows.Err()
	return res
}

func fetchAuthors(ctx context.Context, db *sql.DB) map[int64][]string {
	res := make(map[int64][]string)
	query := "SELECT b.book, a.name FROM books_authors_link b JOIN authors a ON b.author = a.id ORDER BY b.id ASC"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var bookID int64
		var name string
		if err := rows.Scan(&bookID, &name); err == nil && name != "" {
			res[bookID] = append(res[bookID], name)
		}
	}
	_ = rows.Err()
	return res
}

func fetchSeries(ctx context.Context, db *sql.DB) map[int64]string {
	res := make(map[int64]string)
	query := "SELECT b.book, s.name FROM books_series_link b JOIN series s ON b.series = s.id"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var bookID int64
		var name string
		if err := rows.Scan(&bookID, &name); err == nil && name != "" {
			res[bookID] = name
		}
	}
	_ = rows.Err()
	return res
}

func fetchPublishers(ctx context.Context, db *sql.DB) map[int64][]string {
	res := make(map[int64][]string)
	query := "SELECT b.book, p.name FROM books_publishers_link b JOIN publishers p ON b.publisher = p.id"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var bookID int64
		var name string
		if err := rows.Scan(&bookID, &name); err == nil && name != "" {
			res[bookID] = append(res[bookID], name)
		}
	}
	_ = rows.Err()
	return res
}

func fetchLanguages(ctx context.Context, db *sql.DB) map[int64][]string {
	res := make(map[int64][]string)
	query := "SELECT b.book, l.lang_code FROM books_languages_link b JOIN languages l ON b.lang_code = l.id"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var bookID int64
		var lang string
		if err := rows.Scan(&bookID, &lang); err == nil && lang != "" {
			res[bookID] = append(res[bookID], lang)
		}
	}
	_ = rows.Err()
	return res
}

func fetchTags(ctx context.Context, db *sql.DB) map[int64][]string {
	res := make(map[int64][]string)
	query := "SELECT b.book, t.name FROM books_tags_link b JOIN tags t ON b.tag = t.id"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var bookID int64
		var tag string
		if err := rows.Scan(&bookID, &tag); err == nil && tag != "" {
			res[bookID] = append(res[bookID], tag)
		}
	}
	_ = rows.Err()
	return res
}

func fetchRatings(ctx context.Context, db *sql.DB) map[int64]float64 {
	res := make(map[int64]float64)
	query := "SELECT b.book, r.rating FROM books_ratings_link b JOIN ratings r ON b.rating = r.id"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var bookID int64
		var rawRating float64
		if err := rows.Scan(&bookID, &rawRating); err == nil && rawRating > 0 {
			val := rawRating / 2.0
			if val > 5.0 {
				val = 5.0
			}
			res[bookID] = val
		}
	}
	_ = rows.Err()
	return res
}

func fetchIdentifiers(ctx context.Context, db *sql.DB) map[int64]map[string]string {
	res := make(map[int64]map[string]string)
	query := "SELECT book, type, val FROM identifiers"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var bookID int64
		var idType, idVal string
		if err := rows.Scan(&bookID, &idType, &idVal); err == nil && idType != "" && idVal != "" {
			if res[bookID] == nil {
				res[bookID] = make(map[string]string)
			}
			res[bookID][idType] = idVal
		}
	}
	_ = rows.Err()
	return res
}
