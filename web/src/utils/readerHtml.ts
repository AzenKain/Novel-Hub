import DOMPurify from "dompurify";

export const sanitizeReaderHtml = (html: string) => {
  const stripped = html
    .replace(/<title\b[^>]*>[\s\S]*?<\/title>/gi, "")
    .replace(/<meta\b[^>]*>/gi, "")
    .replace(/<\/?(?:html|head|body)\b[^>]*>/gi, "");

  return DOMPurify.sanitize(stripped, {
    USE_PROFILES: { html: true, svg: true },
    ADD_TAGS: ["svg", "image", "g", "use"],
    ADD_ATTR: ["target", "xlink:href", "href", "src", "viewBox", "preserveAspectRatio", "width", "height"],
    FORBID_TAGS: ["script", "style", "iframe", "object", "embed", "base", "frame", "frameset", "math", "link", "video", "audio", "source", "track"],
    FORBID_ATTR: ["srcdoc", "style"],
    ALLOWED_URI_REGEXP: /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|cid|xmpp|blob|data):|[^a-z]|[a-z+.-]+(?:[^a-z+.-:]|$))/i,
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
