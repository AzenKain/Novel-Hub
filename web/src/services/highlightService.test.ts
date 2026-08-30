import { describe, it, expect, vi, beforeEach } from "vitest";
import { highlightService } from "./highlightService";
import { api } from "@/config/api";

vi.mock("@/config/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("highlightService export methods", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls exportAnki with correct URL and blob responseType", async () => {
    const fakeBlob = new Blob(["fake-apkg-data"], { type: "application/apkg" });
    vi.mocked(api.get).mockResolvedValueOnce({ data: fakeBlob });

    const result = await highlightService.exportAnki("book-123");
    expect(api.get).toHaveBeenCalledWith("/highlights/book-123/export.apkg", {
      responseType: "blob",
    });
    expect(result).toBe(fakeBlob);
  });

  it("calls exportCSV with correct URL and blob responseType", async () => {
    const fakeBlob = new Blob(["fake-csv-data"], { type: "text/csv" });
    vi.mocked(api.get).mockResolvedValueOnce({ data: fakeBlob });

    const result = await highlightService.exportCSV("book-123");
    expect(api.get).toHaveBeenCalledWith("/highlights/book-123/export.csv", {
      responseType: "blob",
    });
    expect(result).toBe(fakeBlob);
  });

  it("calls exportMarkdown with correct URL and blob responseType", async () => {
    const fakeBlob = new Blob(["# Book Title"], { type: "text/markdown" });
    vi.mocked(api.get).mockResolvedValueOnce({ data: fakeBlob });

    const result = await highlightService.exportMarkdown("book-123");
    expect(api.get).toHaveBeenCalledWith("/highlights/book-123/export.md", {
      responseType: "blob",
    });
    expect(result).toBe(fakeBlob);
  });
});
