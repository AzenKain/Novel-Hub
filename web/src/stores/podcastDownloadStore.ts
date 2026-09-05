import { create } from "zustand";
import { persist } from "zustand/middleware";

export interface ActivePodcastDownload {
  podcastId: string;
  episodeId: string;
  episodeTitle?: string;
  startedAt: number;
}

interface PodcastDownloadState {
  activeDownloads: Record<string, ActivePodcastDownload>;
  startDownload: (
    podcastId: string,
    episodeId: string,
    episodeTitle?: string,
  ) => void;
  finishDownload: (episodeId: string) => void;
  cancelDownload: (episodeId: string) => void;
  isDownloading: (episodeId: string) => boolean;
  hasActiveDownloads: (podcastId?: string) => boolean;
  getActiveDownloads: () => ActivePodcastDownload[];
}

const DOWNLOAD_TIMEOUT_MS = 15 * 60 * 1000;

export const usePodcastDownloadStore = create<PodcastDownloadState>()(
  persist(
    (set, get) => ({
      activeDownloads: {},

      startDownload: (
        podcastId: string,
        episodeId: string,
        episodeTitle?: string,
      ) => {
        set((state) => ({
          activeDownloads: {
            ...state.activeDownloads,
            [episodeId]: {
              podcastId,
              episodeId,
              episodeTitle,
              startedAt: Date.now(),
            },
          },
        }));
      },

      finishDownload: (episodeId: string) => {
        set((state) => {
          if (!state.activeDownloads[episodeId]) return state;
          const next = { ...state.activeDownloads };
          delete next[episodeId];
          return { activeDownloads: next };
        });
      },

      cancelDownload: (episodeId: string) => {
        set((state) => {
          if (!state.activeDownloads[episodeId]) return state;
          const next = { ...state.activeDownloads };
          delete next[episodeId];
          return { activeDownloads: next };
        });
      },

      isDownloading: (episodeId: string) => {
        const item = get().activeDownloads[episodeId];
        if (!item) return false;
        if (Date.now() - item.startedAt > DOWNLOAD_TIMEOUT_MS) {
          get().finishDownload(episodeId);
          return false;
        }
        return true;
      },

      hasActiveDownloads: (podcastId?: string) => {
        const downloads = Object.values(get().activeDownloads);
        if (downloads.length === 0) return false;
        const now = Date.now();
        const valid = downloads.filter(
          (d) => now - d.startedAt <= DOWNLOAD_TIMEOUT_MS,
        );
        if (podcastId) {
          return valid.some((d) => d.podcastId === podcastId);
        }
        return valid.length > 0;
      },

      getActiveDownloads: () => {
        const downloads = Object.values(get().activeDownloads);
        const now = Date.now();
        return downloads.filter(
          (d) => now - d.startedAt <= DOWNLOAD_TIMEOUT_MS,
        );
      },
    }),
    {
      name: "novelhub-podcast-downloads",
    },
  ),
);
