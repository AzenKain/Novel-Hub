import DOMPurify from "dompurify";

// True when a chapter is only a table of contents / nav page: used to pick a
// non-TOC landing chapter when a book is opened without saved progress.
export const isNavOnlyChapter = (title?: string | null) => {
  if (!title) return false;
  return /^\s*(mục lục|table of contents|contents|目次|목차)\s*$/i.test(title.trim());
};

export const sanitizeReaderHtml = (html: string) => {
  const stripped = html
    .replace(/<title\b[^>]*>[\s\S]*?<\/title>/gi, "")
    .replace(/<meta\b[^>]*>/gi, "")
    .replace(/<\/?(?:html|head|body)\b[^>]*>/gi, "")
    .replace(/<img\b(?![^>]*\bsrc=["'][^"']+)[\s\S]*?>/gi, "")
    .replace(/<img\b[^>]*\bsrc=["']\s*(?:#|about:blank)?\s*["'][^>]*>/gi, "")
    .replace(/<a\b[^>]*>\s*<\/a>/gi, "")
    .replace(/<p\b[^>]*>\s*<\/p>/gi, "");

  const clean = DOMPurify.sanitize(stripped, {
    USE_PROFILES: { html: true, svg: true },
    ADD_TAGS: ["svg", "image", "g", "use", "figure", "figcaption"],
    ADD_ATTR: ["target", "xlink:href", "href", "src", "viewBox", "preserveAspectRatio", "width", "height", "class"],
    FORBID_TAGS: ["script", "style", "iframe", "object", "embed", "base", "frame", "frameset", "math", "link", "video", "audio", "source", "track"],
    FORBID_ATTR: ["srcdoc", "style"],
    ALLOWED_URI_REGEXP: /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|cid|xmpp|blob|data):|[^a-z]|[a-z+.-]+(?:[^a-z+.-:]|$))/i,
  });

  if (typeof document !== "undefined") {
    const doc = new DOMParser().parseFromString(`<body>${clean}</body>`, "text/html");
    const imgs = doc.querySelectorAll("img");
    imgs.forEach((img) => {
      const parent = img.parentElement;
      if (parent && parent.tagName.toLowerCase() === "body") {
        const figure = doc.createElement("figure");
        figure.className = "reader-image-page";
        parent.insertBefore(figure, img);
        figure.appendChild(img);
      } else if (parent && (parent.tagName.toLowerCase() === "p" || parent.tagName.toLowerCase() === "div" || parent.tagName.toLowerCase() === "figure")) {
        const isOnlyChild = parent.children.length === 1 && (!parent.textContent || parent.textContent.trim() === "");
        if (isOnlyChild) {
          parent.classList.add("reader-image-page");
        }
      }
    });
    return doc.body.innerHTML;
  }

  return clean;
};


export const isVisualChapter = (html: string) => {
  const mediaCount = (html.match(/<(?:img|svg|picture|canvas)\b/gi) || [])
    .length;
  if (mediaCount === 0) return false;

  const text = html
    .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, "")
    .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, "")
    .replace(/<[^>]+>/g, "")
    .replace(/&nbsp;|&#160;/gi, " ")
    .replace(/\s+/g, "")
    .trim();

  return text.length <= 250;
};
