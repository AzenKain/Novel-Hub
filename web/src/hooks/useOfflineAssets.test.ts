import { sanitizeReaderHtml } from "@/utils/readerHtml";
import { describe, expect, it } from "vitest";

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
