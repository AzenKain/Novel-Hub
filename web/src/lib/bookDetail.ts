import type { BookEngagementStats, BookFile, MetadataJSON } from "@/types";

const READABLE_FORMATS = new Set([
  "epub",
  "mobi",
  "azw",
  "azw3",
  "amz",
  "pdf",
  "doc",
  "docx",
  "odt",
  "txt",
  "md",
  "markdown",
  "html",
  "htm",
  "rtf",
  "fb2",
  "fbz",
  "zip",
  "cbz",
  "cbr",
  "cbt",
  "cb7",
]);

export const emptyEngagement = (bookId: string): BookEngagementStats => ({
  bookId,
  socialStats: {
    bookId,
    bookmarkCount: 0,
    ratingCount: 0,
    averageRating: 0,
    shareCount: 0,
  },
  downloadStats: { bookId, totalDownloadCount: 0 },
  readStats: { bookId, totalOpenCount: 0, qualifiedReadCount: 0 },
});

export const parseMetadata = (metadataJson?: string): MetadataJSON => {
  if (!metadataJson) return {};
  try {
    return JSON.parse(metadataJson) as MetadataJSON;
  } catch {
    return {};
  }
};

export const getMetaContent = (meta: MetadataJSON, name: string) => {
  const found = meta.meta?.find((item) => (item.name || item.Name) === name);
  return found?.content || found?.Content || "";
};

export const toStringList = (value: unknown): string[] => {
  if (Array.isArray(value)) {
    return value.map((item) => String(item).trim()).filter(Boolean);
  }
  if (typeof value === "string") {
    return value
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);
  }
  return [];
};

export const fileNameFromPath = (path: string) =>
  path.split(/[\\/]/).pop() || path;

const truncateMiddle = (value: string, maxLength: number) => {
  if (value.length <= maxLength) return value;
  const keep = Math.max(4, Math.floor((maxLength - 3) / 2));
  return `${value.slice(0, keep)}...${value.slice(-keep)}`;
};

export const formatFileSize = (bytes: number) => {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );
  return `${(bytes / Math.pow(1024, index)).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
};

export const isReadableFile = (file: BookFile) => {
  const format = file.format?.toLowerCase();
  if (format && READABLE_FORMATS.has(format)) return true;
  const lowerPath = file.path.toLowerCase();
  return Array.from(READABLE_FORMATS).some((ext) =>
    lowerPath.endsWith(`.${ext}`),
  );
};

export const fileSelectLabel = (file: BookFile) => {
  const format = (file.format || "file").toUpperCase();
  return `${format} · ${formatFileSize(file.sizeBytes)} · ${truncateMiddle(
    fileNameFromPath(file.path),
    22,
  )}`;
};

export const metadataHref = (
  nav: string,
  facet: string,
  name: string,
  id?: string,
) => {
  const params = new URLSearchParams({ nav, facet, name });
  if (id) params.set("facet_id", id);
  return `/?${params.toString()}`;
};
