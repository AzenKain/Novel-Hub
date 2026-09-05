import React from "react";
import { useTranslation } from "react-i18next";
import { GripVertical, Edit2, Trash2, PinOff } from "lucide-react";
import {
  useSmartFilterBooksQuery,
  usePinSmartFilterHomeMutation,
} from "@/hooks";
import { HorizontalBookShelf } from "./HorizontalBookShelf";
import type { SmartFilter, Book } from "@/types";

interface SmartFilterShelfProps {
  filter: SmartFilter;
  onEdit: (filter: SmartFilter) => void;
  onDelete: (id: string) => void;
  onBookClick: (book: Book) => void;
  onDragStart: (e: React.DragEvent<HTMLDivElement>) => void;
  onDragOver: (e: React.DragEvent<HTMLDivElement>) => void;
  onDrop: (e: React.DragEvent<HTMLDivElement>) => void;
}

export const SmartFilterShelf: React.FC<SmartFilterShelfProps> = ({
  filter,
  onEdit,
  onDelete,
  onBookClick,
  onDragStart,
  onDragOver,
  onDrop,
}) => {
  const { t } = useTranslation();
  const { data: books = [], isLoading } = useSmartFilterBooksQuery(
    filter.id,
    undefined,
    12,
  );
  const pinHomeMutation = usePinSmartFilterHomeMutation();

  const handleUnpin = (e: React.MouseEvent) => {
    e.stopPropagation();
    pinHomeMutation.mutate({ id: filter.id, isPinned: false });
  };

  return (
    <div
      draggable
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDrop={onDrop}
      className="group/shelf rounded-2xl bg-base-100 shadow-sm border border-base-200 p-4 sm:p-5 transition-all hover:border-primary/30"
    >
      {isLoading ? (
        <div className="flex gap-4 overflow-x-auto pb-2 items-center justify-center min-h-55">
          <span className="loading loading-spinner loading-md text-primary"></span>
        </div>
      ) : (
        <HorizontalBookShelf
          title={filter.name}
          subtitle={t(
            "library.smart_filter_shelf_desc",
            "Dynamic custom shelf",
          )}
          icon={
            <div className="cursor-grab active:cursor-grabbing p-1 text-base-content/30 hover:text-base-content/75 transition-colors">
              <GripVertical className="h-5 w-5 shrink-0" />
            </div>
          }
          headerActions={
            <div className="flex items-center gap-1 opacity-0 group-hover/shelf:opacity-100 transition-opacity">
              <button
                type="button"
                onClick={() => onEdit(filter)}
                className="btn btn-ghost btn-xs btn-square text-base-content/60 cursor-pointer"
                title={t("common.edit", "Edit")}
              >
                <Edit2 className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={handleUnpin}
                className="btn btn-ghost btn-xs btn-square text-base-content/60 cursor-pointer"
                title={t("library.unpin_home", "Unpin from homepage")}
              >
                <PinOff className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() => onDelete(filter.id)}
                className="btn btn-ghost btn-xs btn-square text-error cursor-pointer"
                title={t("common.delete", "Delete")}
              >
                <Trash2 className="w-3.5 h-3.5" />
              </button>
            </div>
          }
          books={books}
          onBookClick={onBookClick}
          emptyMessage={t(
            "library.no_matching_books",
            "No books matching this filter.",
          )}
        />
      )}
    </div>
  );
};
