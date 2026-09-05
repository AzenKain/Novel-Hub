import { DeleteConfirmModal } from "@/components/admin/books/DeleteConfirmModal";
import { TopNav } from "@/components/common/TopNav";
import { ReadListBooksPanel } from "@/components/library/ReadListBooksPanel";
import { ReadListImportReport } from "@/components/library/ReadListImportReport";
import {
  useCreateReadListMutation,
  useDeleteReadListMutation,
  useImportCBLMutation,
  useReadListBooksQuery,
  useReadListsQuery,
  useRemoveReadListBookMutation,
  useReorderReadListMutation,
} from "@/hooks";
import { useAuthStore, useDownloadManagerStore } from "@/stores";
import type { ImportCBLResult, ReadList } from "@/types";
import { hasPermission } from "@/utils/permission";
import { ArrowLeft, ListOrdered, Plus, Trash2, Upload } from "lucide-react";
import React, { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router-dom";
import { toast } from "react-toastify";

export const ReadListPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const user = useAuthStore((s) => s.user);
  const allowDownload = hasPermission(user, "book.download");

  const [activeId, setActiveId] = useState<string | null>(null);
  const [newName, setNewName] = useState("");
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<ReadList | null>(null);
  const [importResult, setImportResult] = useState<ImportCBLResult | null>(null);

  const listsQuery = useReadListsQuery();
  const lists = useMemo(
    () => listsQuery.data?.pages.flatMap((page) => page.data || []) ?? [],
    [listsQuery.data],
  );

  const active = lists.find((list) => list.id === activeId) || null;
  const booksQuery = useReadListBooksQuery(activeId || undefined);
  const books = booksQuery.data || [];

  const createMutation = useCreateReadListMutation();
  const deleteMutation = useDeleteReadListMutation();
  const removeBookMutation = useRemoveReadListBookMutation();
  const reorderMutation = useReorderReadListMutation();
  const importMutation = useImportCBLMutation((result) => {
    setActiveId(result.read_list.id);
    setImportResult(result);
  });

  useEffect(() => {
    if (!activeId && lists.length > 0) setActiveId(lists[0].id);
  }, [activeId, lists]);

  const handleCreate = () => {
    const name = newName.trim();
    if (!name) return;
    createMutation.mutate(
      { name },
      {
        onSuccess: (created) => {
          if (created) setActiveId(created.id);
          setNewName("");
          setIsCreateOpen(false);
        },
      },
    );
  };

  const handleDelete = () => {
    if (!deleteTarget) return;
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => {
        if (activeId === deleteTarget.id) setActiveId(null);
        setDeleteTarget(null);
      },
    });
  };

  const handleImportFile = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (file) importMutation.mutate(file);
  };

  const handleDownloadAll = () => {
    const validBooks = books.map((b) => b.book).filter((b) => b && b.files && b.files.length > 0);
    if (validBooks.length === 0) return;
    const { addBulkDownloads, open } = useDownloadManagerStore.getState();
    addBulkDownloads(
      validBooks.map((b) => ({
        bookId: b.id,
        title: b.title,
        coverUrl: b.cover_url,
        format: b.files?.[0]?.format || "EPUB",
        sizeBytes: b.files?.[0]?.size_bytes,
      })),
      false
    );
    open();
    toast.success(
      t("admin.bulk_download_started", "Added {{count}} books to download queue", {
        count: validBooks.length,
      })
    );
  };

  // The reader walks the rest of the list itself once it knows which list it is inside, so opening
  // position 0 with ?readlist= is the whole of "read in order".
  const openInReader = (bookId: string) => {
    if (!activeId) return;
    navigate(`/reader/${bookId}?readlist=${activeId}`);
  };

  return (
    <div className="min-h-screen bg-base-200/40 flex flex-col">
      <TopNav showSidebarToggle={false} />

      <div className="flex-1 container mx-auto p-4 sm:p-6 lg:p-8 max-w-[1700px] w-full flex flex-col gap-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <Link to="/" className="btn btn-ghost btn-sm gap-1.5 text-primary -ml-2.5">
              <ArrowLeft className="h-4 w-4" />
              {t("library.back_to_library", "Back to Library")}
            </Link>
            <div className="h-6 w-px bg-base-300" />
            <h1 className="flex items-center gap-2 text-xl font-black text-base-content">
              <ListOrdered className="h-5 w-5 text-primary" />
              {t("library.readlists", "Read Lists")}
            </h1>
          </div>

          <div className="flex items-center gap-2">
            <input
              ref={fileInputRef}
              type="file"
              accept=".cbl"
              className="hidden"
              onChange={handleImportFile}
            />
            <button
              className="btn btn-ghost btn-sm gap-1.5 rounded-xl"
              disabled={importMutation.isPending}
              onClick={() => fileInputRef.current?.click()}
            >
              {importMutation.isPending ? (
                <span className="loading loading-spinner loading-xs"></span>
              ) : (
                <Upload className="h-4 w-4" />
              )}
              {t("library.readlist_import_cbl", "Import .cbl")}
            </button>
            <button className="btn btn-primary btn-sm gap-1.5 rounded-xl" onClick={() => setIsCreateOpen(true)}>
              <Plus className="h-4 w-4" />
              {t("library.readlist_new", "New read list")}
            </button>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[20rem_1fr]">
          <aside className="flex flex-col gap-2">
            {listsQuery.isLoading ? (
              <div className="flex items-center justify-center py-16">
                <span className="loading loading-spinner loading-md text-primary"></span>
              </div>
            ) : lists.length === 0 ? (
              <div className="rounded-2xl border border-dashed border-base-300 bg-base-100 p-6 text-center text-sm text-base-content/60">
                {t("library.readlist_empty", "No read lists yet. Create one or import a .cbl file.")}
              </div>
            ) : (
              <ul className="flex flex-col gap-1.5">
                {lists.map((list) => {
                  const isActive = list.id === activeId;
                  return (
                    <li key={list.id}>
                      <div
                        onClick={() => setActiveId(list.id)}
                        className={`group flex cursor-pointer items-center justify-between gap-2 rounded-xl p-3 text-sm font-medium transition-all ${
                          isActive
                            ? "bg-primary text-primary-content shadow-sm"
                            : "bg-base-100 hover:bg-base-200 text-base-content border border-base-200"
                        }`}
                      >
                        <div className="flex min-w-0 flex-1 items-center gap-2.5">
                          <ListOrdered className={`h-4 w-4 shrink-0 ${isActive ? "text-primary-content" : "text-primary"}`} />
                          <div className="min-w-0 flex-1">
                            <p className="truncate font-bold">{list.name}</p>
                            <p className={`text-xs ${isActive ? "text-primary-content/80" : "text-base-content/50"}`}>
                              {list.book_count || 0} {t("library.readlist_books", "books")}
                            </p>
                          </div>
                        </div>
                        <button
                          type="button"
                          className={`btn btn-ghost btn-xs btn-square opacity-0 transition-opacity group-hover:opacity-100 ${
                            isActive ? "hover:bg-primary-focus text-primary-content" : "text-error hover:bg-error/20"
                          }`}
                          onClick={(e) => {
                            e.stopPropagation();
                            setDeleteTarget(list);
                          }}
                          title={t("common.delete", "Delete")}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}

            {listsQuery.hasNextPage && (
              <button
                className="btn btn-ghost btn-sm rounded-xl"
                disabled={listsQuery.isFetchingNextPage}
                onClick={() => void listsQuery.fetchNextPage()}
              >
                {listsQuery.isFetchingNextPage ? (
                  <span className="loading loading-spinner loading-xs"></span>
                ) : (
                  t("common.load_more", "Load more")
                )}
              </button>
            )}
          </aside>

          <section className="flex flex-col gap-4">
            {active ? (
              <div className="flex flex-col gap-4">
                <div className="rounded-2xl border border-base-200 bg-base-100 p-5 shadow-2xs">
                  <h2 className="text-xl font-black text-base-content">{active.name}</h2>
                  {active.description && (
                    <p className="mt-1 text-sm text-base-content/60">{active.description}</p>
                  )}
                </div>
                <ReadListBooksPanel
                  books={books}
                  isLoading={booksQuery.isLoading}
                  isReordering={reorderMutation.isPending}
                  onReorder={(bookIds) => reorderMutation.mutate({ id: active.id, bookIds })}
                  onRemove={(bookId) => removeBookMutation.mutate({ id: active.id, bookId })}
                  onOpenBook={openInReader}
                  onReadInOrder={() => books[0] && openInReader(books[0].book.id)}
                  onDownloadAll={allowDownload ? handleDownloadAll : undefined}
                />
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center gap-3 rounded-2xl border border-dashed border-base-300 bg-base-100 py-20 text-center">
                <div className="grid h-16 w-16 place-items-center rounded-2xl bg-base-200 text-base-content/40">
                  <ListOrdered className="h-8 w-8" />
                </div>
                <p className="text-sm text-base-content/60">
                  {t("library.readlist_select_hint", "Select a read list to see its reading order.")}
                </p>
              </div>
            )}
          </section>
        </div>
      </div>

      <dialog className={`modal ${isCreateOpen ? "modal-open" : ""}`}>
        <div className="modal-box">
          <h3 className="text-lg font-bold">{t("library.readlist_new", "New read list")}</h3>
          <input
            type="text"
            className="input input-bordered mt-4 w-full"
            placeholder={t("library.readlist_name_placeholder", "e.g. Civil War reading order")}
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleCreate()}
          />
          <div className="modal-action">
            <button className="btn btn-ghost" onClick={() => setIsCreateOpen(false)}>
              {t("common.cancel", "Cancel")}
            </button>
            <button
              className="btn btn-primary"
              disabled={!newName.trim() || createMutation.isPending}
              onClick={handleCreate}
            >
              {createMutation.isPending && <span className="loading loading-spinner loading-xs"></span>}
              {t("common.create", "Create")}
            </button>
          </div>
        </div>
      </dialog>

      <DeleteConfirmModal
        open={!!deleteTarget}
        title={t("library.readlist_delete", "Delete read list")}
        message={t("library.readlist_delete_confirm", "Delete \"{{name}}\"? The books themselves stay in your library.", {
          name: deleteTarget?.name || "",
        })}
        loading={deleteMutation.isPending}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
      />

      <ReadListImportReport
        open={!!importResult}
        total={importResult?.total || 0}
        matched={importResult?.matched || 0}
        unmatched={importResult?.unmatched || []}
        onClose={() => setImportResult(null)}
      />
    </div>
  );
};
