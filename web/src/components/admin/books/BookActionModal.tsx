import { getMediaUrl } from "@/config/api";
import { Archive, ArchiveRestore, BookOpen, Image as ImageIcon, Settings, Trash2 } from "lucide-react";
import React from "react";

import type { Book } from "@/types";

type BookActionModalProps = {
  book: Book | null;
  onClose: () => void;
  onRead: (book: Book) => void;
  onEdit: (book: Book) => void;
  onDelete: (book: Book) => void;
  onArchive: (book: Book, archived: boolean) => void;
};

export const BookActionModal: React.FC<BookActionModalProps> = ({
  book,
  onClose,
  onRead,
  onEdit,
  onDelete,
  onArchive,
}) => {
  const isArchived = book?.status === "archived";
  return (
  <dialog className={`modal ${book ? "modal-open" : ""}`}>
    <div className="modal-box max-w-lg overflow-hidden p-0">
      <header className="flex items-start justify-between gap-4 border-b border-base-200 bg-base-200/30 px-5 py-4">
        <div className="min-w-0">
          <p className="text-xs font-black uppercase tracking-wider text-primary">
            Book actions
          </p>
          <h3 className="mt-1 truncate text-xl font-black">{book?.title}</h3>
          <p className="mt-1 truncate text-sm text-base-content/60">
            {book?.author_name || book?.author_id || "Unknown author"}
          </p>
        </div>
        <button onClick={onClose} className="btn btn-ghost btn-circle btn-sm">
          ✕
        </button>
      </header>
      <div className="grid gap-4 p-5 sm:grid-cols-[120px_1fr]">
        <div className="aspect-[3/4.12] overflow-hidden rounded-lg border border-base-200 bg-base-200 shadow-sm">
          {book?.cover_url ? (
            <img
              src={getMediaUrl(book.cover_url)}
              alt={book.title}
              loading="lazy"
              className="h-full w-full object-cover"
            />
          ) : (
            <div className="flex h-full w-full flex-col items-center justify-center gap-2 text-base-content/35">
              <ImageIcon size={26} />
              <span className="text-[10px] font-black uppercase">
                No cover
              </span>
            </div>
          )}
        </div>
        <div className="min-w-0 space-y-3">
          <div className="rounded-lg bg-base-200/45 p-3">
            <div className="text-[10px] font-black uppercase tracking-wider text-base-content/45">
              Files
            </div>
            <div className="mt-1 text-sm font-semibold">
              {book?.files?.length || 0} attached
            </div>
            <div className="mt-1 truncate text-xs uppercase text-base-content/50">
              {book?.files
                ?.map((file) => file.format)
                .filter(Boolean)
                .join(" · ") || "No file"}
            </div>
          </div>
          <div className="rounded-lg bg-base-200/45 p-3">
            <div className="text-[10px] font-black uppercase tracking-wider text-base-content/45">
              Status
            </div>
            <div className="mt-1 text-sm font-semibold capitalize">
              {book?.status || "unknown"}
            </div>
          </div>
        </div>
      </div>
      <footer className="grid grid-cols-2 gap-2 border-t border-base-200 bg-base-100 px-5 py-4">
        <button
          type="button"
          className="btn btn-primary gap-2"
          disabled={!book}
          onClick={() => book && onRead(book)}
        >
          <BookOpen className="h-4 w-4" />
          Read
        </button>
        <button
          type="button"
          className="btn btn-secondary gap-2"
          disabled={!book}
          onClick={() => book && onEdit(book)}
        >
          <Settings className="h-4 w-4" />
          Edit
        </button>
        <button
          type="button"
          className="btn btn-outline gap-2"
          disabled={!book}
          onClick={() => book && onArchive(book, !isArchived)}
        >
          {isArchived ? (
            <>
              <ArchiveRestore className="h-4 w-4" />
              Unarchive
            </>
          ) : (
            <>
              <Archive className="h-4 w-4" />
              Archive
            </>
          )}
        </button>
        <button
          type="button"
          className="btn btn-error btn-outline gap-2"
          disabled={!book}
          onClick={() => book && onDelete(book)}
        >
          <Trash2 className="h-4 w-4" />
          Delete
        </button>
      </footer>
    </div>
    <form method="dialog" className="modal-backdrop">
      <button onClick={onClose}>close</button>
    </form>
  </dialog>
  );
};
