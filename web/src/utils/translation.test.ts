import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  SUPPORTED_LANGUAGES,
  getDefaultTargetLanguage,
  saveTargetLanguagePreference,
  translateText,
} from "./translation";

describe("translation utility", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("should have rich multi-language catalog (>20 languages)", () => {
    expect(SUPPORTED_LANGUAGES.length).toBeGreaterThan(20);
    const codes = SUPPORTED_LANGUAGES.map((l) => l.code);
    expect(codes).toContain("vi");
    expect(codes).toContain("en");
    expect(codes).toContain("ja");
    expect(codes).toContain("zh-CN");
    expect(codes).toContain("fr");
    expect(codes).toContain("es");
    expect(codes).toContain("de");
  });

  it("should retrieve saved preference if present", () => {
    saveTargetLanguagePreference("ja");
    expect(getDefaultTargetLanguage("en")).toBe("ja");
  });

  it("should fallback to current locale if no saved preference", () => {
    expect(getDefaultTargetLanguage("fr")).toBe("fr");
    expect(getDefaultTargetLanguage("en-US")).toBe("en");
    expect(getDefaultTargetLanguage()).toBe("vi");
  });

  it("should successfully translate text using mock fetch", async () => {
    const mockResponse = [
      [["Bonjour le monde", "Hello world", null, null]],
      null,
      "en",
    ];
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: true,
      json: async () => mockResponse,
    } as Response);

    const result = await translateText("Hello world", "fr", "auto");
    expect(result.text).toBe("Bonjour le monde");
    expect(result.engine).toBe("google");
    expect(result.detectedSourceLang).toBe("en");
  });

  it("should fallback to MyMemory if Google fails", async () => {
    vi.spyOn(globalThis, "fetch")
      .mockRejectedValueOnce(new Error("Google network error"))
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          responseData: { translatedText: "Bonjour le monde (MyMemory)" },
        }),
      } as Response);

    const result = await translateText("Hello world", "fr", "auto");
    expect(result.text).toBe("Bonjour le monde (MyMemory)");
    expect(result.engine).toBe("mymemory");
  });
});
