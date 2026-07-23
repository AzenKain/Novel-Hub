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
