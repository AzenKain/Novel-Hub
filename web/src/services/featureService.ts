import { api } from "@/config/api";
import type {
  Book,
  BookDownloadStats,
  BookEngagementStats,
  Bookmark,
  BookRatingSummary,
  BookReadStats,
  BookReview,
  BookSocialStats,
  BookUserState,
  Collection,
  CommonResponse,
  LibraryStats,
  ReadingActivityResult,
  ReadingHistory,
  RecordReadingActivityPayload,
} from "@/types";
import axios from "axios";

const shareClientId = () => {
  const key = "novelhub_share_client_id";
  if (typeof window === "undefined") return "";
  const existing = window.localStorage.getItem(key);
  if (existing) return existing;
  const next =
    typeof crypto !== "undefined" && "randomUUID" in crypto
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  window.localStorage.setItem(key, next);
  return next;
};

export const featureService = {
  getLibraryStats: async (): Promise<CommonResponse<LibraryStats>> => {
    try {
      const res = await api.get("/library/stats");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<LibraryStats>;
      throw error;
    }
  },

  getCollections: async (): Promise<CommonResponse<Collection[]>> => {
    try {
      const res = await api.get("/collections");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<Collection[]>;
      throw error;
    }
  },

  createCollection: async (
    name: string,
  ): Promise<CommonResponse<Collection>> => {
    try {
      const res = await api.post("/collections", { name });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<Collection>;
      throw error;
    }
  },

  getRecentReadingHistory: async (): Promise<
    CommonResponse<ReadingHistory[]>
  > => {
    try {
      const res = await api.get("/reader/history");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<ReadingHistory[]>;
      throw error;
    }
  },

  recordReadingActivity: async (
    payload: RecordReadingActivityPayload,
  ): Promise<CommonResponse<ReadingActivityResult>> => {
    try {
      const res = await api.post("/reader/history", payload);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<ReadingActivityResult>;
      throw error;
    }
  },

  getBookReadStats: async (
    bookId: string,
  ): Promise<CommonResponse<BookReadStats>> => {
    try {
      const res = await api.get(`/reader/stats/${encodeURIComponent(bookId)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookReadStats>;
      throw error;
    }
  },

  getBookDownloadStats: async (
    bookId: string,
  ): Promise<CommonResponse<BookDownloadStats>> => {
    try {
      const res = await api.get(
        `/books/${encodeURIComponent(bookId)}/download-stats`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookDownloadStats>;
      throw error;
    }
  },

  getBookRatingSummary: async (
    bookId: string,
  ): Promise<CommonResponse<BookRatingSummary>> => {
    try {
      const res = await api.get(`/books/${encodeURIComponent(bookId)}/rating`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookRatingSummary>;
      throw error;
    }
  },

  getBookEngagementStats: async (
    bookId: string,
  ): Promise<CommonResponse<BookEngagementStats>> => {
    try {
      const res = await api.get(
        `/books/${encodeURIComponent(bookId)}/engagement`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookEngagementStats>;
      throw error;
    }
  },

  recordShare: async (
    bookId: string,
  ): Promise<CommonResponse<BookSocialStats>> => {
    try {
      const res = await api.post(`/books/${encodeURIComponent(bookId)}/share`, {
        clientId: shareClientId(),
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookSocialStats>;
      throw error;
    }
  },

  listBookReviews: async (
    bookId: string,
    limit = 20,
    offset = 0,
  ): Promise<CommonResponse<BookReview[]>> => {
    try {
      const res = await api.get(
        `/books/${encodeURIComponent(bookId)}/reviews?limit=${limit}&offset=${offset}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookReview[]>;
      throw error;
    }
  },

  getBookUserState: async (
    bookId: string,
  ): Promise<CommonResponse<BookUserState>> => {
    try {
      const res = await api.get(
        `/books/${encodeURIComponent(bookId)}/user-state`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookUserState>;
      throw error;
    }
  },

  setBookmark: async (
    bookId: string,
    bookmarked: boolean,
  ): Promise<CommonResponse<Bookmark | null>> => {
    try {
      const res = await api.put(`/bookmarks/${encodeURIComponent(bookId)}`, {
        bookmarked,
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<Bookmark | null>;
      throw error;
    }
  },

  getBookmarkedBooks: async (
    limit = 100,
    offset = 0,
  ): Promise<CommonResponse<Book[]>> => {
    try {
      const res = await api.get(
        `/bookmarks/books?limit=${limit}&offset=${offset}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<Book[]>;
      throw error;
    }
  },

  upsertBookReview: async (
    bookId: string,
    rating: number,
    review: string,
  ): Promise<CommonResponse<BookReview>> => {
    try {
      const res = await api.put(`/books/${encodeURIComponent(bookId)}/review`, {
        rating,
        review,
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookReview>;
      throw error;
    }
  },

  deleteBookReview: async (bookId: string): Promise<CommonResponse<void>> => {
    try {
      const res = await api.delete(
        `/books/${encodeURIComponent(bookId)}/review`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<void>;
      throw error;
    }
  },

  addBookToCollection: async (
    collectionId: string,
    bookId: string,
  ): Promise<CommonResponse<void>> => {
    try {
      const res = await api.post(`/collections/${encodeURIComponent(collectionId)}/books`, {
        bookId,
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<void>;
      throw error;
    }
  },

  removeBookFromCollection: async (
    collectionId: string,
    bookId: string,
  ): Promise<CommonResponse<void>> => {
    try {
      const res = await api.delete(
        `/collections/${encodeURIComponent(collectionId)}/books/${encodeURIComponent(bookId)}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<void>;
      throw error;
    }
  },
};

