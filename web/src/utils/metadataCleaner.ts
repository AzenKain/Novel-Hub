const minorWords = new Set([
  "and",
  "or",
  "the",
  "of",
  "in",
  "on",
  "a",
  "an",
  "to",
  "for",
  "with",
  "at",
  "by",
  "from",
  "de",
  "la",
]);

export function toTitleCase(s: string): string {
  const words = s.trim().split(/\s+/);
  return words
    .map((w, i) => {
      const lower = w.toLowerCase();
      if (i > 0 && i < words.length - 1 && minorWords.has(lower)) {
        return lower;
      }
      return w.charAt(0).toUpperCase() + w.slice(1).toLowerCase();
    })
    .join(" ");
}

export function stripBrackets(s: string): string {
  return s
    .replace(/\[[^\]]*\]/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

export function stripParentheses(s: string): string {
  return s
    .replace(/\([^)]*\)/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

export function replaceUnderscores(s: string): string {
  return s.replace(/_+/g, " ").replace(/\s+/g, " ").trim();
}

export function splitDashAuthorTitle(
  title: string,
): { author: string; title: string } | null {
  const parts = title.split(/\s+-\s+/);
  if (parts.length >= 2) {
    return {
      author: parts[0].trim(),
      title: parts.slice(1).join(" - ").trim(),
    };
  }
  return null;
}

export function splitDashTitleAuthor(
  title: string,
): { title: string; author: string } | null {
  const parts = title.split(/\s+-\s+/);
  if (parts.length >= 2) {
    return {
      title: parts[0].trim(),
      author: parts.slice(1).join(" - ").trim(),
    };
  }
  return null;
}

export function splitTitleByAuthor(
  title: string,
): { title: string; author: string } | null {
  const m = title.match(/^(.*?)\s+by\s+(.*?)$/i);
  if (m) {
    return {
      title: m[1].trim(),
      author: m[2].trim(),
    };
  }
  return null;
}

export function cleanWhitespace(s: string): string {
  return s.replace(/\s+/g, " ").trim();
}

export function applyCustomRegex(
  text: string,
  pattern: string,
  replacement: string,
): string {
  if (!pattern) return text;
  const reg = new RegExp(pattern, "g");
  return text.replace(reg, replacement).replace(/\s+/g, " ").trim();
}
