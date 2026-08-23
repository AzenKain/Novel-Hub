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
    ADD_TAGS: ["svg", "image", "g", "use", "figure", "figcaption", "video", "audio", "source", "track"],
    ADD_ATTR: ["target", "xlink:href", "href", "src", "viewBox", "preserveAspectRatio", "width", "height", "class", "align", "controls", "preload", "poster", "type", "muted", "loop", "playsinline"],
    FORBID_TAGS: ["script", "style", "iframe", "object", "embed", "base", "frame", "frameset", "math", "link"],
    FORBID_ATTR: ["srcdoc", "style"],
    ALLOWED_URI_REGEXP: /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|cid|xmpp|blob|data):|[^a-z]|[a-z+.-]+(?:[^a-z+.-:]|$))/i,
  });

  if (typeof document !== "undefined") {
    const doc = new DOMParser().parseFromString(`<body>${clean}</body>`, "text/html");
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

    // Auto-detect and tag chapter title paragraphs
    const CHAPTER_TITLE_REGEX = /^(?:chương|chapter|ch\.|hồi|tiết|phần|tập|quyển|vol(?:ume)?\.?|mục|mở đầu|kết thúc|lời bạt|ngoại truyện|minh ho[aạ]|hình minh ho[aạ]|prologue|epilogue|afterword|interlude|side story|extra|bonus|act\b|scene\b|bài|thứ|đoạn|thông tin|giới thiệu|nhân vật)\b/i;
    let foundFirstText = false;
    doc.querySelectorAll<HTMLElement>("p, div, h1, h2, h3, h4, h5, h6").forEach((el) => {
      if (el.classList.contains("reader-image-page") || el.querySelector("img")) return;
      const text = (el.textContent || "").trim();
      if (!text) return;

      const isFirstParagraph = !foundFirstText;
      foundFirstText = true;

      const matchesTitlePattern = CHAPTER_TITLE_REGEX.test(text);
      const isShortHeadingCandidate = isFirstParagraph && text.length <= 65 && !/[.!?]["']?\s*$/.test(text);

      if (matchesTitlePattern || isShortHeadingCandidate || el.classList.contains("title") || el.classList.contains("title-top")) {
        el.classList.add("chapter-title");
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
