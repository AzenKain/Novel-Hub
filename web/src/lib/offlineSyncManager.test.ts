import "fake-indexeddb/auto";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { offlineStore } from "./offlineStore";
import { flushSyncQueue, initOfflineSyncManager } from "./offlineSyncManager";
import { featureService } from "@/services/featureService";
import { readerService } from "@/services/readerService";

vi.mock("@/services/featureService", () => ({
  featureService: {
    recordReadingActivity: vi.fn(),
  },
}));

vi.mock("@/services/readerService", () => ({
  readerService: {
    syncReadingSession: vi.fn(),
  },
}));

describe("offlineSyncManager", () => {
  beforeEach(async () => {
    await offlineStore.clearSyncQueue();
    vi.clearAllMocks();
  });

  it("enqueues and flushes pending progress items successfully when online", async () => {
    vi.mocked(featureService.recordReadingActivity).mockResolvedValue({
      status: true,
      message: "Recorded",
      data: {} as any,
    });

    const progressPayload = {
      book_id: "book-1",
      chapter_id: "chap-1",
      current_cfi: "cfi-123",
      percentage: 50,
    };

    const id = await offlineStore.enqueueSyncItem({
      type: "progress",
      payload: progressPayload,
    });

    let items = await offlineStore.listSyncItems();
    expect(items).toHaveLength(1);
    expect(items[0].id).toBe(id);

    const count = await flushSyncQueue();
    expect(count).toBe(1);
    expect(featureService.recordReadingActivity).toHaveBeenCalledWith(progressPayload);

    items = await offlineStore.listSyncItems();
    expect(items).toHaveLength(0);
  });

  it("enqueues and flushes pending session items successfully when online", async () => {
    vi.mocked(readerService.syncReadingSession).mockResolvedValue({
      status: true,
      message: "Session synced",
    });

    const sessionPayload = {
      book_id: "book-2",
      duration: 300,
      words: 1500,
    };

    await offlineStore.enqueueSyncItem({
      type: "session",
      payload: sessionPayload,
    });

    const count = await flushSyncQueue();
    expect(count).toBe(1);
    expect(readerService.syncReadingSession).toHaveBeenCalledWith("book-2", 300, 1500);

    const items = await offlineStore.listSyncItems();
    expect(items).toHaveLength(0);
  });

  it("registers window online and visibilitychange event listeners", () => {
    const windowSpy = vi.spyOn(window, "addEventListener");
    const documentSpy = vi.spyOn(document, "addEventListener");
    const windowRemoveSpy = vi.spyOn(window, "removeEventListener");
    const documentRemoveSpy = vi.spyOn(document, "removeEventListener");

    const cleanup = initOfflineSyncManager();

    expect(windowSpy).toHaveBeenCalledWith("online", expect.any(Function));
    expect(documentSpy).toHaveBeenCalledWith("visibilitychange", expect.any(Function));

    cleanup();

    expect(windowRemoveSpy).toHaveBeenCalledWith("online", expect.any(Function));
    expect(documentRemoveSpy).toHaveBeenCalledWith("visibilitychange", expect.any(Function));
  });
});
