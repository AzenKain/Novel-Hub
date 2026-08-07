import type { Book } from "./book";

export interface ReadList {
  id: string;
  user_id: string;
  name: string;
  description: string;
  book_count: number;
  created_at: string;
  updated_at: string;
}

export interface ReadListBook {
  position: number;
  book: Book;
}

export interface ReadListNext {
  position: number;
  book?: Book;
  has_next: boolean;
}

export interface CBLUnmatchedEntry {
  series: string;
  number: string;
  reason: string;
}

export interface ImportCBLResult {
  read_list: ReadList;
  total: number;
  matched: number;
  unmatched: CBLUnmatchedEntry[];
}
