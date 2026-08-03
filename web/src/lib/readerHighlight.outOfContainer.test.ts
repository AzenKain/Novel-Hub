import { describe, it, expect } from "vitest";
import { getCharacterOffsetOfRange, saveSelection } from "./readerHighlight";

/**
 * Regression test for the reader highlight bug:
 * "Bấm màu highlight không tô và không gửi request tới backend."
 *
 * Root cause: when a text selection extends OUTSIDE the reader container
 * (e.g. the mouse is released past the content area, near the floating
 * toolbar), saveSelection() still accepts the in-reader start so the toolbar
 * appears — but getCharacterOffsetOfRange() returned null because the range's
 * end container was not within the reader. handleHighlight then silently
 * early-returns (`if (!offset ...) return;`), so no request is sent and no
 * error is logged — exactly the reported symptom.
 *
 * Fix: clamp out-of-container range boundaries to the reader's text bounds so
 * the in-reader portion is still highlighted and the request is still sent.
 */
describe("getCharacterOffsetOfRange with range ending outside the reader", () => {
  it("clamps the end to the container boundary and still resolves an offset", () => {
    const container = document.createElement("div");
    container.className = "reader-content";
    container.innerHTML =
      '<div class="chapter-body"><p id="p1">The quick brown fox jumps.</p></div>';
    document.body.append(container);

    const outside = document.createElement("p");
    outside.textContent = "outside the reader";
    document.body.append(outside);

    const p1 = container.querySelector("#p1")!;
    const range = document.createRange();
    range.setStart(p1.firstChild!, 4); // "quick"
    range.setEnd(outside.firstChild!, 4); // ends OUTSIDE the reader

    // The toolbar would still show: saveSelection accepts the in-reader start.
    expect(saveSelection(container, range)).not.toBeNull();

    // Highlighting must still resolve a document-relative offset (clamped),
    // instead of silently returning null and dropping the request.
    const offset = getCharacterOffsetOfRange(container, range);
    expect(offset).not.toBeNull();
    expect(offset?.start).toBe(4);
    expect(offset?.end).toBe(container.textContent!.length);

    container.remove();
    outside.remove();
  });
});
