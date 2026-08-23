export const POLICY_MODES = [
  { value: "all", labelKey: "settings.policy_mode_all" },
  { value: "disabled", labelKey: "settings.policy_mode_disabled" },
  { value: "selected_libraries", labelKey: "settings.policy_mode_selected_libraries" },
];

export const GUEST_MODES = [
  { value: "all", labelKey: "settings.guest_mode_all" },
  { value: "selected_libraries", labelKey: "settings.guest_mode_selected_libraries" },
];

export const SIDEBAR_LABELS: Record<string, string> = {
  books: "library.books",
  hot_books: "library.hot_books",
  downloaded_books: "library.downloaded_books",
  top_rated_books: "library.top_rated_books",
  bookmarked_books: "library.bookmarked_books",
  read_books: "library.read_books",
  unread_books: "library.unread_books",
  subjects: "library.subjects",
  series: "library.series",
  authors: "library.authors",
  publishers: "library.publishers",
  languages: "library.languages",
  file_formats: "library.file_formats",
  ratings: "library.ratings",
  archived_books: "library.archived_books",
  collections: "library.collections",
};

export const BOOK_FILE_ACCEPT =
  ".epub,.mobi,.azw,.azw3,.amz,.pdf,.doc,.docx,.odt,.txt,.md,.markdown,.html,.htm,.rtf,.fb2,.fbz,.zip,.cbz,.cbr,.cbt,.cb7,.rar,.7z,.mp3,.m4a,.m4b,.flac,.ogg,.wav,.aac,.csv,.tsv,.tex,.latex,.ltx,.pptx,.ppt,.odp,.xlsx,.xls,.ods";

export const READER_PAGE_GAP = 40;
export const READER_CONTENT_MEASURE = 72;
export const MIN_DOUBLE_PAGE_WIDTH = 380;

/**
 * Calculates responsive side-tap zone ratio for turning pages:
 * - On large desktop screens (>= 1024px): 0 (disabled so mouse clicks do not accidentally turn pages)
 * - Scales gradually from 0 at >= 1024px up to 0.30 (30%) on mobile (<= 480px)
 */
export function getSideTapRatio(screenWidth: number): number {
  if (screenWidth >= 1024) {
    return 0;
  }
  if (screenWidth <= 480) {
    return 0.30;
  }
  const factor = (1024 - screenWidth) / (1024 - 480);
  return Number((factor * 0.30).toFixed(3));
}

export const settingsKeyToNavId: Record<string, string> = {
  books: "books",
  hot: "hot",
  hot_books: "hot",
  downloaded_books: "downloaded",
  top_rated_books: "top_rated",
  bookmarked_books: "bookmarks",
  read_books: "read",
  unread_books: "unread",
  subjects: "tags",
  series: "series",
  authors: "authors",
  publishers: "publishers",
  languages: "languages",
  file_formats: "formats",
  ratings: "ratings",
  archived_books: "archived",
  collections: "collections",
};
