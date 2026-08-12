/**
 * EPUB3 Canonical Fragment Identifier (CFI) Generator & Resolver
 * Implements the standard EPUB CFI format: elements are channellized as even numbers,
 * text nodes as odd numbers, with spine chapter offset mapping.
 */

// Helper to determine index path step from parent node to child node
function getCfiStep(parent: Node, child: Node): string {
  let elementIndex = 0;
  let textIndex = 0;

  for (const node of Array.from(parent.childNodes)) {
    if (node.nodeType === Node.ELEMENT_NODE) {
      elementIndex++;
      if (node === child) {
        return `/${elementIndex * 2}`;
      }
    } else if (node.nodeType === Node.TEXT_NODE) {
      textIndex++;
      if (node === child) {
        return `/${textIndex * 2 - 1}`;
      }
    }
  }
  return "";
}

// Helper to construct the DOM path steps from target node up to container
function getDomPath(container: HTMLElement, target: Node): string {
  let path = "";
  let current: Node | null = target;

  while (current && current !== container) {
    const parent: Node | null = current.parentNode;
    if (!parent) break;

    const step = getCfiStep(parent, current);
    path = step + path;
    current = parent;
  }
  return path;
}

/**
 * Generate a standard EPUB CFI position string for a specific node and offset.
 */
export function generateCfi(
  container: HTMLElement,
  node: Node,
  offset: number,
  chapterIndex: number,
  chapterId: string
): string {
  if (!container || !node) return "";
  const domPath = getDomPath(container, node);
  const spineIndex = (chapterIndex + 1) * 2;
  return `epubcfi(/6/${spineIndex}[${chapterId}]!${domPath}:${offset})`;
}

/**
 * Resolve a standard EPUB CFI string to a DOM Node and character offset.
 */
export function resolveCfi(
  container: HTMLElement,
  cfi: string
): { node: Node; offset: number } | null {
  if (!container || !cfi) return null;

  // Match epubcfi(...) structure
  const match = cfi.match(/^epubcfi\((.+)\)$/);
  if (!match) return null;

  const content = match[1];
  const parts = content.split("!");
  if (parts.length < 2) return null;

  // The DOM path part is after the spine indicator '!'
  const domPart = parts[1];
  const [pathPart, offsetPart] = domPart.split(":");
  const offset = offsetPart ? parseInt(offsetPart, 10) : 0;

  const steps = pathPart.split("/").filter(Boolean);
  let current: Node = container;

  for (const step of steps) {
    const val = parseInt(step, 10);
    if (isNaN(val)) return null;

    if (val % 2 === 0) {
      // Even number: Element child
      const targetIndex = val / 2 - 1;
      const elementChildren = Array.from(current.childNodes).filter(
        (n) => n.nodeType === Node.ELEMENT_NODE
      );
      if (targetIndex >= 0 && targetIndex < elementChildren.length) {
        current = elementChildren[targetIndex];
      } else {
        return null;
      }
    } else {
      // Odd number: Text child
      const targetIndex = (val + 1) / 2 - 1;
      const textChildren = Array.from(current.childNodes).filter(
        (n) => n.nodeType === Node.TEXT_NODE
      );
      if (targetIndex >= 0 && targetIndex < textChildren.length) {
        current = textChildren[targetIndex];
      } else {
        return null;
      }
    }
  }

  return { node: current, offset };
}

/**
 * Generate a standard EPUB CFI range string for a DOM selection Range.
 */
export function generateCfiRange(
  container: HTMLElement,
  range: Range,
  chapterIndex: number,
  chapterId: string
): string {
  if (!container || !range) return "";

  const startNode = range.startContainer;
  const endNode = range.endContainer;

  // Find lowest common ancestor
  let lca: Node | null = startNode;
  while (lca && !lca.contains(endNode)) {
    lca = lca.parentNode;
  }
  if (!lca || !container.contains(lca)) {
    lca = container;
  }

  const lcaPath = getDomPath(container, lca);
  const startPath = getDomPath(container, startNode);
  const endPath = getDomPath(container, endNode);

  // Compute paths relative to LCA
  let startRel = startPath.slice(lcaPath.length);
  let endRel = endPath.slice(lcaPath.length);

  // Strip leading slash if relative paths start with it
  if (startRel.startsWith("/")) startRel = startRel.slice(1);
  if (endRel.startsWith("/")) endRel = endRel.slice(1);

  const spineIndex = (chapterIndex + 1) * 2;
  return `epubcfi(/6/${spineIndex}[${chapterId}]!${lcaPath},${startRel}:${range.startOffset},${endRel}:${range.endOffset})`;
}

/**
 * Resolve a standard EPUB CFI range string back into a DOM Range object.
 */
export function resolveCfiRange(container: HTMLElement, cfiRange: string): Range | null {
  if (!container || !cfiRange) return null;

  const match = cfiRange.match(/^epubcfi\((.+)\)$/);
  if (!match) return null;

  const content = match[1];
  const parts = content.split("!");
  if (parts.length < 2) return null;

  const domPart = parts[1];
  const commaParts = domPart.split(",");
  if (commaParts.length < 3) return null;

  const lcaPart = commaParts[0];
  const startPart = commaParts[1];
  const endPart = commaParts[2];

  // Construct absolute paths for start and end
  const startFull = lcaPart + (startPart.startsWith("/") ? "" : "/") + startPart;
  const endFull = lcaPart + (endPart.startsWith("/") ? "" : "/") + endPart;

  const resolvedStart = resolveCfi(container, `epubcfi(${parts[0]}!${startFull})`);
  const resolvedEnd = resolveCfi(container, `epubcfi(${parts[0]}!${endFull})`);

  if (!resolvedStart || !resolvedEnd) return null;

  try {
    const range = document.createRange();
    range.setStart(resolvedStart.node, resolvedStart.offset);
    range.setEnd(resolvedEnd.node, resolvedEnd.offset);
    return range;
  } catch (e) {
    console.error("Failed to construct range from CFI parts", e);
    return null;
  }
}
