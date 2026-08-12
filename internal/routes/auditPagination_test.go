package routes

import (
	"strings"
	"testing"
)

// TestAuditBooksPaginationEnvelopeSplitBrain proves task T0.1: the backend
// emits next_cursor nested under pagination (BuildCursorPaginatedResponse ->
// PaginatedResponse.Pagination.NextCursor = json:"next_cursor"), while the
// frontend reads it from the top level (lastPage.next_cursor). The two sides
// disagree, so infinite scroll can never fetch page 2.
func TestAuditBooksPaginationEnvelopeSplitBrain(t *testing.T) {
	be := readRepoFile(t, "internal/dtos/response/common.go")
	fe := readRepoFile(t, "web/src/hooks/useBooksQuery.ts")

	// Backend nests it under Pagination:
	if !strings.Contains(be, `NextCursor   string `+"`json:\"next_cursor,omitempty\"`") {
		t.Fatal("setup broken: backend next_cursor field not found")
	}
	if !strings.Contains(be, "Pagination *PaginationMeta") {
		t.Fatal("setup broken: PaginatedResponse.Pagination field not found")
	}

	// Frontend reads it top-level:
	if !strings.Contains(fe, "lastPage.next_cursor") {
		t.Fatalf("unexpected: frontend no longer reads top-level next_cursor; contract may be unified")
	}
}

// TestAuditBooksServiceNestsCursorInPagination confirms the service path that
// feeds the frontend builds the nested envelope.
func TestAuditBooksServiceNestsCursorInPagination(t *testing.T) {
	src := readRepoFile(t, "internal/services/bookService.go")
	if !strings.Contains(src, "response.BuildCursorPaginatedResponse(") {
		t.Fatal("setup broken: service no longer builds cursor pagination")
	}
	// The response DTO nests NextCursor under PaginationMeta, so the service's
	// return value has next_cursor at pagination.next_cursor — not top-level.
	common := readRepoFile(t, "internal/dtos/response/common.go")
	if !strings.Contains(common, `NextCursor`) {
		t.Fatal("setup broken")
	}
}
