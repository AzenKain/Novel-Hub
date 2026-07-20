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

export interface DuplicateFileResult {
  hash: string;
  duplicate_count: number;
  file_ids: string;
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
