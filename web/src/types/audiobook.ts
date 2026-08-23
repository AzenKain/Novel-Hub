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

export interface MergeAudioSegment {
  file_id: string;
  start_sec: number;
  end_sec: number;
  gain: number;
}

export interface MergeAudioInput {
  title: string;
  segments: MergeAudioSegment[];
}

export interface AudioChapter {
  title: string;
  start_sec: number;
  end_sec?: number | null;
}

export interface AudioBookmark {
  id: string;
  time_sec: number;
  chapter_title?: string;
  note?: string;
  created_at: string;
}