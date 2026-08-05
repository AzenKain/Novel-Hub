import type { Book, Chapter } from "@/types";

const DB_NAME = "novelhub-offline";
const DB_VERSION = 2;
const BOOKS = "books";
const CHAPTERS = "chapters";
const BLOBS = "blobs";

export type OfflineBook = {
  book: Book;
  chapters: Chapter[];
  savedAt: number;
};

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(BOOKS)) db.createObjectStore(BOOKS);
      if (!db.objectStoreNames.contains(CHAPTERS)) db.createObjectStore(CHAPTERS);
      if (!db.objectStoreNames.contains(BLOBS)) db.createObjectStore(BLOBS);
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
    await run([BOOKS, CHAPTERS, BLOBS], "readwrite", (tx) => {
      tx.objectStore(BOOKS).clear();
      tx.objectStore(CHAPTERS).clear();
      tx.objectStore(BLOBS).clear();
      return null;
    });
  },

  async usage(): Promise<{ usage: number; quota: number } | null> {
    if (!navigator.storage?.estimate) return null;
    const estimate = await navigator.storage.estimate();
    return { usage: estimate.usage || 0, quota: estimate.quota || 0 };
  },
};
