import { api } from "@/config/api";
import type {
  CommonResponse,
  CursorPaginatedResponse,
  ImportCBLResult,
  ReadList,
  ReadListBook,
  ReadListNext,
} from "@/types";
import axios from "axios";

export const readListService = {
  getReadLists: async (
    cursor?: string,
    limit?: number,
  ): Promise<CursorPaginatedResponse<ReadList>> => {
    try {
      const params = new URLSearchParams();
      if (cursor) params.append("cursor", cursor);
      if (limit) params.append("limit", limit.toString());
      const query = params.toString();
      const res = await api.get(`/read-lists${query ? "?" + query : ""}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CursorPaginatedResponse<ReadList>;
      throw error;
    }
  },

  getReadList: async (id: string): Promise<CommonResponse<ReadList>> => {
    try {
      const res = await api.get(`/read-lists/${encodeURIComponent(id)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<ReadList>;
      throw error;
    }
  },

  createReadList: async (
    name: string,
    description?: string,
  ): Promise<CommonResponse<ReadList>> => {
    try {
      const res = await api.post("/read-lists", { name, description });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<ReadList>;
      throw error;
    }
  },

  updateReadList: async (
    id: string,
    name: string,
    description?: string,
  ): Promise<CommonResponse<ReadList>> => {
    try {
      const res = await api.put(`/read-lists/${encodeURIComponent(id)}`, {
        name,
        description,
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<ReadList>;
      throw error;
    }
  },

  deleteReadList: async (id: string): Promise<CommonResponse<null>> => {
    try {
      const res = await api.delete(`/read-lists/${encodeURIComponent(id)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<null>;
      throw error;
    }
  },

  getReadListBooks: async (
    id: string,
  ): Promise<CommonResponse<ReadListBook[]>> => {
    try {
      const res = await api.get(`/read-lists/${encodeURIComponent(id)}/books`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<ReadListBook[]>;
      throw error;
    }
  },

  addBook: async (
    id: string,
    book_id: string,
  ): Promise<CommonResponse<null>> => {
    try {
      const res = await api.post(
        `/read-lists/${encodeURIComponent(id)}/books`,
        { book_id },
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<null>;
      throw error;
    }
  },

  removeBook: async (
    id: string,
    book_id: string,
  ): Promise<CommonResponse<null>> => {
    try {
      const res = await api.delete(
        `/read-lists/${encodeURIComponent(id)}/books/${encodeURIComponent(book_id)}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<null>;
      throw error;
    }
  },

  reorder: async (
    id: string,
    book_ids: string[],
  ): Promise<CommonResponse<null>> => {
    try {
      const res = await api.put(`/read-lists/${encodeURIComponent(id)}/order`, {
        book_ids,
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<null>;
      throw error;
    }
  },

  nextInOrder: async (
    id: string,
    after?: string,
  ): Promise<CommonResponse<ReadListNext>> => {
    try {
      const params = new URLSearchParams();
      if (after) params.append("after", after);
      const query = params.toString();
      const res = await api.get(
        `/read-lists/${encodeURIComponent(id)}/next${query ? "?" + query : ""}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<ReadListNext>;
      throw error;
    }
  },

  importCBL: async (file: File): Promise<CommonResponse<ImportCBLResult>> => {
    try {
      const formData = new FormData();
      formData.append("file", file);
      const res = await api.post("/read-lists/import", formData, {
        headers: { "Content-Type": "multipart/form-data" },
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<ImportCBLResult>;
      throw error;
    }
  },
};
