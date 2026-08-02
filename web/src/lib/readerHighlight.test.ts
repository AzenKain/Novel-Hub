import { beforeEach, describe, expect, it } from "vitest";
import {
  extractTextFromHtml,
  getTextNodeIndex,
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
    });
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

  it("refuses offsets at or before the start, and past the end", () => {
    const container = render("<p>short</p>");

    expect(scrollToTextOffset(container, 0)).toBe(false);
    expect(scrollToTextOffset(container, -5)).toBe(false);
    // Total text is 5 chars; nothing to resolve past it.
    expect(scrollToTextOffset(container, 500)).toBe(false);
    expect(scrolled).toHaveLength(0);
  });
});
