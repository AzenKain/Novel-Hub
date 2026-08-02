import { api } from "@/config/api";
import type {
  Book,
  BookEngagementStats,
  Bookmark,
  BookReview,
  BookUserState,
  Collection,
  CommonResponse,
  CursorPaginatedResponse,
  LibraryStats,
  ReadingActivityResult,
  ReadingHistory,
  RecordReadingActivityPayload,
  SmartCollection,
  SmartCollectionRule,
} from "@/types";
import axios from "axios";

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

  getReadingProgress: async (
    book_id: string,
  ): Promise<CommonResponse<ReadingHistory>> => {
    try {
      const res = await api.get(`/reader/history/progress/${book_id}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<ReadingHistory>;
      throw error;
    }
  },

  getBookEngagementStats: async (
    book_id: string,
  ): Promise<CommonResponse<BookEngagementStats>> => {
    try {
      const res = await api.get(
        `/books/${encodeURIComponent(book_id)}/engagement`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookEngagementStats>;
      throw error;
    }
  },

  listBookReviews: async (
    book_id: string,
    cursor?: string,
    limit = 20,
  ): Promise<CursorPaginatedResponse<BookReview>> => {
    try {
      const params = new URLSearchParams({ limit: limit.toString() });
      if (cursor) params.append("cursor", cursor);
      const res = await api.get(
        `/books/${encodeURIComponent(book_id)}/reviews?${params.toString()}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CursorPaginatedResponse<BookReview>;
      throw error;
    }
  },

  getBookUserState: async (
    book_id: string,
  ): Promise<CommonResponse<BookUserState>> => {
    try {
      const res = await api.get(
        `/books/${encodeURIComponent(book_id)}/user-state`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookUserState>;
      throw error;
    }
  },

  setBookmark: async (
    book_id: string,
    bookmarked: boolean,
  ): Promise<CommonResponse<Bookmark | null>> => {
    try {
      const res = await api.put(`/bookmarks/${encodeURIComponent(book_id)}`, {
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
    book_id: string,
    rating: number,
    review: string,
  ): Promise<CommonResponse<BookReview>> => {
    try {
      const res = await api.put(`/books/${encodeURIComponent(book_id)}/review`, {
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

  deleteBookReview: async (book_id: string): Promise<CommonResponse<void>> => {
    try {
      const res = await api.delete(
        `/books/${encodeURIComponent(book_id)}/review`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<void>;
      throw error;
    }
  },

  adminDeleteBookReview: async (
    book_id: string,
    user_id: string,
  ): Promise<CommonResponse<void>> => {
    try {
      const res = await api.delete(
        `/admin/reviews/${encodeURIComponent(book_id)}/${user_id}`,
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

  getSmartCollections: async (): Promise<CommonResponse<SmartCollection[]>> => {
    try {
      const res = await api.get("/smart-collections/");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<SmartCollection[]>;
      throw error;
    }
  },

  createSmartCollection: async (
    name: string,
    rule: SmartCollectionRule,
  ): Promise<CommonResponse<SmartCollection>> => {
    try {
      const res = await api.post("/smart-collections/", { name, rule });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<SmartCollection>;
      throw error;
    }
  },

  updateSmartCollection: async (
    id: string,
    name: string,
    rule: SmartCollectionRule,
  ): Promise<CommonResponse<SmartCollection>> => {
    try {
      const res = await api.put(`/smart-collections/${encodeURIComponent(id)}`, { name, rule });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<SmartCollection>;
      throw error;
    }
  },

  deleteSmartCollection: async (id: string): Promise<CommonResponse<null>> => {
    try {
      const res = await api.delete(`/smart-collections/${encodeURIComponent(id)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<null>;
      throw error;
    }
  },

  addBookToCollection: async (
    collectionId: string,
    book_id: string,
  ): Promise<CommonResponse<void>> => {
    try {
      const res = await api.post(`/collections/${encodeURIComponent(collectionId)}/books`, {
        book_id,
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
    book_id: string,
  ): Promise<CommonResponse<void>> => {
    try {
      const res = await api.delete(
        `/collections/${encodeURIComponent(collectionId)}/books/${encodeURIComponent(book_id)}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<void>;
      throw error;
    }
  },
};

