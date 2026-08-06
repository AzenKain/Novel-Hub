export interface ReadingHistory {
  user_id: string;
  book_id: string;
  file_id?: string;
  chapter_id: string;
  progress_percent?: number;
  updated_at: string;
  book_title: string;
  book_cover_url?: string;
  chapter_title: string;
  chapter_index: number;
}

export interface ReadingProgress {
  user_id: string;
  book_id: string;
  file_id?: string;
  chapter_id: string;
  chapter_title: string;
  chapter_index: number;
  progress_percent?: number;
  location_cfi?: string;
  location_type?: string;
  opened_count: number;
  qualified_read_count: number;
  last_opened_at?: string;
  last_counted_at?: string;
  updated_at?: string;
}

export interface BookReadStats {
  book_id: string;
  total_open_count: number;
  qualified_read_count: number;
  last_opened_at?: string;
  last_counted_at?: string;
  updated_at?: string;
}

export interface BookDownloadStats {
  book_id: string;
  total_download_count: number;
  last_downloaded_at?: string;
  updated_at?: string;
}

export interface BookReview {
  user_id: string;
  book_id: string;
  rating: number;
  review?: string;
  created_at?: string;
  updated_at?: string;
}

export interface BookRatingSummary {
  book_id: string;
  rating_count: number;
  average_rating: number;
}

export interface BookSocialStats {
  book_id: string;
  bookmark_count: number;
  rating_count: number;
  average_rating: number;
  share_count: number;
  collection_count?: number;
  updated_at?: string;
}

export interface BookEngagementStats {
  book_id: string;
  social_stats: BookSocialStats;
  download_stats: BookDownloadStats;
  read_stats: BookReadStats;
}

export interface Bookmark {
  user_id: string;
  book_id: string;
  created_at: string;
}

export interface BookUserState {
  book_id: string;
  bookmarked: boolean;
  my_review?: BookReview;
  rating_summary: BookRatingSummary;
  social_stats?: BookSocialStats;
  download_stats: BookDownloadStats;
  read_stats: BookReadStats;
  collections: string[];
}

export interface RecordReadingActivityPayload {
  book_id: string;
  file_id?: string;
  chapter_id: string;
  chapter_title?: string;
  chapter_index?: number;
  progress_percent?: number;
  location_cfi?: string;
  location_type?: string;
  event_type?: string;
}

export interface ReadingActivityResult {
  progress: {
    user_id: string;
    book_id: string;
    file_id?: string;
    chapter_id: string;
    chapter_title: string;
    chapter_index: number;
    progress_percent?: number;
    location_cfi?: string;
    location_type?: string;
    opened_count: number;
    qualified_read_count: number;
  };
  stats: BookReadStats;
  counted: boolean;
  cooldown_seconds: number;
}
