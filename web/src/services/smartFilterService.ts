import { api } from "@/config/api";
import { SmartFilter, UpsertSmartFilterPayload, ReorderHomeShelfItem, Book } from "@/types";
import { CommonResponse } from "@/types/common";
import axios from "axios";

export const smartFilterService = {
  list: async (): Promise<CommonResponse<SmartFilter[]>> => {
    try {
      const { data } = await api.get<CommonResponse<SmartFilter[]>>("/smart-filters");
      return data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<SmartFilter[]>;
      throw error;
    }
  },

  get: async (id: string): Promise<CommonResponse<SmartFilter>> => {
    try {
      const { data } = await api.get<CommonResponse<SmartFilter>>(`/smart-filters/${encodeURIComponent(id)}`);
      return data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<SmartFilter>;
      throw error;
    }
  },

  create: async (payload: UpsertSmartFilterPayload): Promise<CommonResponse<SmartFilter>> => {
    try {
      const { data } = await api.post<CommonResponse<SmartFilter>>("/smart-filters", payload);
      return data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<SmartFilter>;
      throw error;
    }
  },

  update: async (id: string, payload: UpsertSmartFilterPayload): Promise<CommonResponse<SmartFilter>> => {
    try {
      const { data } = await api.put<CommonResponse<SmartFilter>>(`/smart-filters/${encodeURIComponent(id)}`, payload);
      return data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<SmartFilter>;
      throw error;
    }
  },

  delete: async (id: string): Promise<CommonResponse<null>> => {
    try {
      const { data } = await api.delete<CommonResponse<null>>(`/smart-filters/${encodeURIComponent(id)}`);
      return data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<null>;
      throw error;
    }
  },

  pinSidebar: async (id: string, isPinned: boolean): Promise<CommonResponse<SmartFilter>> => {
    try {
      const { data } = await api.put<CommonResponse<SmartFilter>>(`/smart-filters/${encodeURIComponent(id)}/pin-sidebar`, { is_pinned: isPinned });
      return data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<SmartFilter>;
      throw error;
    }
  },

  pinHome: async (id: string, isPinned: boolean): Promise<CommonResponse<SmartFilter>> => {
    try {
      const { data } = await api.put<CommonResponse<SmartFilter>>(`/smart-filters/${encodeURIComponent(id)}/pin-home`, { is_pinned: isPinned });
      return data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<SmartFilter>;
      throw error;
    }
  },

  reorderHome: async (shelves: ReorderHomeShelfItem[]): Promise<CommonResponse<null>> => {
    try {
      const { data } = await api.put<CommonResponse<null>>("/smart-filters/reorder-home", { shelves });
      return data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<null>;
      throw error;
    }
  },

  getBooks: async (
    id: string,
    params?: { library_id?: string; cursor?: string; limit?: number }
  ): Promise<{ status: boolean; data: Book[]; next_cursor?: string }> => {
    try {
      const { data } = await api.get<{ status: boolean; data: Book[]; next_cursor?: string }>(`/smart-filters/${encodeURIComponent(id)}/books`, { params });
      return data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as { status: boolean; data: Book[]; next_cursor?: string };
      throw error;
    }
  },
};
