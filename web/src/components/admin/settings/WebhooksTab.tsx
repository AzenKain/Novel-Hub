import {
  useCreateWebhookMutation,
  useDeleteWebhookMutation,
  useTestWebhookMutation,
  useUpdateWebhookMutation,
  useWebhooksQuery,
} from "@/hooks";
import { useWebhookStore } from "@/stores";
import type { CreateWebhookInput } from "@/types";
import {
  CheckCircle2,
  Edit3,
  Globe,
  Plus,
  RefreshCw,
  Send,
  Shield,
  Trash2,
  XCircle,
} from "lucide-react";
import React from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";
import { useShallow } from "zustand/react/shallow";
import { WebhookModal } from "./WebhookModal";
import { ConfirmModal } from "@/components/common";

import { useQueryClient } from "@tanstack/react-query";

export const WebhooksTab: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [page, setPage] = React.useState(1);
  const pageSize = 10;
  const offset = (page - 1) * pageSize;

  const { data, isLoading, isFetching, refetch } = useWebhooksQuery(
    pageSize,
    offset,
  );
  const webhooks = data?.webhooks || [];
  const totalPages = data?.totalPages || 1;

  const [deleteWebhookId, setDeleteWebhookId] = React.useState<string | null>(
    null,
  );
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
  } = useWebhookStore(
    useShallow((state) => ({
      modalOpen: state.modalOpen,
      editingWebhook: state.editingWebhook,
      openCreateModal: state.openCreateModal,
      openEditModal: state.openEditModal,
      closeModal: state.closeModal,
    })),
  );

  const handleSaveModal = (input: CreateWebhookInput) => {
    if (editingWebhook) {
      updateWebhookMutation.mutate(
        { id: editingWebhook.id, input },
        {
          onSuccess: () => {
            toast.success(t("admin.save", "Saved successfully"));
            closeModal();
          },
          onError: (err: any) => {
            toast.error(
              err?.message || t("error.unknown", "Failed to save webhook"),
            );
          },
        },
      );
    } else {
      createWebhookMutation.mutate(input, {
        onSuccess: () => {
          toast.success(t("common.success", "Webhook created successfully"));
          closeModal();
        },
        onError: (err: any) => {
          toast.error(
            err?.message || t("error.unknown", "Failed to save webhook"),
          );
        },
      });
    }
  };

  const handleDelete = (id: string) => {
    setDeleteWebhookId(id);
  };

  const handleConfirmDelete = () => {
    if (!deleteWebhookId) return;
    deleteWebhookMutation.mutate(deleteWebhookId, {
      onSuccess: () => {
        toast.success(t("admin.deleted", "Deleted"));
        setDeleteWebhookId(null);
      },
      onError: (err: any) => {
        toast.error(
          err?.message || t("error.unknown", "Failed to delete webhook"),
        );
      },
    });
  };

  const handleTestPing = (id: string) => {
    testWebhookMutation.mutate(id, {
      onSuccess: () =>
        toast.success(t("common.success", "Test ping sent successfully!")),
      onError: (err: any) =>
        toast.error(err?.message || t("error.unknown", "Webhook ping failed")),
    });
  };

  const getPlatformBadge = (type: string) => {
    switch (type) {
      case "discord":
        return (
          <span className="badge badge-primary gap-1 font-medium">Discord</span>
        );
      case "telegram":
        return (
          <span className="badge badge-info gap-1 font-medium">Telegram</span>
        );
      case "slack":
        return (
          <span className="badge badge-warning gap-1 font-medium">Slack</span>
        );
      default:
        return (
          <span className="badge badge-neutral gap-1 font-medium">
            {t("settings.webhook_generic_json")}
          </span>
        );
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 sm:gap-4">
        <div className="flex items-center justify-between gap-3 w-full sm:w-auto">
          <div className="min-w-0 flex-1">
            <h2 className="text-base sm:text-lg font-bold flex items-center gap-2">
              <Globe className="h-5 w-5 text-primary shrink-0" />
              <span className="truncate">
                {t("admin.webhooks", "Webhooks & Outbound Integrations")}
              </span>
            </h2>
            <p className="text-xs text-base-content/60 mt-0.5 line-clamp-1 sm:line-clamp-none">
              {t(
                "admin.webhooks_subtitle",
                "Configure real-time event notifications for Discord, Telegram, Slack, and automation webhooks.",
              )}
            </p>
          </div>
          <button
            onClick={async () => {
              await queryClient.invalidateQueries({
                queryKey: ["admin", "webhooks"],
              });
              await refetch();
              toast.info(t("common.refreshed", "Data refreshed"));
            }}
            className="btn btn-square btn-ghost btn-sm sm:hidden shrink-0"
            title={t("settings.refresh", "Refresh")}
            disabled={isFetching}
          >
            <RefreshCw
              className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`}
            />
          </button>
        </div>

        <div className="flex items-center gap-2 w-full sm:w-auto justify-end">
          <button
            onClick={async () => {
              await queryClient.invalidateQueries({
                queryKey: ["admin", "webhooks"],
              });
              await refetch();
              toast.info(t("common.refreshed", "Data refreshed"));
            }}
            className="btn btn-square btn-ghost btn-sm hidden sm:inline-flex shrink-0"
            title={t("settings.refresh", "Refresh")}
            disabled={isFetching}
          >
            <RefreshCw
              className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`}
            />
          </button>
          <button
            onClick={openCreateModal}
            className="btn btn-primary btn-sm gap-2 w-full sm:w-auto"
          >
            <Plus className="h-4 w-4" />
            {t("admin.add_webhook", "Add Webhook")}
          </button>
        </div>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-12">
          <span className="loading loading-spinner loading-md text-primary"></span>
        </div>
      ) : webhooks.length === 0 ? (
        <div className="bg-base-200 border border-base-300 rounded-xl p-8 text-center flex flex-col items-center gap-3">
          <Globe className="h-10 w-10 text-base-content/30" />
          <div className="font-semibold text-sm">
            {t("admin.no_webhooks", "No webhooks configured")}
          </div>
          <p className="text-xs text-base-content/60 max-w-sm">
            {t(
              "admin.no_webhooks_desc",
              "Connect your library to Discord channels or custom API endpoints to get notified about new book additions.",
            )}
          </p>
          <button
            onClick={openCreateModal}
            className="btn btn-primary btn-xs gap-1 mt-2"
          >
            <Plus className="h-3.5 w-3.5" />{" "}
            {t("admin.add_webhook", "Add Webhook")}
          </button>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {webhooks.map((wh) => (
            <div
              key={wh.id}
              className="flex flex-col sm:flex-row items-start sm:items-center justify-between p-4 bg-base-200/50 border border-base-300 rounded-xl gap-4 hover:border-base-300 transition-colors"
            >
              <div className="flex flex-col gap-1 min-w-0 flex-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="font-bold text-sm">{wh.name}</span>
                  {getPlatformBadge(wh.template_type)}
                  {wh.is_active ? (
                    <span className="badge badge-success badge-xs gap-1 text-[10px] font-medium">
                      <CheckCircle2 className="h-3 w-3" /> Active
                    </span>
                  ) : (
                    <span className="badge badge-ghost badge-xs gap-1 text-[10px] font-medium opacity-60">
                      <XCircle className="h-3 w-3" /> {t("common.disabled")}
                    </span>
                  )}
                  {wh.secret && (
                    <span
                      className="badge badge-outline badge-xs gap-1 text-[10px] font-mono"
                      title={t("settings.webhook_hmac_enabled")}
                    >
                      <Shield className="h-3 w-3" /> HMAC
                    </span>
                  )}
                </div>
                <div className="text-xs text-base-content/70 font-mono truncate max-w-lg">
                  {wh.url}
                </div>
              </div>

              <div className="flex items-center gap-2 shrink-0 self-end sm:self-center">
                <button
                  type="button"
                  onClick={() => handleTestPing(wh.id)}
                  disabled={
                    testWebhookMutation.isPending &&
                    testWebhookMutation.variables === wh.id
                  }
                  className="btn btn-outline btn-xs gap-1"
                >
                  {testWebhookMutation.isPending &&
                  testWebhookMutation.variables === wh.id ? (
                    <RefreshCw className="h-3 w-3 animate-spin" />
                  ) : (
                    <Send className="h-3 w-3" />
                  )}
                  Test Ping
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
          {totalPages > 1 && (
            <div className="flex justify-end gap-2 mt-4 items-center">
              <button
                className="btn btn-sm btn-outline"
                disabled={page === 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                {t("common.previous", "Previous")}
              </button>
              <span className="text-xs font-semibold px-2 text-base-content/75">
                {t("admin.page_of", "Page {{page}} of {{totalPages}}", {
                  page,
                  totalPages,
                })}
              </span>
              <button
                className="btn btn-sm btn-outline"
                disabled={page === totalPages}
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              >
                {t("common.next", "Next")}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Modern Full Builder & Live Preview Modal */}
      <WebhookModal
        open={modalOpen}
        editingWebhook={editingWebhook}
        onClose={closeModal}
        onSave={handleSaveModal}
        isSaving={
          createWebhookMutation.isPending || updateWebhookMutation.isPending
        }
      />

      <ConfirmModal
        open={deleteWebhookId !== null}
        title={t("review.confirm_delete_title", "Delete Webhook")}
        message={t(
          "review.confirm_delete",
          "Are you sure you want to delete this webhook?",
        )}
        onClose={() => setDeleteWebhookId(null)}
        onConfirm={handleConfirmDelete}
        variant="danger"
        loading={deleteWebhookMutation.isPending}
      />
    </div>
  );
};
