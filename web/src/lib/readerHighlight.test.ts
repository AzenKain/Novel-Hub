import { beforeEach, describe, expect, it } from "vitest";
import {
  extractTextFromHtml,
  getCharacterOffsetOfRange,
  getTextNodeIndex,
  getVisibleTtsStartPoint,
  resolveToTextNode,
  saveSelection,
  scrollToTextOffset,
} from "./readerHighlight";

function render(html: string): HTMLElement {
  const container = document.createElement("div");
  container.innerHTML = html;
  document.body.appendChild(container);
  return container;
}

function textNodes(container: HTMLElement): Node[] {
  const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, null);
  const out: Node[] = [];
  while (walker.nextNode()) out.push(walker.currentNode);
  return out;
}

beforeEach(() => {
  document.body.innerHTML = "";
});

describe("getCharacterOffsetOfRange", () => {
  it("maps single- and multi-node selections", () => {
    const container = render("hello <b>world</b>!");
    const range = document.createRange();
    range.setStart(container.firstChild!, 1);
    range.setEnd(container.querySelector("b")!.firstChild!, 3);
    expect(getCharacterOffsetOfRange(container, range)).toEqual({ start: 1, end: 9 });

    const single = document.createRange();
    single.setStart(container.firstChild!, 1);
    single.setEnd(container.firstChild!, 4);
    expect(getCharacterOffsetOfRange(container, single)).toEqual({ start: 1, end: 4 });
  });

  it("maps element boundary containers to document offsets", () => {
    const container = render("<p>first</p><p>second</p>");
    const root = container;
    const range = document.createRange();
    range.setStart(root, 0);
    range.setEnd(root, root.childNodes.length);
    expect(getCharacterOffsetOfRange(container, range)).toEqual({ start: 0, end: 11 });
  });
  it("handles whitespace, equal, and unresolved ranges", () => {
    const container = render("   ");
    const whitespace = document.createRange();
    whitespace.selectNodeContents(container);
    expect(getCharacterOffsetOfRange(container, whitespace)).toEqual({ start: 0, end: 3 });
    const equal = document.createRange();
    equal.setStart(container.firstChild!, 1);
    equal.setEnd(container.firstChild!, 1);
    expect(getCharacterOffsetOfRange(container, equal)).toBeNull();
    const outside = document.createTextNode("outside");
    const unresolved = document.createRange();
    unresolved.setStart(outside, 0);
    unresolved.setEnd(outside, 2);
    expect(getCharacterOffsetOfRange(container, unresolved)).toBeNull();
  });
});
describe("getTextNodeIndex", () => {
  it("numbers text nodes in document order across nesting", () => {
    const container = render("<p>one</p><div><span>two</span>three</div>");
    const nodes = textNodes(container);

    expect(nodes.map((n) => n.textContent)).toEqual(["one", "two", "three"]);
    nodes.forEach((node, i) => {
      expect(getTextNodeIndex(container, node)).toBe(i);
    });
  });

  it("returns -1 for a node outside the container", () => {
    const container = render("<p>inside</p>");
    const stranger = render("<p>outside</p>");

    expect(getTextNodeIndex(container, textNodes(stranger)[0])).toBe(-1);
  });
});

describe("resolveToTextNode", () => {
  it("passes a text node through, clamping an offset past its length", () => {
    const container = render("<p>hello</p>");
    const node = textNodes(container)[0];

    expect(resolveToTextNode(container, node, 2)).toEqual({ textNode: node, textOffset: 2 });
    // A selection can report an offset past the node when the DOM shifted.
    expect(resolveToTextNode(container, node, 99)).toEqual({ textNode: node, textOffset: 5 });
  });

  it("maps an element + child index onto the text node underneath", () => {
    const container = render("<div><p>first</p><p>second</p></div>");
    const outer = container.firstElementChild as HTMLElement;

    const resolved = resolveToTextNode(container, outer, 1);
    expect(resolved?.textNode.textContent).toBe("second");
    expect(resolved?.textOffset).toBe(0);
  });

  it("rejects nodes belonging to a different container", () => {
    const container = render("<p>inside</p>");
    const stranger = render("<p>outside</p>");

    expect(resolveToTextNode(container, textNodes(stranger)[0], 0)).toBeNull();
    expect(resolveToTextNode(container, stranger.firstElementChild as HTMLElement, 0)).toBeNull();
  });
});

describe("saveSelection", () => {
  it("records the trimmed text plus a node index the DOM can be re-walked with", () => {
    const container = render("<p>alpha</p><p>  beta  </p>");
    const target = textNodes(container)[1];

    const range = document.createRange();
    range.setStart(target, 0);
    range.setEnd(target, target.textContent!.length);

    expect(saveSelection(container, range)).toEqual({
      selectedText: "beta",
      textNodeIndex: 1,
      offset: 0,
      startIndex: 5,
      endIndex: 13,
    });
  });

  it("captures document-relative char offsets so highlighting survives a later DOM rebuild", () => {
    const container = render("<p>alpha</p><p>  beta  </p>");
    const target = textNodes(container)[1];

    const range = document.createRange();
    range.setStart(target, 0);
    range.setEnd(target, target.textContent!.length);

    const saved = saveSelection(container, range);
    expect(saved?.startIndex).toBe(5);
    expect(saved?.endIndex).toBe(13);
    expect(saved?.endIndex).toBeGreaterThan(saved!.startIndex);
  });

  it("returns null when the range starts outside the container", () => {
    const container = render("<p>inside</p>");
    const stranger = render("<p>outside</p>");

    const range = document.createRange();
    const node = textNodes(stranger)[0];
    range.setStart(node, 0);
    range.setEnd(node, 3);

    expect(saveSelection(container, range)).toBeNull();
  });
});

describe("extractTextFromHtml", () => {
  it("drops script and style content and collapses whitespace", () => {
    const text = extractTextFromHtml(
      "<div>Hello\n\n  world<script>evil()</script><style>.a{color:red}</style></div>",
    );

    expect(text).toBe("Hello world");
  });

  it("returns an empty string for empty input", () => {
    expect(extractTextFromHtml("")).toBe("");
  });

  it("falls back to the raw html when the rendered container yields nothing", () => {
    const empty = render("");

    expect(extractTextFromHtml("<p>from html</p>", empty)).toBe("from html");
  });
});

describe("scrollToTextOffset", () => {
  // jsdom has no layout, so Element.scrollIntoView does not exist and the real
  // call would throw straight into the function's catch. Stub it so the test
  // measures offset resolution rather than jsdom's gaps.
  let scrolled: Element[];
  beforeEach(() => {
    scrolled = [];
    Element.prototype.scrollIntoView = function () {
      scrolled.push(this);
    };
  });

  it("resolves an offset that lands inside a later text node", () => {
    const container = render("<p>0123456789</p><p>abcdefghij</p>");

    expect(scrollToTextOffset(container, 12)).toBe(true);
    // Offset 12 is 2 chars into the second paragraph, so that is what scrolls.
    expect(scrolled).toHaveLength(1);
    expect(scrolled[0].textContent).toBe("abcdefghij");
  });

  it("refuses offsets before the start, and past the end", () => {
    const container = render("<p>short</p>");

    expect(scrollToTextOffset(container, 0)).toBe(true);
    expect(scrollToTextOffset(container, -5)).toBe(false);
    // Total text is 5 chars; nothing to resolve past it.
    expect(scrollToTextOffset(container, 500)).toBe(false);
  });
});

describe("getVisibleTtsStartPoint", () => {
  it("extracts from offset 0 when whole text node is visible at the beginning", () => {
    const container = render("<p>First line of chapter</p><p>Second line</p>");
    container.classList.add("reader-mode-single");
    const result = getVisibleTtsStartPoint(container);
    expect(result.text).toContain("First line of chapter");
    expect(result.startPoint?.offset).toBe(0);
  });

  it("extracts from overflowed text offset on current page when paragraph starts on previous page", () => {
    const container = render("<p>Paragraph one line one. Paragraph one line two on page two.</p><p>Paragraph two.</p>");
    container.classList.add("reader-mode-single");
    Object.defineProperty(container, "clientWidth", { value: 800, configurable: true });

    const originalGetBCR = container.getBoundingClientRect;
    container.getBoundingClientRect = () => ({
      left: 100,
      right: 900,
      top: 50,
      bottom: 700,
      width: 800,
      height: 650,
      x: 100,
      y: 50,
      toJSON: () => {},
    });

    const originalGetClientRects = Range.prototype.getClientRects;
    Range.prototype.getClientRects = function () {
      const nodeText = this.startContainer.textContent || "";
      if (nodeText.includes("Paragraph one line one")) {
        // If range starts at or after "Paragraph one line two" (offset 24)
        if (this.startOffset >= 24) {
          return [
            {
              left: 100,
              right: 800,
              top: 60,
              bottom: 90,
              width: 700,
              height: 30,
              x: 100,
              y: 60,
              toJSON: () => {},
            } as DOMRect,
          ] as unknown as DOMRectList;
        }
        // Entire node or start < 24: has 2 line boxes (one on previous page, one on page two)
        return [
          {
            left: -700,
            right: 0,
            top: 60,
            bottom: 90,
            width: 700,
            height: 30,
            x: -700,
            y: 60,
            toJSON: () => {},
          } as DOMRect,
          {
            left: 100,
            right: 800,
            top: 60,
            bottom: 90,
            width: 700,
            height: 30,
            x: 100,
            y: 60,
            toJSON: () => {},
          } as DOMRect,
        ] as unknown as DOMRectList;
      }
      return originalGetClientRects.call(this);
    };

    try {
      const result = getVisibleTtsStartPoint(container);
      expect(result.startPoint).not.toBeNull();
      expect(result.startPoint?.textNodeIndex).toBe(0);
      expect(result.startPoint?.offset).toBe(24);
      expect(result.text).toBe("Paragraph one line two on page two.Paragraph two.");
      expect(result.text.startsWith("Paragraph one line two")).toBe(true);
    } finally {
      Range.prototype.getClientRects = originalGetClientRects;
      container.getBoundingClientRect = originalGetBCR;
    }
  });
});
