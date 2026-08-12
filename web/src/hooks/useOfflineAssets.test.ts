import { sanitizeReaderHtml } from "@/utils/readerHtml";
import { describe, expect, it, beforeEach, vi } from "vitest";
import { useOfflineAssets } from "./useOfflineAssets";
import { offlineStore } from "@/lib/offlineStore";
import "fake-indexeddb/auto";

// Mock React hooks to allow direct invocation of the custom hook in a non-React context
vi.mock("react", () => ({
  useRef: (initial: any) => ({ current: initial }),
  useCallback: (fn: any) => fn,
  useEffect: () => {},
}));

describe("sanitizeReaderHtml with offline blob URLs", () => {
  it("preserves blob: URLs in img src attributes", () => {
    const html = `<div class="novelhub-cbz-page"><img src="blob:http://localhost:3434/1234-5678" alt="Page 1" /></div>`;
    const sanitized = sanitizeReaderHtml(html);
    expect(sanitized).toContain('src="blob:http://localhost:3434/1234-5678"');
  });

  it("preserves blob: URLs in SVG image xlink:href attributes", () => {
    const html = `<svg viewBox="0 0 100 100"><image xlink:href="blob:http://localhost:3434/1234-5678" width="100" height="100"/></svg>`;
    const sanitized = sanitizeReaderHtml(html);
    expect(sanitized).toContain('xlink:href="blob:http://localhost:3434/1234-5678"');
  });
});

describe("useOfflineAssets hook", () => {
  beforeEach(async () => {
    await offlineStore.clearAll();
  });

  it("resolves blob URL from offline store", async () => {
    // Mock URL.createObjectURL and URL.revokeObjectURL
    const originalCreate = URL.createObjectURL;
    const originalRevoke = URL.revokeObjectURL;
    URL.createObjectURL = () => "blob:mocked-url";
    URL.revokeObjectURL = () => {};

    try {
      await offlineStore.saveBlob("b1", "file:f1", new Blob(["audio content"]));

      const result = useOfflineAssets("b1");
      const url = await result.resolveBlobURL("file:f1");
      
      expect(url).toBe("blob:mocked-url");
    } finally {
      URL.createObjectURL = originalCreate;
      URL.revokeObjectURL = originalRevoke;
    }
  });
});
