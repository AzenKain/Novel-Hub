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
  seriesIndex?: string;
}

export interface BookFile {
  id: string;
  bookId: string;
  path: string;
  format: string;
  sizeBytes: number;
  modTime: string;
  hash?: string;
  state?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Book {
  id: string;
  libraryId?: string;
  title: string;
  authorId?: string;
  authorName?: string;
  description?: string;
  coverUrl?: string;
  filePath?: string;
  status: string;
  metadataJson?: string;
  files?: BookFile[];
  createdAt: string;
  updatedAt: string;
}

export interface Chapter {
  id: string;
  bookId: string;
  title: string;
  contentPath?: string;
  chapterIndex: number;
  createdAt: string;
  updatedAt: string;
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
}

export interface SearchDeepResult {
  book_id: string;
  chapter_id: string;
  title: string;
}

export interface DuplicateFileDetail {
  fileId: string;
  bookId: string;
  bookTitle: string;
  bookCoverUrl?: string;
  libraryId: string;
  format: string;
  sizeBytes: number;
  path: string;
  createdAt: string;
}

export interface DuplicateGroupResult {
  hash: string;
  duplicateCount: number;
  files: DuplicateFileDetail[];
}

export interface DuplicateFileResult {
  hash: string;
  duplicateCount?: number;
  duplicate_count?: number;
  fileIds?: string;
  file_ids?: string;
}

export interface OnlineMetadataResult {
  title: string;
  creator?: string;
  publisher?: string;
  language?: string;
  description?: string;
  subject?: string;
  coverImage?: string;
  series?: string;
  seriesIndex?: string;
}

export interface MetadataCount {
  id: string;
  name: string;
  bookCount: number;
  coverUrl?: string;
}

export interface BootstrapResponse {
  book: Book;
  chapters: Chapter[];
}

export interface Highlight {
  id: string;
  userId: number;
  bookId: string;
  chapterId: string;
  textContent: string;
  startIndex: number;
  endIndex: number;
  color: string;
  note?: string;
  createdAt: string;
  updatedAt: string;
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
