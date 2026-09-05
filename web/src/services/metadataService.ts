import { api } from "@/config/api";
import type {
  CommonResponse,
  MetadataCount,
  MetadataFacetParams,
  OnlineMetadataResult,
  PaginatedResponse,
} from "@/types";
import axios from "axios";

function facetQuery(params?: MetadataFacetParams): string {
  const search = new URLSearchParams();
  if (params?.cursor) search.set("cursor", params.cursor);
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.search?.trim()) search.set("search", params.search.trim());
  if (params?.alpha && params.alpha !== "All")
    search.set("alpha", params.alpha);
  const query = search.toString();
  return query ? `?${query}` : "";
}

function cleanQuery(query: string): string {
  // Remove content in parenthesis or brackets
  let cleaned = query.replace(/\s*[([{}].*?[)\]}]\s*/g, "");
  // Remove volume/chapter numbers
  cleaned = cleaned.replace(
    /\s*(?:tập|vol(?:ume)?|quyển|chương|chuong)\b.*/gi,
    "",
  );
  // Remove punctuation
  cleaned = cleaned.replace(/[^\p{L}\p{N}\s]/gu, "");
  cleaned = cleaned.trim();
  return cleaned || query.trim();
}

async function searchGoogleBooks(
  query: string,
): Promise<OnlineMetadataResult[]> {
  try {
    const url = `https://www.googleapis.com/books/v1/volumes?q=${encodeURIComponent(query)}&maxResults=5`;
    const res = await axios.get(url);
    const data = res.data;
    if (!data.items) return [];

    return data.items.map((item: any) => {
      const vol = item.volumeInfo;
      let cover = "";
      if (vol.imageLinks) {
        cover = vol.imageLinks.thumbnail || vol.imageLinks.smallThumbnail || "";
        cover = cover.replace("http://", "https://");
      }
      return {
        title: vol.title,
        creator: vol.authors ? vol.authors.join(", ") : "",
        publisher: vol.publisher,
        language: vol.language,
        description: vol.description,
        cover_image: cover,
      } as OnlineMetadataResult;
    });
  } catch (err: any) {
    if (
      err?.response?.status === 429 ||
      err?.response?.data?.error?.status === "RESOURCE_EXHAUSTED"
    ) {
      throw new Error(
        "Google Books API quota exceeded for the day. Please select AniList or Open Library.",
      );
    }
    throw err;
  }
}

async function searchOpenLibrary(
  query: string,
): Promise<OnlineMetadataResult[]> {
  const url = `https://openlibrary.org/search.json?q=${encodeURIComponent(query)}&limit=5`;
  const res = await axios.get(url);
  const data = res.data;
  if (!data.docs) return [];

  return data.docs.map((doc: any) => {
    let cover = "";
    if (doc.cover_i) {
      cover = `https://covers.openlibrary.org/b/id/${doc.cover_i}-L.jpg`;
    }
    return {
      title: doc.title,
      creator: doc.author_name ? doc.author_name.join(", ") : "",
      publisher: doc.publisher ? doc.publisher[0] : "",
      language: doc.language ? doc.language[0] : "",
      cover_image: cover,
    } as OnlineMetadataResult;
  });
}

async function searchAniList(query: string): Promise<OnlineMetadataResult[]> {
  const graphqlQuery = `
  query ($search: String) {
    Page(page: 1, perPage: 10) {
      media(search: $search, type: MANGA) {
        title {
          romaji
          english
          native
        }
        format
        countryOfOrigin
        description
        coverImage {
          large
        }
        staff {
          edges {
            role
            node {
              name {
                full
              }
            }
          }
        }
        genres
      }
    }
  }
  `;

  const res = await axios.post(
    "https://graphql.anilist.co",
    {
      query: graphqlQuery,
      variables: { search: query },
    },
    {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
    },
  );

  const data = res.data;
  const media = data.data?.Page?.media || [];

  return media.map((m: any) => {
    const title = m.title.romaji || m.title.english || m.title.native;
    let author = "";
    if (m.staff?.edges) {
      const authors = m.staff.edges
        .filter((e: any) => {
          const role = e.role ? e.role.toLowerCase() : "";
          return (
            role.includes("story") ||
            role.includes("author") ||
            role.includes("writer") ||
            role.includes("original")
          );
        })
        .map((e: any) => e.node?.name?.full)
        .filter(Boolean);
      if (authors.length > 0) author = Array.from(new Set(authors)).join(", ");
    }
    return {
      title,
      creator: author,
      description: m.description ? m.description.replace(/<[^>]*>?/gm, "") : "",
      cover_image: m.coverImage?.large || "",
      language: m.countryOfOrigin,
      subject: m.genres ? m.genres.join(", ") : "",
    } as OnlineMetadataResult;
  });
}

export const metadataService = {
  async listAuthors(
    params?: MetadataFacetParams,
  ): Promise<PaginatedResponse<MetadataCount>> {
    const res = await api.get(`/metadata/authors${facetQuery(params)}`);
    return res.data;
  },

  async listSeries(
    params?: MetadataFacetParams,
  ): Promise<PaginatedResponse<MetadataCount>> {
    const res = await api.get(`/metadata/series${facetQuery(params)}`);
    return res.data;
  },

  async listPublishers(
    params?: MetadataFacetParams,
  ): Promise<PaginatedResponse<MetadataCount>> {
    const res = await api.get(`/metadata/publishers${facetQuery(params)}`);
    return res.data;
  },

  async listLanguages(
    params?: MetadataFacetParams,
  ): Promise<PaginatedResponse<MetadataCount>> {
    const res = await api.get(`/metadata/languages${facetQuery(params)}`);
    return res.data;
  },

  async listTags(
    params?: MetadataFacetParams,
  ): Promise<PaginatedResponse<MetadataCount>> {
    const res = await api.get(`/metadata/tags${facetQuery(params)}`);
    return res.data;
  },

  async listFormats(
    params?: MetadataFacetParams,
  ): Promise<PaginatedResponse<MetadataCount>> {
    const res = await api.get(`/metadata/formats${facetQuery(params)}`);
    return res.data;
  },

  async listRatings(
    params?: MetadataFacetParams,
  ): Promise<PaginatedResponse<MetadataCount>> {
    const res = await api.get(`/metadata/ratings${facetQuery(params)}`);
    return res.data;
  },

  async searchOnline(
    query: string,
    source: string,
  ): Promise<OnlineMetadataResult[]> {
    if (!query) return [];

    switch (source) {
      case "google":
        return searchGoogleBooks(query);
      case "anilist":
        return searchAniList(cleanQuery(query));
      case "openlibrary":
        return searchOpenLibrary(cleanQuery(query));
      default: {
        // Fallback strategy: try AniList first, then OpenLibrary, then Google Books
        try {
          const aniListResults = await searchAniList(cleanQuery(query));
          if (aniListResults.length > 0) return aniListResults;
        } catch (e) {
          console.warn("AniList search failed", e);
        }

        try {
          const olResults = await searchOpenLibrary(cleanQuery(query));
          if (olResults.length > 0) return olResults;
        } catch (e) {
          console.warn("OpenLibrary search failed", e);
        }

        try {
          const gbResults = await searchGoogleBooks(query);
          if (gbResults.length > 0) return gbResults;
        } catch (e) {
          console.warn("Google Books search failed", e);
        }

        return [];
      }
    }
  },
};
