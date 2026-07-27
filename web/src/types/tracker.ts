export interface ConnectTrackerInput {
  provider: string;
  access_token: string;
}

export interface MapTrackerInput {
  book_id: string;
  provider: string;
  external_series_id: string;
}

export interface SyncProgressInput {
  book_id: string;
  title: string;
  progress: number;
}

export interface TrackerSearchResult {
  provider: string;
  external_series_id: string;
}
