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
  metadata_json?: string;
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
