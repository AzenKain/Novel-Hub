import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Trash2, FolderInput, Tag, AlertTriangle, Loader2 } from "lucide-react";
import { useLibrariesQuery } from "@/hooks/useLibraryQueries";
import {
  useBulkDeleteBooksMutation,
  useBulkMoveBooksMutation,
  useBulkAddTagsMutation,
} from "@/hooks/useBooksQuery";

interface BulkDeleteModalProps {
  isOpen: boolean;
  bookIds: string[];
  onClose: () => void;
  onSuccess: () => void;
}

export const BulkDeleteModal: React.FC<BulkDeleteModalProps> = ({
  isOpen,
  bookIds,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const deleteMutation = useBulkDeleteBooksMutation();

  if (!isOpen) return null;

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync(bookIds);
      onSuccess();
      onClose();
    } catch {}
  };

  return (
    <div className="modal modal-open">
      <div className="modal-box relative border border-base-content/10 shadow-2xl">
        <h3 className="font-bold text-lg text-error flex items-center gap-2">
          <AlertTriangle className="h-5 w-5" />
          {t("library.bulk_delete_title", "Delete Selected Books")}
        </h3>
        <p className="py-4 text-sm text-base-content/80">
          {t(
            "library.bulk_delete_confirm",
            "Are you sure you want to delete {{count}} selected books? This action cannot be undone and will remove all associated files.",
            { count: bookIds.length },
          )}
        </p>
        <div className="modal-action">
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={onClose}
            disabled={deleteMutation.isPending}
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            type="button"
            className="btn btn-error btn-sm gap-2"
            onClick={handleDelete}
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Trash2 className="h-4 w-4" />
            )}
            {t("common.delete", "Delete")}
          </button>
        </div>
      </div>
      <div className="modal-backdrop" onClick={onClose} />
    </div>
  );
};

interface BulkMoveModalProps {
  isOpen: boolean;
  bookIds: string[];
  onClose: () => void;
  onSuccess: () => void;
}

export const BulkMoveModal: React.FC<BulkMoveModalProps> = ({
  isOpen,
  bookIds,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const { data: libraries, isLoading } = useLibrariesQuery();
  const [selectedLibraryId, setSelectedLibraryId] = useState<string>("");
  const moveMutation = useBulkMoveBooksMutation();

  if (!isOpen) return null;

  const handleMove = async () => {
    if (!selectedLibraryId) return;
    try {
      await moveMutation.mutateAsync({
        bookIds,
        targetLibraryId: selectedLibraryId,
      });
      onSuccess();
      onClose();
    } catch {}
  };

  return (
    <div className="modal modal-open">
      <div className="modal-box relative border border-base-content/10 shadow-2xl">
        <h3 className="font-bold text-lg flex items-center gap-2">
          <FolderInput className="h-5 w-5 text-primary" />
          {t("library.bulk_move_title", "Move Books to Library")}
        </h3>
        <p className="py-2 text-sm text-base-content/70">
          {t(
            "library.bulk_move_subtitle",
            "Select target library for {{count}} selected books:",
            { count: bookIds.length },
          )}
        </p>

        <div className="py-3">
          {isLoading ? (
            <div className="flex justify-center p-4">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : (
            <select
              className="select select-bordered w-full text-sm"
              value={selectedLibraryId}
              onChange={(e) => setSelectedLibraryId(e.target.value)}
            >
              <option value="" disabled>
                {t("library.select_target_library", "Select target library...")}
              </option>
              {libraries?.map((lib) => (
                <option key={lib.id} value={lib.id}>
                  {lib.name}
                </option>
              ))}
            </select>
          )}
        </div>

        <div className="modal-action">
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={onClose}
            disabled={moveMutation.isPending}
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            type="button"
            className="btn btn-primary btn-sm gap-2"
            onClick={handleMove}
            disabled={!selectedLibraryId || moveMutation.isPending}
          >
            {moveMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <FolderInput className="h-4 w-4" />
            )}
            {t("library.move", "Move")}
          </button>
        </div>
      </div>
      <div className="modal-backdrop" onClick={onClose} />
    </div>
  );
};

interface BulkTagModalProps {
  isOpen: boolean;
  bookIds: string[];
  onClose: () => void;
  onSuccess: () => void;
}

export const BulkTagModal: React.FC<BulkTagModalProps> = ({
  isOpen,
  bookIds,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const [tagInput, setTagInput] = useState<string>("");
  const tagMutation = useBulkAddTagsMutation();

  if (!isOpen) return null;

  const handleAddTags = async () => {
    const tags = tagInput
      .split(",")
      .map((t) => t.trim())
      .filter((t) => t.length > 0);

    if (tags.length === 0) return;

    try {
      await tagMutation.mutateAsync({
        bookIds,
        tagNames: tags,
      });
      onSuccess();
      onClose();
    } catch {}
  };

  return (
    <div className="modal modal-open">
      <div className="modal-box relative border border-base-content/10 shadow-2xl">
        <h3 className="font-bold text-lg flex items-center gap-2">
          <Tag className="h-5 w-5 text-primary" />
          {t("library.bulk_tag_title", "Add Tags to Selected Books")}
        </h3>
        <p className="py-2 text-sm text-base-content/70">
          {t(
            "library.bulk_tag_subtitle",
            "Enter tags separated by commas to add to {{count}} books:",
            { count: bookIds.length },
          )}
        </p>

        <div className="py-3">
          <input
            type="text"
            className="input input-bordered w-full text-sm"
            placeholder={t(
              "library.tag_input_placeholder",
              "e.g. Fantasy, Action, Sci-Fi",
            )}
            value={tagInput}
            onChange={(e) => setTagInput(e.target.value)}
          />
        </div>

        <div className="modal-action">
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={onClose}
            disabled={tagMutation.isPending}
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            type="button"
            className="btn btn-primary btn-sm gap-2"
            onClick={handleAddTags}
            disabled={!tagInput.trim() || tagMutation.isPending}
          >
            {tagMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Tag className="h-4 w-4" />
            )}
            {t("library.add_tags", "Add Tags")}
          </button>
        </div>
      </div>
      <div className="modal-backdrop" onClick={onClose} />
    </div>
  );
};
