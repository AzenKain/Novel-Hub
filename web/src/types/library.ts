export interface Library {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface LibraryStats {
  totalBooks: number;
  needReview: number;
  seriesTracked: number;
}

export interface Collection {
  id: string;
  userId: string;
  name: string;
  createdAt: string;
  updatedAt: string;
}
