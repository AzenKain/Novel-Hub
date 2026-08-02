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

export interface ReadingGoal {
  userId: string;
  targetWordsPerDay: number;
  targetBooksPerYear: number;
  updatedAt: string;
}

// Mirrors the backend SmartCollectionRuleDto — the subset of search params a
// saved filter can replay. Keys are snake_case because they go straight into the
// /library query string.
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
  userId: string;
  name: string;
  rule: SmartCollectionRule;
  createdAt: string;
  updatedAt: string;
}
