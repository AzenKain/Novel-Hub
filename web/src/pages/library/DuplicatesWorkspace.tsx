import React, { useState } from "react";
import { useDuplicatesQuery, useDeleteBookFileMutation, usePotentialDuplicatesQuery, useMergeBooksMutation } from "@/hooks";
import { Loader2, Trash2, Copy, CheckCircle, ShieldCheck, GitMerge } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { getMediaUrl } from "@/config/api";
import { toast } from "react-toastify";
import { DeleteConfirmModal } from "@/components/admin";

type ConfirmState =
  | { type: "single"; file_id: string; title: string }
  | { type: "keepOne"; keepFileId: string; toDeleteFileIds: string[] }
  | { type: "merge"; sourceId: string; sourceTitle: string; targetId: string; targetTitle: string }
  | null;

export const DuplicatesWorkspace = () => {
  const { t } = useTranslation();
  const { data: duplicateGroups = [], isLoading: loading, refetch } = useDuplicatesQuery();
  const { data: potentialDuplicates = [], isLoading: loadingPotential } = usePotentialDuplicatesQuery();
  const deleteFileMutation = useDeleteBookFileMutation();
  const mergeMutation = useMergeBooksMutation();
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

  const openDeleteSingle = (file_id: string, title: string) => {
    setConfirmState({ type: "single", file_id, title });
  };

  const openKeepOnlyOne = (keepFileId: string, groupFiles: { file_id: string }[]) => {
    const toDelete = groupFiles.filter((f) => f.file_id !== keepFileId).map((f) => f.file_id);
    if (toDelete.length === 0) return;
    setConfirmState({ type: "keepOne", keepFileId, toDeleteFileIds: toDelete });
  };

  const handleConfirmDelete = async () => {
    if (!confirmState) return;
    setIsDeleting(true);
    try {
      if (confirmState.type === "single") {
        setDeletingId(confirmState.file_id);
        await deleteFileMutation.mutateAsync(confirmState.file_id);
        toast.success(t("common.success", "File deleted successfully"));
      } else if (confirmState.type === "keepOne") {
        let deleted = 0;
        const failures: string[] = [];
        for (const id of confirmState.toDeleteFileIds) {
          try {
            await deleteFileMutation.mutateAsync(id);
            deleted++;
          } catch (err) {
            failures.push(err instanceof Error ? err.message : String(err));
          }
        }
        if (failures.length > 0) {
          toast.error(
            t("admin.duplicates_partial_cleanup", "Deleted {{deleted}} file(s), {{failed}} failed: {{reason}}", {
              deleted,
              failed: failures.length,
              reason: failures[0],
            })
          );
        } else {
          toast.success(t("common.success", "Duplicates cleaned up"));
        }
      } else if (confirmState.type === "merge") {
        await mergeMutation.mutateAsync({
          source_id: confirmState.sourceId,
          target_id: confirmState.targetId,
        });
        toast.success(t("admin.merge_books_success", "Books merged"));
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
        title: t("admin.confirm_delete_file_title", "Delete Duplicate File"),
        message: (
          <span>
            {t("admin.confirm_delete_file_msg", "Are you sure you want to delete file for")} <strong>"{confirmState.title}"</strong>?
          </span>
        ),
      };
    }
    if (confirmState.type === "keepOne") {
      return {
        title: t("admin.confirm_keep_one_title", "Keep Only This Copy"),
        message: (
          <span>
            {t("admin.confirm_keep_one_msg", "Are you sure you want to keep this copy and delete {{count}} duplicate copy(ies)?", { count: confirmState.toDeleteFileIds.length })}
          </span>
        ),
      };
    }
    return {
      title: t("admin.confirm_merge_title", "Merge Books"),
      message: (
        <span>
          {t("admin.confirm_merge_msg", 'Merge "{{source}}" into "{{target}}"? All chapters, files and reading data of the first book will be folded into the second, then the first is deleted.', {
            source: confirmState.sourceTitle,
            target: confirmState.targetTitle,
          })}
        </span>
      ),
    };
  };

  return (
    <div className="p-8 min-h-screen bg-base-200 max-w-7xl mx-auto">
      <div className="mb-8 flex justify-between items-center bg-base-100 p-6 rounded-2xl shadow-sm border border-base-200">
        <div>
          <h1 className="text-3xl font-bold mb-2 flex items-center gap-3">
            <Copy className="w-8 h-8 text-primary" />
            {t("library.duplicate_files", "Duplicate Files")}
          </h1>
          <p className="opacity-60">{t("library.manage_duplicates", "Manage identical files detected by SHA-256 hash.")}</p>
        </div>
        <Link to="/" className="btn btn-ghost">
          {t("library.back_to_library", "Back to Library")}
        </Link>
      </div>

      {loading ? (
        <div className="flex justify-center p-12">
          <Loader2 className="animate-spin text-primary w-8 h-8" />
        </div>
      ) : (
        <>
          {/* Fuzzy title/author matches: not hash-identical, but likely the same work */}
          <div className="card bg-base-100 shadow-sm border border-base-200">
            <div className="card-body p-6 gap-4">
              <div className="flex items-center gap-2 border-b border-base-200 pb-3">
                <div className="grid h-9 w-9 place-items-center rounded-xl bg-primary/10 text-primary">
                  <GitMerge className="h-5 w-5" />
                </div>
                <div>
                  <h2 className="font-bold text-lg">{t("library.potential_matches", "Potential Matches")}</h2>
                  <p className="text-xs opacity-60">{t("library.potential_matches_desc", "Same title and author with minor differences — merge the duplicates into one book.")}</p>
                </div>
              </div>

              {loadingPotential ? (
                <div className="flex justify-center p-6">
                  <Loader2 className="animate-spin text-primary w-6 h-6" />
                </div>
              ) : potentialDuplicates.length === 0 ? (
                <p className="text-center py-6 opacity-60">{t("library.no_potential_matches", "No potential matches found.")}</p>
              ) : (
                <div className="flex flex-col gap-3">
                  {potentialDuplicates.map((pair, idx) => (
                    <div key={idx} className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-3 p-3.5 rounded-xl border border-base-200 bg-base-200/40">
                      <div className="flex flex-col gap-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="badge badge-warning badge-sm font-bold gap-1">
                            {Math.round(pair.similarity * 100)}%
                          </span>
                          {pair.author_name && <span className="text-xs opacity-60">{pair.author_name}</span>}
                        </div>
                        <div className="text-sm font-semibold truncate">{pair.source_title}</div>
                        <div className="text-xs opacity-50">→ {t("library.merge_into", "merge into")} →</div>
                        <div className="text-sm font-semibold truncate">{pair.target_title}</div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0 self-end lg:self-center">
                        <button
                          onClick={() => setConfirmState({ type: "merge", sourceId: pair.source_id, sourceTitle: pair.source_title, targetId: pair.target_id, targetTitle: pair.target_title })}
                          disabled={mergeMutation.isPending}
                          className="btn btn-primary btn-xs font-bold"
                        >
                          {mergeMutation.isPending ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <GitMerge className="w-3.5 h-3.5" />}
                          {t("library.merge_into_target", "Merge into second")}
                        </button>
                        <button
                          onClick={() => setConfirmState({ type: "merge", sourceId: pair.target_id, sourceTitle: pair.target_title, targetId: pair.source_id, targetTitle: pair.source_title })}
                          disabled={mergeMutation.isPending}
                          className="btn btn-ghost btn-xs"
                        >
                          {t("library.merge_into_source", "Merge into first")}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {duplicateGroups.length === 0 ? (
            <div className="text-center p-12 bg-base-100 rounded-2xl border border-base-200 shadow-sm flex flex-col items-center gap-3">
              <CheckCircle className="w-12 h-12 text-success" />
              <p className="font-semibold text-lg">{t("library.no_duplicates", "No duplicate files found. Your library is clean!")}</p>
            </div>
          ) : (
            <div className="flex flex-col gap-6">
          {duplicateGroups.map((group) => (
            <div key={group.hash} className="card bg-base-100 shadow-sm border border-base-200">
              <div className="card-body p-6 gap-4">
                <div className="flex justify-between items-center border-b border-base-200 pb-3">
                  <div className="flex items-center gap-2">
                    <span className="badge badge-warning font-bold gap-1">
                      <Copy className="w-3.5 h-3.5" />
                      {group.files?.length || 0} {t("library.identical_files", "Identical Files")}
                    </span>
                    <code className="text-xs font-mono opacity-60">SHA-256: {group.hash}</code>
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-3">
                  {group.files?.map((file, idx) => (
                    <div
                      key={file.file_id}
                      className="flex flex-col sm:flex-row items-start sm:items-center justify-between p-3.5 rounded-xl border border-base-200 bg-base-200/40 gap-4"
                    >
                      <div className="flex items-center gap-3 min-w-0 flex-1">
                        {file.book_cover_url ? (
                          <img
                            src={getMediaUrl(file.book_cover_url)}
                            alt={file.book_title}
                            className="w-12 h-16 object-cover rounded-lg shadow-sm border border-base-300 shrink-0"
                          />
                        ) : (
                          <div className="w-12 h-16 bg-base-300 rounded-lg flex items-center justify-center text-xs font-bold text-base-content/40 shrink-0">
                            NO COVER
                          </div>
                        )}
                        <div className="flex flex-col min-w-0 flex-1 gap-1">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="font-bold text-sm text-base-content truncate">{file.book_title}</span>
                            {idx === 0 && (
                              <span className="badge badge-success badge-xs font-bold text-[10px] gap-1">
                                <ShieldCheck className="w-3 h-3" />
                                {t("admin.primary_copy", "Primary Copy")}
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-3 text-xs text-base-content/60 flex-wrap font-mono">
                            <span className="badge badge-outline text-[11px] font-bold uppercase">{file.format}</span>
                            <span>{formatSize(file.size_bytes)}</span>
                          </div>
                        </div>
                      </div>

                      <div className="flex items-center gap-2 shrink-0 self-end sm:self-center">
                        <button
                          onClick={() => openKeepOnlyOne(file.file_id, group.files)}
                          className="btn btn-ghost btn-xs text-primary"
                        >
                          {t("admin.keep_this_only", "Keep Only This")}
                        </button>
                        <button
                          onClick={() => openDeleteSingle(file.file_id, file.book_title)}
                          disabled={deletingId === file.file_id || isDeleting}
                          className="btn btn-ghost btn-square btn-sm text-error"
                        >
                          {deletingId === file.file_id ? <Loader2 className="w-4 h-4 animate-spin" /> : <Trash2 className="w-4 h-4" />}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ))}
        </div>
        )}
        </>
      )}

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
