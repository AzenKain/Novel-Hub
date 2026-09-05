import { describe, it, expect } from "vitest";
import { getCharacterOffsetOfRange, saveSelection } from "./readerHighlight";

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
    range.setStart(p1.firstChild!, 4);
    range.setEnd(outside.firstChild!, 4);

    expect(saveSelection(container, range)).not.toBeNull();

    const offset = getCharacterOffsetOfRange(container, range);
    expect(offset).not.toBeNull();
    expect(offset?.start).toBe(4);
    expect(offset?.end).toBe(container.textContent!.length);

    container.remove();
    outside.remove();
  });
});
