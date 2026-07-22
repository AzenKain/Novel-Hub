import React from 'react';
import { Trash2, FolderInput, Tag, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';

interface BulkActionToolbarProps {
  selectedCount: number;
  onClearSelection: () => void;
  onBulkMove: () => void;
  onBulkAddTags: () => void;
  onBulkDelete: () => void;
}

export const BulkActionToolbar: React.FC<BulkActionToolbarProps> = ({
  selectedCount,
  onClearSelection,
  onBulkMove,
  onBulkAddTags,
  onBulkDelete,
}) => {
  const { t } = useTranslation();

  if (selectedCount === 0) return null;

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-40 bg-base-300/95 border border-base-content/20 backdrop-blur-md px-5 py-3 rounded-full shadow-2xl flex items-center gap-3 animate-in fade-in slide-in-from-bottom-4 duration-200">
      <span className="text-xs font-bold text-base-content px-2">
        {selectedCount} {t('library.selected', 'selected')}
      </span>

      <div className="h-4 w-px bg-base-content/20" />

      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onBulkMove}
          className="btn btn-neutral btn-xs gap-1.5 font-medium"
        >
          <FolderInput className="h-3.5 w-3.5" />
          {t('library.move_library', 'Move Library')}
        </button>

        <button
          type="button"
          onClick={onBulkAddTags}
          className="btn btn-neutral btn-xs gap-1.5 font-medium"
        >
          <Tag className="h-3.5 w-3.5" />
          {t('library.add_tags', 'Add Tags')}
        </button>

        <button
          type="button"
          onClick={onBulkDelete}
          className="btn btn-error btn-xs gap-1.5 font-medium"
        >
          <Trash2 className="h-3.5 w-3.5" />
          {t('common.delete', 'Delete')}
        </button>

        <div className="h-4 w-px bg-base-content/20" />

        <button
          type="button"
          onClick={onClearSelection}
          className="btn btn-ghost btn-xs btn-circle"
          title={t('common.cancel', 'Cancel')}
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
};
