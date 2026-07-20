export const POLICY_MODES = [
  { value: "all", label: "All" },
  { value: "disabled", label: "Disabled" },
  { value: "selected_libraries", label: "Selected Libraries" },
];

export const GUEST_MODES = [
  { value: "all", label: "All Libraries" },
  { value: "selected_libraries", label: "Selected Libraries" },
  { value: "login_required", label: "Login Required" },
];

export const SIDEBAR_LABELS: Record<string, string> = {
  books: "All Books",
  hot_books: "Hot Books",
  downloaded_books: "Downloaded Books",
  top_rated_books: "Top Rated Books",
  bookmarked_books: "Bookmarked Books",
  read_books: "Read Books",
  unread_books: "Unread Books",
  subjects: "Subjects",
  series: "Series",
  authors: "Authors",
  publishers: "Publishers",
  languages: "Languages",
  file_formats: "File Formats",
  ratings: "Ratings",
  archived_books: "Archived Books",
  collections: "Collections",
};

export const BOOK_FILE_ACCEPT =
  ".epub,.mobi,.azw,.azw3,.amz,.pdf,.doc,.docx,.odt,.txt,.md,.markdown,.html,.htm,.rtf,.fb2,.fbz,.zip,.cbz,.cbr,.cbt,.cb7";

export const READER_PAGE_GAP = 40;
export const READER_CONTENT_MEASURE = 72;
export const MIN_DOUBLE_PAGE_WIDTH = 380;

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
