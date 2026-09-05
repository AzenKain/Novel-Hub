import { describe, it, expect, vi } from "vitest";
import { bookDoctorService } from "./bookDoctorService";
import { api } from "@/config/api";

vi.mock("@/config/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe("bookDoctorService", () => {
  it("calls validate endpoint correctly", async () => {
    const mockReport = {
      valid: true,
      errors: 0,
      warnings: 0,
      infos: 0,
      issues: [],
    };
    (api.get as any).mockResolvedValueOnce({
      data: { status: true, data: mockReport },
    });

    const res = await bookDoctorService.validateBook("book-123", "file-456");
    expect(api.get).toHaveBeenCalledWith("/books/book-123/doctor/validate", {
      params: { file_id: "file-456" },
    });
    expect(res).toEqual(mockReport);
  });

  it("calls repair endpoint with options", async () => {
    const mockResult = {
      success: true,
      fixed_count: 3,
      logs: ["[XHTML] Repaired entities"],
      report: { valid: true, errors: 0, warnings: 0, infos: 0, issues: [] },
    };
    (api.post as any).mockResolvedValueOnce({
      data: { status: true, data: mockResult },
    });

    const res = await bookDoctorService.repairBook(
      "book-123",
      { fix_xhtml: true, normalize_mimetype: true },
      "file-456",
    );
    expect(api.post).toHaveBeenCalledWith(
      "/books/book-123/doctor/repair",
      { fix_xhtml: true, normalize_mimetype: true },
      { params: { file_id: "file-456" } },
    );
    expect(res).toEqual(mockResult);
  });
});
