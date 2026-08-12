export interface Podcast {
  id: string;
  library_id: string;
  feed_url: string;
  title: string;
  description?: string | null;
  cover_url?: string | null;
  author?: string | null;
  auto_download: boolean;
  last_checked_at?: string | null;
  episode_count?: number;
  created_at: string;
  updated_at: string;
}

export interface PodcastEpisode {
  id: string;
  podcast_id: string;
  guid: string;
  title: string;
  description?: string | null;
  audio_url: string;
  duration_sec?: number | null;
  published_at?: string | null;
  downloaded: boolean;
  book_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface SubscribePodcastInput {
  feed_url: string;
  library_id: string;
}

export interface UpdatePodcastInput {
  auto_download?: boolean;
}