import {
  useCreateWebhookMutation,
  useDeleteWebhookMutation,
  useTestWebhookMutation,
  useUpdateWebhookMutation,
  useWebhooksQuery,
} from "@/hooks";
import { useWebhookStore } from "@/stores";
import type { CreateWebhookInput } from "@/types";
import { CheckCircle2, Edit3, Globe, Plus, RefreshCw, Send, Shield, Trash2, XCircle } from "lucide-react";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

import { useShallow } from "zustand/react/shallow";

export const WebhooksTab: React.FC = () => {
  const { t } = useTranslation();

  const { data: webhooks = [], isLoading } = useWebhooksQuery();
  const createWebhookMutation = useCreateWebhookMutation();
  const updateWebhookMutation = useUpdateWebhookMutation();
  const deleteWebhookMutation = useDeleteWebhookMutation();
  const testWebhookMutation = useTestWebhookMutation();

  const {
    modalOpen,
    editingWebhook,
    openCreateModal,
    openEditModal,
    closeModal,
  } = useWebhookStore(useShallow((state) => ({
    modalOpen: state.modalOpen,
    editingWebhook: state.editingWebhook,
    openCreateModal: state.openCreateModal,
    openEditModal: state.openEditModal,
    closeModal: state.closeModal,
  })));

  const [form, setForm] = useState<CreateWebhookInput>({
    name: "",
    url: "",
    template_type: "generic",
    secret: "",
    custom_headers: "",
    events: ["book.created"],
    is_active: true,
  });

  useEffect(() => {
    if (editingWebhook) {
      setForm({
        name: editingWebhook.name,
        url: editingWebhook.url,
        template_type: editingWebhook.template_type,
        secret: editingWebhook.secret || "",
        custom_headers: editingWebhook.custom_headers || "",
        events: editingWebhook.events || ["book.created"],
        is_active: editingWebhook.is_active,
      });
    } else {
      setForm({
        name: "",
        url: "",
        template_type: "generic",
        secret: "",
        custom_headers: "",
        events: ["book.created"],
        is_active: true,
      });
    }
  }, [editingWebhook, modalOpen]);

  const handleSave = (e: React.SyntheticEvent) => {
    e.preventDefault();
    if (!form.name.trim() || !form.url.trim()) return;

    if (editingWebhook) {
      updateWebhookMutation.mutate(
        { id: editingWebhook.id, input: form },
        {
          onSuccess: () => {
            toast.success(t("admin.save", "Saved successfully"));
            closeModal();
          },
          onError: (err: any) => {
            toast.error(err?.message || t("error.unknown", "Failed to save webhook"));
          },
        }
      );
    } else {
      createWebhookMutation.mutate(form, {
        onSuccess: () => {
          toast.success(t("common.success", "Webhook created successfully"));
          closeModal();
        },
        onError: (err: any) => {
          toast.error(err?.message || t("error.unknown", "Failed to save webhook"));
        },
      });
    }
  };

  const handleDelete = (id: string) => {
    if (!confirm(t("review.confirm_delete", "Are you sure you want to delete this webhook?"))) return;
    deleteWebhookMutation.mutate(id, {
      onSuccess: () => toast.success(t("admin.deleted", "Deleted")),
      onError: (err: any) => toast.error(err?.message || t("error.unknown", "Failed to delete webhook")),
    });
  };

  const handleTestPing = (id: string) => {
    testWebhookMutation.mutate(id, {
      onSuccess: () => toast.success(t("common.success", "Test ping sent successfully!")),
      onError: (err: any) => toast.error(err?.message || t("error.unknown", "Webhook ping failed")),
    });
  };

  const toggleEvent = (evt: string) => {
    setForm((prev) => {
      const current = prev.events || [];
      if (current.includes(evt)) {
        return { ...prev, events: current.filter((e) => e !== evt) };
      } else {
        return { ...prev, events: [...current, evt] };
      }
    });
  };

  const getPlatformBadge = (type: string) => {
    switch (type) {
      case "discord":
        return <span className="badge badge-primary gap-1 font-medium">Discord</span>;
      case "telegram":
        return <span className="badge badge-info gap-1 font-medium">Telegram</span>;
      case "slack":
        return <span className="badge badge-warning gap-1 font-medium">Slack</span>;
      default:
        return <span className="badge badge-neutral gap-1 font-medium">Generic JSON</span>;
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-bold flex items-center gap-2">
            <Globe className="h-5 w-5 text-primary" />
            Webhooks Integration
          </h2>
          <p className="text-xs text-base-content/60">
            Dispatch real-time HTTP event notifications to Discord, Telegram, Slack, n8n, or custom servers.
          </p>
        </div>
        <button type="button" onClick={openCreateModal} className="btn btn-primary btn-sm gap-1.5">
          <Plus className="h-4 w-4" />
          {t("common.create", "Add Webhook")}
        </button>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-12">
          <RefreshCw className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : webhooks.length === 0 ? (
        <div className="bg-base-200 border border-base-300 rounded-xl p-8 text-center flex flex-col items-center gap-2">
          <Globe className="h-10 w-10 text-base-content/30" />
          <div className="font-semibold text-sm">{t("library.no_items", "No Webhooks Configured")}</div>
          <p className="text-xs text-base-content/60 max-w-sm">
            Add a webhook URL to receive instant alerts when new books are uploaded or reading goals are met.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3">
          {webhooks.map((wh) => (
            <div
              key={wh.id}
              className="bg-base-200 border border-base-300 p-4 rounded-xl flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4"
            >
              <div className="flex flex-col gap-1 min-w-0 flex-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="font-bold text-sm">{wh.name}</span>
                  {getPlatformBadge(wh.template_type)}
                  {wh.is_active ? (
                    <span className="badge badge-success badge-xs gap-1">
                      <CheckCircle2 className="h-3 w-3" /> Active
                    </span>
                  ) : (
                    <span className="badge badge-ghost badge-xs gap-1 opacity-50">
                      <XCircle className="h-3 w-3" /> Inactive
                    </span>
                  )}
                  {wh.secret && (
                    <span className="badge badge-outline badge-xs gap-1 text-xs">
                      <Shield className="h-3 w-3" /> HMAC
                    </span>
                  )}
                </div>
                <div className="text-xs text-base-content/70 font-mono truncate max-w-lg">{wh.url}</div>
                <div className="flex items-center gap-1.5 flex-wrap mt-1">
                  {wh.events.map((evt) => (
                    <span key={evt} className="bg-base-300 text-base-content/80 text-[10px] px-1.5 py-0.5 rounded font-mono">
                      {evt}
                    </span>
                  ))}
                </div>
              </div>

              <div className="flex items-center gap-2 shrink-0 self-end sm:self-center">
                <button
                  type="button"
                  onClick={() => handleTestPing(wh.id)}
                  disabled={testWebhookMutation.isPending && testWebhookMutation.variables === wh.id}
                  className="btn btn-outline btn-xs gap-1"
                >
                  <Send className="h-3.5 w-3.5" />
                  {testWebhookMutation.isPending && testWebhookMutation.variables === wh.id ? "Testing..." : "Test Ping"}
                </button>

                <button
                  type="button"
                  onClick={() => openEditModal(wh)}
                  className="btn btn-ghost btn-xs btn-square"
                >
                  <Edit3 className="h-4 w-4" />
                </button>

                <button
                  type="button"
                  onClick={() => handleDelete(wh.id)}
                  className="btn btn-ghost btn-xs btn-square text-error"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Modal Form */}
      {modalOpen && (
        <div className="modal modal-open">
          <div className="modal-box max-w-lg">
            <h3 className="font-bold text-lg mb-4">
              {editingWebhook ? t("admin.edit", "Edit Webhook") : t("common.create", "Add New Webhook")}
            </h3>
            <form onSubmit={handleSave} className="flex flex-col gap-4">
              <div>
                <label className="label text-xs font-bold">{t("admin.name", "Webhook Name")}</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Discord Library Channel"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  className="input input-bordered input-sm w-full"
                />
              </div>

              <div>
                <label className="label text-xs font-bold">Endpoint URL</label>
                <input
                  type="url"
                  required
                  placeholder="https://discord.com/api/webhooks/..."
                  value={form.url}
                  onChange={(e) => setForm({ ...form, url: e.target.value })}
                  className="input input-bordered input-sm w-full font-mono text-xs"
                />
              </div>

              <div>
                <label className="label text-xs font-bold">Payload Template Platform</label>
                <select
                  value={form.template_type}
                  onChange={(e) => setForm({ ...form, template_type: e.target.value as any })}
                  className="select select-bordered select-sm w-full font-medium"
                >
                  <option value="generic">Generic JSON (Custom / n8n / Zapier)</option>
                  <option value="discord">Discord Webhook Embed</option>
                  <option value="telegram">Telegram Bot HTML</option>
                  <option value="slack">Slack Block Kit</option>
                </select>
              </div>

              <div>
                <label className="label text-xs font-bold">Secret Key (HMAC SHA-256 Signature)</label>
                <input
                  type="text"
                  placeholder="Optional secret key for X-NovelHub-Signature header"
                  value={form.secret || ""}
                  onChange={(e) => setForm({ ...form, secret: e.target.value })}
                  className="input input-bordered input-sm w-full font-mono text-xs"
                />
              </div>

              <div>
                <label className="label text-xs font-bold">Subscribed Events</label>
                <div className="flex flex-wrap gap-2">
                  {["book.created", "reading.completed", "metadata.updated"].map((evt) => (
                    <label
                      key={evt}
                      className={`btn btn-xs cursor-pointer ${
                        form.events.includes(evt) ? "btn-primary" : "btn-outline"
                      }`}
                      onClick={() => toggleEvent(evt)}
                    >
                      {evt}
                    </label>
                  ))}
                </div>
              </div>

              <div className="form-control">
                <label className="label cursor-pointer justify-start gap-3">
                  <input
                    type="checkbox"
                    checked={form.is_active}
                    onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                    className="checkbox checkbox-primary checkbox-sm"
                  />
                  <span className="label-text font-medium text-xs">Enable Webhook</span>
                </label>
              </div>

              <div className="modal-action gap-2 mt-4">
                <button type="button" onClick={closeModal} className="btn btn-ghost btn-sm">
                  {t("common.cancel", "Cancel")}
                </button>
                <button
                  type="submit"
                  disabled={createWebhookMutation.isPending || updateWebhookMutation.isPending}
                  className="btn btn-primary btn-sm"
                >
                  {createWebhookMutation.isPending || updateWebhookMutation.isPending ? (
                    <span className="loading loading-spinner loading-xs"></span>
                  ) : (
                    t("common.save", "Save Webhook")
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
