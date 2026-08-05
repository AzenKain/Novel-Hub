import { API_BASE, api } from "@/config/api";
import type {
  Book,
  BookFile,
  BookSeriesContext,
  Chapter,
  CursorPaginatedResponse,
  CommonResponse,
  DuplicateGroupResult,
  SearchBookParams,
  SearchDeepResult,
} from "@/types";
import axios from "axios";

export const bookService = {
  async getBooks(
    params: SearchBookParams = {},
  ): Promise<CursorPaginatedResponse<Book>> {
    const query = new URLSearchParams();
    if (params.cursor) query.append("cursor", params.cursor);
    if (params.limit) query.append("limit", params.limit.toString());
    if (params.search) query.append("search", params.search);
    if (params.library_id) query.append("library_id", params.library_id);
    if (params.nav) query.append("nav", params.nav);
    if (params.collection) query.append("collection", params.collection);
    if (params.chip) query.append("chip", params.chip);
    if (params.facet) query.append("facet", params.facet);
    if (params.facet_id) query.append("facet_id", params.facet_id);

    const qs = query.toString();
    try {
      const res = await api.get(`/books${qs ? `?${qs}` : ""}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CursorPaginatedResponse<Book>;
      throw error;
    }
  },

  async getBook(id: string): Promise<CommonResponse<Book>> {
    try {
      const res = await api.get(`/books/${id}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<Book>;
      throw error;
    }
  },

  async getBookSeries(id: string): Promise<CommonResponse<BookSeriesContext>> {
    try {
      const res = await api.get(`/books/${id}/series`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookSeriesContext>;
      throw error;
    }
  },

  getDownloadUrl(id: string, file_id?: string): string {
    const query = file_id ? `?file_id=${encodeURIComponent(file_id)}` : "";
    return `${API_BASE}/books/${id}/download${query}`;
  },

  async listFiles(id: string): Promise<CommonResponse<BookFile[]>> {
    try {
      const res = await api.get(`/books/${id}/files`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<BookFile[]>;
      throw error;
    }
  },

  async getChapters(book_id: string): Promise<CommonResponse<Chapter[]>> {
    try {
      const res = await api.get(`/books/${book_id}/chapters`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<Chapter[]>;
      throw error;
    }
  },

  async searchDeep(
    query: string,
    limit = 20,
    offset = 0,
  ): Promise<CommonResponse<SearchDeepResult[]>> {
    const qs = new URLSearchParams({
      q: query,
      limit: limit.toString(),
      offset: offset.toString(),
    }).toString();
    try {
      const res = await api.get(`/books/search/deep?${qs}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<SearchDeepResult[]>;
      throw error;
    }
  },

  async getDuplicates(): Promise<CommonResponse<DuplicateGroupResult[]>> {
    try {
      const res = await api.get(`/books/files/duplicates`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<DuplicateGroupResult[]>;
      throw error;
    }
  },

  async updateMetadata(
    id: string,
    data: {
      title: string;
      author?: string;
      description?: string;
      publisher?: string;
      language?: string;
      date?: string;
      subjects?: string[];
      series?: string;
      series_index?: string;
    },
  ): Promise<CommonResponse<void>> {
    try {
      const res = await api.put(`/books/${id}/metadata`, data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<void>;
      throw error;
    }
  },

  async listImages(book_id: string): Promise<CommonResponse<string[]>> {
    try {
      const res = await api.get(`/reader/${book_id}/images`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<string[]>;
      throw error;
    }
  },

  async updateCover(
    book_id: string,
    data: { cover?: File; cover_url?: string; epub_image_path?: string },
  ): Promise<CommonResponse<{ cover_url: string }>> {
    const formData = new FormData();
    if (data.cover) {
      formData.append("cover", data.cover);
    } else if (data.cover_url) {
      formData.append("cover_url", data.cover_url);
    } else if (data.epub_image_path) {
      formData.append("epub_image_path", data.epub_image_path);
    }

    try {
      const res = await api.post(`/reader/${book_id}/cover`, formData, {
        headers: {
          "Content-Type": "multipart/form-data",
        },
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<{ cover_url: string }>;
      throw error;
    }
  },

  async archiveBook(
    id: string,
    archived: boolean,
  ): Promise<CommonResponse<void>> {
    try {
      const res = await api.patch(`/books/${id}/archive`, { archived });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<void>;
      throw error;
    }
  },

  async deleteBook(id: string): Promise<CommonResponse<void>> {
    try {
      const res = await api.delete(`/books/${id}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<void>;
      throw error;
    }
  },

  async sendToEmail(id: string, recipientEmail: string): Promise<CommonResponse<void>> {
    try {
      const res = await api.post(`/books/${id}/send-email`, { recipient_email: recipientEmail });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<void>;
      throw error;
    }
  },

  async bulkDeleteBooks(bookIds: string[]): Promise<CommonResponse<import("@/types").BulkOperationResponse>> {
    try {
      const res = await api.post(`/books/bulk-delete`, { book_ids: bookIds });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<import("@/types").BulkOperationResponse>;
      throw error;
    }
  },

  async bulkMoveBooks(bookIds: string[], targetLibraryId: string): Promise<CommonResponse<import("@/types").BulkOperationResponse>> {
    try {
      const res = await api.post(`/books/bulk-move`, { book_ids: bookIds, target_library_id: targetLibraryId });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<import("@/types").BulkOperationResponse>;
      throw error;
    }
  },

  async bulkAssignCollections(bookIds: string[], collectionIds: string[]): Promise<CommonResponse<import("@/types").BulkOperationResponse>> {
    try {
      const res = await api.post(`/books/bulk-assign-collections`, { book_ids: bookIds, collection_ids: collectionIds });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<import("@/types").BulkOperationResponse>;
      throw error;
    }
  },

  async bulkAddTags(bookIds: string[], tagNames: string[]): Promise<CommonResponse<import("@/types").BulkOperationResponse>> {
    try {
      const res = await api.post(`/books/bulk-add-tags`, { book_ids: bookIds, tag_names: tagNames });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<import("@/types").BulkOperationResponse>;
      throw error;
    }
  },
};
