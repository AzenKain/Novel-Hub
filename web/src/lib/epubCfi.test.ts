import { describe, it, expect } from "vitest";
import { generateCfi, resolveCfi, generateCfiRange, resolveCfiRange } from "./epubCfi";

describe("EPUB CFI Utilities", () => {
  it("generates and resolves element and text node CFIs correctly", () => {
    const container = document.createElement("div");
    container.innerHTML = `
      <div>
        <p>First paragraph text.</p>
        <p>Second paragraph with <b>bold text</b> and <i>italic</i> tags.</p>
      </div>
    `;

    const firstParagraphTextNode = container.querySelector("p")?.firstChild!;
    expect(firstParagraphTextNode).toBeDefined();

    // Sibling order:
    // div is the first child element of container -> /2
    // Inside div:
    // whitespace -> /1
    // p (first) -> /2
    // whitespace -> /3
    // p (second) -> /4
    // Inside first p:
    // text node -> /1
    const cfi = generateCfi(container, firstParagraphTextNode, 6, 0, "chapter-1");
    expect(cfi).toBe("epubcfi(/6/2[chapter-1]!/2/2/1:6)");

    const resolved = resolveCfi(container, cfi);
    expect(resolved).not.toBeNull();
    expect(resolved!.node).toBe(firstParagraphTextNode);
    expect(resolved!.offset).toBe(6);
  });

  it("handles complex mixed child element nodes", () => {
    const container = document.createElement("div");
    container.innerHTML = `<p>Hello <span>World</span> and <b>everyone</b>!</p>`;

    const bTextNode = container.querySelector("b")?.firstChild!;
    expect(bTextNode).toBeDefined();

    // Sibling order inside p:
    // text "Hello " -> /1
    // span -> /2
    // text " and " -> /3
    // b -> /4
    // Inside b:
    // text "everyone" -> /1
    const cfi = generateCfi(container, bTextNode, 3, 1, "chapter-2");
    expect(cfi).toBe("epubcfi(/6/4[chapter-2]!/2/4/1:3)");

    const resolved = resolveCfi(container, cfi);
    expect(resolved).not.toBeNull();
    expect(resolved!.node).toBe(bTextNode);
    expect(resolved!.offset).toBe(3);
  });

  it("generates and resolves range CFIs accurately", () => {
    const container = document.createElement("div");
    container.innerHTML = `<div><p>Hello <span>World</span> text</p></div>`;

    const startNode = container.querySelector("p")?.firstChild!; // text "Hello "
    const endNode = container.querySelector("span")?.firstChild!; // text "World"

    const range = document.createRange();
    range.setStart(startNode, 2);
    range.setEnd(endNode, 4);

    const cfiRange = generateCfiRange(container, range, 0, "chapter-1");
    // LCA of startNode and endNode is p element (/2/2)
    // startNode inside p is text node index 1 (/1)
    // endNode inside span (/2) is text node index 1 (/2/1)
    expect(cfiRange).toBe("epubcfi(/6/2[chapter-1]!/2/2,1:2,2/1:4)");

    const resolvedRange = resolveCfiRange(container, cfiRange);
    expect(resolvedRange).not.toBeNull();
    expect(resolvedRange!.startContainer).toBe(startNode);
    expect(resolvedRange!.startOffset).toBe(2);
    expect(resolvedRange!.endContainer).toBe(endNode);
    expect(resolvedRange!.endOffset).toBe(4);
  });
});
