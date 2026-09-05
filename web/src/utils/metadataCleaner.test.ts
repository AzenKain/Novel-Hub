import { describe, it, expect } from "vitest";
import {
  stripBrackets,
  stripParentheses,
  replaceUnderscores,
  toTitleCase,
  splitDashAuthorTitle,
  splitDashTitleAuthor,
  splitTitleByAuthor,
  cleanWhitespace,
  applyCustomRegex,
} from "./metadataCleaner";

describe("metadataCleaner tests", () => {
  describe("stripBrackets", () => {
    it("removes square bracket tags from titles", () => {
      expect(stripBrackets("[TangThuVien] Solo Leveling [VIP] [Full]")).toBe(
        "Solo Leveling",
      );
      expect(stripBrackets("[Audiobook] Dune [Unabridged]")).toBe("Dune");
      expect(stripBrackets("Normal Title")).toBe("Normal Title");
    });
  });

  describe("stripParentheses", () => {
    it("removes round bracket notes from titles", () => {
      expect(
        stripParentheses("Lord of the Mysteries (Bản Dịch Chuẩn) (Full)"),
      ).toBe("Lord of the Mysteries");
      expect(stripParentheses("Harry Potter (Book 1)")).toBe("Harry Potter");
    });
  });

  describe("replaceUnderscores", () => {
    it("converts underscores to spaces and cleans spacing", () => {
      expect(replaceUnderscores("Harry_Potter_and_the_Sorcerers_Stone")).toBe(
        "Harry Potter and the Sorcerers Stone",
      );
      expect(replaceUnderscores("__My__Book__Title__")).toBe("My Book Title");
    });
  });

  describe("toTitleCase", () => {
    it("formats titles into Title Case with minor words handling", () => {
      expect(toTitleCase("harry potter and the goblet of fire")).toBe(
        "Harry Potter and the Goblet of Fire",
      );
      expect(toTitleCase("THE LORD OF THE RINGS")).toBe(
        "The Lord of the Rings",
      );
      expect(toTitleCase("the catcher in the rye")).toBe(
        "The Catcher in the Rye",
      );
    });
  });

  describe("splitDashAuthorTitle and splitDashTitleAuthor", () => {
    it('splits "Author - Title"', () => {
      const res = splitDashAuthorTitle("Brandon Sanderson - Mistborn");
      expect(res).not.toBeNull();
      expect(res?.author).toBe("Brandon Sanderson");
      expect(res?.title).toBe("Mistborn");
    });

    it('splits "Title - Author"', () => {
      const res = splitDashTitleAuthor("Mistborn - Brandon Sanderson");
      expect(res).not.toBeNull();
      expect(res?.title).toBe("Mistborn");
      expect(res?.author).toBe("Brandon Sanderson");
    });

    it("returns null if no delimiter exists", () => {
      expect(splitDashAuthorTitle("Mistborn")).toBeNull();
    });
  });

  describe("splitTitleByAuthor", () => {
    it('splits "Title by Author" cleanly', () => {
      const res = splitTitleByAuthor("Dune by Frank Herbert");
      expect(res).not.toBeNull();
      expect(res?.title).toBe("Dune");
      expect(res?.author).toBe("Frank Herbert");
    });
  });

  describe("applyCustomRegex", () => {
    it("replaces regex patterns properly", () => {
      expect(
        applyCustomRegex("Chương 123: Hồi Kết", "^Chương \\d+:\\s*", ""),
      ).toBe("Hồi Kết");
      expect(
        applyCustomRegex("Chapter 01 - Prologue", "^Chapter \\d+\\s*-\\s*", ""),
      ).toBe("Prologue");
    });
  });

  describe("cleanWhitespace", () => {
    it("normalizes spaces and trims ends", () => {
      expect(cleanWhitespace("   A    Long    Title   ")).toBe("A Long Title");
    });
  });
});
