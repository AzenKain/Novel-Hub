import DOMPurify from "dompurify";

// True when a chapter is only a table of contents / nav page: used to pick a
// non-TOC landing chapter when a book is opened without saved progress.
export const isNavOnlyChapter = (title?: string | null) => {
  if (!title) return false;
  return /^\s*(mục lục|table of contents|contents|目次|목차)\s*$/i.test(title.trim());
};

export const sanitizeReaderHtml = (html: string) => {
  const normalized = (html || "").normalize("NFC");
  const stripped = normalized
    .replace(/<title\b[^>]*>[\s\S]*?<\/title>/gi, "")
    .replace(/<meta\b[^>]*>/gi, "")
    .replace(/<\/?(?:html|head|body)\b[^>]*>/gi, "")
    .replace(/<img\b(?![^>]*\bsrc=["'][^"']+)[\s\S]*?>/gi, "")
    .replace(/<img\b[^>]*\bsrc=["']\s*(?:#|about:blank)?\s*["'][^>]*>/gi, "")
    .replace(/<a\b[^>]*>\s*<\/a>/gi, "")
    .replace(/<p\b[^>]*>\s*<\/p>/gi, "");

  const clean = DOMPurify.sanitize(stripped, {
    USE_PROFILES: { html: true, svg: true },
    ADD_TAGS: ["svg", "image", "g", "use", "figure", "figcaption", "video", "audio", "source", "track"],
    ADD_ATTR: ["target", "xlink:href", "href", "src", "viewBox", "preserveAspectRatio", "width", "height", "class", "align", "controls", "preload", "poster", "type", "muted", "loop", "playsinline", "style", "hidden"],
    FORBID_TAGS: ["script", "style", "iframe", "object", "embed", "base", "frame", "frameset", "math", "link"],
    FORBID_ATTR: ["srcdoc"],
    ALLOWED_URI_REGEXP: /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|cid|xmpp|blob|data):|[^a-z]|[a-z+.-]+(?:[^a-z+.-:]|$))/i,
  });

  if (typeof document !== "undefined") {
    const doc = new DOMParser().parseFromString(`<body>${clean}</body>`, "text/html");

    // Remove explicitly hidden elements (e.g. hidden duplicate titles, hidden watermarks/copyright)
    doc.querySelectorAll("[hidden], [style*='display: none'], [style*='display:none'], [style*='visibility: hidden'], [style*='visibility:hidden'], .d-none, .hidden, .invisible").forEach((el) => {
      el.remove();
    });

    doc.querySelectorAll('[align="justify" i], [style*="justify" i]').forEach((el) => {
      el.setAttribute("align", "left");
      const style = el.getAttribute("style");
      if (style) {
        const cleaned = style
          .replace(/text-align\s*:\s*justify\s*;?/gi, "")
          .replace(/text-justify\s*:\s*[^;]+;?/gi, "")
          .replace(/^\s*;\s*/, "")
          .trim();
        el.setAttribute("style", cleaned);
      }
    });

    const imgs = doc.querySelectorAll("img");
    imgs.forEach((img) => {
      let highestOnlyChild: HTMLElement | null = null;
      let current = img.parentElement;
      while (current && current.tagName.toLowerCase() !== "body" && current.tagName.toLowerCase() !== "html") {
        const text = (current.textContent || "").trim();
        const imgCount = current.querySelectorAll("img").length;
        if (imgCount === 1 && text === "") {
          highestOnlyChild = current;
        }
        current = current.parentElement;
      }

      if (highestOnlyChild) {
        let node: HTMLElement | null = img.parentElement;
        while (node && node !== highestOnlyChild.parentElement) {
          node.classList.add("reader-image-page");
          Array.from(node.childNodes).forEach((child) => {
            if (child.nodeType === Node.TEXT_NODE && (child.textContent || "").trim() === "") {
              child.remove();
            } else if (child.nodeType === Node.ELEMENT_NODE && (child as HTMLElement).tagName.toLowerCase() === "br") {
              child.remove();
            }
          });
          node = node.parentElement;
        }
      } else {
        const parent = img.parentElement;
        if (parent && parent.tagName.toLowerCase() === "body") {
          const figure = doc.createElement("figure");
          figure.className = "reader-image-page";
          parent.insertBefore(figure, img);
          figure.appendChild(img);
        }
      }
    });

    return doc.body.innerHTML;
  }

  return clean;
};

