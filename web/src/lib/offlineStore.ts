import type { Book, Chapter } from "@/types";

const DB_NAME = "novelhub-offline";
const DB_VERSION = 3;
const BOOKS = "books";
const CHAPTERS = "chapters";
const BLOBS = "blobs";
const SYNC_QUEUE = "sync_queue";

export type OfflineBook = {
  book: Book;
  chapters: Chapter[];
  savedAt: number;
};

export type PendingSyncItem = {
  id: string;
  type: "progress" | "session";
  payload: any;
  createdAt: number;
};

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(BOOKS)) db.createObjectStore(BOOKS);
      if (!db.objectStoreNames.contains(CHAPTERS)) db.createObjectStore(CHAPTERS);
      if (!db.objectStoreNames.contains(BLOBS)) db.createObjectStore(BLOBS);
      if (!db.objectStoreNames.contains(SYNC_QUEUE)) db.createObjectStore(SYNC_QUEUE);
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

function run<T>(
  storeNames: string | string[],
  mode: IDBTransactionMode,
  work: (tx: IDBTransaction) => IDBRequest<T> | null,
): Promise<T | undefined> {
  return openDB().then(
    (db) =>
      new Promise<T | undefined>((resolve, reject) => {
        const tx = db.transaction(storeNames, mode);
        const request = work(tx);
        tx.oncomplete = () => {
          db.close();
          resolve(request ? request.result : undefined);
        };
        tx.onerror = () => {
          db.close();
          reject(tx.error);
        };
      }),
  );
}

const scopedKey = (bookId: string, suffix: string) => `${bookId}:${suffix}`;

// Deleting by key range rather than by walking the book entry: a download interrupted before
// the books row was written would otherwise leave its chapters and blobs behind forever.
const bookRange = (bookId: string) => IDBKeyRange.bound(`${bookId}:`, `${bookId}:￿`);

export const offlineStore = {
  async saveBook(entry: OfflineBook): Promise<void> {
    await run(BOOKS, "readwrite", (tx) => tx.objectStore(BOOKS).put(entry, entry.book.id));
  },

  async getBook(bookId: string): Promise<OfflineBook | undefined> {
    return run<OfflineBook>(BOOKS, "readonly", (tx) => tx.objectStore(BOOKS).get(bookId));
  },

  async listBooks(): Promise<OfflineBook[]> {
    const all = await run<OfflineBook[]>(BOOKS, "readonly", (tx) => tx.objectStore(BOOKS).getAll());
    return all || [];
  },

  async saveChapter(bookId: string, chapterId: string, html: string): Promise<void> {
    await run(CHAPTERS, "readwrite", (tx) =>
      tx.objectStore(CHAPTERS).put(html, scopedKey(bookId, chapterId)),
    );
  },

  async getChapter(bookId: string, chapterId: string): Promise<string | undefined> {
    return run<string>(CHAPTERS, "readonly", (tx) =>
      tx.objectStore(CHAPTERS).get(scopedKey(bookId, chapterId)),
    );
  },

  async saveBlob(bookId: string, path: string, blob: Blob): Promise<void> {
    await run(BLOBS, "readwrite", (tx) => tx.objectStore(BLOBS).put(blob, scopedKey(bookId, path)));
  },

  async getBlob(bookId: string, path: string): Promise<Blob | undefined> {
    return run<Blob>(BLOBS, "readonly", (tx) => tx.objectStore(BLOBS).get(scopedKey(bookId, path)));
  },

  async deleteBook(bookId: string): Promise<void> {
    await run([BOOKS, CHAPTERS, BLOBS], "readwrite", (tx) => {
      tx.objectStore(BOOKS).delete(bookId);
      tx.objectStore(CHAPTERS).delete(bookRange(bookId));
      tx.objectStore(BLOBS).delete(bookRange(bookId));
      return null;
    });
  },

  async clearAll(): Promise<void> {
    await run([BOOKS, CHAPTERS, BLOBS, SYNC_QUEUE], "readwrite", (tx) => {
      tx.objectStore(BOOKS).clear();
      tx.objectStore(CHAPTERS).clear();
      tx.objectStore(BLOBS).clear();
      tx.objectStore(SYNC_QUEUE).clear();
      return null;
    });
  },

  async enqueueSyncItem(item: Omit<PendingSyncItem, "id" | "createdAt">): Promise<string> {
    const id = `${Date.now()}:${Math.random().toString(36).slice(2, 9)}`;
    const fullItem: PendingSyncItem = {
      id,
      type: item.type,
      payload: item.payload,
      createdAt: Date.now(),
    };
    await run(SYNC_QUEUE, "readwrite", (tx) => tx.objectStore(SYNC_QUEUE).put(fullItem, id));
    return id;
  },

  async listSyncItems(): Promise<PendingSyncItem[]> {
    const all = await run<PendingSyncItem[]>(SYNC_QUEUE, "readonly", (tx) =>
      tx.objectStore(SYNC_QUEUE).getAll(),
    );
    return (all || []).sort((a, b) => a.createdAt - b.createdAt);
  },

  async deleteSyncItem(id: string): Promise<void> {
    await run(SYNC_QUEUE, "readwrite", (tx) => {
      tx.objectStore(SYNC_QUEUE).delete(id);
      return null;
    });
  },

  async clearSyncQueue(): Promise<void> {
    await run(SYNC_QUEUE, "readwrite", (tx) => {
      tx.objectStore(SYNC_QUEUE).clear();
      return null;
    });
  },

  async usage(): Promise<{ usage: number; quota: number } | null> {
    if (!navigator.storage?.estimate) return null;
    const estimate = await navigator.storage.estimate();
    return { usage: estimate.usage || 0, quota: estimate.quota || 0 };
  },
};
