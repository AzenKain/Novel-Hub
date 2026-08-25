import { resolveCfiRange } from "./epubCfi";
import { sanitizeReaderHtml } from "@/utils/readerHtml";

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
      ::highlight(search-result-match) {
        background-color: rgba(250, 204, 21, 0.75) !important;
        color: #000000 !important;
        outline: 2px solid #ca8a04 !important;
        border-radius: 4px !important;
      }
      ::highlight(reader-active-selection) {
        background-color: rgba(59, 130, 246, 0.35) !important;
        color: inherit !important;
        border-radius: 3px !important;
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
      ::highlight(user-highlight-orange) {
        background-color: rgba(254, 215, 170, 0.65) !important;
        color: inherit !important;
        border-bottom: 2px dashed #f97316 !important;
      }
      ::highlight(user-highlight-purple) {
        background-color: rgba(233, 213, 255, 0.65) !important;
        color: inherit !important;
        border-bottom: 2px dashed #a855f7 !important;
      }
      ::highlight(user-highlight-red) {
        background-color: rgba(254, 202, 202, 0.65) !important;
        color: inherit !important;
        border-bottom: 2px dashed #ef4444 !important;
      }
      ::highlight(user-highlight-cyan) {
        background-color: rgba(153, 246, 228, 0.65) !important;
        color: inherit !important;
        border-bottom: 2px dashed #14b8a6 !important;
      }

      /* E-Ink Theme Optimizations (High Contrast, No Blurs, Jitter-Free) */
      [data-theme="e-ink"] .reader-selection-toolbar {
        animation: none !important;
        transition: none !important;
        box-shadow: none !important;
        border: 2px solid #000000 !important;
      }
      [data-theme="e-ink"] ::highlight(reader-active-selection) {
        background-color: rgba(0, 0, 0, 0.15) !important;
        color: inherit !important;
      }
      [data-theme="e-ink"] ::highlight(user-highlight-yellow),
      [data-theme="e-ink"] ::highlight(user-highlight-default),
      [data-theme="e-ink"] ::highlight(user-highlight-green),
      [data-theme="e-ink"] ::highlight(user-highlight-blue),
      [data-theme="e-ink"] ::highlight(user-highlight-pink),
      [data-theme="e-ink"] ::highlight(user-highlight-orange),
      [data-theme="e-ink"] ::highlight(user-highlight-purple),
      [data-theme="e-ink"] ::highlight(user-highlight-red),
      [data-theme="e-ink"] ::highlight(user-highlight-cyan) {
        background-color: rgba(0, 0, 0, 0.12) !important;
        color: inherit !important;
        border-bottom: 2px dashed #000000 !important;
      }
    `;
    document.head.appendChild(style);
  }
};

export const setActiveSelectionHighlight = (
  range: Range | null,
  container?: HTMLElement | null,
  saved?: SavedSelection | null
) => {
  if (typeof CSS !== "undefined" && "highlights" in CSS && typeof Highlight !== "undefined") {
    ensureHighlightStyle();
    try {
      if (range && !range.collapsed) {
        let liveRange = range;
        if (container && saved && typeof saved.startIndex === "number" && typeof saved.endIndex === "number") {
          const fresh = createRangeFromCharOffset(container, saved.startIndex, saved.endIndex);
          if (fresh) {
            liveRange = fresh;
          }
        }
        // @ts-ignore
        const activeSelHl = new Highlight(liveRange);
        activeSelHl.priority = 1;
        // @ts-ignore
        CSS.highlights.set("reader-active-selection", activeSelHl);
      } else {
        // @ts-ignore
        CSS.highlights.delete("reader-active-selection");
      }
    } catch (e) {}
  }
};

export const clearActiveSelectionHighlight = () => {
  if (typeof CSS !== "undefined" && "highlights" in CSS) {
    try {
      // @ts-ignore
      CSS.highlights.delete("reader-active-selection");
    } catch (e) {}
  }
};

export const clearHighlight = () => {
  invalidateTextNodesCache();
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
  if (!selectedText) return null;

  const offsets = getCharacterOffsetOfRange(container, range);
  if (!offsets || offsets.end <= offsets.start) return null;

  let textNodeIndex = -1;
  let textOffset = 0;

  const resolved = resolveToTextNode(container, range.startContainer, range.startOffset);
  if (resolved) {
    textNodeIndex = getTextNodeIndex(container, resolved.textNode);
    textOffset = resolved.textOffset;
  }

  if (textNodeIndex < 0) {
    const fallbackRange = createRangeFromCharOffset(container, offsets.start, offsets.end);
    if (fallbackRange) {
      const fallbackResolved = resolveToTextNode(container, fallbackRange.startContainer, fallbackRange.startOffset);
      if (fallbackResolved) {
        textNodeIndex = getTextNodeIndex(container, fallbackResolved.textNode);
        textOffset = fallbackResolved.textOffset;
      }
    }
  }

  return {
    selectedText,
    textNodeIndex: Math.max(0, textNodeIndex),
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

interface TextNodeCache {
  container: HTMLElement;
  nodes: Text[];
  lengths: number[];
}

let cachedTree: TextNodeCache | null = null;

export const getTextNodesCache = (container: HTMLElement): TextNodeCache => {
  if (
    cachedTree &&
    cachedTree.container === container &&
    container.isConnected &&
    cachedTree.nodes.length > 0 &&
    cachedTree.nodes[0].isConnected
  ) {
    return cachedTree;
  }
  const treeWalker = document.createTreeWalker(
    container,
    NodeFilter.SHOW_TEXT,
    null
  );
  const nodes: Text[] = [];
  const lengths: number[] = [];
  while (treeWalker.nextNode()) {
    const node = treeWalker.currentNode as Text;
    nodes.push(node);
    lengths.push(node.textContent?.length || 0);
  }
  cachedTree = { container, nodes, lengths };
  return cachedTree;
};

export const invalidateTextNodesCache = () => {
  cachedTree = null;
};

export const highlightTextRangeFromNode = (
  container: HTMLElement,
  startPoint: TtsStartPoint | null,
  charIndex: number,
  length: number
) => {
  ensureHighlightStyle();
  if (!container || length <= 0) return;

  const cache = getTextNodesCache(container);
  const { nodes, lengths } = cache;
  if (nodes.length === 0) return;

  const startIndex = startPoint && startPoint.textNodeIndex >= 0 && startPoint.textNodeIndex < nodes.length
    ? startPoint.textNodeIndex
    : 0;
  const startOffsetInFirstNode = startPoint ? Math.max(0, startPoint.offset) : 0;

  let startNode: Text | null = null;
  let startNodeOffset = 0;
  let endNode: Text | null = null;
  let endNodeOffset = 0;

  let currentGlobalOffset = 0;

  for (let i = startIndex; i < nodes.length; i++) {
    const node = nodes[i];
    const nodeLen = lengths[i];
    const offsetInNode = (i === startIndex) ? startOffsetInFirstNode : 0;
    const availableInNode = Math.max(0, nodeLen - offsetInNode);

    if (!startNode) {
      if (currentGlobalOffset + availableInNode > charIndex || (currentGlobalOffset + availableInNode >= charIndex && i === nodes.length - 1)) {
        startNode = node;
        startNodeOffset = offsetInNode + Math.max(0, charIndex - currentGlobalOffset);
      }
    }

    if (startNode && !endNode) {
      const targetEndOffset = charIndex + length;
      if (currentGlobalOffset + availableInNode >= targetEndOffset || i === nodes.length - 1) {
        endNode = node;
        endNodeOffset = Math.min(nodeLen, offsetInNode + Math.max(0, targetEndOffset - currentGlobalOffset));
        break;
      }
    }

    currentGlobalOffset += availableInNode;
  }

  if (!startNode) return;
  if (!endNode) {
    endNode = startNode;
    endNodeOffset = Math.min(startNode.textContent?.length || 0, startNodeOffset + length);
  }

  try {
    const range = document.createRange();
    range.setStart(startNode, Math.min(startNodeOffset, startNode.textContent?.length || 0));
    range.setEnd(endNode, Math.min(endNodeOffset, endNode.textContent?.length || 0));

    if (
      typeof CSS !== "undefined" &&
      "highlights" in CSS &&
      typeof Highlight !== "undefined"
    ) {
      // @ts-ignore
      const ttsHl = new Highlight(range);
      ttsHl.priority = 3;
      // @ts-ignore
      CSS.highlights.set("tts-active-word", ttsHl);
    }

    // Auto-scroll or auto-flip page for active word smoothly via rAF
    if (typeof requestAnimationFrame !== "undefined") {
      requestAnimationFrame(() => {
        try {
          const isPaged = container.classList?.contains("reader-mode-single") || container.classList?.contains("reader-mode-double");
          const pagedContainer = (container.querySelector("body") || container) as HTMLElement;

          if (isPaged && pagedContainer.clientWidth > 0 && typeof pagedContainer.scrollTo === "function") {
            const rect = range.getBoundingClientRect();
            const containerRect = pagedContainer.getBoundingClientRect();
            const pageGap = 40;
            const scrollStep = pagedContainer.clientWidth + pageGap;
            const currentScrollLeft = pagedContainer.scrollLeft;
            const relativeLeft = (rect.left - containerRect.left) + currentScrollLeft;
            const targetPageIndex = Math.max(0, Math.floor(relativeLeft / scrollStep));
            const currentCalculatedPage = Math.round(currentScrollLeft / scrollStep);

            if (targetPageIndex !== currentCalculatedPage && targetPageIndex >= 0) {
              pagedContainer.scrollTo({
                left: targetPageIndex * scrollStep,
                behavior: "smooth",
              });
            }
          } else {
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
        } catch (e) {}
      });
    }
  } catch (e) {}
};

export const highlightTextRange = (
  container: HTMLElement,
  startChar: number,
  length: number
) => {
  highlightTextRangeFromNode(container, null, startChar, length);
};

/**
 * Determine the first visible text node and starting offset currently in view,
 * both for CSS multi-column paged modes and vertical continuous scroll mode.
 */
export const getVisibleTtsStartPoint = (
  container: HTMLElement,
  scrollContainer?: HTMLElement | null
): { text: string; startPoint: TtsStartPoint | null } => {
  if (!container) return { text: "", startPoint: null };

  const isPaged = container.classList?.contains("reader-mode-single") || container.classList?.contains("reader-mode-double");
  const pagedContainer = (container.querySelector("body") || container) as HTMLElement;

  const treeWalker = document.createTreeWalker(
    container,
    NodeFilter.SHOW_TEXT,
    null
  );

  let textNodeIndex = 0;
  let targetNode: Node | null = null;
  let targetNodeIndex = 0;
  let targetOffset = 0;
  let firstNonEmptyNode: Node | null = null;
  let firstNonEmptyNodeIndex = 0;

  if (isPaged && (pagedContainer.clientWidth > 0 || (typeof window !== "undefined" && window.innerWidth > 0))) {
    const containerRect = pagedContainer.clientWidth > 0
      ? pagedContainer.getBoundingClientRect()
      : { left: 0, right: typeof window !== "undefined" ? window.innerWidth : 1024, top: 0, bottom: typeof window !== "undefined" ? window.innerHeight : 800, width: 1024, height: 800 };

    while (treeWalker.nextNode()) {
      const node = treeWalker.currentNode;
      const text = (node.textContent || "").trim();
      if (text.length > 0) {
        if (!firstNonEmptyNode) {
          firstNonEmptyNode = node;
          firstNonEmptyNodeIndex = textNodeIndex;
        }
        try {
          const range = document.createRange();
          range.selectNodeContents(node);
          const rawRects = Array.from(range.getClientRects());
          const rects = rawRects.length > 0 ? rawRects : [range.getBoundingClientRect()];

          let hasVisibleRect = false;
          let firstVisibleRectIndex = -1;

          for (let i = 0; i < rects.length; i++) {
            const r = rects[i];
            if (r.width === 0 && r.height === 0) continue;
            if (
              r.right > containerRect.left + 2 &&
              r.left < containerRect.right - 2 &&
              r.bottom > containerRect.top + 2 &&
              r.top < containerRect.bottom - 2
            ) {
              hasVisibleRect = true;
              if (firstVisibleRectIndex === -1) {
                firstVisibleRectIndex = i;
              }
            }
          }

          if (hasVisibleRect) {
            targetNode = node;
            targetNodeIndex = textNodeIndex;

            if (firstVisibleRectIndex === 0) {
              targetOffset = 0;
            } else {
              // The text node started on a previous page and overflowed onto this page.
              // Binary search to find the exact character offset where the visible portion begins.
              const nodeLen = node.textContent?.length || 0;
              let low = 0;
              let high = nodeLen;
              let foundOffset = 0;

              while (low <= high) {
                const mid = Math.floor((low + high) / 2);
                const subRange = document.createRange();
                subRange.setStart(node, mid);
                subRange.setEnd(node, nodeLen);
                const subRects = subRange.getClientRects();
                if (subRects.length === 0) {
                  high = mid - 1;
                  continue;
                }
                const firstSub = subRects[0];
                if (firstSub.right <= containerRect.left + 2) {
                  // Still on previous page
                  low = mid + 1;
                } else {
                  foundOffset = mid;
                  high = mid - 1;
                }
              }
              targetOffset = foundOffset;
            }
            break;
          }
        } catch (e) {}
      }
      textNodeIndex++;
    }
  } else {
    // Vertical scroll mode
    const scrollParent = scrollContainer || container.closest(".overflow-y-auto") || container.parentElement;
    const scrollRect = scrollParent ? scrollParent.getBoundingClientRect() : { top: 0, bottom: window.innerHeight, left: 0, right: window.innerWidth };

    while (treeWalker.nextNode()) {
      const node = treeWalker.currentNode;
      const text = (node.textContent || "").trim();
      if (text.length > 0) {
        if (!firstNonEmptyNode) {
          firstNonEmptyNode = node;
          firstNonEmptyNodeIndex = textNodeIndex;
        }
        try {
          const range = document.createRange();
          range.selectNodeContents(node);
          const rawRects = Array.from(range.getClientRects());
          const rects = rawRects.length > 0 ? rawRects : [range.getBoundingClientRect()];

          let hasVisibleRect = false;
          let firstVisibleRectIndex = -1;

          for (let i = 0; i < rects.length; i++) {
            const r = rects[i];
            if (r.width === 0 && r.height === 0) continue;
            if (
              r.bottom >= scrollRect.top + 10 &&
              r.top < scrollRect.bottom - 10
            ) {
              hasVisibleRect = true;
              if (firstVisibleRectIndex === -1) {
                firstVisibleRectIndex = i;
              }
            }
          }

          if (hasVisibleRect) {
            targetNode = node;
            targetNodeIndex = textNodeIndex;

            if (firstVisibleRectIndex === 0) {
              targetOffset = 0;
            } else {
              const nodeLen = node.textContent?.length || 0;
              let low = 0;
              let high = nodeLen;
              let foundOffset = 0;

              while (low <= high) {
                const mid = Math.floor((low + high) / 2);
                const subRange = document.createRange();
                subRange.setStart(node, mid);
                subRange.setEnd(node, nodeLen);
                const subRects = subRange.getClientRects();
                if (subRects.length === 0) {
                  high = mid - 1;
                  continue;
                }
                const firstSub = subRects[0];
                if (firstSub.bottom < scrollRect.top + 10) {
                  low = mid + 1;
                } else {
                  foundOffset = mid;
                  high = mid - 1;
                }
              }
              targetOffset = foundOffset;
            }
            break;
          }
        } catch (e) {}
      }
      textNodeIndex++;
    }
  }

  if (!targetNode && firstNonEmptyNode) {
    targetNode = firstNonEmptyNode;
    targetNodeIndex = firstNonEmptyNodeIndex;
    targetOffset = 0;
  }

  if (targetNode) {
    try {
      const fromHereRange = document.createRange();
      fromHereRange.selectNodeContents(container);
      fromHereRange.setStart(targetNode, targetOffset);
      const rawText = fromHereRange.toString();
      if (rawText.trim()) {
        return {
          text: rawText,
          startPoint: {
            textNodeIndex: targetNodeIndex,
            offset: targetOffset,
          },
        };
      }
    } catch (e) {}
  }

  // Fallback: entire container text
  const fullText = extractTextFromHtml(container.innerHTML, container);
  return {
    text: fullText,
    startPoint: null,
  };
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
        "script, style, head, title, meta, noscript, svg, nav, .reader-settings-panel, button, [hidden], [style*='display: none'], [style*='display:none'], [style*='visibility: hidden'], [style*='visibility:hidden'], .d-none, .hidden, .invisible"
      )
      .forEach((el) => el.remove());
    const rawText = container.innerText || container.textContent || "";
    const cleaned = rawText.replace(/\s+/g, " ").trim();
    if (cleaned) return cleaned;
  }

  const cleanHtml = sanitizeReaderHtml(html);
  const temp = document.createElement("div");
  temp.innerHTML = cleanHtml;
  temp
    .querySelectorAll(
      "script, style, head, title, meta, noscript, svg, nav, [hidden], [style*='display: none'], [style*='display:none'], [style*='visibility: hidden'], [style*='visibility:hidden'], .d-none, .hidden, .invisible"
    )
    .forEach((el) => el.remove());
  const rawText = temp.textContent || temp.innerText || "";
  return rawText.replace(/\s+/g, " ").trim();
};

/**
 * Resolve a character offset (from in-book search) to a DOM range over the
 * rendered text nodes and scroll it into view. Returns true if resolved.
 */
export const scrollToTextOffset = (container: HTMLElement, startChar: number): boolean => {
  if (!container || startChar < 0) return false;

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
    if (currentOffset + nodeLength > startChar || (startChar === 0 && nodeLength > 0)) {
      startNode = node;
      startNodeOffset = Math.max(0, startChar - currentOffset);
      break;
    }
    currentOffset += nodeLength;
  }

  if (!startNode) return false;

  try {
    const range = document.createRange();
    range.setStart(startNode, startNodeOffset);
    range.setEnd(startNode, Math.min(startNodeOffset + 1, startNode.textContent?.length || 1));

    // Check if container is in CSS multi-column paginated mode
    const isPaged = container.classList?.contains("reader-mode-single") || container.classList?.contains("reader-mode-double");
    const pagedContainer = (container.querySelector("body") || container) as HTMLElement;

    if (isPaged && pagedContainer.clientWidth > 0 && typeof range.getBoundingClientRect === "function") {
      const rangeRect = range.getBoundingClientRect();
      const containerRect = pagedContainer.getBoundingClientRect();
      const pageGap = 40;
      const scrollStep = pagedContainer.clientWidth + pageGap;
      const currentScrollLeft = pagedContainer.scrollLeft;
      const relativeLeft = (rangeRect.left - containerRect.left) + currentScrollLeft;
      const pageIndex = Math.max(0, Math.floor(relativeLeft / scrollStep));
      pagedContainer.scrollTo({
        left: pageIndex * scrollStep,
        behavior: "smooth",
      });
      return true;
    }

    // Vertical scroll mode
    const el = range.startContainer.nodeType === Node.ELEMENT_NODE
      ? (range.startContainer as HTMLElement)
      : (range.startContainer.parentElement);
    el?.scrollIntoView({ block: "center", behavior: "smooth" });
    return true;
  } catch (e) {
    return false;
  }
};

let searchHighlightTimer: ReturnType<typeof setTimeout> | null = null;
let activeClearListener: (() => void) | null = null;

export const clearSearchHighlight = () => {
  if (searchHighlightTimer) {
    clearTimeout(searchHighlightTimer);
    searchHighlightTimer = null;
  }
  if (activeClearListener) {
    document.removeEventListener("pointerdown", activeClearListener, true);
    document.removeEventListener("keydown", activeClearListener, true);
    activeClearListener = null;
  }
  try {
    if (typeof CSS !== "undefined" && "highlights" in CSS) {
      // @ts-ignore
      CSS.highlights.delete("search-result-match");
    }
  } catch {}
};

/**
 * Find search term or snippet in the rendered reader DOM, scroll to it,
 * and visually highlight the match with CSS Highlight API or pulse.
 */
export const scrollToSearchMatch = (
  container: HTMLElement,
  searchTerm: string,
  snippet?: string
): boolean => {
  if (!container || (!searchTerm && !snippet)) return false;
  ensureHighlightStyle();

  const cleanTerm = (searchTerm || "").trim().toLowerCase();

  let markWord = "";
  let snippetPhrase = "";
  if (snippet) {
    const markMatch = snippet.match(/<mark[^>]*>(.*?)<\/mark>/i);
    if (markMatch) {
      markWord = markMatch[1].replace(/<[^>]+>/g, "").trim().toLowerCase();
    }
    snippetPhrase = snippet
      .replace(/<[^>]+>/g, "")
      .replace(/&[a-z0-9#]+;/gi, " ")
      .replace(/\s*\.\.\.\s*/g, " ")
      .trim()
      .toLowerCase();
  }

  const treeWalker = document.createTreeWalker(
    container,
    NodeFilter.SHOW_TEXT,
    null
  );

  const textNodes: Node[] = [];
  const nodeOffsets: { node: Node; start: number; len: number }[] = [];
  let accumulated = "";

  while (treeWalker.nextNode()) {
    const node = treeWalker.currentNode;
    const txt = node.textContent || "";
    textNodes.push(node);
    nodeOffsets.push({ node, start: accumulated.length, len: txt.length });
    accumulated += txt;
  }

  if (textNodes.length === 0) return false;

  const lowerAccum = accumulated.toLowerCase();
  let matchIndex = -1;
  let matchLength = cleanTerm.length || 1;

  if (snippetPhrase && snippetPhrase.length > 5) {
    matchIndex = lowerAccum.indexOf(snippetPhrase);
    if (matchIndex !== -1) {
      matchLength = snippetPhrase.length;
      if (markWord) {
        const markInSnippet = snippetPhrase.indexOf(markWord);
        if (markInSnippet !== -1) {
          matchIndex += markInSnippet;
          matchLength = markWord.length;
        }
      }
    } else {
      const words = snippetPhrase.split(/\s+/).filter((w) => w.length > 1);
      for (let w = Math.min(words.length, 5); w >= 2; w--) {
        for (let i = 0; i <= words.length - w; i++) {
          const sub = words.slice(i, i + w).join(" ");
          const subIdx = lowerAccum.indexOf(sub);
          if (subIdx !== -1) {
            matchIndex = subIdx;
            matchLength = sub.length;
            break;
          }
        }
        if (matchIndex !== -1) break;
      }
    }
  }

  if (matchIndex === -1 && markWord) {
    matchIndex = lowerAccum.indexOf(markWord);
    if (matchIndex !== -1) matchLength = markWord.length;
  }

  if (matchIndex === -1 && cleanTerm) {
    matchIndex = lowerAccum.indexOf(cleanTerm);
    if (matchIndex !== -1) matchLength = cleanTerm.length;
  }

  if (matchIndex === -1) return false;

  let targetNode: Node | null = null;
  let targetOffset = 0;
  for (const entry of nodeOffsets) {
    if (entry.start + entry.len > matchIndex) {
      targetNode = entry.node;
      targetOffset = Math.max(0, matchIndex - entry.start);
      break;
    }
  }

  if (!targetNode) return false;

  try {
    const range = document.createRange();
    range.setStart(targetNode, targetOffset);
    const nodeLen = targetNode.textContent?.length || 0;
    const endOff = Math.min(nodeLen, targetOffset + matchLength);
    range.setEnd(targetNode, endOff);

    if (
      typeof CSS !== "undefined" &&
      "highlights" in CSS &&
      typeof Highlight !== "undefined"
    ) {
      clearSearchHighlight();

      // @ts-ignore
      const searchHl = new Highlight(range);
      searchHl.priority = 10;
      // @ts-ignore
      CSS.highlights.set("search-result-match", searchHl);

      searchHighlightTimer = setTimeout(() => {
        clearSearchHighlight();
      }, 2000);

      activeClearListener = () => {
        clearSearchHighlight();
      };

      setTimeout(() => {
        if (activeClearListener) {
          document.addEventListener("pointerdown", activeClearListener, true);
          document.addEventListener("keydown", activeClearListener, true);
        }
      }, 150);
    }

    const isPaged = container.classList?.contains("reader-mode-single") || container.classList?.contains("reader-mode-double");
    const pagedContainer = (container.querySelector("body") || container) as HTMLElement;

    if (isPaged && pagedContainer.clientWidth > 0 && typeof range.getBoundingClientRect === "function") {
      const rangeRect = range.getBoundingClientRect();
      const containerRect = pagedContainer.getBoundingClientRect();
      const pageGap = 40;
      const scrollStep = pagedContainer.clientWidth + pageGap;
      const currentScrollLeft = pagedContainer.scrollLeft;
      const relativeLeft = (rangeRect.left - containerRect.left) + currentScrollLeft;
      const pageIndex = Math.max(0, Math.floor(relativeLeft / scrollStep));
      pagedContainer.scrollTo({
        left: pageIndex * scrollStep,
        behavior: "smooth",
      });
      return true;
    }

    const el = range.startContainer.nodeType === Node.ELEMENT_NODE
      ? (range.startContainer as HTMLElement)
      : range.startContainer.parentElement;
    if (el) {
      el.scrollIntoView({ block: "center", behavior: "smooth" });
    }
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
  const cache = getTextNodesCache(container);
  const { nodes, lengths } = cache;
  if (nodes.length === 0) return null;

  let currentOffset = 0;
  let startNode: Node | null = null;
  let startNodeOffset = 0;
  let endNode: Node | null = null;
  let endNodeOffset = 0;

  for (let i = 0; i < nodes.length; i++) {
    const node = nodes[i];
    const nodeLen = lengths[i];

    if (!startNode) {
      if (currentOffset + nodeLen > startChar || (currentOffset + nodeLen >= startChar && (i === nodes.length - 1 || startChar === 0))) {
        startNode = node;
        startNodeOffset = Math.max(0, startChar - currentOffset);
      }
    }
    if (startNode && !endNode) {
      if (currentOffset + nodeLen >= endChar || i === nodes.length - 1) {
        endNode = node;
        endNodeOffset = Math.min(nodeLen, endChar - currentOffset);
        break;
      }
    }
    currentOffset += nodeLen;
  }

  if (startNode && !endNode) {
    endNode = startNode;
    endNodeOffset = Math.min(startNode.textContent?.length || 0, startNodeOffset + (endChar - startChar));
  }

  if (startNode && endNode) {
    try {
      const range = document.createRange();
      range.setStart(startNode, Math.min(startNodeOffset, startNode.textContent?.length || 0));
      range.setEnd(endNode, Math.min(endNodeOffset, endNode.textContent?.length || 0));
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
    orange: [],
    purple: [],
    red: [],
    cyan: [],
    default: [],
  };

  if (!highlights || highlights.length === 0) {
    if (typeof CSS !== "undefined" && "highlights" in CSS && typeof Highlight !== "undefined") {
      for (const color of Object.keys(colorGroups)) {
        try {
          // @ts-ignore
          CSS.highlights.delete(`user-highlight-${color}`);
        } catch (e) {}
      }
    }
    return;
  }

  const cache = getTextNodesCache(container);
  const fullText = cache.nodes.map(n => n.textContent || "").join("").normalize("NFC");

  for (const h of highlights) {
    if (!h.text_content || !h.text_content.trim()) continue;
    let range: Range | null = null;

    if (h.cfi_range) {
      range = resolveCfiRange(container, h.cfi_range);
    }

    if (!range && typeof h.start_index === "number" && typeof h.end_index === "number" && h.end_index > h.start_index) {
      range = createRangeFromCharOffset(container, h.start_index, h.end_index);
      if (range) {
        const normRange = range.toString().normalize("NFC").replace(/\s+/g, " ").trim();
        const normTarget = (h.text_content || "").normalize("NFC").replace(/\s+/g, " ").trim();
        if (normRange !== normTarget && !normRange.includes(normTarget) && !normTarget.includes(normRange)) {
          range = null;
        }
      }
    }

    if (!range) {
      const targetText = (h.text_content || "").normalize("NFC").trim();
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
      const rawColor = (h.color || "").toLowerCase().trim();
      let colorKey = "yellow";
      if (rawColor === "#fef08a" || rawColor === "#facc15" || rawColor === "yellow") colorKey = "yellow";
      else if (rawColor === "#bbf7d0" || rawColor === "#4ade80" || rawColor === "green") colorKey = "green";
      else if (rawColor === "#bfdbfe" || rawColor === "#60a5fa" || rawColor === "blue") colorKey = "blue";
      else if (rawColor === "#fbcfe8" || rawColor === "#f472b6" || rawColor === "pink") colorKey = "pink";
      else if (rawColor === "#fed7aa" || rawColor === "#fb923c" || rawColor === "orange") colorKey = "orange";
      else if (rawColor === "#e9d5ff" || rawColor === "#c084fc" || rawColor === "purple") colorKey = "purple";
      else if (rawColor === "#fecaca" || rawColor === "#f87171" || rawColor === "red") colorKey = "red";
      else if (rawColor === "#99f6e4" || rawColor === "#2dd4bf" || rawColor === "cyan") colorKey = "cyan";
      else if (colorGroups[rawColor]) colorKey = rawColor;

      colorGroups[colorKey].push(range);
    }
  }

  if (typeof CSS !== "undefined" && "highlights" in CSS && typeof Highlight !== "undefined") {
    for (const [color, ranges] of Object.entries(colorGroups)) {
      try {
        if (ranges.length > 0) {
          // @ts-ignore
          const userHl = new Highlight(...ranges);
          userHl.priority = 2;
          // @ts-ignore
          CSS.highlights.set(`user-highlight-${color}`, userHl);
        } else {
          // @ts-ignore
          CSS.highlights.delete(`user-highlight-${color}`);
        }
      } catch (e) {}
    }
  }
};
