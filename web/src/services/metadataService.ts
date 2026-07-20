import { api } from "@/config/api";
import type { CommonResponse, MetadataCount, OnlineMetadataResult } from "@/types";
import axios from "axios";

function cleanQuery(query: string): string {
  // Remove content in parenthesis or brackets
  let cleaned = query.replace(/\s*[([{}].*?[)\]}]\s*/g, "");
  // Remove volume/chapter numbers
  cleaned = cleaned.replace(/\s*(?:tập|vol(?:ume)?|quyển|chương|chuong)\b.*/gi, "");
  // Remove punctuation
  cleaned = cleaned.replace(/[^\p{L}\p{N}\s]/gu, "");
  return cleaned.trim();
}

async function searchGoogleBooks(query: string): Promise<OnlineMetadataResult[]> {
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
      coverImage: cover,
    } as OnlineMetadataResult;
  });
}

async function searchOpenLibrary(query: string): Promise<OnlineMetadataResult[]> {
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
      coverImage: cover,
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

  const res = await axios.post("https://graphql.anilist.co", {
    query: graphqlQuery,
    variables: { search: query }
  }, {
    headers: { "Content-Type": "application/json", "Accept": "application/json" }
  });
  
  const data = res.data;
  const media = data.data?.Page?.media || [];

  return media.map((m: any) => {
    const title = m.title.romaji || m.title.english || m.title.native;
    let author = "";
    if (m.staff?.edges) {
      const authors = m.staff.edges
        .filter((e: any) => e.role && e.role.toLowerCase().includes("story"))
        .map((e: any) => e.node?.name?.full);
      if (authors.length > 0) author = authors.join(", ");
    }
    return {
      title,
      creator: author,
      description: m.description ? m.description.replace(/<[^>]*>?/gm, '') : "",
      coverImage: m.coverImage?.large || "",
      language: m.countryOfOrigin,
      subject: m.genres ? m.genres.join(", ") : "",
    } as OnlineMetadataResult;
  });
}

export const metadataService = {
  async listAuthors(): Promise<CommonResponse<MetadataCount[]>> {
    const res = await api.get("/metadata/authors");
    return res.data;
  },

  async listSeries(): Promise<CommonResponse<MetadataCount[]>> {
    const res = await api.get("/metadata/series");
    return res.data;
  },

  async listPublishers(): Promise<CommonResponse<MetadataCount[]>> {
    const res = await api.get("/metadata/publishers");
    return res.data;
  },

  async listLanguages(): Promise<CommonResponse<MetadataCount[]>> {
    const res = await api.get("/metadata/languages");
    return res.data;
  },

  async listTags(): Promise<CommonResponse<MetadataCount[]>> {
    const res = await api.get("/metadata/tags");
    return res.data;
  },

  async listFormats(): Promise<CommonResponse<MetadataCount[]>> {
    const res = await api.get("/metadata/formats");
    return res.data;
  },

  async searchOnline(query: string, source: string): Promise<OnlineMetadataResult[]> {
    if (!query) return [];

    switch (source) {
      case "google":
        return searchGoogleBooks(query);
      case "anilist":
        return searchAniList(query);
      case "openlibrary":
        return searchOpenLibrary(query);
      default: {
        // Fallback strategy: try anilist first, then google books, then open library
        try {
          const aniListResults = await searchAniList(query);
          if (aniListResults.length > 0) return aniListResults;
        } catch (e) {
          console.warn("AniList search failed", e);
        }

        try {
          const gbResults = await searchGoogleBooks(query);
          if (gbResults.length > 0) return gbResults;
        } catch (e) {
          console.warn("Google Books search failed", e);
        }

        try {
          const olResults = await searchOpenLibrary(cleanQuery(query));
          if (olResults.length > 0) return olResults;
        } catch (e) {
          console.warn("OpenLibrary search failed", e);
        }
        
        return [];
      }
    }
  }
};
