package constants

import "testing"

// The constants replaced string literals scattered across the repositories. A typo here
// silently stops invalidating that key — nothing fails, the cache just goes stale forever.
// These are the exact strings that were in the code before the swap.
func TestCacheKeyValuesUnchanged(t *testing.T) {
	for name, got := range map[string]string{
		"CacheKeyBookSearchPattern":        CacheKeyBookSearchPattern,
		"CacheKeyLibraryStats":             CacheKeyLibraryStats,
		"CacheKeyBookAllPattern":           CacheKeyBookAllPattern,
		"CacheKeyBookIDsPattern":           CacheKeyBookIDsPattern,
		"CacheKeyBookFilePattern":          CacheKeyBookFilePattern,
		"CacheKeyBookFileNamespacePattern": CacheKeyBookFileNamespacePattern,
		"CacheKeyBookFileAllPattern":       CacheKeyBookFileAllPattern,
		"CacheKeyBookFileDupesPattern":     CacheKeyBookFileDupesPattern,
		"CacheKeyBookTrackerMapPattern":    CacheKeyBookTrackerMapPattern,
		"CacheKeyChapterPattern":           CacheKeyChapterPattern,
		"CacheKeyChapterByBookPattern":     CacheKeyChapterByBookPattern,
		"CacheKeyCollectionOwnedPattern":   CacheKeyCollectionOwnedPattern,
		"CacheKeyReadListOwnedPattern":     CacheKeyReadListOwnedPattern,
		"CacheKeyReadListCountsPattern":    CacheKeyReadListCountsPattern,
		"CacheKeyFTSPattern":               CacheKeyFTSPattern,
		"CacheKeyFTSBookSearchPattern":     CacheKeyFTSBookSearchPattern,
		"CacheKeyFTSSearchPattern":         CacheKeyFTSSearchPattern,
		"CacheKeyMetadataPattern":          CacheKeyMetadataPattern,
		"CacheKeyMetadataCountPattern":     CacheKeyMetadataCountPattern,
		"CacheKeyAllReviewsPattern":        CacheKeyAllReviewsPattern,
		"CacheKeyRoleNamePattern":          CacheKeyRoleNamePattern,
		"CacheKeyUserAllPattern":           CacheKeyUserAllPattern,
		"CacheKeySettingsAll":              CacheKeySettingsAll,
		"CacheKeyUserSearch":               CacheKeyUserSearch,
		"CacheKeyUserCount":                CacheKeyUserCount,
	} {
		want := map[string]string{
			"CacheKeyBookSearchPattern": "book:search*", "CacheKeyLibraryStats": "feature:library_stats",
			"CacheKeyBookAllPattern": "book:*", "CacheKeyBookIDsPattern": "book_ids*",
			"CacheKeyBookFilePattern": "book_file*", "CacheKeyBookFileNamespacePattern": "book_file:*",
			"CacheKeyBookFileAllPattern": "book_file:all*", "CacheKeyBookFileDupesPattern": "book_file:duplicates*",
			"CacheKeyBookTrackerMapPattern": "book_tracker_mapping*", "CacheKeyChapterPattern": "chapter*",
			"CacheKeyChapterByBookPattern": "chapter:book:*", "CacheKeyCollectionOwnedPattern": "collection:owned:*",
			"CacheKeyReadListOwnedPattern":  "read_list:owned:*",
			"CacheKeyReadListCountsPattern": "read_list:counts:*",
			"CacheKeyFTSPattern":            "fts:*", "CacheKeyFTSBookSearchPattern": "fts:book-search*",
			"CacheKeyFTSSearchPattern": "fts:search*", "CacheKeyMetadataPattern": "metadata:*",
			"CacheKeyMetadataCountPattern": "metadata_count:*", "CacheKeyAllReviewsPattern": "feature:all_reviews*",
			"CacheKeyRoleNamePattern": "role:name:*", "CacheKeyUserAllPattern": "user:*",
			"CacheKeySettingsAll": "settings:all", "CacheKeyUserSearch": "user:search*",
			"CacheKeyUserCount": "user:count*",
		}[name]
		if got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}
