import "fake-indexeddb/auto";
import { beforeEach, describe, expect, it } from "vitest";
import { offlineStore } from "./offlineStore";
import type { Book, Chapter } from "@/types";

const book = (id: string): Book => ({ id, title: `Book ${id}`, library_id: "lib" } as Book);
const chapter = (id: string, index: number): Chapter =>
  ({ id, book_id: "b1", title: `Ch ${index}`, chapter_index: index } as Chapter);

describe("offlineStore", () => {
  beforeEach(async () => {
    await offlineStore.clearAll();
  });

  it("round-trips a book with its chapters", async () => {
    const chapters = [chapter("c1", 0), chapter("c2", 1)];
    await offlineStore.saveBook({ book: book("b1"), chapters, savedAt: 1 });
    await offlineStore.saveChapter("b1", "c1", "<p>one</p>");
    await offlineStore.saveChapter("b1", "c2", "<p>two</p>");

    const stored = await offlineStore.getBook("b1");
    expect(stored?.chapters).toHaveLength(2);
    expect(await offlineStore.getChapter("b1", "c2")).toBe("<p>two</p>");
  });

  // Keyed by "bookId:chapterId", so deleting one book must not touch another's chapters.
  it("deletes only the requested book's chapters", async () => {
    await offlineStore.saveBook({ book: book("b1"), chapters: [chapter("c1", 0)], savedAt: 1 });
    await offlineStore.saveChapter("b1", "c1", "<p>first book</p>");
    await offlineStore.saveBook({ book: book("b2"), chapters: [chapter("c1", 0)], savedAt: 1 });
    await offlineStore.saveChapter("b2", "c1", "<p>second book</p>");

    await offlineStore.deleteBook("b1");

    expect(await offlineStore.getBook("b1")).toBeUndefined();
    expect(await offlineStore.getChapter("b1", "c1")).toBeUndefined();
    expect(await offlineStore.getChapter("b2", "c1")).toBe("<p>second book</p>");
  });

  // Anything left after logout is readable by the next person on a shared device.
  it("clearAll leaves nothing behind for the next user", async () => {
    await offlineStore.saveBook({ book: book("b1"), chapters: [chapter("c1", 0)], savedAt: 1 });
    await offlineStore.saveChapter("b1", "c1", "<p>private</p>");
    await offlineStore.saveBlob("b1", "asset:page1.jpg", new Blob(["private page"]));

    await offlineStore.clearAll();

    expect(await offlineStore.listBooks()).toEqual([]);
    expect(await offlineStore.getChapter("b1", "c1")).toBeUndefined();
    expect(await offlineStore.getBlob("b1", "asset:page1.jpg")).toBeUndefined();
  });

  // fake-indexeddb structured-clones a Blob into a plain object under jsdom, so these assert
  // that the right key was stored and returned rather than reading the bytes back.
  it("stores comic pages and raw files under separate keys", async () => {
    await offlineStore.saveBlob("b1", "asset:pages/001.jpg", new Blob(["page one"]));
    await offlineStore.saveBlob("b1", "file:f1", new Blob(["audio bytes"]));

    expect(await offlineStore.getBlob("b1", "asset:pages/001.jpg")).toBeDefined();
    expect(await offlineStore.getBlob("b1", "file:f1")).toBeDefined();
    expect(await offlineStore.getBlob("b1", "asset:pages/002.jpg")).toBeUndefined();
  });

  it("deletes only the requested book's blobs", async () => {
    await offlineStore.saveBlob("b1", "asset:cover.jpg", new Blob(["first"]));
    await offlineStore.saveBlob("b2", "asset:cover.jpg", new Blob(["second"]));

    await offlineStore.deleteBook("b1");

    expect(await offlineStore.getBlob("b1", "asset:cover.jpg")).toBeUndefined();
    expect(await offlineStore.getBlob("b2", "asset:cover.jpg")).toBeDefined();
  });

  // A download that dies before the books row is written used to leave its chapters and blobs
  // orphaned, because deletion walked the entry's chapter list to find what to remove.
  it("deletes a partial download that never got a book row", async () => {
    await offlineStore.saveChapter("b1", "c1", "<p>half</p>");
    await offlineStore.saveBlob("b1", "asset:page1.jpg", new Blob(["half"]));

    await offlineStore.deleteBook("b1");

    expect(await offlineStore.getChapter("b1", "c1")).toBeUndefined();
    expect(await offlineStore.getBlob("b1", "asset:page1.jpg")).toBeUndefined();
  });
});
