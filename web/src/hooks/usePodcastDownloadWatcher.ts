import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { usePodcastDownloadStore } from "@/stores";
import { podcastService } from "@/services";
import i18n from "@/i18n";
import { toast } from "react-toastify";

export function usePodcastDownloadWatcher() {
  const queryClient = useQueryClient();
  const activeDownloads = usePodcastDownloadStore(
    (state) => state.activeDownloads,
  );
  const finishDownload = usePodcastDownloadStore(
    (state) => state.finishDownload,
  );
  const isCheckingRef = useRef(false);

  useEffect(() => {
    const downloadEntries = Object.values(activeDownloads);
    if (downloadEntries.length === 0) return;

    const uniquePodcastIds = Array.from(
      new Set(downloadEntries.map((d) => d.podcastId)),
    );

    const checkDownloads = async () => {
      if (isCheckingRef.current) return;
      isCheckingRef.current = true;

      try {
        for (const podcastId of uniquePodcastIds) {
          const res = await podcastService.listEpisodes(podcastId);
          if (res.status && res.data) {
            const currentDownloads =
              usePodcastDownloadStore.getState().activeDownloads;
            for (const ep of res.data) {
              if (ep.downloaded && currentDownloads[ep.id]) {
                const title = currentDownloads[ep.id].episodeTitle || ep.title;
                finishDownload(ep.id);
                toast.success(
                  i18n.t(
                    "podcasts.download_completed",
                    'Episode "{{title}}" downloaded successfully',
                    { title },
                  ),
                  { toastId: `podcast-download-${ep.id}` },
                );
                void queryClient.invalidateQueries({
                  queryKey: ["podcastEpisodes", podcastId],
                });
                void queryClient.invalidateQueries({ queryKey: ["podcasts"] });
                void queryClient.invalidateQueries({ queryKey: ["books"] });
              }
            }
          }
        }
      } catch {
        // Ignore background polling errors
      } finally {
        isCheckingRef.current = false;
      }
    };

    // Run immediately on mount or download list change
    void checkDownloads();

    const intervalId = setInterval(() => {
      void checkDownloads();
    }, 2500);

    return () => clearInterval(intervalId);
  }, [activeDownloads, finishDownload, queryClient]);
}
