import React, { useState, useEffect, useMemo } from 'react';
import {
  X,
  Copy,
  Layers,
  Save,
  Check,
  Image as ImageIcon,
  Loader2,
  Plus,
  ArrowDown10,
  Upload,
  Link as LinkIcon,
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
  subjectMode?: 'append' | 'remove' | 'replace';
  selectedTagToRemove?: string;
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

  const [coverUrlModal, setCoverUrlModal] = useState<{
    isOpen: boolean;
    bookIndex: number | null;
    url: string;
  }>({
    isOpen: false,
    bookIndex: null,
    url: '',
  });

  const allExistingSubjects = useMemo<string[]>(() => {
    const set = new Set<string>();
    items.forEach((it) => it.subjects.forEach((s) => set.add(s)));
    return Array.from(set).sort();
  }, [items]);

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

  const getFieldLabel = (field: SyncFieldType): string => {
    switch (field) {
      case 'author':
        return t('book.author_lower', 'author');
      case 'series':
        return t('book.series_lower', 'series');
      case 'publisher':
        return t('book.publisher_lower', 'publisher');
      case 'language':
        return t('book.language_lower', 'language');
      case 'age_rating':
        return t('book.age_rating_lower', 'age rating');
      case 'subjects':
        return t('book.tags_lower', 'tags');
      case 'description':
        return t('book.description_lower', 'description');
      case 'cover':
        return t('book.cover_lower', 'cover');
      default:
        return field;
    }
  };

  // Open Sync confirmation modal for a specific field on a book
  const handleOpenSyncModal = (sourceIndex: number, field: SyncFieldType) => {
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
          pendingCover: sourceItem.pendingCover || (sourceItem.coverPreview ? { type: 'url', value: sourceItem.coverPreview } : null),
        };
        break;
    }

    // Default: select all books in the batch
    const allIds = new Set<string>(items.map((it) => it.id));

    setSyncState({
      isOpen: true,
      sourceIndex,
      field,
      fieldName: getFieldLabel(field),
      fieldValue,
      selectedBookIds: allIds,
      subjectMode: field === 'subjects' ? 'append' : undefined,
      selectedTagToRemove:
        field === 'subjects' && Array.isArray(fieldValue) && fieldValue.length > 0
          ? fieldValue[0]
          : '',
    });
  };

  // Execute sync field to selected books
  const handleApplySync = () => {
    const { field, fieldValue, selectedBookIds, subjectMode, selectedTagToRemove } = syncState;

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
            if (subjectMode === 'append') {
              const toAdd = (fieldValue as string[]) || [];
              updates.subjects = Array.from(new Set([...item.subjects, ...toAdd]));
            } else if (subjectMode === 'remove') {
              const tagToRemove = selectedTagToRemove?.trim().toLowerCase();
              if (tagToRemove) {
                updates.subjects = item.subjects.filter((s) => s.toLowerCase() !== tagToRemove);
              }
            } else {
              updates.subjects = [...(fieldValue as string[])];
            }
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
            const rawVal = item.pendingCover.value as string;
            if (rawVal.startsWith('http://') || rawVal.startsWith('https://')) {
              await bookService.updateCover(item.id, { cover_url: rawVal });
            } else {
              try {
                const fullMediaUrl = getMediaUrl(rawVal, item.id);
                const res = await fetch(fullMediaUrl);
                if (res.ok) {
                  const blob = await res.blob();
                  const file = new File([blob], 'cover.jpg', { type: blob.type || 'image/jpeg' });
                  await bookService.updateCover(item.id, { cover: file });
                }
              } catch (err) {
                console.error('Failed to copy cover image blob:', err);
              }
            }
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
                className="btn btn-sm btn-circle bg-base-200 hover:bg-base-300 text-base-content border border-base-300 shadow-sm flex items-center justify-center transition-all hover:scale-105"
                aria-label={t("common.close", "Close")}
              >
                <X className="w-4 h-4 text-base-content stroke-[2.5]" />
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
                {/* Book Card Header */}
                <div className="flex items-center justify-between gap-3 pb-3 mb-5 border-b border-base-200/80">
                  <div className="flex items-center gap-2.5 min-w-0">
                    <span className="font-mono text-xs font-bold px-2.5 py-1 rounded-lg bg-primary/10 text-primary border border-primary/20 shrink-0">
                      #{index + 1}
                    </span>
                    <span className="text-sm font-bold text-base-content truncate" title={item.original.title}>
                      {item.original.title}
                    </span>
                  </div>
                  {item.modified && (
                    <span className="badge badge-sm badge-warning font-semibold text-[11px] gap-1 shrink-0">
                      <Check className="w-3 h-3" />
                      {t('common.modified', 'Modified')}
                    </span>
                  )}
                </div>

                {/* Main Content: Left Column (Cover) & Right Column (Form Fields) */}
                <div className="flex flex-col md:flex-row items-start gap-6">
                  {/* Left Column: Cover Image & Actions */}
                  <div className="w-full md:w-36 shrink-0 flex flex-col items-center gap-3">
                    <div className="w-32 h-44 sm:w-36 sm:h-50 rounded-2xl bg-base-300 border border-base-300/80 overflow-hidden shadow-md relative group flex items-center justify-center">
                      {item.coverPreview ? (
                        <img
                          src={
                            item.coverPreview.startsWith('blob:') || item.coverPreview.startsWith('http')
                              ? item.coverPreview
                              : getMediaUrl(item.coverPreview, item.id, item.original.updated_at)
                          }
                          alt={item.title}
                          className="w-full h-full object-cover"
                        />
                      ) : (
                        <div className="flex flex-col items-center gap-1.5 text-base-content/40">
                          <ImageIcon className="w-8 h-8" />
                          <span className="text-[10px] font-bold uppercase">{t('library.no_cover')}</span>
                        </div>
                      )}
                      {/* Hover Overlay */}
                      <label className="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 flex flex-col items-center justify-center gap-1.5 cursor-pointer transition-opacity text-white p-2">
                        <Upload className="w-5 h-5 text-primary" />
                        <span className="text-[11px] font-bold text-center leading-tight">{t('library.change_image')}</span>
                        <input
                          type="file"
                          accept="image/*"
                          className="hidden"
                          onChange={(e) => {
                            const file = e.target.files?.[0];
                            if (file) {
                              const preview = URL.createObjectURL(file);
                              updateItem(index, {
                                coverPreview: preview,
                                pendingCover: { type: 'file', value: file },
                              });
                            }
                          }}
                        />
                      </label>
                    </div>

                    {/* Cover Actions Button Group */}
                    <div className="flex flex-col gap-1.5 w-full">
                      <div className="grid grid-cols-2 gap-1.5 w-full">
                        <label className="btn btn-xs btn-outline btn-primary gap-1 h-7 min-h-0 px-1 text-[11px] font-semibold cursor-pointer rounded-lg" title={t('library.upload_image_title')}>
                          <Upload className="w-3 h-3" />
                          <span>{t('library.upload_image')}</span>
                          <input
                            type="file"
                            accept="image/*"
                            className="hidden"
                            onChange={(e) => {
                              const file = e.target.files?.[0];
                              if (file) {
                                const preview = URL.createObjectURL(file);
                                updateItem(index, {
                                  coverPreview: preview,
                                  pendingCover: { type: 'file', value: file },
                                });
                              }
                            }}
                          />
                        </label>
                        <button
                          type="button"
                          onClick={() => {
                            setCoverUrlModal({
                              isOpen: true,
                              bookIndex: index,
                              url: item.coverPreview?.startsWith("http") ? item.coverPreview : "",
                            });
                          }}
                          className="btn btn-xs btn-ghost border border-base-300 gap-1 h-7 min-h-0 px-1 text-[11px] font-semibold rounded-lg hover:bg-base-200"
                          title={t('admin.paste_image_url')}
                        >
                          <LinkIcon className="w-3 h-3" />
                          <span>{t('common.url')}</span>
                        </button>
                      </div>

                      <button
                        type="button"
                        onClick={() => handleOpenSyncModal(index, 'cover')}
                        className="btn btn-xs btn-outline btn-secondary gap-1 h-7 min-h-0 text-[11px] font-semibold rounded-lg w-full"
                        title={t('library.apply_cover_to_others')}
                      >
                        <Copy className="w-3 h-3" />
                        <span>{t('library.sync_cover', 'Sync cover')}</span>
                      </button>
                    </div>
                  </div>

                  {/* Right Column: Metadata Form Fields */}
                  <div className="flex-1 min-w-0 space-y-4 w-full">
                    {/* Title */}
                    <div className="flex flex-col gap-1">
                      <label className="text-xs font-bold text-base-content/80">
                        {t('book.title', 'Title')}
                      </label>
                      <input
                        type="text"
                        value={item.title}
                        onChange={(e) => updateItem(index, { title: e.target.value })}
                        placeholder={t('book.title_placeholder', 'Book title')}
                        className="input input-md input-bordered w-full font-bold text-sm bg-base-100 rounded-xl"
                      />
                    </div>

                {/* Form Fields Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
                  {/* Author */}
                  <div className="flex flex-col gap-1.5">
                    <div className="flex items-center justify-between h-7">
                      <span className="text-xs font-bold text-base-content/80">
                        {t('book.author', 'Author')}
                      </span>
                      <button
                        type="button"
                        onClick={() => handleOpenSyncModal(index, 'author')}
                        className="btn btn-ghost btn-xs text-primary gap-1 h-7 min-h-0 px-2.5 text-xs font-semibold rounded-lg hover:bg-primary/10"
                      >
                        <Copy className="w-3.5 h-3.5" />
                        {t('library.sync_field_to_books', 'Sync')}
                      </button>
                    </div>
                    <input
                      type="text"
                      value={item.author}
                      onChange={(e) => updateItem(index, { author: e.target.value })}
                      placeholder="e.g. J.K. Rowling"
                      className="input input-md input-bordered w-full bg-base-100 text-sm rounded-xl font-medium"
                    />
                  </div>

                  {/* Publisher */}
                  <div className="flex flex-col gap-1.5">
                    <div className="flex items-center justify-between h-7">
                      <span className="text-xs font-bold text-base-content/80">
                        {t('book.publisher', 'Publisher')}
                      </span>
                      <button
                        type="button"
                        onClick={() => handleOpenSyncModal(index, 'publisher')}
                        className="btn btn-ghost btn-xs text-primary gap-1 h-7 min-h-0 px-2.5 text-xs font-semibold rounded-lg hover:bg-primary/10"
                      >
                        <Copy className="w-3.5 h-3.5" />
                        {t('library.sync_field_to_books', 'Sync')}
                      </button>
                    </div>
                    <input
                      type="text"
                      value={item.publisher}
                      onChange={(e) => updateItem(index, { publisher: e.target.value })}
                      placeholder="e.g. Bloomsbury"
                      className="input input-md input-bordered w-full bg-base-100 text-sm rounded-xl font-medium"
                    />
                  </div>

                  {/* Language */}
                  <div className="flex flex-col gap-1.5">
                    <div className="flex items-center justify-between h-7">
                      <span className="text-xs font-bold text-base-content/80">
                        {t('book.language', 'Language')}
                      </span>
                      <button
                        type="button"
                        onClick={() => handleOpenSyncModal(index, 'language')}
                        className="btn btn-ghost btn-xs text-primary gap-1 h-7 min-h-0 px-2.5 text-xs font-semibold rounded-lg hover:bg-primary/10"
                      >
                        <Copy className="w-3.5 h-3.5" />
                        {t('library.sync_field_to_books', 'Sync')}
                      </button>
                    </div>
                    <input
                      type="text"
                      value={item.language}
                      onChange={(e) => updateItem(index, { language: e.target.value })}
                      placeholder="vi, en, ja..."
                      className="input input-md input-bordered w-full bg-base-100 text-sm font-mono rounded-xl font-medium"
                    />
                  </div>

                  {/* Series & Series Index Group (Spans 2 cols on md/lg) */}
                  <div className="flex flex-col gap-1.5 md:col-span-2 lg:col-span-2">
                    <div className="flex items-center justify-between h-7">
                      <div className="flex items-center justify-between flex-1 pr-2">
                        <span className="text-xs font-bold text-base-content/80">
                          {t('book.series', 'Series')}
                        </span>
                        <button
                          type="button"
                          onClick={() => handleOpenSyncModal(index, 'series')}
                          className="btn btn-ghost btn-xs text-primary gap-1 h-7 min-h-0 px-2.5 text-xs font-semibold rounded-lg hover:bg-primary/10"
                        >
                          <Copy className="w-3.5 h-3.5" />
                          {t('library.sync_field_to_books', 'Sync')}
                        </button>
                      </div>
                      <span className="text-xs font-bold text-base-content/80 w-24 pl-2 border-l border-base-300">
                        {t('book.series_index', 'Index')}
                      </span>
                    </div>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={item.series}
                        onChange={(e) => updateItem(index, { series: e.target.value })}
                        placeholder="e.g. Harry Potter"
                        className="input input-md input-bordered flex-1 bg-base-100 text-sm rounded-xl min-w-0 font-medium"
                      />
                      <input
                        type="text"
                        value={item.series_index}
                        onChange={(e) => updateItem(index, { series_index: e.target.value })}
                        placeholder="1, 2, 3..."
                        className="input input-md input-bordered w-24 bg-base-100 text-sm font-mono rounded-xl shrink-0 text-center font-medium"
                      />
                    </div>
                  </div>

                  {/* Age Rating */}
                  <div className="flex flex-col gap-1.5">
                    <div className="flex items-center justify-between h-7">
                      <span className="text-xs font-bold text-base-content/80">
                        {t('book.age_rating', 'Age Rating')}
                      </span>
                      <button
                        type="button"
                        onClick={() => handleOpenSyncModal(index, 'age_rating')}
                        className="btn btn-ghost btn-xs text-primary gap-1 h-7 min-h-0 px-2.5 text-xs font-semibold rounded-lg hover:bg-primary/10"
                      >
                        <Copy className="w-3.5 h-3.5" />
                        {t('library.sync_field_to_books', 'Sync')}
                      </button>
                    </div>
                    <select
                      value={item.age_rating}
                      onChange={(e) => updateItem(index, { age_rating: e.target.value })}
                      className="select select-md select-bordered w-full bg-base-100 text-sm rounded-xl font-medium"
                    >
                      <option value="">{t('common.none', 'None')}</option>
                      <option value="safe">{t('book.age_safe')}</option>
                      <option value="teen">{t('book.age_teen')}</option>
                      <option value="mature">{t('book.age_mature')}</option>
                      <option value="explicit">{t('book.age_explicit')}</option>
                    </select>
                  </div>
                </div>

                {/* Subjects / Tags row */}
                <div className="mt-5 pt-4 border-t border-base-200/80">
                  <div className="flex items-center justify-between h-7 mb-2">
                    <span className="text-xs font-bold text-base-content/80">
                      {t('book.tags', 'Tags / Subjects')}
                    </span>
                    <button
                      type="button"
                      onClick={() => handleOpenSyncModal(index, 'subjects')}
                      className="btn btn-ghost btn-xs text-primary gap-1 h-7 min-h-0 px-2.5 text-xs font-semibold rounded-lg hover:bg-primary/10"
                    >
                      <Copy className="w-3.5 h-3.5" />
                      {t('library.sync_field_to_books', 'Sync')}
                    </button>
                  </div>
                  <div className="flex flex-wrap gap-2 items-center p-3.5 bg-base-200/40 border border-base-200 rounded-xl min-h-13">
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
                    <div className="flex items-center gap-2 flex-1 min-w-55">
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
                        className="btn btn-sm btn-primary rounded-lg gap-1 font-bold"
                      >
                        <Plus className="w-3.5 h-3.5" />
                        {t('common.add', 'Add')}
                      </button>
                    </div>
                  </div>
                </div>

                {/* Description row */}
                <div className="mt-4">
                  <div className="flex items-center justify-between h-7 mb-2">
                    <span className="text-xs font-bold text-base-content/80">
                      {t('book.description', 'Description')}
                    </span>
                    <button
                      type="button"
                      onClick={() => handleOpenSyncModal(index, 'description')}
                      className="btn btn-ghost btn-xs text-primary gap-1 h-7 min-h-0 px-2.5 text-xs font-semibold rounded-lg hover:bg-primary/10"
                    >
                      <Copy className="w-3.5 h-3.5" />
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
                className="btn btn-ghost btn-sm sm:btn-md rounded-xl font-semibold"
              >
                {t('common.cancel', 'Cancel')}
              </button>

              <button
                type="button"
                onClick={handleSaveAll}
                disabled={isSaving || items.filter((it) => it.modified).length === 0}
                className="btn btn-primary btn-sm sm:btn-md gap-2 font-bold rounded-xl px-6 shadow-lg shadow-primary/20"
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
        <dialog className="modal modal-open z-60 bg-black/60 backdrop-blur-sm">
          <div className="modal-box w-11/12 max-w-2xl 2xl:max-w-3xl bg-base-100 shadow-2xl p-0 overflow-hidden flex flex-col max-h-[90vh] rounded-3xl border border-base-300">
            {/* Sync Header */}
            <header className="px-6 py-4 border-b border-base-200 bg-base-200/50 flex items-center justify-between shrink-0">
              <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-xl bg-primary/10 text-primary">
                  <Copy className="w-5 h-5" />
                </div>
                <div>
                  <h4 className="font-black text-lg text-base-content">
                    {t('library.sync_field_title', 'Sync {{field}} to other books', {
                      field: syncState.fieldName,
                    })}
                  </h4>
                  <p className="text-xs text-base-content/60 mt-0.5">
                    {t('library.sync_modal_subtitle', 'Choose which books will receive this value')}
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setSyncState((prev) => ({ ...prev, isOpen: false }))}
                className="btn btn-sm btn-circle bg-base-200 hover:bg-base-300 text-base-content border border-base-300 shadow-sm flex items-center justify-center transition-all hover:scale-105"
                aria-label={t("common.close", "Close")}
              >
                <X className="w-4 h-4 text-base-content stroke-[2.5]" />
              </button>
            </header>

            {/* Sync Body */}
            <div className="p-6 space-y-5 overflow-y-auto flex-1">
              {/* Value Preview Banner */}
              <div className="rounded-2xl bg-primary/5 border border-primary/15 p-4.5">
                <span className="text-xs font-bold text-primary block uppercase tracking-wider mb-1.5">
                  {t('library.value_to_sync', 'Value to synchronize')}:
                </span>
                <div className="text-base font-bold text-base-content wrap-break-word leading-relaxed">
                  {typeof syncState.fieldValue === 'string'
                    ? syncState.fieldValue || <span className="italic font-normal opacity-50">({t('common.empty', 'Empty')})</span>
                    : Array.isArray(syncState.fieldValue)
                    ? syncState.fieldValue.length > 0 ? syncState.fieldValue.join(', ') : <span className="italic font-normal opacity-50">({t('common.empty', 'Empty')})</span>
                    : syncState.field === 'cover'
                    ? t('library.cover_image', 'Cover image')
                    : String(syncState.fieldValue)}
                </div>
              </div>

              {/* Subject Operation Mode Selector (when syncing tags/subjects) */}
              {syncState.field === 'subjects' && (
                <div className="space-y-3 p-4 bg-base-200/50 rounded-2xl border border-base-200">
                  <span className="text-xs font-bold text-base-content/80 block uppercase tracking-wider">
                    {t('library.subject_operation_mode', 'Tag / Subject apply mode')}:
                  </span>
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                    <button
                      type="button"
                      onClick={() => setSyncState((prev) => ({ ...prev, subjectMode: 'append' }))}
                      className={`p-3 rounded-xl border text-left transition-all ${
                        syncState.subjectMode === 'append'
                          ? 'border-primary bg-primary/10 text-primary font-bold shadow-xs'
                          : 'border-base-300 bg-base-100 opacity-70 hover:opacity-100'
                      }`}
                    >
                      <div className="text-xs font-bold">{t('library.tag_mode_append')}</div>
                      <div className="text-[11px] font-normal opacity-70 mt-0.5">{t('library.tag_mode_append_desc')}</div>
                    </button>
                    <button
                      type="button"
                      onClick={() => setSyncState((prev) => ({ ...prev, subjectMode: 'remove' }))}
                      className={`p-3 rounded-xl border text-left transition-all ${
                        syncState.subjectMode === 'remove'
                          ? 'border-error bg-error/10 text-error font-bold shadow-xs'
                          : 'border-base-300 bg-base-100 opacity-70 hover:opacity-100'
                      }`}
                    >
                      <div className="text-xs font-bold">{t('library.tag_mode_remove')}</div>
                      <div className="text-[11px] font-normal opacity-70 mt-0.5">{t('library.tag_mode_remove_desc')}</div>
                    </button>
                    <button
                      type="button"
                      onClick={() => setSyncState((prev) => ({ ...prev, subjectMode: 'replace' }))}
                      className={`p-3 rounded-xl border text-left transition-all ${
                        syncState.subjectMode === 'replace'
                          ? 'border-primary bg-primary/10 text-primary font-bold shadow-xs'
                          : 'border-base-300 bg-base-100 opacity-70 hover:opacity-100'
                      }`}
                    >
                      <div className="text-xs font-bold">{t('library.tag_mode_replace')}</div>
                      <div className="text-[11px] font-normal opacity-70 mt-0.5">{t('library.tag_mode_replace_desc')}</div>
                    </button>
                  </div>

                  {syncState.subjectMode === 'remove' && (
                    <div className="mt-3 pt-3 border-t border-base-300 space-y-2">
                      <label className="text-xs font-semibold text-base-content/80 block">
                        {t('library.select_tag_to_remove', 'Select a tag to remove')}:
                      </label>
                      <div className="flex flex-wrap gap-1.5 max-h-32 overflow-y-auto">
                        {allExistingSubjects.map((tag) => (
                          <button
                            key={tag}
                            type="button"
                            onClick={() => setSyncState((prev) => ({ ...prev, selectedTagToRemove: tag }))}
                            className={`badge badge-sm cursor-pointer py-2.5 px-3 rounded-lg border transition-all ${
                              syncState.selectedTagToRemove?.toLowerCase() === tag.toLowerCase()
                                ? 'badge-error text-white font-bold'
                                : 'badge-ghost hover:badge-neutral'
                            }`}
                          >
                            {tag}
                          </button>
                        ))}
                      </div>
                      <input
                        type="text"
                        value={syncState.selectedTagToRemove || ''}
                        onChange={(e) => setSyncState((prev) => ({ ...prev, selectedTagToRemove: e.target.value }))}
                        placeholder={t('library.or_type_tag_to_remove', 'Or type a tag name to remove...')}
                        className="input input-sm input-bordered w-full rounded-xl text-xs mt-2"
                      />
                    </div>
                  )}
                </div>
              )}

              {/* Target Selection Tools */}
              <div className="flex items-center justify-between text-sm px-1">
                <span className="font-bold text-base-content/80 text-sm">
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
                    className="btn btn-ghost btn-sm text-primary font-bold hover:bg-primary/10 rounded-lg"
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
                    className="btn btn-ghost btn-sm text-base-content/60 hover:bg-base-200 rounded-lg"
                  >
                    {t('common.deselect_all', 'Deselect all')}
                  </button>
                </div>
              </div>

              {/* Target Books Checkbox List */}
              <div className="space-y-2.5 max-h-80 overflow-y-auto pr-1">
                {items.map((item, idx) => {
                  const isChecked = syncState.selectedBookIds.has(item.id);
                  const isSource = idx === syncState.sourceIndex;

                  return (
                    <label
                      key={item.id}
                      className={`flex items-center gap-4 p-3.5 rounded-2xl border cursor-pointer transition-all ${
                        isChecked
                          ? 'border-primary/40 bg-primary/5 text-base-content shadow-xs'
                          : 'border-base-200 bg-base-100 text-base-content/70 hover:border-base-300'
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
                        className="checkbox checkbox-md checkbox-primary rounded-lg shrink-0"
                      />
                      <div className="w-12 h-16 rounded-xl bg-base-300 overflow-hidden shrink-0 border border-base-300 shadow-xs flex items-center justify-center">
                        {item.coverPreview ? (
                          <img
                            src={
                              item.coverPreview.startsWith('blob:') || item.coverPreview.startsWith('http')
                                ? item.coverPreview
                                : getMediaUrl(item.coverPreview, item.id, item.original.updated_at)
                            }
                            alt={item.title}
                            className="w-full h-full object-cover"
                          />
                        ) : (
                          <div className="text-xs font-bold opacity-40">
                            #{idx + 1}
                          </div>
                        )}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="font-bold text-sm text-base-content truncate flex items-center gap-2">
                          <span className="truncate">{item.title}</span>
                          {isSource && (
                            <span className="badge badge-sm badge-primary font-bold shrink-0">
                              {t('common.source', 'Source')}
                            </span>
                          )}
                        </div>
                        <div className="text-xs text-base-content/60 truncate mt-0.5">
                          {item.author || item.original.author_name || t('book.author_unknown', 'Unknown author')}
                        </div>
                      </div>
                    </label>
                  );
                })}
              </div>
            </div>

            {/* Sync Footer */}
            <footer className="px-6 py-4 border-t border-base-200 bg-base-200/50 flex items-center justify-end gap-3 shrink-0">
              <button
                type="button"
                onClick={() => setSyncState((prev) => ({ ...prev, isOpen: false }))}
                className="btn btn-md btn-ghost rounded-xl font-semibold"
              >
                {t('common.cancel', 'Cancel')}
              </button>
              <button
                type="button"
                onClick={handleApplySync}
                disabled={syncState.selectedBookIds.size === 0}
                className="btn btn-md btn-primary gap-2 font-bold rounded-xl px-6 shadow-md shadow-primary/20"
              >
                <Check className="w-4 h-4" />
                {t('common.apply', 'Apply')} ({syncState.selectedBookIds.size})
              </button>
            </footer>
          </div>
        </dialog>
      )}

      {/* ======================= COVER URL INPUT MODAL ======================= */}
      {coverUrlModal.isOpen && coverUrlModal.bookIndex !== null && (
        <dialog className="modal modal-open z-70 bg-black/60 backdrop-blur-xs">
          <div className="modal-box max-w-md p-6 rounded-3xl border border-base-300 shadow-2xl bg-base-100 animate-in fade-in zoom-in-95 duration-150">
            <div className="flex items-center justify-between pb-3 mb-4 border-b border-base-200">
              <div className="flex items-center gap-2.5 font-bold text-base text-base-content">
                <div className="grid h-9 w-9 place-items-center rounded-xl bg-primary/10 text-primary">
                  <LinkIcon className="w-4 h-4" />
                </div>
                <div>
                  <div className="font-bold text-base leading-tight">
                    {t('admin.cover_url_title', 'Enter cover image link')}
                  </div>
                  <div className="text-xs text-base-content/50 font-normal mt-0.5">
                    {t('library.book_num', { n: coverUrlModal.bookIndex + 1 })}: {items[coverUrlModal.bookIndex]?.title}
                  </div>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setCoverUrlModal({ isOpen: false, bookIndex: null, url: '' })}
                className="btn btn-sm btn-circle bg-base-200 hover:bg-base-300 text-base-content border border-base-300 shadow-sm flex items-center justify-center transition-all hover:scale-105"
                aria-label={t("common.close", "Close")}
              >
                <X className="w-4 h-4 text-base-content stroke-[2.5]" />
              </button>
            </div>

            <form
              onSubmit={(e) => {
                e.preventDefault();
                if (coverUrlModal.url.trim() && coverUrlModal.bookIndex !== null) {
                  updateItem(coverUrlModal.bookIndex, {
                    coverPreview: coverUrlModal.url.trim(),
                    pendingCover: { type: 'url', value: coverUrlModal.url.trim() },
                  });
                }
                setCoverUrlModal({ isOpen: false, bookIndex: null, url: '' });
              }}
              className="space-y-4"
            >
              <div>
                <label className="text-xs font-bold text-base-content/80 mb-1.5 block">
                  {t('admin.image_url_label', 'Online cover image URL')}
                </label>
                <input
                  type="url"
                  autoFocus
                  required
                  placeholder="https://example.com/cover.jpg"
                  value={coverUrlModal.url}
                  onChange={(e) =>
                    setCoverUrlModal((prev) => ({ ...prev, url: e.target.value }))
                  }
                  className="input input-bordered input-md w-full bg-base-100 rounded-xl text-sm font-medium focus:outline-hidden"
                />
                <span className="text-[11px] text-base-content/50 mt-1 block">
                  {t('admin.cover_url_hint', 'Supports direct image formats (JPG, PNG, WEBP, GIF)')}
                </span>
              </div>

              {/* URL Preview if text is entered */}
              {coverUrlModal.url.trim() && (
                <div className="p-3 bg-base-200/50 rounded-2xl border border-base-200 flex items-center gap-3">
                  <div className="w-12 h-16 rounded-xl bg-base-300 overflow-hidden shrink-0 border border-base-300 flex items-center justify-center">
                    <img
                      src={coverUrlModal.url}
                      alt={t('common.preview')}
                      className="w-full h-full object-cover"
                      onError={(e) => {
                        (e.target as HTMLElement).style.display = 'none';
                      }}
                    />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="text-xs font-bold text-base-content truncate">
                      {t('common.preview', 'Image preview')}
                    </div>
                    <div className="text-[11px] text-base-content/60 truncate font-mono mt-0.5">
                      {coverUrlModal.url}
                    </div>
                  </div>
                </div>
              )}

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setCoverUrlModal({ isOpen: false, bookIndex: null, url: '' })}
                  className="btn btn-md btn-ghost rounded-xl font-semibold"
                >
                  {t('common.cancel', 'Cancel')}
                </button>
                <button
                  type="submit"
                  disabled={!coverUrlModal.url.trim()}
                  className="btn btn-md btn-primary rounded-xl px-5 font-bold shadow-md shadow-primary/20"
                >
                  {t('common.apply', 'Apply')}
                </button>
              </div>
            </form>
          </div>
        </dialog>
      )}
    </>
  );
};
