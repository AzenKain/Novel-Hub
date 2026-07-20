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

export function metadataInitial(name: string) {
  const first = name.trim().charAt(0).toUpperCase();
  if (!first) return "#";
  return /[A-ZĐ]/.test(first) ? first : "#";
}

export function filterMetadataItems(
  items: MetadataCount[],
  query: string,
  alpha: string,
  sort: "name-asc" | "name-desc" | "count-desc",
) {
  const normalizedQuery = query.trim().toLowerCase();
  return [...items]
    .filter((item) => {
      const matchesQuery =
        !normalizedQuery || item.name.toLowerCase().includes(normalizedQuery);
      const matchesAlpha =
        alpha === "All" || metadataInitial(item.name) === alpha;
      return matchesQuery && matchesAlpha;
    })
    .sort((a, b) => {
      if (sort === "count-desc") {
        return b.bookCount - a.bookCount || a.name.localeCompare(b.name);
      }
      const result = a.name.localeCompare(b.name, undefined, {
        sensitivity: "base",
      });
      return sort === "name-desc" ? -result : result;
    });
}
