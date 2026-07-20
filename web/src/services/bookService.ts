import { API_BASE, api } from "@/config/api";
import type {
  Book,
  BookFile,
  Chapter,
  CommonResponse,
  DuplicateFileResult,
  OnlineMetadataResult,
  SearchBookParams,
  SearchDeepResult,
} from "@/types";
import axios from "axios";

export const bookService = {
  async getBooks(
    params: SearchBookParams = {},
  ): Promise<CommonResponse<Book[]>> {
    const query = new URLSearchParams();
    if (params.page) query.append("page", params.page.toString());
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
        return error.response.data as CommonResponse<Book[]>;
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

  getDownloadUrl(id: string, fileId?: string): string {
    const query = fileId ? `?file_id=${encodeURIComponent(fileId)}` : "";
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

  async uploadFiles(
    id: string,
    formData: FormData,
  ): Promise<
    CommonResponse<{ uploaded: number; total: number; files: BookFile[] }>
  > {
    try {
      const res = await api.post(`/books/${id}/files`, formData, {
        headers: {
          "Content-Type": "multipart/form-data",
        },
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<{
          uploaded: number;
          total: number;
          files: BookFile[];
        }>;
      throw error;
    }
  },

  async getChapters(bookId: string): Promise<CommonResponse<Chapter[]>> {
    try {
      const res = await api.get(`/books/${bookId}/chapters`);
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

  async getDuplicates(): Promise<CommonResponse<DuplicateFileResult[]>> {
    try {
      const res = await api.get(`/books/files/duplicates`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<DuplicateFileResult[]>;
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
      seriesIndex?: string;
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

  async getOnlineMetadata(
    id: string,
    source: string = "fallback",
  ): Promise<CommonResponse<OnlineMetadataResult[]>> {
    try {
      const res = await api.get(
        `/books/${id}/metadata/online?source=${source}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<OnlineMetadataResult[]>;
      throw error;
    }
  },

  async listImages(bookId: string): Promise<CommonResponse<string[]>> {
    try {
      const res = await api.get(`/reader/${bookId}/images`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<string[]>;
      throw error;
    }
  },

  async updateCover(
    bookId: string,
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
      const res = await api.post(`/reader/${bookId}/cover`, formData, {
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
};
