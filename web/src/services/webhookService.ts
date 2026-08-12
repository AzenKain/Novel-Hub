import { api } from "@/config/api";
import type { CommonResponse, CreateWebhookInput, PaginatedResponse, Webhook } from "@/types";
import axios from "axios";

export const webhookService = {
  async listWebhooks(limit?: number, offset?: number): Promise<PaginatedResponse<Webhook>> {
    try {
      const res = await api.get("/admin/webhooks", {
        params: { limit, offset },
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as PaginatedResponse<Webhook>;
      }
      throw error;
    }
  },

  async createWebhook(input: CreateWebhookInput): Promise<CommonResponse<Webhook>> {
    try {
      const res = await api.post("/admin/webhooks", input);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Webhook>;
      }
      throw error;
    }
  },

  async updateWebhook(id: string, input: CreateWebhookInput): Promise<CommonResponse<Webhook>> {
    try {
      const res = await api.put(`/admin/webhooks/${id}`, input);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Webhook>;
      }
      throw error;
    }
  },

  async deleteWebhook(id: string): Promise<CommonResponse<void>> {
    try {
      const res = await api.delete(`/admin/webhooks/${id}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  },

  async testPingWebhook(id: string): Promise<CommonResponse<void>> {
    try {
      const res = await api.post(`/admin/webhooks/${id}/test`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  },
};
