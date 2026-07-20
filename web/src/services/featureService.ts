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
  CursorPaginatedResponse,
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

  getCollections: async (
    cursor?: string,
    limit?: number,
  ): Promise<CommonResponse<Collection[]>> => {
    try {
      const params = new URLSearchParams();
      if (cursor) params.append("cursor", cursor);
      if (limit) params.append("limit", limit.toString());
      const query = params.toString();
      const res = await api.get(`/collections${query ? "?" + query : ""}`);
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

  getRecentReadingHistory: async (
    cursor?: string,
    limit?: number,
  ): Promise<CursorPaginatedResponse<ReadingHistory>> => {
    try {
      const params = new URLSearchParams();
      if (cursor) params.append("cursor", cursor);
      if (limit) params.append("limit", limit.toString());
      const query = params.toString();
      const res = await api.get(`/reader/history${query ? "?" + query : ""}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CursorPaginatedResponse<ReadingHistory>;
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
    cursor?: string,
    limit = 20,
  ): Promise<CursorPaginatedResponse<BookReview>> => {
    try {
      const params = new URLSearchParams({ limit: limit.toString() });
      if (cursor) params.append("cursor", cursor);
      const res = await api.get(
        `/books/${encodeURIComponent(bookId)}/reviews?${params.toString()}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CursorPaginatedResponse<BookReview>;
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
    cursor?: string,
    limit = 100,
  ): Promise<CursorPaginatedResponse<Book>> => {
    try {
      const params = new URLSearchParams({ limit: limit.toString() });
      if (cursor) params.append("cursor", cursor);
      const res = await api.get(
        `/bookmarks/books?${params.toString()}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CursorPaginatedResponse<Book>;
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

  adminDeleteBookReview: async (
    bookId: string,
    userId: number,
  ): Promise<CommonResponse<void>> => {
    try {
      const res = await api.delete(
        `/admin/reviews/${encodeURIComponent(bookId)}/${userId}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<void>;
      throw error;
    }
  },

  updateCollection: async (
    id: string,
    name: string,
  ): Promise<CommonResponse<Collection>> => {
    try {
      const res = await api.put(`/collections/${id}`, { name });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<Collection>;
      throw error;
    }
  },

  deleteCollection: async (id: string): Promise<CommonResponse<null>> => {
    try {
      const res = await api.delete(`/collections/${id}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<null>;
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

