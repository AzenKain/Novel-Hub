import { describe, it, expect } from "vitest";
import {
  getAvailableChips,
  sanitizeChip,
  isCatalogView,
} from "./libraryFilter";

describe("libraryFilter utility", () => {
  describe("getAvailableChips", () => {
    it("returns only All and No cover for read books tab", () => {
      const chips = getAvailableChips("read");
      expect(chips).toEqual(["All", "No cover"]);
      expect(chips).not.toContain("Unread");
      expect(chips).not.toContain("Reading");
    });

    it("returns only All and No cover for unread books tab", () => {
      const chips = getAvailableChips("unread");
      expect(chips).toEqual(["All", "No cover"]);
      expect(chips).not.toContain("Unread");
      expect(chips).not.toContain("Reading");
    });

    it("returns only All and No cover when activeSmartFilterId is present", () => {
      const chips = getAvailableChips("books", "smart-filter-123");
      expect(chips).toEqual(["All", "No cover"]);
      expect(chips).not.toContain("Unread");
      expect(chips).not.toContain("Reading");
    });

    it("returns all chips for All Books (books)", () => {
      const chips = getAvailableChips("books");
      expect(chips).toEqual(["All", "Reading", "Unread", "No cover"]);
    });

    it("returns all chips for other sidebar tabs like hot, downloaded, top_rated, bookmarks, archived", () => {
      for (const nav of ["hot", "downloaded", "top_rated", "bookmarks", "archived"]) {
        expect(getAvailableChips(nav)).toEqual([
          "All",
          "Reading",
          "Unread",
          "No cover",
        ]);
      }
    });
  });

  describe("sanitizeChip", () => {
    it("falls back to All if rawChip is not in availableChips", () => {
      const available = ["All", "No cover"];
      expect(sanitizeChip("Unread", available)).toBe("All");
      expect(sanitizeChip("Reading", available)).toBe("All");
      expect(sanitizeChip(undefined, available)).toBe("All");
      expect(sanitizeChip(null, available)).toBe("All");
    });

    it("keeps valid chip if in availableChips", () => {
      const available = ["All", "No cover"];
      expect(sanitizeChip("No cover", available)).toBe("No cover");
      expect(sanitizeChip("All", available)).toBe("All");
    });
  });

  describe("isCatalogView", () => {
    it("returns false for home page (empty or books with no search/collection/facet)", () => {
      expect(
        isCatalogView({
          effectiveNav: "books",
          debouncedSearch: "",
          effectiveCollection: "",
          activeSmartFilterId: null,
          hasFacetSection: false,
        }),
      ).toBe(false);

      expect(
        isCatalogView({
          effectiveNav: "",
          debouncedSearch: "",
          effectiveCollection: "",
          activeSmartFilterId: null,
          hasFacetSection: false,
        }),
      ).toBe(false);
    });

    it("returns true for specific navigation tabs like read, unread, bookmarks, hot", () => {
      expect(isCatalogView({ effectiveNav: "read" })).toBe(true);
      expect(isCatalogView({ effectiveNav: "unread" })).toBe(true);
      expect(isCatalogView({ effectiveNav: "bookmarks" })).toBe(true);
      expect(isCatalogView({ effectiveNav: "hot" })).toBe(true);
    });

    it("returns true when searching, filtering by collection, smart filter, or facet", () => {
      expect(isCatalogView({ effectiveNav: "books", debouncedSearch: "test" })).toBe(true);
      expect(isCatalogView({ effectiveNav: "books", effectiveCollection: "Favorites" })).toBe(true);
      expect(isCatalogView({ effectiveNav: "books", activeSmartFilterId: "sf-1" })).toBe(true);
      expect(isCatalogView({ effectiveNav: "books", hasFacetSection: true })).toBe(true);
    });
  });
});
