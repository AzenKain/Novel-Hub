import React from "react";
import { Trash2, FolderInput, Tag, X, Layers, Download } from "lucide-react";
import { useTranslation } from "react-i18next";

interface BulkActionToolbarProps {
  selectedCount: number;
  onClearSelection: () => void;
  onBulkMove: () => void;
  onBulkAddTags: () => void;
  onBulkDelete: () => void;
  onBulkEditMetadata?: () => void;
  onBulkDownload?: () => void;
}

export const BulkActionToolbar: React.FC<BulkActionToolbarProps> = ({
  selectedCount,
  onClearSelection,
  onBulkMove,
  onBulkAddTags,
  onBulkDelete,
  onBulkEditMetadata,
  onBulkDownload,
}) => {
  const { t } = useTranslation();

  if (selectedCount === 0) return null;

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-40 bg-base-300/95 border border-base-content/20 backdrop-blur-md px-3 sm:px-5 py-2 sm:py-3 rounded-2xl sm:rounded-full shadow-2xl flex items-center gap-2 sm:gap-3 max-w-[calc(100vw-1.5rem)] overflow-x-auto no-scrollbar animate-in fade-in slide-in-from-bottom-4 duration-200">
      <span className="text-xs font-bold text-base-content px-1.5 sm:px-2 whitespace-nowrap shrink-0">
        {selectedCount} {t("library.selected", "selected")}
      </span>

      <div className="h-4 w-px bg-base-content/20 shrink-0" />

      <div className="flex items-center gap-1.5 sm:gap-2 shrink-0">
        {onBulkDownload && (
          <button
            type="button"
            onClick={onBulkDownload}
            className="btn btn-primary btn-xs gap-1.5 font-medium shrink-0"
            title={t("common.download", "Download")}
          >
            <Download className="h-3.5 w-3.5 shrink-0" />
            <span className="hidden xs:inline">{t("common.download", "Download")}</span>
          </button>
        )}

        {onBulkEditMetadata && (
          <button
            type="button"
            onClick={onBulkEditMetadata}
            className="btn btn-neutral btn-xs gap-1.5 font-medium shrink-0"
            title={t("library.bulk_edit_metadata_title", "Edit Metadata")}
          >
            <Layers className="h-3.5 w-3.5 shrink-0" />
            <span className="hidden sm:inline">{t("library.bulk_edit_metadata_title", "Edit Metadata")}</span>
          </button>
        )}

        <button
          type="button"
          onClick={onBulkMove}
          className="btn btn-neutral btn-xs gap-1.5 font-medium shrink-0"
          title={t("library.move_library", "Move Library")}
        >
          <FolderInput className="h-3.5 w-3.5 shrink-0" />
          <span className="hidden sm:inline">{t("library.move_library", "Move Library")}</span>
        </button>

        <button
          type="button"
          onClick={onBulkAddTags}
          className="btn btn-neutral btn-xs gap-1.5 font-medium shrink-0"
          title={t("library.add_tags", "Add Tags")}
        >
          <Tag className="h-3.5 w-3.5 shrink-0" />
          <span className="hidden sm:inline">{t("library.add_tags", "Add Tags")}</span>
        </button>

        <button
          type="button"
          onClick={onBulkDelete}
          className="btn btn-error btn-xs gap-1.5 font-medium shrink-0"
          title={t("common.delete", "Delete")}
        >
          <Trash2 className="h-3.5 w-3.5 shrink-0" />
          <span className="hidden xs:inline">{t("common.delete", "Delete")}</span>
        </button>

        <div className="h-4 w-px bg-base-content/20 shrink-0" />

        <button
          type="button"
          onClick={onClearSelection}
          className="btn btn-ghost btn-xs btn-circle shrink-0"
          title={t("common.cancel", "Cancel")}
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
};
