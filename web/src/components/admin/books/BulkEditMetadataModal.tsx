import React, { useState, useEffect } from 'react';
import {
  X,
  Copy,
  Layers,
  Save,
  Check,
  Image as ImageIcon,
  Loader2,
  Plus,
  ArrowDown10
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { toast } from 'react-toastify';
import { getMediaUrl } from '@/config/api';
import { bookService } from '@/services';
import type { Book } from '@/types';

interface BulkEditItem {
  id: string;
  original: Book;
  title: string;
  author: string;
  series: string;
  series_index: string;
  publisher: string;
  language: string;
  subjects: string[];
  description: string;
  age_rating: string;
  tagInput: string;
  coverPreview: string | null;
  pendingCover: { type: 'file' | 'url'; value: File | string } | null;
  modified: boolean;
}

interface BulkEditMetadataModalProps {
  isOpen: boolean;
  books: Book[];
  onClose: () => void;
  onSuccess: () => void;
}

type SyncFieldType =
  | 'author'
  | 'series'
  | 'publisher'
  | 'language'
  | 'subjects'
  | 'description'
  | 'age_rating'
  | 'cover';

interface SyncModalState {
  isOpen: boolean;
  sourceIndex: number;
  field: SyncFieldType;
  fieldName: string;
  fieldValue: any;
  selectedBookIds: Set<string>;
}

export const BulkEditMetadataModal: React.FC<BulkEditMetadataModalProps> = ({
  isOpen,
  books,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const [items, setItems] = useState<BulkEditItem[]>([]);
  const [isSaving, setIsSaving] = useState(false);
  const [saveProgress, setSaveProgress] = useState<{ current: number; total: number } | null>(null);

  // Sync Field Modal State
  const [syncState, setSyncState] = useState<SyncModalState>({
    isOpen: false,
    sourceIndex: 0,
    field: 'author',
    fieldName: '',
    fieldValue: '',
    selectedBookIds: new Set<string>(),
  });

  // Initialize editable state when modal opens or books change
  useEffect(() => {
    if (!isOpen || books.length === 0) {
      setItems([]);
      return;
    }

    const initialItems: BulkEditItem[] = books.map((b) => {
      let series = '';
      let series_index = '';
      let publisher = '';
      let language = '';
      let subjects: string[] = [];

      if (b.metadata_json) {
        try {
          const meta = JSON.parse(b.metadata_json);
          series = meta.series || '';
          series_index = meta.series_index || '';
          publisher = meta.publisher || (meta.publishers && meta.publishers[0]) || '';
          language = meta.language || (meta.languages && meta.languages[0]) || '';
          if (Array.isArray(meta.subject)) {
            subjects = meta.subject;
          }
        } catch {}
      }

      return {
        id: b.id,
        original: b,
        title: b.title || '',
        author: b.author_name || b.author_id || '',
        series,
        series_index,
        publisher,
        language,
        subjects,
        description: b.description || '',
        age_rating: b.age_rating || '',
        tagInput: '',
        coverPreview: b.cover_url || null,
        pendingCover: null,
        modified: false,
      };
    });

    setItems(initialItems);
  }, [isOpen, books]);

  if (!isOpen) return null;

  const updateItem = (index: number, updates: Partial<BulkEditItem>) => {
    setItems((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], ...updates, modified: true };
      return next;
    });
  };

  // Open Sync confirmation modal for a specific field on a book
  const handleOpenSyncModal = (sourceIndex: number, field: SyncFieldType, fieldName: string) => {
    const sourceItem = items[sourceIndex];
    let fieldValue: any = '';

    switch (field) {
      case 'author':
        fieldValue = sourceItem.author;
        break;
      case 'series':
        fieldValue = sourceItem.series;
        break;
      case 'publisher':
        fieldValue = sourceItem.publisher;
        break;
      case 'language':
        fieldValue = sourceItem.language;
        break;
      case 'subjects':
        fieldValue = [...sourceItem.subjects];
        break;
      case 'description':
        fieldValue = sourceItem.description;
        break;
      case 'age_rating':
        fieldValue = sourceItem.age_rating;
        break;
      case 'cover':
        fieldValue = {
          preview: sourceItem.coverPreview,
          pendingCover: sourceItem.pendingCover,
        };
        break;
    }

    // Default: select all books in the batch
    const allIds = new Set<string>(items.map((it) => it.id));

    setSyncState({
      isOpen: true,
      sourceIndex,
      field,
      fieldName,
      fieldValue,
      selectedBookIds: allIds,
    });
  };

  // Execute sync field to selected books
  const handleApplySync = () => {
    const { field, fieldValue, selectedBookIds } = syncState;

    setItems((prev) =>
      prev.map((item) => {
        if (!selectedBookIds.has(item.id)) return item;

        const updates: Partial<BulkEditItem> = { modified: true };

        switch (field) {
          case 'author':
            updates.author = fieldValue as string;
            break;
          case 'series':
            updates.series = fieldValue as string;
            break;
          case 'publisher':
            updates.publisher = fieldValue as string;
            break;
          case 'language':
            updates.language = fieldValue as string;
            break;
          case 'subjects':
            updates.subjects = [...(fieldValue as string[])];
            break;
          case 'description':
            updates.description = fieldValue as string;
            break;
          case 'age_rating':
            updates.age_rating = fieldValue as string;
            break;
          case 'cover':
            updates.coverPreview = fieldValue.preview;
            updates.pendingCover = fieldValue.pendingCover;
            break;
        }

        return { ...item, ...updates };
      })
    );

    const count = selectedBookIds.size;
    toast.success(
      t('library.sync_applied', 'Applied {{field}} to {{count}} books', {
        field: syncState.fieldName,
        count,
      })
    );
    setSyncState((prev) => ({ ...prev, isOpen: false }));
  };

  // Auto-sequence series indices 1..N
  const handleAutoSequenceSeriesIndex = () => {
    setItems((prev) =>
      prev.map((item, idx) => ({
        ...item,
        series_index: String(idx + 1),
        modified: true,
      }))
    );
    toast.success(
      t('library.series_index_sequenced', 'Sequenced series index (1 to {{count}})', { count: items.length })
    );
  };

  // Save all modified books
  const handleSaveAll = async () => {
    const toSave = items.filter((it) => it.modified);
    if (toSave.length === 0) {
      onClose();
      return;
    }

    setIsSaving(true);
    setSaveProgress({ current: 0, total: toSave.length });

    let successCount = 0;
    let failCount = 0;

    for (let i = 0; i < toSave.length; i++) {
      const item = toSave[i];
      setSaveProgress({ current: i + 1, total: toSave.length });

      try {
        await bookService.updateMetadata(item.id, {
          title: item.title.trim() || item.original.title,
          author: item.author.trim(),
          series: item.series.trim(),
          series_index: item.series_index.trim(),
          publisher: item.publisher.trim(),
          language: item.language.trim(),
          subjects: item.subjects,
          description: item.description.trim(),
        });

        if (item.pendingCover) {
          if (item.pendingCover.type === 'file') {
            await bookService.updateCover(item.id, { cover: item.pendingCover.value as File });
          } else if (item.pendingCover.type === 'url') {
            await bookService.updateCover(item.id, { cover_url: item.pendingCover.value as string });
          }
        }
        successCount++;
      } catch (err) {
        console.error(`Failed to update metadata for book ${item.id}`, err);
        failCount++;
      }
    }

    setIsSaving(false);
    setSaveProgress(null);

    if (failCount === 0) {
      toast.success(
        t('library.bulk_saved_success', 'Successfully updated metadata for {{count}} books', {
          count: successCount,
        })
      );
    } else {
      toast.warn(
        t('library.bulk_saved_partial', 'Updated {{success}} books, {{fail}} failed', {
          success: successCount,
          fail: failCount,
        })
      );
    }

    onSuccess();
    onClose();
  };

  return (
    <>
      <dialog className="modal modal-open z-50">
        <div className="modal-box w-11/12 max-w-[96vw] 2xl:max-w-[1700px] h-[95vh] max-h-[96vh] bg-base-100 shadow-2xl p-0 overflow-hidden flex flex-col rounded-2xl">
          {/* Header */}
          <header className="px-6 py-4 border-b border-base-200 bg-base-200/40 flex items-center justify-between shrink-0">
            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-xl bg-primary/10 text-primary">
                <Layers className="w-6 h-6" />
              </div>
              <div>
                <h3 className="text-xl font-black text-base-content flex items-center gap-2">
                  {t('library.bulk_edit_metadata_title', 'Bulk Edit Metadata')}
                  <span className="badge badge-primary badge-md font-bold">
                    {items.length} {t('library.selected', 'books')}
                  </span>
                </h3>
                <p className="text-xs sm:text-sm text-base-content/60 mt-0.5">
                  {t(
                    'library.bulk_edit_metadata_desc',
                    'Edit and synchronize fields across multiple books simultaneously'
                  )}
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handleAutoSequenceSeriesIndex}
                className="btn btn-sm sm:btn-md btn-outline gap-1.5 text-xs sm:text-sm font-semibold rounded-xl"
                title={t('library.auto_index_desc', 'Set series index 1, 2, 3... in order')}
              >
                <ArrowDown10 className="w-4 h-4 sm:w-5 sm:h-5" />
                <span className="hidden sm:inline">1..N {t('library.series_index', 'Index')}</span>
              </button>
              <button
                type="button"
                onClick={onClose}
                disabled={isSaving}
                className="btn btn-sm sm:btn-md btn-circle btn-ghost"
              >
                ✕
              </button>
            </div>
          </header>

          {/* Body: Scrollable list of books */}
          <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-6">
            {items.map((item, index) => (
              <div
                key={item.id}
                className={`rounded-2xl border p-5 sm:p-6 transition-all ${
                  item.modified
                    ? 'border-primary/50 bg-base-100 shadow-md ring-2 ring-primary/20'
                    : 'border-base-300/80 bg-base-200/25'
                }`}
              >
                {/* Book Header */}
                <div className="flex items-start gap-4 sm:gap-5 mb-5 pb-4 border-b border-base-200/80">
                  {/* Book Index & Thumbnail */}
                  <div className="flex flex-col items-center gap-2 shrink-0">
                    <span className="text-xs font-mono font-bold px-2.5 py-0.5 rounded-lg bg-base-300 text-base-content/80">
                      #{index + 1}
                    </span>
                    <div className="w-16 h-24 rounded-xl bg-base-300 border border-base-300 overflow-hidden shadow-sm flex items-center justify-center relative group">
                      {item.coverPreview ? (
                        <img
                          src={
                            item.coverPreview.startsWith('blob:') || item.coverPreview.startsWith('http')
                              ? item.coverPreview
                              : getMediaUrl(item.coverPreview, item.id)
                          }
                          alt={item.title}
                          className="w-full h-full object-cover"
                        />
                      ) : (
                        <ImageIcon className="w-6 h-6 text-base-content/30" />
                      )}
                    </div>
                  </div>

                  {/* Title & Quick Info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-2 mb-1.5">
                      <label className="text-xs font-bold text-base-content/70 uppercase tracking-wider">
                        {t('book.title', 'Title')}
                      </label>
                      <button
                        type="button"
                        onClick={() => handleOpenSyncModal(index, 'cover', t('book.cover', 'Cover'))}
                        className="btn btn-ghost btn-xs text-primary gap-1 h-7 min-h-0 px-2.5 text-xs font-semibold rounded-lg hover:bg-primary/10"
                      >
                        <Copy className="w-3.5 h-3.5" />
                        {t('library.sync_cover', 'Sync Cover')}
                      </button>
                    </div>
                    <input
                      type="text"
                      value={item.title}
                      onChange={(e) => updateItem(index, { title: e.target.value })}
                      placeholder={t('book.title_placeholder', 'Book title')}
                      className="input input-md input-bordered w-full font-bold text-base bg-base-100 rounded-xl"
                    />
                  </div>
                </div>

                {/* Form Fields Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
                  {/* Author */}
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <span className="text-xs font-bold text-base-content/70 uppercase tracking-wider">
                        {t('book.author', 'Author')}
                      </span>
                      <button
                        type="button"
                        onClick={() => handleOpenSyncModal(index, 'author', t('book.author', 'Author'))}
                        className="btn btn-ghost btn-xs text-primary gap-1 h-6 min-h-0 px-2 text-xs font-semibold rounded-lg hover:bg-primary/10"
                      >
                        <Copy className="w-3 h-3" />
                        {t('library.sync_field_to_books', 'Sync')}
                      </button>
                    </div>
                    <input
                      type="text"
                      value={item.author}
                      onChange={(e) => updateItem(index, { author: e.target.value })}
                      placeholder="e.g. J.K. Rowling"
                      className="input input-md input-bordered w-full bg-base-100 text-sm rounded-xl"
                    />
                  </div>

                  {/* Series */}
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <span className="text-xs font-bold text-base-content/70 uppercase tracking-wider">
                        {t('book.series', 'Series')}
                      </span>
                      <button
                        type="button"
                        onClick={() => handleOpenSyncModal(index, 'series', t('book.series', 'Series'))}
                        className="btn btn-ghost btn-xs text-primary gap-1 h-6 min-h-0 px-2 text-xs font-semibold rounded-lg hover:bg-primary/10"
                      >
                        <Copy className="w-3 h-3" />
                        {t('library.sync_field_to_books', 'Sync')}
                      </button>
                    </div>
                    <input
                      type="text"
                      value={item.series}
                      onChange={(e) => updateItem(index, { series: e.target.value })}
                      placeholder="e.g. Harry Potter"
                      className="input input-md input-bordered w-full bg-base-100 text-sm rounded-xl"
                    />
                  </div>

                  {/* Series Index */}
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <span className="text-xs font-bold text-base-content/70 uppercase tracking-wider">
                        {t('book.series_index', 'Series Index')}
                      </span>
                    </div>
                    <input
                      type="text"
                      value={item.series_index}
                      onChange={(e) => updateItem(index, { series_index: e.target.value })}
                      placeholder="1, 2, 3.5..."
                      className="input input-md input-bordered w-full bg-base-100 text-sm font-mono rounded-xl"
                    />
                  </div>

                  {/* Publisher */}
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <span className="text-xs font-bold text-base-content/70 uppercase tracking-wider">
                        {t('book.publisher', 'Publisher')}
                      </span>
                      <button
                        type="button"
                        onClick={() => handleOpenSyncModal(index, 'publisher', t('book.publisher', 'Publisher'))}
                        className="btn btn-ghost btn-xs text-primary gap-1 h-6 min-h-0 px-2 text-xs font-semibold rounded-lg hover:bg-primary/10"
                      >
                        <Copy className="w-3 h-3" />
                        {t('library.sync_field_to_books', 'Sync')}
                      </button>
                    </div>
                    <input
                      type="text"
                      value={item.publisher}
                      onChange={(e) => updateItem(index, { publisher: e.target.value })}
                      placeholder="e.g. Bloomsbury"
                      className="input input-md input-bordered w-full bg-base-100 text-sm rounded-xl"
                    />
                  </div>

                  {/* Language */}
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <span className="text-xs font-bold text-base-content/70 uppercase tracking-wider">
                        {t('book.language', 'Language')}
                      </span>
                      <button
                        type="button"
                        onClick={() => handleOpenSyncModal(index, 'language', t('book.language', 'Language'))}
                        className="btn btn-ghost btn-xs text-primary gap-1 h-6 min-h-0 px-2 text-xs font-semibold rounded-lg hover:bg-primary/10"
                      >
                        <Copy className="w-3 h-3" />
                        {t('library.sync_field_to_books', 'Sync')}
                      </button>
                    </div>
                    <input
                      type="text"
                      value={item.language}
                      onChange={(e) => updateItem(index, { language: e.target.value })}
                      placeholder="vi, en, ja..."
                      className="input input-md input-bordered w-full bg-base-100 text-sm font-mono rounded-xl"
                    />
                  </div>

                  {/* Age Rating */}
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <span className="text-xs font-bold text-base-content/70 uppercase tracking-wider">
                        {t('book.age_rating', 'Age Rating')}
                      </span>
                      <button
                        type="button"
                        onClick={() => handleOpenSyncModal(index, 'age_rating', t('book.age_rating', 'Age Rating'))}
                        className="btn btn-ghost btn-xs text-primary gap-1 h-6 min-h-0 px-2 text-xs font-semibold rounded-lg hover:bg-primary/10"
                      >
                        <Copy className="w-3 h-3" />
                        {t('library.sync_field_to_books', 'Sync')}
                      </button>
                    </div>
                    <select
                      value={item.age_rating}
                      onChange={(e) => updateItem(index, { age_rating: e.target.value })}
                      className="select select-md select-bordered w-full bg-base-100 text-sm rounded-xl font-medium"
                    >
                      <option value="">{t('common.none', 'None')}</option>
                      <option value="safe">Safe / All Ages</option>
                      <option value="teen">Teen / 13+</option>
                      <option value="mature">Mature / 16+</option>
                      <option value="explicit">Explicit / 18+</option>
                    </select>
                  </div>
                </div>

                {/* Subjects / Tags row */}
                <div className="mt-4 pt-4 border-t border-base-200/60">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-xs font-bold text-base-content/70 uppercase tracking-wider">
                      {t('book.tags', 'Tags / Subjects')}
                    </span>
                    <button
                      type="button"
                      onClick={() => handleOpenSyncModal(index, 'subjects', t('book.tags', 'Tags'))}
                      className="btn btn-ghost btn-xs text-primary gap-1 h-6 min-h-0 px-2 text-xs font-semibold rounded-lg hover:bg-primary/10"
                    >
                      <Copy className="w-3 h-3" />
                      {t('library.sync_field_to_books', 'Sync')}
                    </button>
                  </div>
                  <div className="flex flex-wrap gap-2 items-center p-3 bg-base-200/40 border border-base-200 rounded-xl min-h-12">
                    {item.subjects.map((sub, sIdx) => (
                      <span key={sIdx} className="badge badge-md badge-primary/10 text-primary border border-primary/20 gap-1.5 py-3 px-3 text-xs font-medium rounded-lg">
                        {sub}
                        <button
                          type="button"
                          onClick={() => {
                            const next = item.subjects.filter((_, i) => i !== sIdx);
                            updateItem(index, { subjects: next });
                          }}
                          className="hover:text-error ml-0.5 cursor-pointer font-bold text-xs"
                        >
                          ✕
                        </button>
                      </span>
                    ))}
                    <div className="flex items-center gap-1.5 flex-1 min-w-[200px]">
                      <input
                        type="text"
                        value={item.tagInput}
                        onChange={(e) => updateItem(index, { tagInput: e.target.value })}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ',') {
                            e.preventDefault();
                            const val = item.tagInput.trim().replace(/^,+|,+$/g, '');
                            if (val && !item.subjects.includes(val)) {
                              updateItem(index, {
                                subjects: [...item.subjects, val],
                                tagInput: '',
                              });
                            }
                          }
                        }}
                        placeholder={t('book.add_tag_placeholder', 'Add tag (press Enter)...')}
                        className="input input-sm input-bordered bg-base-100 text-xs flex-1 rounded-lg"
                      />
                      <button
                        type="button"
                        onClick={() => {
                          const val = item.tagInput.trim().replace(/^,+|,+$/g, '');
                          if (val && !item.subjects.includes(val)) {
                            updateItem(index, {
                              subjects: [...item.subjects, val],
                              tagInput: '',
                            });
                          }
                        }}
                        disabled={!item.tagInput.trim()}
                        className="btn btn-sm btn-primary rounded-lg gap-1"
                      >
                        <Plus className="w-3.5 h-3.5" />
                        {t('common.add', 'Add')}
                      </button>
                    </div>
                  </div>
                </div>

                {/* Description row */}
                <div className="mt-4">
                  <div className="flex items-center justify-between mb-1.5">
                    <span className="text-xs font-bold text-base-content/70 uppercase tracking-wider">
                      {t('book.description', 'Description')}
                    </span>
                    <button
                      type="button"
                      onClick={() => handleOpenSyncModal(index, 'description', t('book.description', 'Description'))}
                      className="btn btn-ghost btn-xs text-primary gap-1 h-6 min-h-0 px-2 text-xs font-semibold rounded-lg hover:bg-primary/10"
                    >
                      <Copy className="w-3 h-3" />
                      {t('library.sync_field_to_books', 'Sync')}
                    </button>
                  </div>
                  <textarea
                    rows={3}
                    value={item.description}
                    onChange={(e) => updateItem(index, { description: e.target.value })}
                    placeholder={t('book.description_placeholder', 'Book summary / description...')}
                    className="textarea textarea-bordered textarea-md w-full bg-base-100 text-sm leading-relaxed rounded-xl resize-y"
                  />
                </div>
              </div>
            ))}
          </div>

          {/* Footer Actions */}
          <footer className="px-6 py-4 border-t border-base-200 bg-base-200/40 flex items-center justify-between shrink-0">
            <div className="text-xs sm:text-sm text-base-content/70">
              {saveProgress ? (
                <span className="flex items-center gap-2 font-bold text-primary">
                  <Loader2 className="w-4 h-4 animate-spin" />
                  {t('library.saving_progress', 'Saving {{current}} of {{total}} books...', {
                    current: saveProgress.current,
                    total: saveProgress.total,
                  })}
                </span>
              ) : (
                <span>
                  {items.filter((it) => it.modified).length > 0 ? (
                    <span className="text-warning font-semibold">
                      {t('library.modified_count', '{{count}} books modified', {
                        count: items.filter((it) => it.modified).length,
                      })}
                    </span>
                  ) : (
                    <span className="text-base-content/50">
                      {t('library.no_changes_yet', 'No changes made yet')}
                    </span>
                  )}
                </span>
              )}
            </div>

            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={onClose}
                disabled={isSaving}
                className="btn btn-ghost btn-sm sm:btn-md rounded-xl"
              >
                {t('common.cancel', 'Cancel')}
              </button>

              <button
                type="button"
                onClick={handleSaveAll}
                disabled={isSaving || items.filter((it) => it.modified).length === 0}
                className="btn btn-primary btn-sm sm:btn-md gap-2 !text-white font-bold rounded-xl shadow-lg shadow-primary/20"
              >
                {isSaving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                {t('common.save_all', 'Save All Changes')}
              </button>
            </div>
          </footer>
        </div>
      </dialog>

      {/* ======================= SYNC FIELD POPUP MODAL ======================= */}
      {syncState.isOpen && (
        <dialog className="modal modal-open z-60 bg-black/40 backdrop-blur-xs">
          <div className="modal-box w-11/12 max-w-lg bg-base-100 shadow-2xl p-0 overflow-hidden flex flex-col max-h-[85vh] rounded-2xl border border-base-300">
            {/* Sync Header */}
            <header className="px-5 py-3.5 border-b border-base-200 bg-base-200/40 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Copy className="w-4 h-4 text-primary" />
                <h4 className="font-bold text-sm">
                  {t('library.sync_field_title', 'Sync {{field}} to other books', {
                    field: syncState.fieldName,
                  })}
                </h4>
              </div>
              <button
                type="button"
                onClick={() => setSyncState((prev) => ({ ...prev, isOpen: false }))}
                className="btn btn-xs btn-circle btn-ghost"
              >
                ✕
              </button>
            </header>

            {/* Sync Body */}
            <div className="p-4 space-y-3 overflow-y-auto flex-1">
              {/* Value Preview Banner */}
              <div className="rounded-lg bg-primary/5 border border-primary/15 p-3 text-xs">
                <span className="font-bold text-primary block mb-1">
                  {t('library.value_to_sync', 'Value to synchronize')}:
                </span>
                <span className="text-base-content font-medium block truncate">
                  {typeof syncState.fieldValue === 'string'
                    ? syncState.fieldValue || `(${t('common.empty', 'Empty')})`
                    : Array.isArray(syncState.fieldValue)
                    ? syncState.fieldValue.join(', ') || `(${t('common.empty', 'Empty')})`
                    : syncState.field === 'cover'
                    ? t('library.cover_image', 'Cover Image')
                    : String(syncState.fieldValue)}
                </span>
              </div>

              {/* Target Selection Tools */}
              <div className="flex items-center justify-between text-xs px-1">
                <span className="font-semibold text-base-content/70">
                  {t('library.select_target_books', 'Select target books')}:
                </span>
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() =>
                      setSyncState((prev) => ({
                        ...prev,
                        selectedBookIds: new Set<string>(items.map((it) => it.id)),
                      }))
                    }
                    className="btn btn-ghost btn-xs text-primary font-medium"
                  >
                    {t('common.select_all', 'Select all')}
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setSyncState((prev) => ({
                        ...prev,
                        selectedBookIds: new Set<string>(),
                      }))
                    }
                    className="btn btn-ghost btn-xs text-base-content/60"
                  >
                    {t('common.deselect_all', 'Deselect all')}
                  </button>
                </div>
              </div>

              {/* Target Books Checkbox List */}
              <div className="space-y-1.5 max-h-60 overflow-y-auto pr-1">
                {items.map((item, idx) => {
                  const isChecked = syncState.selectedBookIds.has(item.id);
                  const isSource = idx === syncState.sourceIndex;

                  return (
                    <label
                      key={item.id}
                      className={`flex items-center gap-3 p-2 rounded-lg border text-xs cursor-pointer transition-colors ${
                        isChecked
                          ? 'border-primary/30 bg-primary/5 text-base-content'
                          : 'border-base-200 bg-base-100 text-base-content/60'
                      }`}
                    >
                      <input
                        type="checkbox"
                        checked={isChecked}
                        onChange={(e) => {
                          const next = new Set(syncState.selectedBookIds);
                          if (e.target.checked) next.add(item.id);
                          else next.delete(item.id);
                          setSyncState((prev) => ({ ...prev, selectedBookIds: next }));
                        }}
                        className="checkbox checkbox-xs checkbox-primary"
                      />
                      <div className="w-7 h-10 rounded bg-base-300 overflow-hidden shrink-0">
                        {item.coverPreview ? (
                          <img
                            src={
                              item.coverPreview.startsWith('blob:') || item.coverPreview.startsWith('http')
                                ? item.coverPreview
                                : getMediaUrl(item.coverPreview, item.id)
                            }
                            alt={item.title}
                            className="w-full h-full object-cover"
                          />
                        ) : (
                          <div className="w-full h-full grid place-items-center text-[8px] font-bold opacity-40">
                            #{idx + 1}
                          </div>
                        )}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="font-bold truncate flex items-center gap-1.5">
                          <span className="truncate">{item.title}</span>
                          {isSource && (
                            <span className="badge badge-xs badge-primary font-bold shrink-0">
                              {t('common.source', 'Source')}
                            </span>
                          )}
                        </div>
                        <div className="text-[10px] opacity-60 truncate">
                          {item.author || item.original.author_name || 'Unknown author'}
                        </div>
                      </div>
                    </label>
                  );
                })}
              </div>
            </div>

            {/* Sync Footer */}
            <footer className="px-5 py-3 border-t border-base-200 bg-base-200/40 flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={() => setSyncState((prev) => ({ ...prev, isOpen: false }))}
                className="btn btn-xs btn-ghost"
              >
                {t('common.cancel', 'Cancel')}
              </button>
              <button
                type="button"
                onClick={handleApplySync}
                disabled={syncState.selectedBookIds.size === 0}
                className="btn btn-xs btn-primary gap-1 !text-white font-bold"
              >
                <Check className="w-3.5 h-3.5" />
                {t('common.apply', 'Apply')} ({syncState.selectedBookIds.size})
              </button>
            </footer>
          </div>
        </dialog>
      )}
    </>
  );
};
