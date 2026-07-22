import { api } from "@/config/api";
import type { CommonResponse, CreateWebhookInput, Webhook } from "@/types";

export const webhookService = {
  async listWebhooks(): Promise<CommonResponse<Webhook[]>> {
    const res = await api.get("/admin/webhooks");
    return res.data;
  },

  async createWebhook(input: CreateWebhookInput): Promise<CommonResponse<Webhook>> {
    const res = await api.post("/admin/webhooks", input);
    return res.data;
  },

  async updateWebhook(id: string, input: CreateWebhookInput): Promise<CommonResponse<Webhook>> {
    const res = await api.put(`/admin/webhooks/${id}`, input);
    return res.data;
  },

  async deleteWebhook(id: string): Promise<CommonResponse<void>> {
    const res = await api.delete(`/admin/webhooks/${id}`);
    return res.data;
  },

  async testPingWebhook(id: string): Promise<CommonResponse<void>> {
    const res = await api.post(`/admin/webhooks/${id}/test`);
    return res.data;
  },
};
