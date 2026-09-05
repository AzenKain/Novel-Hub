package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"
)

func seedReviews(t *testing.T, db *sql.DB, users, books, reviews int) {
	t.Helper()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO libraries (id,name) VALUES ('lib-1','L')`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < users; i++ {
		if _, err := tx.Exec(`INSERT INTO users (id,email,password_hash,full_name) VALUES (?,?,?,?)`,
			fmt.Sprintf("u-%06d", i), fmt.Sprintf("u%d@e.com", i), "x", fmt.Sprintf("User %d", i)); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < books; i++ {
		if _, err := tx.Exec(`INSERT INTO books (id,library_id,title,status) VALUES (?,?,?,?)`,
			fmt.Sprintf("b-%06d", i), "lib-1", fmt.Sprintf("Book %d", i), "active"); err != nil {
			t.Fatal(err)
		}
	}
	base := time.Now().Add(-time.Duration(reviews) * time.Minute)
	for i := 0; i < reviews; i++ {
		if _, err := tx.Exec(`INSERT INTO book_reviews (user_id,book_id,rating,review,created_at,updated_at) VALUES (?,?,?,?,?,?)`,
			fmt.Sprintf("u-%06d", i/books), fmt.Sprintf("b-%06d", i%books), i%5+1, "r",
			base.Add(time.Duration(i)*time.Minute), base.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

// Without idx_book_reviews_updated the admin review list is a full SCAN plus a temp b-tree per page: 85ms at 32k rows against 248µs with it.
func TestListAllReviewsPlanStaysIndexed(t *testing.T) {
	db := probeDB(t)
	seedReviews(t, db, 500, 2000, 2000)

	rows, err := db.Query("EXPLAIN QUERY PLAN " + strings.TrimSuffix(listAllReviewsSQL, "\n"))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	plan := ""
	for rows.Next() {
		var a, b, c int
		var detail string
		if err := rows.Scan(&a, &b, &c, &detail); err != nil {
			t.Fatal(err)
		}
		plan += detail + "\n"
	}
	if strings.Contains(plan, "TEMP B-TREE") {
		t.Errorf("ListAllReviews sorts in a temp b-tree; idx_book_reviews_updated(updated_at DESC) is gone or unusable:\n%s", plan)
	}
	if !strings.Contains(plan, "idx_book_reviews_updated") {
		t.Errorf("ListAllReviews no longer reads idx_book_reviews_updated:\n%s", plan)
	}
	if !strings.Contains(plan, "CO-ROUTINE") {
		t.Errorf("the deferred subquery was flattened away, so users/books are joined for every row before the LIMIT:\n%s", plan)
	}
}

const listAllReviewsSQL = `SELECT br.user_id, br.book_id, br.rating, br.review, br.created_at, br.updated_at,
       u.full_name as user_name, u.email as user_email,
       b.title as book_title
FROM (SELECT user_id, book_id, rating, review, created_at, updated_at
      FROM book_reviews ORDER BY updated_at DESC LIMIT 50 OFFSET 0) br
JOIN users u ON u.id = br.user_id
JOIN books b ON b.id = br.book_id
ORDER BY br.updated_at DESC`
