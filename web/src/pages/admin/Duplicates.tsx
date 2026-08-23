import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Loader2, Copy, CheckCircle, Trash2, ShieldCheck, HardDrive, CheckSquare, Square, RefreshCw } from "lucide-react";
import { useDuplicatesQuery, useDeleteBookFileMutation } from "@/hooks";
import { getMediaUrl } from "@/config/api";
import { toast } from "react-toastify";
import { DeleteConfirmModal } from "@/components/admin";
import { useQueryClient } from "@tanstack/react-query";

type ConfirmState =
  | { type: "single"; file_id: string; title: string }
  | { type: "selected"; file_ids: string[] }
  | { type: "keepOne"; keepFileId: string; toDeleteFileIds: string[] }
  | null;

export const Duplicates: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { data: duplicateGroups = [], isLoading: loading, isFetching, refetch } = useDuplicatesQuery();
  const deleteFileMutation = useDeleteBookFileMutation();
  const [selectedFileIds, setSelectedFileIds] = useState<Set<string>>(new Set());
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [confirmState, setConfirmState] = useState<ConfirmState>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const formatSize = (bytes: number) => {
    if (!bytes) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  const totalFiles = duplicateGroups.reduce((sum, g) => sum + (g.files?.length || 0), 0);
  const wastedBytes = duplicateGroups.reduce((sum, g) => {
    if (!g.files || g.files.length <= 1) return sum;
    const singleSize = g.files[0]?.size_bytes || 0;
    return sum + singleSize * (g.files.length - 1);
  }, 0);

  const toggleSelectFile = (file_id: string) => {
    setSelectedFileIds((prev) => {
      const next = new Set(prev);
      if (next.has(file_id)) next.delete(file_id);
      else next.add(file_id);
      return next;
    });
  };

  const selectAllDuplicates = () => {
    const all = new Set<string>();
    duplicateGroups.forEach((g) => {
      if (g.files && g.files.length > 1) {
        g.files.slice(1).forEach((f) => all.add(f.file_id));
      }
    });
    setSelectedFileIds(all);
  };

  const deselectAll = () => setSelectedFileIds(new Set());

  const openDeleteSingle = (file_id: string, title: string) => {
    setConfirmState({ type: "single", file_id, title });
  };

  const openDeleteSelected = () => {
    if (selectedFileIds.size === 0) return;
    setConfirmState({ type: "selected", file_ids: Array.from(selectedFileIds) });
  };

  const openKeepOneOnly = (keepFileId: string, allFiles: { file_id: string }[]) => {
    const toDeleteFileIds = allFiles.map((f) => f.file_id).filter((id) => id !== keepFileId);
    if (toDeleteFileIds.length === 0) return;
    setConfirmState({ type: "keepOne", keepFileId, toDeleteFileIds });
  };

  const handleConfirmDelete = async () => {
    if (!confirmState) return;
    setIsDeleting(true);

    try {
      if (confirmState.type === "single") {
        setDeletingId(confirmState.file_id);
        await deleteFileMutation.mutateAsync(confirmState.file_id);
        setSelectedFileIds((prev) => {
          const next = new Set(prev);
          next.delete(confirmState.file_id);
          return next;
        });
        toast.success(t("common.success", "File deleted successfully"));
      } else if (confirmState.type === "selected") {
        for (const file_id of confirmState.file_ids) {
          setDeletingId(file_id);
          await deleteFileMutation.mutateAsync(file_id);
        }
        setSelectedFileIds(new Set());
        toast.success(t("common.success", "Selected files deleted successfully"));
      } else if (confirmState.type === "keepOne") {
        for (const file_id of confirmState.toDeleteFileIds) {
          setDeletingId(file_id);
          await deleteFileMutation.mutateAsync(file_id);
        }
        setSelectedFileIds((prev) => {
          const next = new Set(prev);
          confirmState.toDeleteFileIds.forEach((id) => next.delete(id));
          return next;
        });
        toast.success(t("common.success", "Duplicate copies removed successfully"));
      }
      void refetch();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setIsDeleting(false);
      setDeletingId(null);
      setConfirmState(null);
    }
  };

  const renderModalContent = () => {
    if (!confirmState) return { title: "", message: null };
    if (confirmState.type === "single") {
      return {
        title: t("common.delete", "Delete File"),
        message: (
          <span>
            {t("admin.confirm_delete_file", "Are you sure you want to delete file")}{" "}
            <strong>"{confirmState.title}"</strong>?
          </span>
        ),
      };
    }
    if (confirmState.type === "selected") {
      return {
        title: t("admin.delete_selected", "Delete Selected Files"),
        message: (
          <span>
            {t("admin.confirm_delete_selected", `Are you sure you want to delete ${confirmState.file_ids.length} selected file(s)?`)}
          </span>
        ),
      };
    }
    return {
      title: t("admin.keep_this_only", "Keep Only This Copy"),
      message: (
        <span>
          {t("admin.confirm_keep_one_msg", `Are you sure you want to keep this copy and delete ${confirmState.toDeleteFileIds.length} duplicate copy(ies)?`)}
        </span>
      ),
    };
  };

  return (
    <div className="flex flex-col h-full bg-base-100">
      {/* Header Bar */}
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("admin.duplicates_title", "Duplicate Files Management")}</h1>
          <p className="text-sm text-base-content/60 mt-1">
            {t("admin.duplicates_subtitle", "Detect and clean up identical ebook files uploaded across the library by SHA-256 hash.")}
          </p>
        </div>

        {/* Stats & Refresh Controls */}
        <div className="flex items-center gap-3 flex-wrap">
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-base-200/50 border border-base-200 text-xs">
            <Copy className="w-3.5 h-3.5 text-primary" />
            <span className="text-base-content/60">{t("admin.total_groups", "Duplicate Groups")}:</span>
            <span className="font-bold text-primary">{duplicateGroups.length}</span>
          </div>
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-base-200/50 border border-base-200 text-xs">
            <HardDrive className="w-3.5 h-3.5 text-warning" />
            <span className="text-base-content/60">{t("admin.space_wasted", "Wasted Space")}:</span>
            <span className="font-bold text-warning">{formatSize(wastedBytes)}</span>
          </div>
          <button
            onClick={async () => {
              await queryClient.invalidateQueries({ queryKey: ["admin", "duplicates"] });
              await refetch();
              toast.info(t("common.refreshed", "Data refreshed"));
            }}
            className="btn btn-square btn-ghost btn-sm sm:btn-md"
            title={t("settings.refresh", "Refresh")}
            disabled={isFetching}
          >
            <RefreshCw className={`h-5 w-5 ${isFetching ? "animate-spin" : ""}`} />
          </button>
        </div>
      </header>

      {/* Main Content Area */}
      <div className="flex-1 overflow-auto p-4 sm:p-6 lg:p-8">
        <div className="max-w-7xl mx-auto w-full space-y-6">
          {/* Bulk Action Controls */}
          {duplicateGroups.length > 0 && (
            <div className="flex flex-wrap items-center justify-between gap-3 bg-base-100 p-3 px-4 rounded-xl border border-base-200 shadow-sm">
              <div className="flex items-center gap-2">
                <button
                  onClick={selectedFileIds.size > 0 ? deselectAll : selectAllDuplicates}
                  className="btn btn-ghost btn-sm gap-1.5 text-xs"
                >
                  {selectedFileIds.size > 0 ? (
                    <CheckSquare className="w-4 h-4 text-primary" />
                  ) : (
                    <Square className="w-4 h-4 text-base-content/40" />
                  )}
                  {selectedFileIds.size > 0
                    ? t("admin.deselect_all", "Deselect All")
                    : t("admin.select_all_duplicates", "Select All Duplicates (Keep 1/Group)")}
                </button>
                {selectedFileIds.size > 0 && (
                  <span className="text-xs font-semibold text-primary bg-primary/10 px-2.5 py-1 rounded-lg">
                    {selectedFileIds.size} {t("common.selected", "selected")}
                  </span>
                )}
              </div>

              {selectedFileIds.size > 0 && (
                <button
                  onClick={openDeleteSelected}
                  className="btn btn-error btn-sm gap-2 text-xs"
                >
                  <Trash2 className="w-4 h-4" />
                  {t("admin.delete_selected", "Delete Selected Files")} ({selectedFileIds.size})
                </button>
              )}
            </div>
          )}

          {/* Body Content */}
          {loading ? (
            <div className="flex justify-center py-16">
              <Loader2 className="animate-spin text-primary w-8 h-8" />
            </div>
          ) : duplicateGroups.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-base-300 bg-base-100 p-12 sm:p-16 text-center flex flex-col items-center justify-center gap-3 shadow-xs">
              <div className="grid h-14 w-14 place-items-center rounded-2xl bg-success/10 text-success mb-1">
                <CheckCircle className="h-7 w-7" />
              </div>
              <div>
                <h3 className="text-base sm:text-lg font-bold text-base-content">{t("library.no_duplicates", "No Duplicate Files Found")}</h3>
                <p className="text-xs sm:text-sm text-base-content/60 mt-1 max-w-sm">
                  {t("admin.no_duplicates_desc", "All files in your library are unique. There are no SHA-256 hash collisions.")}
                </p>
              </div>
            </div>
          ) : (
            <div className="flex flex-col gap-6">
              {duplicateGroups.map((group) => (
                <div key={group.hash} className="bg-base-100 border border-base-200 p-5 rounded-2xl shadow-sm flex flex-col gap-4">
                  {/* Group Header */}
                  <div className="flex justify-between items-center flex-wrap gap-2 pb-3 border-b border-base-200">
                    <div className="flex items-center gap-2">
                      <span className="badge badge-warning font-bold gap-1 p-2 text-xs">
                        <Copy className="w-3.5 h-3.5" />
                        {group.files?.length || 0} {t("library.identical_files", "Identical Files")}
                      </span>
                      <code className="text-xs font-mono text-base-content/70 truncate max-w-lg">
                        SHA-256: {group.hash}
                      </code>
                    </div>
                  </div>

                  {/* Duplicate File Cards */}
                  <div className="grid grid-cols-1 gap-3">
                    {group.files?.map((file, idx) => {
                      const isSelected = selectedFileIds.has(file.file_id);
                      const isItemDeleting = deletingId === file.file_id;

                      return (
                        <div
                          key={file.file_id}
                          className={`flex flex-col sm:flex-row items-start sm:items-center justify-between p-3.5 rounded-xl border transition-all gap-4 ${
                            isSelected
                              ? "bg-primary/5 border-primary/40 shadow-sm"
                              : "bg-base-200/40 border-base-200 hover:bg-base-200/70"
                          }`}
                        >
                          <div className="flex items-center gap-3 min-w-0 flex-1">
                            <input
                              type="checkbox"
                              checked={isSelected}
                              onChange={() => toggleSelectFile(file.file_id)}
                              className="checkbox checkbox-primary checkbox-sm shrink-0"
                            />

                            {/* Cover Thumbnail */}
                            {file.book_cover_url ? (
                              <img
                                src={getMediaUrl(file.book_cover_url, file.book_id)}
                                alt={file.book_title}
                                className="w-12 h-16 object-cover rounded-lg shadow-sm border border-base-300 shrink-0"
                              />
                            ) : (
                              <div className="w-12 h-16 bg-base-300 rounded-lg flex items-center justify-center text-xs font-bold text-base-content/40 shrink-0">
                                NO COVER
                              </div>
                            )}

                            {/* File & Book Info */}
                            <div className="flex flex-col min-w-0 flex-1 gap-1">
                              <div className="flex items-center gap-2 flex-wrap">
                                <span className="font-bold text-sm text-base-content truncate">
                                  {file.book_title || t("common.untitled", "Untitled Book")}
                                </span>
                                {idx === 0 && (
                                  <span className="badge badge-success badge-xs font-bold text-[10px] gap-1">
                                    <ShieldCheck className="w-3 h-3" />
                                    {t("admin.primary_copy", "Primary Copy")}
                                  </span>
                                )}
                              </div>
                              <div className="text-xs text-base-content/60 font-mono truncate">
                                {file.path}
                              </div>
                              <div className="flex items-center gap-3 text-[11px] text-base-content/50 mt-0.5">
                                <span className="font-semibold text-primary">{formatSize(file.size_bytes)}</span>
                                <span>•</span>
                                <span className="uppercase font-bold">{file.format}</span>
                              </div>
                            </div>
                          </div>

                          {/* Action Buttons */}
                          <div className="flex items-center gap-2 shrink-0 self-end sm:self-center">
                            <button
                              onClick={() => openKeepOneOnly(file.file_id, group.files || [])}
                              disabled={isDeleting || (group.files?.length || 0) <= 1}
                              className="btn btn-outline btn-xs gap-1 font-normal"
                              title={t("admin.confirm_keep_one", "Keep this file and delete duplicate copies?")}
                            >
                              <ShieldCheck className="w-3.5 h-3.5 text-success" />
                              {t("admin.keep_this_only", "Keep Only This")}
                            </button>
                            <button
                              onClick={() => openDeleteSingle(file.file_id, file.book_title)}
                              disabled={isItemDeleting || isDeleting}
                              className="btn btn-ghost btn-square btn-sm text-error hover:bg-error/10"
                              title={t("common.delete", "Delete")}
                            >
                              {isItemDeleting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
                            </button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      <DeleteConfirmModal
        open={Boolean(confirmState)}
        title={renderModalContent().title}
        message={renderModalContent().message}
        loading={isDeleting}
        onClose={() => setConfirmState(null)}
        onConfirm={() => void handleConfirmDelete()}
      />
    </div>
  );
};
