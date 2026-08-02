export interface Library {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface LibraryStats {
  total_books: number;
  need_review: number;
  series_tracked: number;
}

export interface Collection {
  id: string;
  user_id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface ReadingGoal {
  user_id: string;
  target_words_per_day: number;
  target_books_per_year: number;
  updated_at: string;
}

export interface SmartCollectionRule {
  search?: string;
  library_id?: string;
  nav?: string;
  collection?: string;
  chip?: string;
  facet?: string;
  facet_id?: string;
}

export interface SmartCollection {
  id: string;
  user_id: string;
  name: string;
  rule: SmartCollectionRule;
  created_at: string;
  updated_at: string;
}
