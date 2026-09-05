import { describe, it, expect, beforeEach } from "vitest";
import { usePodcastDownloadStore } from "./podcastDownloadStore";

describe("podcastDownloadStore", () => {
  beforeEach(() => {
    usePodcastDownloadStore.setState({ activeDownloads: {} });
  });

  it("adds and tracks downloading episodes", () => {
    const store = usePodcastDownloadStore.getState();
    expect(store.isDownloading("ep-1")).toBe(false);
    expect(store.hasActiveDownloads()).toBe(false);

    store.startDownload("pod-1", "ep-1", "Episode 1 Title");

    const updated = usePodcastDownloadStore.getState();
    expect(updated.isDownloading("ep-1")).toBe(true);
    expect(updated.isDownloading("ep-2")).toBe(false);
    expect(updated.hasActiveDownloads()).toBe(true);
    expect(updated.hasActiveDownloads("pod-1")).toBe(true);
    expect(updated.hasActiveDownloads("pod-2")).toBe(false);

    const activeList = updated.getActiveDownloads();
    expect(activeList).toHaveLength(1);
    expect(activeList[0].episodeTitle).toBe("Episode 1 Title");
  });

  it("finishes and cancels downloads cleanly", () => {
    const store = usePodcastDownloadStore.getState();
    store.startDownload("pod-1", "ep-1", "Episode 1");
    store.startDownload("pod-1", "ep-2", "Episode 2");

    expect(
      usePodcastDownloadStore.getState().getActiveDownloads(),
    ).toHaveLength(2);

    usePodcastDownloadStore.getState().finishDownload("ep-1");
    expect(usePodcastDownloadStore.getState().isDownloading("ep-1")).toBe(
      false,
    );
    expect(usePodcastDownloadStore.getState().isDownloading("ep-2")).toBe(true);
    expect(usePodcastDownloadStore.getState().hasActiveDownloads("pod-1")).toBe(
      true,
    );

    usePodcastDownloadStore.getState().cancelDownload("ep-2");
    expect(usePodcastDownloadStore.getState().isDownloading("ep-2")).toBe(
      false,
    );
    expect(usePodcastDownloadStore.getState().hasActiveDownloads()).toBe(false);
  });
});
