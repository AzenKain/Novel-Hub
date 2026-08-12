export interface AudiobookChapter {
  id: string;
  book_id: string;
  file_id?: string | null;
  chapter_index: number;
  title: string;
  start_sec: number;
  end_sec?: number | null;
  created_at: string;
  updated_at: string;
}

export interface UpsertAudiobookChapterInput {
  file_id?: string | null;
  chapter_index: number;
  title: string;
  start_sec: number;
  end_sec?: number | null;
}

export interface LookupAudiobookChaptersInput {
  asin: string;
}

export interface MergeAudioInput {
  title: string;
  file_ids: string[];
}