import { describe, expect, it } from "vitest";

import booksPage from "../pages/admin/Books.tsx?raw";
import libraryHooks from "./useLibraryQueries.ts?raw";
import bookAdminStore from "../stores/bookAdminStore.ts?raw";

describe("library list ownership", () => {
  it("keeps no libraries copy in bookAdminStore", () => {
    expect(bookAdminStore).not.toMatch(/\blibraries\b/);
    expect(bookAdminStore).not.toContain("loadLibraries");
    expect(bookAdminStore).not.toContain("libraryService");
  });

  it("reads libraries through the query the mutations invalidate", () => {
    expect(booksPage).toContain("useLibrariesQuery()");
    expect(booksPage).not.toContain("state.libraries");

    expect(libraryHooks).toContain('queryKey: ["libraries"]');
    for (const hook of [
      "useCreateLibraryMutation",
      "useUpdateLibraryMutation",
      "useDeleteLibraryMutation",
    ]) {
      expect(libraryHooks).toContain(hook);
    }
    expect(
      libraryHooks.match(/invalidateLibraries\(queryClient\)/g)?.length,
    ).toBe(3);
  });

  it("clears the deleted library from filter and upload target", () => {
    expect(booksPage).toContain('setSelectedLibraryId("")');
    expect(booksPage).toContain('setUploadLibraryId("")');
  });
});
