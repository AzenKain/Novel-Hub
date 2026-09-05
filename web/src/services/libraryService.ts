import { api } from "@/config/api";
import type { CommonResponse, Library } from "@/types";
import axios from "axios";

export const libraryService = {
  async getLibraries(): Promise<CommonResponse<Library[]>> {
    try {
      const res = await api.get(`/libraries`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<Library[]>;
      throw error;
    }
  },

  async createLibrary(data: {
    name: string;
  }): Promise<CommonResponse<Library>> {
    try {
      const res = await api.post(`/libraries`, data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<Library>;
      throw error;
    }
  },

  async updateLibrary(
    id: string,
    data: { name: string },
  ): Promise<CommonResponse<Library>> {
    try {
      const res = await api.put(`/libraries/${id}`, data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<Library>;
      throw error;
    }
  },

  async deleteLibrary(id: string): Promise<CommonResponse<null>> {
    try {
      const res = await api.delete(`/libraries/${id}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<null>;
      throw error;
    }
  },

  async setupLibraryInbox(id: string): Promise<CommonResponse<string>> {
    try {
      const res = await api.post(`/libraries/${id}/inbox/setup`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<string>;
      throw error;
    }
  },
};
