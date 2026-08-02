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

export const emptyEngagement = (book_id: string): BookEngagementStats => ({
  book_id,
  social_stats: {
    book_id,
    bookmark_count: 0,
    rating_count: 0,
    average_rating: 0,
    share_count: 0,
  },
  download_stats: { book_id, total_download_count: 0 },
  read_stats: { book_id, total_open_count: 0, qualified_read_count: 0 },
});

export const parseMetadata = (metadata_json?: string): MetadataJSON => {
  if (!metadata_json) return {};
  try {
    return JSON.parse(metadata_json) as MetadataJSON;
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

export const formatUploadSpeed = (bytesPerSec: number) => {
  if (!Number.isFinite(bytesPerSec) || bytesPerSec <= 0) return "0 B/s";
  const units = ["B/s", "KB/s", "MB/s", "GB/s"];
  const index = Math.min(
    Math.floor(Math.log(bytesPerSec) / Math.log(1024)),
    units.length - 1,
  );
  return `${(bytesPerSec / Math.pow(1024, index)).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
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
  return `${format} · ${formatFileSize(file.size_bytes)} · ${truncateMiddle(
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
