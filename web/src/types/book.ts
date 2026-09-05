export interface MetadataJSON {
  title?: string;
  creator?: string;
  creators?: string[];
  description?: string;
  publisher?: string;
  publishers?: string[];
  language?: string;
  languages?: string[];
  date?: string;
  dates?: string[];
  subject?: string[];
  identifier?: { id: string; value: string }[];
  meta?: { name?: string; content?: string; Name?: string; Content?: string }[];
  series?: string;
  series_index?: string;
}

export interface BookFile {
  id: string;
  book_id: string;
  path: string;
  format: string;
  size_bytes: number;
  mod_time: string;
  hash?: string;
  state?: string;
  created_at: string;
  updated_at: string;
}

export interface Book {
  id: string;
  library_id?: string;
  title: string;
  author_id?: string;
  author_name?: string;
  description?: string;
  cover_url?: string;
  file_path?: string;
  status: string;
  age_rating?: string;
  content_warnings?: string[];
  metadata_json?: string;
  google_books_id?: string;
  anilist_id?: string;
  openlibrary_id?: string;
  files?: BookFile[];
  created_at: string;
  updated_at: string;
}

export interface Chapter {
  id: string;
  book_id: string;
  title: string;
  content_path?: string;
  chapter_index: number;
  created_at: string;
  updated_at: string;
}

export interface SearchBookParams {
  cursor?: string;
  limit?: number;
  search?: string;
  library_id?: string;
  nav?: string;
  collection?: string;
  chip?: string;
  facet?: string;
  facet_id?: string;
  t?: number;
  sort?: string;
}

export interface SearchDeepResult {
  book_id: string;
  chapter_id: string;
  title: string;
}

export interface DuplicateFileDetail {
  file_id: string;
  book_id: string;
  book_title: string;
  book_cover_url?: string;
  library_id: string;
  format: string;
  size_bytes: number;
  path: string;
  created_at: string;
}

export interface DuplicateGroupResult {
  hash: string;
  duplicate_count: number;
  files: DuplicateFileDetail[];
}

export interface PotentialDuplicateResult {
  source_id: string;
  source_title: string;
  target_id: string;
  target_title: string;
  author_name: string;
  similarity: number;
}

export interface MergeBooksPayload {
  source_id: string;
  target_id: string;
}

export interface ConvertBookPayload {
  file_id: string;
  target_format: string;
}

export interface ConvertBookResult {
  job_id: string;
}

export interface BulkConvertItem {
  book_id: string;
  file_id: string;
  target_format: string;
}

export interface BulkConvertPayload {
  items: BulkConvertItem[];
}

export interface BulkConvertResult {
  job_ids: string[];
}

export interface DuplicateFileResult {
  hash: string;
  duplicate_count?: number;
  file_ids?: string;
}

export interface OnlineMetadataResult {
  title: string;
  creator?: string;
  publisher?: string;
  language?: string;
  description?: string;
  subject?: string;
  cover_image?: string;
  series?: string;
  series_index?: string;
}

export interface MetadataCount {
  id: string;
  name: string;
  book_count: number;
  cover_url?: string;
}

export interface MetadataFacetParams {
  cursor?: string;
  limit?: number;
  search?: string;
  alpha?: string;
}

export interface BootstrapResponse {
  book: Book;
  chapters: Chapter[];
}

export interface BookSeriesEntry {
  series_id: string;
  series_name: string;
  series_index?: string;
}

export interface NextInSeries {
  series_id: string;
  series_name: string;
  book_id: string;
  title: string;
  cover_url?: string;
  series_index?: string;
}

export interface BookSeriesContext {
  series: BookSeriesEntry[];
  next?: NextInSeries;
}

export interface Highlight {
  id: string;
  user_id: string;
  book_id: string;
  chapter_id: string;
  text_content: string;
  start_index: number;
  end_index: number;
  color: string;
  note?: string;
  cfi_range?: string;
  created_at: string;
  updated_at: string;
}

export interface SearchSnippet {
  chapter_id: string;
  chapter_title: string;
  chapter_index: number;
  snippet: string;
  offset: number;
}

export interface UploadCommitParams {
  target: "library" | "book";
  library_id?: string;
  book_id?: string;
  filename: string;
  total_chunks: number;
}

export interface BulkDeleteBooksInput {
  book_ids: string[];
}

export interface BulkMoveBooksInput {
  book_ids: string[];
  target_library_id: string;
}

export interface BulkAssignCollectionsInput {
  book_ids: string[];
  collection_ids: string[];
}

export interface BulkAddTagsInput {
  book_ids: string[];
  tag_names: string[];
}

export interface BulkOperationResponse {
  success_count: number;
  failed_count: number;
  errors?: Record<string, string>;
}

export interface UpdateBookMetadataRequest {
  title: string;
  author?: string;
  description?: string;
  publisher?: string;
  language?: string;
  date?: string;
  subjects?: string[];
  series?: string;
  series_index?: string;
  age_rating?: string;
  rating?: number;
}

export interface BulkUpdateMetadataItem {
  book_id: string;
  title?: string;
  author?: string;
  series_index?: string;
  publisher?: string;
  language?: string;
  description?: string;
}

export interface BulkUpdateMetadataRequest {
  book_ids: string[];
  author?: string;
  series?: string;
  auto_index_series?: boolean;
  publisher?: string;
  language?: string;
  add_tags?: string[];
  remove_tags?: string[];
  items?: BulkUpdateMetadataItem[];
}

export interface BookCardProps {
  book: Book;
  onClick: (book: Book) => void;
  compact?: boolean;
  selected?: boolean;
  selectionIndex?: number;
  onSelectToggle?: (book: Book) => void;
}

export interface ImageBookmark {
  id: string;
  image_url: string;
  chapter_id?: string;
  chapter_title?: string;
  page_index?: number;
  note?: string;
  created_at: string;
}

export interface ActiveImageTarget {
  image_url: string;
  chapter_id?: string;
  chapter_title?: string;
  page_index?: number;
  x?: number;
  y?: number;
}

export type UploadProgressStats = {
  progress: number;
  uploadedBytes: number;
  totalBytes: number;
  speedBytesPerSec: number;
};
