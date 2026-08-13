package routes

import (
	"strings"
	"testing"
)

// TestAuditBooksPaginationEnvelopeUnified confirms the backend and frontend
// agree on the top-level next_cursor.
func TestAuditBooksPaginationEnvelopeUnified(t *testing.T) {
	be := readRepoFile(t, "internal/dtos/response/common.go")
	fe := readRepoFile(t, "web/src/hooks/useBooksQuery.ts")

	// Backend has NextCursor in CursorPaginatedResponse:
	if !strings.Contains(be, `NextCursor *string `+"`json:\"next_cursor,omitempty\"`") {
		t.Fatal("setup broken: backend next_cursor field not found in CursorPaginatedResponse")
	}

	// Frontend reads it top-level:
	if !strings.Contains(fe, "lastPage.next_cursor") {
		t.Fatalf("unexpected: frontend no longer reads top-level next_cursor")
	}
}

// TestAuditBooksServiceUsesCursorPaginatedResponse confirms the service path that
// feeds the frontend builds the top-level next_cursor response.
func TestAuditBooksServiceUsesCursorPaginatedResponse(t *testing.T) {
	src := readRepoFile(t, "internal/services/bookService.go")
	if !strings.Contains(src, "response.CursorPaginatedResponse{") {
		t.Fatal("setup broken: service no longer builds CursorPaginatedResponse")
	}
	common := readRepoFile(t, "internal/dtos/response/common.go")
	if !strings.Contains(common, `NextCursor`) {
		t.Fatal("setup broken")
	}
}
