import { getMediaUrl } from "@/config/api";
import type { ReadListBook } from "@/types";
import { BookOpen, ChevronDown, ChevronUp, GripVertical, Play, Trash2 } from "lucide-react";
import React, { useState } from "react";
import { useTranslation } from "react-i18next";

type ReadListBooksPanelProps = {
  books: ReadListBook[];
  isLoading: boolean;
  isReordering: boolean;
  onReorder: (bookIds: string[]) => void;
  onRemove: (bookId: string) => void;
  onOpenBook: (bookId: string) => void;
  onReadInOrder: () => void;
};

// The server rejects an order that does not name every stored book, so a move always sends the whole
// id array back rather than the pair that changed.
const movedOrder = (books: ReadListBook[], from: number, to: number) => {
  const next = [...books];
  const [entry] = next.splice(from, 1);
  next.splice(to, 0, entry);
  return next.map((item) => item.book.id);
};

export const ReadListBooksPanel: React.FC<ReadListBooksPanelProps> = ({
  books,
  isLoading,
  isReordering,
  onReorder,
  onRemove,
  onOpenBook,
  onReadInOrder,
}) => {
  const { t } = useTranslation();
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);

  const handleDrop = (dropIndex: number) => {
    if (draggedIndex === null || draggedIndex === dropIndex) return;
    onReorder(movedOrder(books, draggedIndex, dropIndex));
    setDraggedIndex(null);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <span className="loading loading-spinner loading-lg text-primary"></span>
      </div>
    );
  }

  if (books.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20 rounded-2xl border border-dashed border-base-300 bg-base-100 text-center">
        <div className="grid h-16 w-16 place-items-center rounded-2xl bg-base-200 text-base-content/40">
          <BookOpen className="h-8 w-8" />
        </div>
        <div>
          <p className="font-bold text-base text-base-content/80">
            {t("library.readlist_books_empty", "This read list is empty")}
          </p>
          <p className="mt-1 text-xs text-base-content/50">
            {t("library.readlist_books_empty_hint", "Open a book and add it to this read list to start the order.")}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <button className="btn btn-primary btn-sm w-fit gap-1.5 rounded-xl" onClick={onReadInOrder}>
        <Play className="h-4 w-4" />
        {t("library.readlist_read_in_order", "Read in order")}
      </button>

      <ul className="flex flex-col gap-2">
        {books.map((entry, index) => {
          const coverUrl = entry.book.cover_url ? getMediaUrl(entry.book.cover_url) : null;
          return (
            <li
              key={entry.book.id}
              draggable
              onDragStart={() => setDraggedIndex(index)}
              onDragOver={(e) => e.preventDefault()}
              onDrop={() => handleDrop(index)}
              className="flex items-center gap-3 rounded-xl border border-base-200 bg-base-100 p-3 shadow-2xs transition-all hover:border-primary/40 hover:shadow-md"
            >
              <GripVertical className="h-4 w-4 shrink-0 cursor-grab text-base-content/30 active:cursor-grabbing" />
              <span className="w-7 shrink-0 text-center font-mono text-xs font-bold text-base-content/40">
                {index + 1}
              </span>
              <button
                className="flex min-w-0 flex-1 items-center gap-3 border-none bg-transparent text-left"
                onClick={() => onOpenBook(entry.book.id)}
              >
                <div className="relative aspect-[3/4.2] w-11 shrink-0 overflow-hidden rounded-lg border border-base-200 bg-base-200">
                  {coverUrl ? (
                    <img src={coverUrl} alt={entry.book.title} loading="lazy" className="h-full w-full object-cover" />
                  ) : (
                    <div className="grid h-full w-full place-items-center bg-primary/10 text-primary">
                      <BookOpen className="h-5 w-5" />
                    </div>
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-bold text-base-content">{entry.book.title}</p>
                  {entry.book.author_name && (
                    <p className="mt-0.5 truncate text-xs text-base-content/60">{entry.book.author_name}</p>
                  )}
                </div>
              </button>

              <div className="flex shrink-0 items-center gap-1">
                <button
                  className="btn btn-ghost btn-xs btn-square"
                  disabled={index === 0 || isReordering}
                  onClick={() => onReorder(movedOrder(books, index, index - 1))}
                  title={t("library.readlist_move_up", "Move up")}
                >
                  <ChevronUp className="h-3.5 w-3.5" />
                </button>
                <button
                  className="btn btn-ghost btn-xs btn-square"
                  disabled={index === books.length - 1 || isReordering}
                  onClick={() => onReorder(movedOrder(books, index, index + 1))}
                  title={t("library.readlist_move_down", "Move down")}
                >
                  <ChevronDown className="h-3.5 w-3.5" />
                </button>
                <button
                  className="btn btn-ghost btn-xs btn-square text-error hover:bg-error/10"
                  onClick={() => onRemove(entry.book.id)}
                  title={t("library.readlist_remove_book", "Remove from read list")}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
};
