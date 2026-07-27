import DOMPurify from "dompurify";

export const sanitizeReaderHtml = (html: string) => {
  const stripped = html
    .replace(/<title\b[^>]*>[\s\S]*?<\/title>/gi, "")
    .replace(/<meta\b[^>]*>/gi, "")
    .replace(/<\/?(?:html|head|body)\b[^>]*>/gi, "");

  return DOMPurify.sanitize(stripped, {
    USE_PROFILES: { html: true },
    ADD_ATTR: ["target"],
    FORBID_TAGS: ["script", "style", "iframe", "object", "embed", "base", "frame", "frameset", "svg", "math", "link", "video", "audio", "source", "track"],
    FORBID_ATTR: ["srcdoc", "style"],
    ALLOW_UNKNOWN_PROTOCOLS: false,
  });
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

  return text.length <= 24;
};
