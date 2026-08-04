import type { MetadataCount } from "@/types";

export const metadataNavIds = [
  "tags",
  "series",
  "authors",
  "publishers",
  "languages",
  "formats",
];

export const alphabetFilters = [
  "All",
  "#",
  "A",
  "B",
  "C",
  "D",
  "E",
  "F",
  "G",
  "H",
  "I",
  "K",
  "L",
  "M",
  "N",
  "O",
  "P",
  "R",
  "S",
  "T",
  "U",
  "V",
  "Y",
  "Z",
  "Đ",
];

export function sortMetadataItems(
  items: MetadataCount[],
  sort: "name-asc" | "name-desc" | "count-desc",
) {
  return [...items].sort((a, b) => {
    if (sort === "count-desc") {
      return b.book_count - a.book_count || a.name.localeCompare(b.name);
    }
    const result = a.name.localeCompare(b.name, undefined, {
      sensitivity: "base",
    });
    return sort === "name-desc" ? -result : result;
  });
}
