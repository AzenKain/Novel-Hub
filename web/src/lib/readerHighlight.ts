import { resolveCfiRange } from "./epubCfi";

/**
 * Helper functions for reader text extraction, word highlighting and text selection offsets.
 */

const ensureHighlightStyle = () => {
  if (typeof document === "undefined") return;
  const styleId = "tts-active-word-style";
  if (!document.getElementById(styleId)) {
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = `
      ::highlight(tts-active-word) {
        background-color: rgba(59, 130, 246, 0.45) !important;
        color: inherit !important;
        text-decoration: underline !important;
        text-decoration-color: #2563eb !important;
        text-underline-offset: 4px !important;
        border-radius: 4px !important;
      }
      ::highlight(user-highlight-yellow), ::highlight(user-highlight-default) {
        background-color: rgba(254, 240, 138, 0.65) !important;
        color: inherit !important;
        border-bottom: 2px dashed #eab308 !important;
      }
      ::highlight(user-highlight-green) {
        background-color: rgba(187, 247, 208, 0.65) !important;
        color: inherit !important;
        border-bottom: 2px dashed #22c55e !important;
      }
      ::highlight(user-highlight-blue) {
        background-color: rgba(191, 219, 254, 0.65) !important;
        color: inherit !important;
        border-bottom: 2px dashed #3b82f6 !important;
      }
      ::highlight(user-highlight-pink) {
        background-color: rgba(251, 207, 232, 0.65) !important;
        color: inherit !important;
        border-bottom: 2px dashed #ec4899 !important;
      }
      ::highlight(user-highlight-purple) {
        background-color: rgba(233, 213, 255, 0.65) !important;
        color: inherit !important;
        border-bottom: 2px dashed #a855f7 !important;
      }
    `;
    document.head.appendChild(style);
  }
};

export const clearHighlight = () => {
  if (typeof CSS !== "undefined" && "highlights" in CSS) {
    try {
      // @ts-ignore
      CSS.highlights.delete("tts-active-word");
    } catch (e) {}
  }
};

export interface TtsStartPoint {
  textNodeIndex: number;
  offset: number;
}

export interface SavedSelection {
  selectedText: string;
  textNodeIndex: number;
  offset: number;
  /**
   * Document-relative character offsets captured at selection time, so the
   * highlight can be created even if the reader DOM is rebuilt (and the
   * cloned Range invalidated) between selecting text and clicking a color.
   */
  startIndex: number;
  endIndex: number;
}

export const getCharacterOffsetOfRange = (
  container: HTMLElement,
  range: Range
): { start: number; end: number } | null => {
  if (!container || !range) return null;
  const isWithin = (node: Node) => node === container || container.contains(node);

  // A range covering the full container text; reused for the spanning check and
  // to compute the container's total text length (clamp ceiling).
  let fullRange: Range;
  try {
    fullRange = document.createRange();
    fullRange.selectNodeContents(container);
  } catch {
    return null;
  }
  const total = fullRange.toString().length;

  const boundaryOffset = (node: Node, offset: number): number | null => {
    try {
      const before = document.createRange();
      before.selectNodeContents(container);
      before.setEnd(node, offset);
      return before.toString().length;
    } catch {
      return null;
    }
  };

  // Clamp boundaries that fall outside the reader container to the container's
  // text bounds. A selection released near the floating toolbar can extend
  // beyond the reader; rejecting it outright would make the toolbar appear but
  // leave highlighting silently doing nothing (no request, no error). Clamp so
  // the in-reader portion is still highlighted — but only when the selection
  // actually overlaps the reader. A selection entirely outside the reader
  // still resolves to null so we never highlight the wrong text.
  const startIn = isWithin(range.startContainer);
  const endIn = isWithin(range.endContainer);

  if (!startIn && !endIn) {
    // Both boundaries are outside; only act if the range actually spans the
    // container (start before it, end after it). Otherwise the selection does
    // not touch the reader at all and must stay unresolved. compareBoundaryPoints
    // can throw when the range lives in a different tree (e.g. a detached node),
    // which we treat as "no overlap".
    let spansContainer = false;
    try {
      spansContainer =
        range.compareBoundaryPoints(Range.START_TO_START, fullRange) < 0 &&
        range.compareBoundaryPoints(Range.END_TO_END, fullRange) > 0;
    } catch {
      spansContainer = false;
    }
    if (!spansContainer) return null;
    return { start: 0, end: total };
  }

  let startNode = range.startContainer;
  let startOffset = range.startOffset;
  let endNode = range.endContainer;
  let endOffset = range.endOffset;

  if (!startIn) {
    // Selection starts before the reader → clamp to the very beginning.
    startNode = container;
    startOffset = 0;
  }
  if (!endIn) {
    // Selection ends after the reader → clamp to the very end.
    endNode = container;
    endOffset = container.childNodes.length;
  }

  const start = boundaryOffset(startNode, startOffset);
  const end = boundaryOffset(endNode, endOffset);

  if (start === null || end === null) return null;
  if (end <= start) return null;
  // Guard against a clamped end overshooting the container's true text length.
  if (total >= 0 && end > total) {
    if (start >= total) return null;
    return { start, end: total };
  }
  return { start, end };
};

export const getTextNodeIndex = (container: HTMLElement, targetNode: Node): number => {
  const treeWalker = document.createTreeWalker(
    container,
    NodeFilter.SHOW_TEXT,
    null
  );
  let index = 0;
  while (treeWalker.nextNode()) {
    if (treeWalker.currentNode === targetNode) {
      return index;
    }
    index++;
  }
  return -1;
};

export const resolveToTextNode = (
  container: HTMLElement,
  node: Node,
  offset: number
): { textNode: Node; textOffset: number } | null => {
  if (!node || !container) return null;

  // Case 1: If node is already a TEXT_NODE
  if (node.nodeType === Node.TEXT_NODE) {
    const parent = node.parentNode;
    if (parent && container.contains(parent)) {
      return {
        textNode: node,
        textOffset: Math.min(offset, node.textContent?.length || 0),
      };
    }
    return null;
  }

  // Case 2: If node is an ELEMENT_NODE (e.g., DIV, P, SECTION)
  if (node.nodeType === Node.ELEMENT_NODE) {
    const elem = node as HTMLElement;
    if (!container.contains(elem) && elem !== container) return null;

    const childNodes = elem.childNodes;
    let targetChild: Node | null = null;
    if (offset < childNodes.length) {
      targetChild = childNodes[offset];
    } else if (childNodes.length > 0) {
      targetChild = childNodes[childNodes.length - 1];
    } else {
      targetChild = elem;
    }

    const treeWalker = document.createTreeWalker(
      container,
      NodeFilter.SHOW_TEXT,
      null
    );

    while (treeWalker.nextNode()) {
      const textNode = treeWalker.currentNode;
      if (targetChild === textNode || targetChild.contains(textNode)) {
        return { textNode, textOffset: 0 };
      }
      if (
        targetChild.compareDocumentPosition(textNode) &
        Node.DOCUMENT_POSITION_FOLLOWING
      ) {
        return { textNode, textOffset: 0 };
      }
    }
  }

  return null;
};

export const saveSelection = (container: HTMLElement, range: Range): SavedSelection | null => {
  const selectedText = range.toString().trim();
  const resolved = resolveToTextNode(container, range.startContainer, range.startOffset);
  if (!resolved) return null;

  const { textNode, textOffset } = resolved;
  const textNodeIndex = getTextNodeIndex(container, textNode);
  if (textNodeIndex < 0) return null;

  // Capture document-relative char offsets now, while the range still points
  // at live text nodes. The caller stores this and can reuse it later even if
  // the reader DOM gets rebuilt (which would invalidate a cloned Range).
  const offsets = getCharacterOffsetOfRange(container, range);
  if (!offsets || offsets.end <= offsets.start) return null;

  return {
    selectedText,
    textNodeIndex,
    offset: textOffset,
    startIndex: offsets.start,
    endIndex: offsets.end,
  };
};

export const getTextFromHereFromSaved = (
  container: HTMLElement,
  saved: SavedSelection
): string => {
  const treeWalker = document.createTreeWalker(
    container,
    NodeFilter.SHOW_TEXT,
    null
  );

  let currentIndex = 0;
  let targetNode: Node | null = null;

  while (treeWalker.nextNode()) {
    if (currentIndex === saved.textNodeIndex) {
      targetNode = treeWalker.currentNode;
      break;
    }
    currentIndex++;
  }

  if (!targetNode) return saved.selectedText;

  try {
    const fromHereRange = document.createRange();
    fromHereRange.selectNodeContents(container);
    fromHereRange.setStart(targetNode, saved.offset);
    return fromHereRange.toString().trim();
  } catch (e) {
    return saved.selectedText;
  }
};

export const highlightTextRangeFromNode = (
  container: HTMLElement,
  startPoint: TtsStartPoint | null,
  charIndex: number,
  length: number
) => {
  ensureHighlightStyle();
  window.getSelection()?.removeAllRanges();

  if (!startPoint || startPoint.textNodeIndex < 0) {
    return highlightTextRange(container, charIndex, length);
  }

  const treeWalker = document.createTreeWalker(
    container,
    NodeFilter.SHOW_TEXT,
    null
  );

  let currentIndex = 0;
  let currentNode: Node | null = null;
  while (treeWalker.nextNode()) {
    if (currentIndex === startPoint.textNodeIndex) {
      currentNode = treeWalker.currentNode;
      break;
    }
    currentIndex++;
  }

  if (!currentNode) {
    return highlightTextRange(container, charIndex, length);
  }
  
  const targetStartChar = startPoint.offset + charIndex;
  const targetEndChar = targetStartChar + length;

  let startNode: Node | null = null;
  let startNodeOffset = 0;
  let endNode: Node | null = null;
  let endNodeOffset = 0;

  let accumulated = 0;

  while (currentNode) {
    const nodeLen = currentNode.textContent?.length || 0;

    if (!startNode) {
      if (accumulated + nodeLen >= targetStartChar) {
        startNode = currentNode;
        startNodeOffset = targetStartChar - accumulated;
      }
    }

    if (startNode && !endNode) {
      if (accumulated + nodeLen >= targetEndChar) {
        endNode = currentNode;
        endNodeOffset = targetEndChar - accumulated;
        break;
      }
    }

    accumulated += nodeLen;
    currentNode = treeWalker.nextNode();
  }

  if (startNode && !endNode) {
    endNode = startNode;
    endNodeOffset = startNode.textContent?.length || 0;
  }

  if (startNode && endNode) {
    try {
      const range = document.createRange();
      range.setStart(startNode, startNodeOffset);
      range.setEnd(endNode, endNodeOffset);

      if (
        typeof CSS !== "undefined" &&
        "highlights" in CSS &&
        typeof Highlight !== "undefined"
      ) {
        // @ts-ignore
        CSS.highlights.set("tts-active-word", new Highlight(range));
      }

      // Auto-scroll active word into view smoothly
      const rect = range.getBoundingClientRect();
      if (rect.top < 100 || rect.bottom > window.innerHeight - 100) {
        const scrollable =
          container.closest(".overflow-y-auto") || container.parentElement;
        if (scrollable) {
          const containerRect = scrollable.getBoundingClientRect();
          const scrollTop =
            scrollable.scrollTop +
            (rect.top - containerRect.top) -
            scrollable.clientHeight / 2;
          scrollable.scrollTo({ top: Math.max(0, scrollTop), behavior: "smooth" });
        }
      }
    } catch (e) {}
  }
};

export const highlightTextRange = (
  container: HTMLElement,
  startChar: number,
  length: number
) => {
  ensureHighlightStyle();
  window.getSelection()?.removeAllRanges();

  const treeWalker = document.createTreeWalker(
    container,
    NodeFilter.SHOW_TEXT,
    null
  );
  let currentOffset = 0;
  let startNode: Node | null = null;
  let startNodeOffset = 0;
  let endNode: Node | null = null;
  let endNodeOffset = 0;

  while (treeWalker.nextNode()) {
    const node = treeWalker.currentNode;
    const nodeLength = node.textContent?.length || 0;

    if (!startNode && currentOffset + nodeLength > startChar) {
      startNode = node;
      startNodeOffset = startChar - currentOffset;
    }

    if (
      startNode &&
      !endNode &&
      currentOffset + nodeLength >= startChar + length
    ) {
      endNode = node;
      endNodeOffset = startChar + length - currentOffset;
      break;
    }

    currentOffset += nodeLength;
  }

  if (startNode && endNode) {
    const range = document.createRange();
    range.setStart(startNode, startNodeOffset);
    range.setEnd(endNode, endNodeOffset);

    if (typeof CSS !== "undefined" && "highlights" in CSS) {
      try {
        // @ts-ignore
        CSS.highlights.set("tts-active-word", new Highlight(range));
      } catch (e) {}
    }

    // Auto-scroll active word into view smoothly
    const rect = range.getBoundingClientRect();
    if (rect.top < 100 || rect.bottom > window.innerHeight - 100) {
      const scrollable =
        container.closest(".overflow-y-auto") || container.parentElement;
      if (scrollable) {
        const containerRect = scrollable.getBoundingClientRect();
        const scrollTop =
          scrollable.scrollTop +
          (rect.top - containerRect.top) -
          scrollable.clientHeight / 2;
        scrollable.scrollTo({ top: Math.max(0, scrollTop), behavior: "smooth" });
      }
    }
  }
};

export const extractTextFromHtml = (
  html: string,
  renderedContainer?: HTMLElement | null
) => {
  if (typeof document === "undefined" || !html) return "";

  if (renderedContainer) {
    const container = renderedContainer.cloneNode(true) as HTMLElement;
    container
      .querySelectorAll(
        "script, style, head, noscript, svg, nav, .reader-settings-panel, button"
      )
      .forEach((el) => el.remove());
    const rawText = container.innerText || container.textContent || "";
    const cleaned = rawText.replace(/\s+/g, " ").trim();
    if (cleaned) return cleaned;
  }

  const temp = document.createElement("div");
  temp.innerHTML = html;
  temp
    .querySelectorAll("script, style, head, noscript, svg, nav")
    .forEach((el) => el.remove());
  const rawText = temp.textContent || temp.innerText || "";
  return rawText.replace(/\s+/g, " ").trim();
};

/**
 * Resolve a character offset (from in-book search) to a DOM range over the
 * rendered text nodes and scroll it into view. Returns true if resolved.
 */
export const scrollToTextOffset = (container: HTMLElement, startChar: number): boolean => {
  if (!container || startChar <= 0) return false;

  const treeWalker = document.createTreeWalker(
    container,
    NodeFilter.SHOW_TEXT,
    null
  );
  let currentOffset = 0;
  let startNode: Node | null = null;
  let startNodeOffset = 0;

  while (treeWalker.nextNode()) {
    const node = treeWalker.currentNode;
    const nodeLength = node.textContent?.length || 0;
    if (currentOffset + nodeLength > startChar) {
      startNode = node;
      startNodeOffset = startChar - currentOffset;
      break;
    }
    currentOffset += nodeLength;
  }

  if (!startNode) return false;

  try {
    const range = document.createRange();
    range.setStart(startNode, startNodeOffset);
    range.setEnd(startNode, Math.min(startNodeOffset + 1, startNode.textContent?.length || 1));
    const el = range.startContainer.nodeType === Node.ELEMENT_NODE
      ? (range.startContainer as HTMLElement)
      : (range.startContainer.parentElement);
    el?.scrollIntoView({ block: "center", behavior: "smooth" });
    return true;
  } catch (e) {
    return false;
  }
};

export const getSelectionInfo = (container: HTMLElement, range: Range) => {
  const selectedText = range.toString().trim();

  const resolved = resolveToTextNode(container, range.startContainer, range.startOffset);

  if (!resolved) {
    return {
      selectedText,
      textFromHere: selectedText,
      startPoint: null,
    };
  }

  const { textNode, textOffset } = resolved;
  const textNodeIndex = getTextNodeIndex(container, textNode);

  let textFromHere = selectedText;
  try {
    const fromHereRange = document.createRange();
    fromHereRange.selectNodeContents(container);
    fromHereRange.setStart(textNode, textOffset);
    textFromHere = fromHereRange.toString().trim();
  } catch (e) {}

  return {
    selectedText,
    textFromHere,
    startPoint: {
      textNodeIndex,
      offset: textOffset,
    } as TtsStartPoint,
  };
};

export const createRangeFromCharOffset = (container: HTMLElement, startChar: number, endChar: number): Range | null => {
  if (!container || startChar < 0 || endChar <= startChar) return null;
  const treeWalker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, null);
  let currentOffset = 0;
  let startNode: Node | null = null;
  let startNodeOffset = 0;
  let endNode: Node | null = null;
  let endNodeOffset = 0;

  while (treeWalker.nextNode()) {
    const node = treeWalker.currentNode;
    const nodeLen = node.textContent?.length || 0;

    if (!startNode && currentOffset + nodeLen > startChar) {
      startNode = node;
      startNodeOffset = startChar - currentOffset;
    }
    if (!endNode && currentOffset + nodeLen >= endChar) {
      endNode = node;
      endNodeOffset = endChar - currentOffset;
      break;
    }
    currentOffset += nodeLen;
  }

  if (startNode && endNode) {
    try {
      const range = document.createRange();
      range.setStart(startNode, startNodeOffset);
      range.setEnd(endNode, endNodeOffset);
      return range;
    } catch (e) {
      return null;
    }
  }
  return null;
};

export interface HighlightEntity {
  id: string;
  text_content: string;
  start_index: number;
  end_index: number;
  color: string;
  cfi_range?: string;
}

export const applyUserHighlights = (
  container: HTMLElement | null,
  highlights: HighlightEntity[]
) => {
  if (typeof document === "undefined" || !container) return;
  ensureHighlightStyle();

  const colorGroups: Record<string, Range[]> = {
    yellow: [],
    green: [],
    blue: [],
    pink: [],
    purple: [],
    default: [],
  };

  if (!highlights) return;

  const treeWalker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, null);
  let fullText = "";
  while (treeWalker.nextNode()) {
    fullText += treeWalker.currentNode.textContent || "";
  }

  for (const h of highlights) {
    if (!h.text_content || !h.text_content.trim()) continue;
    let range: Range | null = null;

    if (h.cfi_range) {
      range = resolveCfiRange(container, h.cfi_range);
    }

    if (!range && h.start_index >= 0 && h.end_index > h.start_index) {
      range = createRangeFromCharOffset(container, h.start_index, h.end_index);
      if (range) {
        const normRange = range.toString().replace(/\s+/g, " ").trim();
        const normTarget = h.text_content.replace(/\s+/g, " ").trim();
        if (normRange !== normTarget && !normRange.includes(normTarget) && !normTarget.includes(normRange)) {
          range = null;
        }
      }
    }

    if (!range) {
      const targetText = h.text_content.trim();
      let idx = fullText.indexOf(targetText);
      if (idx === -1) {
        try {
          const regexPattern = targetText
            .replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
            .replace(/\s+/g, "\\s+");
          const match = new RegExp(regexPattern).exec(fullText);
          if (match) {
            idx = match.index;
            range = createRangeFromCharOffset(container, idx, idx + match[0].length);
          }
        } catch (e) {}
      } else {
        range = createRangeFromCharOffset(container, idx, idx + targetText.length);
      }
    }

    if (range) {
      let colorKey = h.color;
      if (colorKey === "#fef08a") colorKey = "yellow";
      else if (colorKey === "#bbf7d0") colorKey = "green";
      else if (colorKey === "#bfdbfe") colorKey = "blue";
      else if (colorKey === "#e9d5ff") colorKey = "purple";

      const c = colorKey && colorGroups[colorKey] ? colorKey : "yellow";
      colorGroups[c].push(range);
    }
  }

  if (typeof CSS !== "undefined" && "highlights" in CSS && typeof Highlight !== "undefined") {
    for (const [color, ranges] of Object.entries(colorGroups)) {
      try {
        if (ranges.length > 0) {
          // @ts-ignore
          CSS.highlights.set(`user-highlight-${color}`, new Highlight(...ranges));
        } else {
          // @ts-ignore
          CSS.highlights.delete(`user-highlight-${color}`);
        }
      } catch (e) {}
    }
  }
};
