import React from 'react';
import { Trash2, FolderInput, Tag, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';

interface BulkActionToolbarProps {
  selectedCount: number;
  onClearSelection: () => void;
  onBulkDelete: () => void;
}

export const BulkActionToolbar: React.FC<BulkActionToolbarProps> = ({
  selectedCount,
  onClearSelection,
  onBulkDelete,
}) => {
  const { t } = useTranslation();

  if (selectedCount === 0) return null;

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-40 bg-base-300/95 border border-base-content/20 backdrop-blur-md px-4 py-2.5 rounded-full shadow-2xl flex items-center gap-3 animate-bounce-in">
      <span className="text-xs font-bold text-base-content px-2">
        {selectedCount} {t('library.selected', 'selected')}
      </span>

      <div className="h-4 w-px bg-base-content/20" />

      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onBulkDelete}
          className="btn btn-error btn-xs gap-1 font-medium"
        >
          <Trash2 className="h-3.5 w-3.5" />
          {t('common.delete', 'Delete')}
        </button>

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
