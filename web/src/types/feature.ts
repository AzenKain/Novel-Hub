export interface ReadingHistory {
  userId: string;
  bookId: string;
  fileId?: string;
  chapterId: string;
  progressPercent?: number;
  locationCfi?: string;
  locationType?: string;
  updatedAt: string;
  bookTitle: string;
  bookCoverUrl?: string;
  chapterTitle: string;
  chapterIndex: number;
}

export interface BookReadStats {
  bookId: string;
  totalOpenCount: number;
  qualifiedReadCount: number;
  lastOpenedAt?: string;
  lastCountedAt?: string;
  updatedAt?: string;
}

export interface BookDownloadStats {
  bookId: string;
  totalDownloadCount: number;
  lastDownloadedAt?: string;
  updatedAt?: string;
}

export interface BookReview {
  userId: string;
  bookId: string;
  rating: number;
  review?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface BookRatingSummary {
  bookId: string;
  ratingCount: number;
  averageRating: number;
}

export interface BookSocialStats {
  bookId: string;
  bookmarkCount: number;
  ratingCount: number;
  averageRating: number;
  shareCount: number;
  collectionCount?: number;
  updatedAt?: string;
}

export interface BookEngagementStats {
  bookId: string;
  socialStats: BookSocialStats;
  downloadStats: BookDownloadStats;
  readStats: BookReadStats;
}

export interface Bookmark {
  userId: string;
  bookId: string;
  createdAt: string;
}

export interface BookUserState {
  bookId: string;
  bookmarked: boolean;
  myReview?: BookReview;
  ratingSummary: BookRatingSummary;
  socialStats?: BookSocialStats;
  downloadStats: BookDownloadStats;
  readStats: BookReadStats;
  collections: string[];
}

export interface RecordReadingActivityPayload {
  bookId: string;
  fileId?: string;
  chapterId: string;
  chapterTitle?: string;
  chapterIndex?: number;
  progressPercent?: number;
  locationCfi?: string;
  locationType?: string;
  eventType?: string;
}

export interface ReadingActivityResult {
  progress: {
    userId: string;
    bookId: string;
    fileId?: string;
    chapterId: string;
    chapterTitle: string;
    chapterIndex: number;
    progressPercent?: number;
    locationCfi?: string;
    locationType?: string;
    openedCount: number;
    qualifiedReadCount: number;
  };
  stats: BookReadStats;
  counted: boolean;
  cooldownSeconds: number;
}
