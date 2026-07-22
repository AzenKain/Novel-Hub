import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Loader2, Copy, CheckCircle, Trash2, ShieldCheck, HardDrive } from "lucide-react";
import { useDuplicatesQuery, useDeleteBookFileMutation } from "@/hooks";
import { getMediaUrl } from "@/config/api";
import { toast } from "react-toastify";

export const DuplicatesTab: React.FC = () => {
  const { t } = useTranslation();
  const { data: duplicateGroups = [], isLoading: loading, refetch } = useDuplicatesQuery();
  const deleteFileMutation = useDeleteBookFileMutation();
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const formatSize = (bytes: number) => {
    if (!bytes) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  const handleDeleteSingle = async (fileId: string, title: string) => {
    if (!window.confirm(t("admin.confirm_delete_file", `Are you sure you want to delete file for "${title}"?`))) return;
    setDeletingId(fileId);
    try {
      await deleteFileMutation.mutateAsync(fileId);
      toast.success(t("common.success", "File deleted successfully"));
      void refetch();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setDeletingId(null);
    }
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
    <div className="flex flex-col gap-6">
      <div>
        <h2 className="text-lg font-bold flex items-center gap-2">
          <Copy className="h-5 w-5 text-primary" />
          {t("library.duplicate_files", "Duplicate Files Management")}
        </h2>
        <p className="text-xs text-base-content/60">
          {t("library.manage_duplicates", "Manage identical files detected across the library by SHA-256 hash checksums.")}
        </p>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="animate-spin text-primary w-6 h-6" />
        </div>
      ) : duplicateGroups.length === 0 ? (
        <div className="bg-base-200 border border-base-300 rounded-xl p-8 text-center flex flex-col items-center gap-2">
          <CheckCircle className="h-10 w-10 text-success/60" />
          <div className="font-semibold text-sm">
            {t("library.no_duplicates", "No duplicate files found. Your library is clean!")}
          </div>
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          {duplicateGroups.map((group) => (
            <div key={group.hash} className="bg-base-200/50 border border-base-300 p-4 rounded-xl flex flex-col gap-3">
              <div className="flex justify-between items-center flex-wrap gap-2 pb-2 border-b border-base-300">
                <div className="flex items-center gap-2">
                  <span className="badge badge-warning font-bold text-xs">
                    {group.files?.length || 0} {t("library.identical_files", "Identical Files")}
                  </span>
                  <code className="text-xs font-mono text-base-content/70 truncate max-w-md">
                    SHA-256: {group.hash}
                  </code>
                </div>
              </div>

              <div className="flex flex-col gap-2">
                {group.files?.map((file, idx) => (
                  <div
                    key={file.fileId}
                    className="flex flex-col sm:flex-row items-start sm:items-center justify-between p-3 bg-base-100 rounded-lg border border-base-300 gap-3"
                  >
                    <div className="flex items-center gap-3 min-w-0 flex-1">
                      {file.bookCoverUrl ? (
                        <img
                          src={getMediaUrl(file.bookCoverUrl)}
                          alt={file.bookTitle}
                          className="w-10 h-14 object-cover rounded shadow-sm border border-base-300 shrink-0"
                        />
                      ) : (
                        <div className="w-10 h-14 bg-base-300 rounded flex items-center justify-center text-[10px] font-bold text-base-content/40 shrink-0">
                          NO COVER
                        </div>
                      )}
                      <div className="flex flex-col min-w-0 flex-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-semibold text-sm truncate">{file.bookTitle}</span>
                          {idx === 0 && (
                            <span className="badge badge-success badge-xs font-bold text-[10px]">
                              {t("admin.primary_copy", "Primary Copy")}
                            </span>
                          )}
                        </div>
                        <div className="flex items-center gap-2 text-xs text-base-content/60 flex-wrap font-mono">
                          <span className="badge badge-outline text-[10px]">{file.format}</span>
                          <span>{formatSize(file.sizeBytes)}</span>
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-2 shrink-0 self-end sm:self-center">
                      <button
                        onClick={() => void handleKeepOnlyOne(file.fileId, group.files)}
                        className="btn btn-ghost btn-xs text-primary"
                      >
                        {t("admin.keep_this_only", "Keep Only This")}
                      </button>
                      <button
                        onClick={() => void handleDeleteSingle(file.fileId, file.bookTitle)}
                        disabled={deletingId === file.fileId}
                        className="btn btn-ghost btn-square btn-xs text-error"
                      >
                        {deletingId === file.fileId ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Trash2 className="w-3.5 h-3.5" />}
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
