import { featureService } from "@/services/featureService";
import { readerService } from "@/services/readerService";
import { offlineStore } from "./offlineStore";

let isSyncing = false;

export async function flushSyncQueue(): Promise<number> {
  if (isSyncing || (typeof navigator !== "undefined" && !navigator.onLine)) {
    return 0;
  }
  isSyncing = true;
  let syncedCount = 0;

  try {
    const items = await offlineStore.listSyncItems();
    for (const item of items) {
      try {
        if (item.type === "progress") {
          const res = await featureService.recordReadingActivity(item.payload);
          if (res.status) {
            await offlineStore.deleteSyncItem(item.id);
            syncedCount++;
          }
        } else if (item.type === "session") {
          const res = await readerService.syncReadingSession(
            item.payload.book_id,
            item.payload.duration,
            item.payload.words,
          );
          if (res.status) {
            await offlineStore.deleteSyncItem(item.id);
            syncedCount++;
          }
        }
      } catch (err) {
        if (typeof navigator !== "undefined" && !navigator.onLine) {
          break;
        }
      }
    }
  } finally {
    isSyncing = false;
  }

  return syncedCount;
}

export function initOfflineSyncManager(): () => void {
  if (typeof window === "undefined") return () => {};

  const handleOnline = () => {
    void flushSyncQueue();
  };

  const handleVisibilityChange = () => {
    if (document.visibilityState === "visible" && navigator.onLine) {
      void flushSyncQueue();
    }
  };

  window.addEventListener("online", handleOnline);
  document.addEventListener("visibilitychange", handleVisibilityChange);

  if (navigator.onLine) {
    void flushSyncQueue();
  }

  return () => {
    window.removeEventListener("online", handleOnline);
    document.removeEventListener("visibilitychange", handleVisibilityChange);
  };
}
