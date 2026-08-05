import { describe, expect, it } from "vitest";

// bookAdminStore used to keep its own `libraries` array, fetched straight from the service.
// Every mutation invalidated the ["libraries"] query key, which no consumer of that copy read —
// so renaming a library updated the modal it was renamed in and nothing else, and creating one
// left every other page stale until a reload.
//
// Read as raw text rather than imported: what broke is *where the list lives*, and a runtime
// import cannot see that. Vite's ?raw keeps this in the browser-only toolchain — asserting on
// the files with node:fs would mean adding @types/node for one test.
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
    for (const hook of ["useCreateLibraryMutation", "useUpdateLibraryMutation", "useDeleteLibraryMutation"]) {
      expect(libraryHooks).toContain(hook);
    }
    // One helper invalidates for all three; without it a mutation silently leaves the list stale.
    expect(libraryHooks.match(/invalidateLibraries\(queryClient\)/g)?.length).toBe(3);
  });

  it("clears the deleted library from filter and upload target", () => {
    expect(booksPage).toContain('setSelectedLibraryId("")');
    expect(booksPage).toContain('setUploadLibraryId("")');
  });
});
