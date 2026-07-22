import type { Webhook } from "@/types";
import { create } from "zustand";

interface WebhookState {
  modalOpen: boolean;
  editingWebhook: Webhook | null;

  openCreateModal: () => void;
  openEditModal: (webhook: Webhook) => void;
  closeModal: () => void;
}

export const useWebhookStore = create<WebhookState>((set) => ({
  modalOpen: false,
  editingWebhook: null,

  openCreateModal: () => set({ modalOpen: true, editingWebhook: null }),
  openEditModal: (webhook) => set({ modalOpen: true, editingWebhook: webhook }),
  closeModal: () => set({ modalOpen: false, editingWebhook: null }),
}));
