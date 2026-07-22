import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Loader2, Copy, CheckCircle, Trash2, ShieldCheck, HardDrive, CheckSquare, Square } from "lucide-react";
import { useDuplicatesQuery, useDeleteBookFileMutation } from "@/hooks";
import { getMediaUrl } from "@/config/api";
import { toast } from "react-toastify";

export const Duplicates: React.FC = () => {
  const { t } = useTranslation();
  const { data: duplicateGroups = [], isLoading: loading, refetch } = useDuplicatesQuery();
  const deleteFileMutation = useDeleteBookFileMutation();
  const [selectedFileIds, setSelectedFileIds] = useState<Set<string>>(new Set());
  const [deletingId, setDeletingId] = useState<string | null>(null);

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
    const singleSize = g.files[0]?.sizeBytes || 0;
    return sum + singleSize * (g.files.length - 1);
  }, 0);

  const toggleSelectFile = (fileId: string) => {
    setSelectedFileIds((prev) => {
      const next = new Set(prev);
      if (next.has(fileId)) next.delete(fileId);
      else next.add(fileId);
      return next;
    });
  };

  const selectAllDuplicates = () => {
    const all = new Set<string>();
    duplicateGroups.forEach((g) => {
      if (g.files && g.files.length > 1) {
        // Keep the first one, select the rest for deletion
        g.files.slice(1).forEach((f) => all.add(f.fileId));
      }
    });
    setSelectedFileIds(all);
  };

  const deselectAll = () => setSelectedFileIds(new Set());

  const handleDeleteSingle = async (fileId: string, title: string) => {
    if (!window.confirm(t("admin.confirm_delete_file", `Are you sure you want to delete file for "${title}"?`))) return;
    setDeletingId(fileId);
    try {
      await deleteFileMutation.mutateAsync(fileId);
      toast.success(t("common.success", "File deleted successfully"));
      setSelectedFileIds((prev) => {
        const next = new Set(prev);
        next.delete(fileId);
        return next;
      });
      void refetch();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setDeletingId(null);
    }
  };

  const handleDeleteSelected = async () => {
    if (selectedFileIds.size === 0) return;
    if (!window.confirm(t("admin.confirm_delete_selected", `Delete ${selectedFileIds.size} selected duplicate files?`))) return;
    
    let deletedCount = 0;
    for (const id of Array.from(selectedFileIds)) {
      try {
        await deleteFileMutation.mutateAsync(id);
        deletedCount++;
      } catch (err) {
        toast.error(`Error deleting file ${id}: ${String(err)}`);
      }
    }
    toast.success(`Successfully deleted ${deletedCount} files`);
    setSelectedFileIds(new Set());
    void refetch();
  };

  const handleKeepOnlyOne = async (keepFileId: string, groupFiles: { fileId: string }[]) => {
    const toDelete = groupFiles.filter((f) => f.fileId !== keepFileId).map((f) => f.fileId);
    if (toDelete.length === 0) return;
    if (!window.confirm(t("admin.confirm_keep_one", `Keep this file and delete ${toDelete.length} duplicate copy(ies)?`))) return;

    for (const id of toDelete) {
      try {
        await deleteFileMutation.mutateAsync(id);
      } catch (err) {
        toast.error(`Error: ${String(err)}`);
      }
    }
    toast.success(t("common.success", "Duplicates cleaned up"));
    void refetch();
  };

  return (
    <div className="p-6 max-w-7xl mx-auto flex flex-col gap-6">
      {/* Header & Stats Bar */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 bg-base-100 p-6 rounded-2xl shadow-sm border border-base-200">
        <div className="flex items-center gap-3">
          <div className="p-3 bg-primary/10 text-primary rounded-xl">
            <Copy className="h-6 w-6" />
          </div>
          <div>
            <h1 className="text-2xl font-bold">{t("admin.duplicates_title", "Duplicate Files Management")}</h1>
            <p className="text-xs text-base-content/60">
              {t("admin.duplicates_subtitle", "Detect and clean up identical ebook files uploaded across the library by SHA-256 hash.")}
            </p>
          </div>
        </div>

        {/* Stats */}
        <div className="flex items-center gap-3 flex-wrap">
          <div className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-base-200/50 border border-base-200 text-xs">
            <Copy className="w-4 h-4 text-primary" />
            <div>
              <div className="text-base-content/60">{t("admin.total_groups", "Duplicate Groups")}</div>
              <div className="font-bold text-sm text-primary">{duplicateGroups.length}</div>
            </div>
          </div>
          <div className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-base-200/50 border border-base-200 text-xs">
            <HardDrive className="w-4 h-4 text-warning" />
            <div>
              <div className="text-base-content/60">{t("admin.space_wasted", "Wasted Space")}</div>
              <div className="font-bold text-sm text-warning">{formatSize(wastedBytes)}</div>
            </div>
          </div>
        </div>
      </div>

      {/* Bulk Action Controls */}
      {duplicateGroups.length > 0 && (
        <div className="flex flex-wrap items-center justify-between gap-3 bg-base-100 p-3 px-4 rounded-xl border border-base-200">
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
              onClick={() => void handleDeleteSelected()}
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
        <div className="bg-base-100 border border-base-200 rounded-2xl p-12 text-center flex flex-col items-center gap-3 shadow-sm">
          <CheckCircle className="h-12 w-12 text-success" />
          <h3 className="text-lg font-bold">{t("library.no_duplicates", "No Duplicate Files Found")}</h3>
          <p className="text-xs text-base-content/60 max-w-md">
            {t("admin.no_duplicates_desc", "All files in your library are unique. There are no SHA-256 hash collisions.")}
          </p>
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
                  const isSelected = selectedFileIds.has(file.fileId);
                  const isDeleting = deletingId === file.fileId;

                  return (
                    <div
                      key={file.fileId}
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
                          onChange={() => toggleSelectFile(file.fileId)}
                          className="checkbox checkbox-primary checkbox-sm shrink-0"
                        />

                        {/* Cover Thumbnail */}
                        {file.bookCoverUrl ? (
                          <img
                            src={getMediaUrl(file.bookCoverUrl)}
                            alt={file.bookTitle}
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
                              {file.bookTitle || t("common.untitled", "Untitled Book")}
                            </span>
                            {idx === 0 && (
                              <span className="badge badge-success badge-xs font-bold text-[10px] gap-1">
                                <ShieldCheck className="w-3 h-3" />
                                {t("admin.primary_copy", "Primary Copy")}
                              </span>
                            )}
                          </div>

                          <div className="flex items-center gap-3 text-xs text-base-content/60 flex-wrap font-mono">
                            <span className="badge badge-outline text-[11px] font-bold uppercase">{file.format || "FILE"}</span>
                            <span>{formatSize(file.sizeBytes)}</span>
                            <span className="truncate max-w-md text-base-content/40" title={file.path}>
                              {file.path}
                            </span>
                          </div>
                        </div>
                      </div>

                      {/* Item Actions */}
                      <div className="flex items-center gap-2 shrink-0 self-end sm:self-center">
                        <button
                          onClick={() => void handleKeepOnlyOne(file.fileId, group.files)}
                          className="btn btn-ghost btn-xs text-primary font-medium hover:bg-primary/10"
                          title={t("admin.keep_only_this", "Keep only this copy and delete others")}
                        >
                          {t("admin.keep_this_only", "Keep Only This")}
                        </button>
                        <button
                          onClick={() => void handleDeleteSingle(file.fileId, file.bookTitle)}
                          disabled={isDeleting}
                          className="btn btn-ghost btn-square btn-sm text-error hover:bg-error/10"
                          title={t("common.delete", "Delete")}
                        >
                          {isDeleting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
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
  );
};
