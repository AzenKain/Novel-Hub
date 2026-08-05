package services

import (
	"context"
	"testing"

	_ "modernc.org/sqlite"

	"novelhub/internal/dtos/request"
	"novelhub/internal/repositories"
	"novelhub/pkg/cache"
	"novelhub/pkg/database"
)

// The facet DTO is now passed straight through from the controller instead of being copied into
// a service-local struct. The clamp has to stay in the service regardless: pkg/validator only
// runs on the HTTP path, so a caller that constructs the DTO directly would otherwise ask the
// database for an unbounded page.
func TestListFacetClampsLimitEvenWhenTheValidatorNeverRan(t *testing.T) {
	_, db := newActivityService(t)
	c := cache.NewRamCache()
	svc := &metadataService{
		bookRepo: repositories.NewBookDBRepository(db, c),
		libraries: &libraryService{
			libraryRepo: repositories.NewLibraryRepository(db, c),
			permissions: allowAllPermissions{},
			settings:    NewSettingsService(repositories.NewSettingsRepository(db, c), database.NewTxManager(db)),
		},
	}

	for _, q := range []*request.MetadataFacetDto{
		{Limit: 5000},
		{Limit: 0},
		{Limit: -1},
		nil,
	} {
		res, err := svc.ListAuthors(context.Background(), q, nil)
		if err != nil {
			t.Fatalf("%#v: %v", q, err)
		}
		if res.Pagination == nil {
			t.Fatalf("%#v returned no pagination metadata", q)
		}
		if res.Pagination.PageSize > 100 {
			t.Errorf("%#v produced page size %d, above the 100 cap", q, res.Pagination.PageSize)
		}
		if res.Pagination.PageSize <= 0 {
			t.Errorf("%#v produced page size %d, which would return nothing", q, res.Pagination.PageSize)
		}
	}
}
