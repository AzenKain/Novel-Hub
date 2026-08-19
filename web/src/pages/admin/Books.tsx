import { BookActionModal, CalibreImportModal, DeleteConfirmModal, ManageLibrariesModal, UploadBooksModal, ConvertBookModal, MergeAudiobookModal, BulkConvertModal, BulkEditMetadataModal } from "@/components/admin";
import { BookCard } from "@/components/ui";
import { BulkDeleteModal } from "@/components/library";
import { getMediaUrl } from "@/config/api";
import { BOOK_FILE_ACCEPT } from "@/constants";
import { useBooksQuery, useCalibreImportMutation, useCreateLibraryMutation, useDebounce, useDeleteLibraryMutation, useLibrariesQuery, useUpdateLibraryMutation } from "@/hooks";
import { fileNameFromPath, formatFileSize, parseMetadata } from "@/lib/bookDetail";
import { useAuthStore, useBookAdminStore } from "@/stores";
import type { Book, BookFile } from "@/types";
import { hasPermission } from "@/utils/permission";
import { BookOpen, DatabaseBackup, Edit3, Eye, FilePlus2, FileText, Globe, Image as ImageIcon, LayoutGrid, Link as LinkIcon, List, Loader2, RefreshCw, Save, Search, Trash2, Upload, AudioLines, Layers, Plus } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { toast } from "react-toastify";
import { useShallow } from "zustand/react/shallow";

const MERGEABLE_FORMATS = new Set(["m4a", "m4b", "mp3", "flac", "ogg", "wav", "aac"]);

export function Books() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const store = useBookAdminStore(useShallow((state) => ({
    search: state.search, selectedLibraryId: state.selectedLibraryId,
    editingBook: state.editingBook, formData: state.formData, submitting: state.submitting,
    bookFiles: state.bookFiles, uploadingBookFiles: state.uploadingBookFiles,
    coverTab: state.coverTab, epubImages: state.epubImages, loadingImages: state.loadingImages, linkUrl: state.linkUrl, coverPreview: state.coverPreview,
    searchSource: state.searchSource, onlineSearchQuery: state.onlineSearchQuery, searching: state.searching, searchResults: state.searchResults,
    showUploadModal: state.showUploadModal, uploadLibraryId: state.uploadLibraryId, uploading: state.uploading,
    uploadProgress: state.uploadProgress, uploadSpeed: state.uploadSpeed, uploadCurrentFile: state.uploadCurrentFile, uploadBytesText: state.uploadBytesText, uploadBatchInfo: state.uploadBatchInfo,
    showLibraryModal: state.showLibraryModal, newLibraryName: state.newLibraryName,
    bookToDelete: state.bookToDelete, libraryToDelete: state.libraryToDelete,
    setSearch: state.setSearch, setSelectedLibraryId: state.setSelectedLibraryId, setSearchSource: state.setSearchSource, setOnlineSearchQuery: state.setOnlineSearchQuery, setCoverTab: state.setCoverTab, setLinkUrl: state.setLinkUrl,
    setFormData: state.setFormData, setShowUploadModal: state.setShowUploadModal, setUploadLibraryId: state.setUploadLibraryId, setShowLibraryModal: state.setShowLibraryModal,
    setNewLibraryName: state.setNewLibraryName,
    setEditingBook: state.setEditingBook, setSearchResults: state.setSearchResults, setBookToDelete: state.setBookToDelete, setLibraryToDelete: state.setLibraryToDelete,
    openEditModal: state.openEditModal, handleSearchOnline: state.handleSearchOnline, handleSelectResult: state.handleSelectResult,
    handleSelectEpubImage: state.handleSelectEpubImage, handleImageUpload: state.handleImageUpload, handleLinkUpload: state.handleLinkUpload, handleEditSubmit: state.handleEditSubmit,
    handleUploadBookFiles: state.handleUploadBookFiles, handleUploadFiles: state.handleUploadFiles, deleteBook: state.deleteBook, archiveBook: state.archiveBook,
    deleteBookFile: state.deleteBookFile
  })));
  const [actionBook, setActionBook] = useState<Book | null>(null);
  const [convertBook, setConvertBook] = useState<Book | null>(null);
  const [mergeBook, setMergeBook] = useState<Book | null>(null);
  const [viewMode, setViewMode] = useState<'grid' | 'table'>('table');
  const [selectedBookIds, setSelectedBookIds] = useState<string[]>([]);
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);
  const [showBulkConvertModal, setShowBulkConvertModal] = useState(false);
  const [showBulkEditModal, setShowBulkEditModal] = useState(false);
  const [tagInput, setTagInput] = useState("");

  const handleAddTag = () => {
    const raw = tagInput.trim();
    if (!raw) return;
    const newTags = raw.split(',').map(s => s.trim().replace(/^,+|,+$/g, '')).filter(Boolean);
    const existing = formData.subjects || [];
    const merged = Array.from(new Set([...existing, ...newTags]));
    setFormData({ subjects: merged });
    setTagInput("");
  };

  const {
    search, selectedLibraryId,
    editingBook, formData, submitting,
    bookFiles, uploadingBookFiles,
    coverTab, epubImages, loadingImages, linkUrl, coverPreview,
    searchSource, onlineSearchQuery, searching, searchResults,
    showUploadModal, uploadLibraryId, uploading,
    uploadProgress, uploadSpeed, uploadCurrentFile, uploadBytesText, uploadBatchInfo,
    showLibraryModal, newLibraryName,
    bookToDelete, libraryToDelete,
    setSearch, setSelectedLibraryId, setSearchSource, setOnlineSearchQuery, setCoverTab, setLinkUrl,
    setFormData, setShowUploadModal, setUploadLibraryId, setShowLibraryModal,
    setNewLibraryName,
    setEditingBook, setSearchResults, setBookToDelete, setLibraryToDelete,
    openEditModal, handleSearchOnline, handleSelectResult,
    handleSelectEpubImage, handleImageUpload, handleLinkUpload, handleEditSubmit,
    handleUploadBookFiles, handleUploadFiles, deleteBookFile
  } = store;

  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const canImportCalibre = hasPermission(user, "calibre.sync");
  const [showCalibreModal, setShowCalibreModal] = useState(false);
  const [deleteFileId, setDeleteFileId] = useState<string | null>(null);
  const calibreImportMutation = useCalibreImportMutation();
  const { data: libraries = [] } = useLibrariesQuery();
  const createLibraryMutation = useCreateLibraryMutation();
  const updateLibraryMutation = useUpdateLibraryMutation();
  const deleteLibraryMutation = useDeleteLibraryMutation();

  const handleCreateLibrary = (e: React.SyntheticEvent) => {
    e.preventDefault();
    const name = newLibraryName.trim();
    if (!name) return;
    createLibraryMutation.mutate(name, {
      onSuccess: () => {
        setNewLibraryName("");
        toast.success(t("admin.library_created", "Library created"));
      },
    });
  };

  const debouncedSearch = useDebounce(search, 500);
  const {
    data: booksPages,
    isLoading,
    isFetching,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    refetch,
  } = useBooksQuery(useMemo(() => ({
    limit: 24,
    search: debouncedSearch || undefined,
    library_id: selectedLibraryId || undefined,
  }), [debouncedSearch, selectedLibraryId]));

  const books = useMemo(() => booksPages?.pages.flatMap((page) => page.data || []) ?? [], [booksPages]);

  const selectedBooks = useMemo(() => {
    return books.filter((b) => selectedBookIds.includes(b.id));
  }, [books, selectedBookIds]);

  const allSelectedAudioFiles = useMemo(() => {
    return selectedBooks.flatMap((b) => b.files || []).filter((f) => MERGEABLE_FORMATS.has(f.format.toLowerCase()));
  }, [selectedBooks]);

  const canBulkMerge = allSelectedAudioFiles.length >= 2;

  const canBulkConvert = selectedBooks.some((b) => b.files && b.files.length > 0);

  useEffect(() => {
    setSelectedBookIds([]);
  }, [debouncedSearch, selectedLibraryId]);

  const isAllSelected = books.length > 0 && books.every(b => selectedBookIds.includes(b.id));

  const toggleSelectAll = () => {
    if (isAllSelected) {
      setSelectedBookIds([]);
    } else {
      setSelectedBookIds(books.map(b => b.id));
    }
  };

  const toggleSelectBook = (id: string, e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    setSelectedBookIds(prev =>
      prev.includes(id) ? prev.filter(item => item !== id) : [...prev, id]
    );
  };

  const getImageAssetUrl = (imagePath: string) => {
    if (!editingBook) return "";
    return `/api/v1/reader/${editingBook.id}/asset/${imagePath}`;
  };

  return (
    <div className="flex flex-col h-full bg-base-100">
      {/* Header Bar matching Roles/Users design */}
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("admin.books", "Books")}</h1>
          <p className="text-sm text-base-content/60 mt-1">{t("admin.books_subtitle", "Manage EPUB novel files, libraries, and calibre imports.")}</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => void refetch()}
            className="btn btn-square btn-ghost btn-sm sm:btn-md"
            title={t("settings.refresh", "Refresh list")}
            disabled={isFetching}
          >
            <RefreshCw className={`h-5 w-5 ${isFetching ? "animate-spin" : ""}`} />
          </button>
        </div>
      </header>

      {/* Main Content Area */}
      <div className="flex-1 overflow-auto p-4 sm:p-6 lg:p-8">
        <div className="max-w-7xl mx-auto w-full space-y-6">
          {/* Toolbar & Action Controls */}
          <div className="flex flex-col xl:flex-row items-stretch xl:items-center justify-between gap-3 sm:gap-4 mb-6">
            {/* Search, Library Filter & View Switcher */}
            <div className="flex flex-1 flex-col sm:flex-row items-stretch sm:items-center gap-2.5 sm:gap-3 min-w-0">
              <div className="relative flex-1 min-w-[200px] sm:min-w-[240px] max-w-md">
                <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/40 pointer-events-none" />
                <input
                  type="text"
                  placeholder={t("admin.search_placeholder", "Search books...")}
                  className="input input-bordered input-sm sm:input-md w-full pl-9 bg-base-100 rounded-xl"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
              </div>

              <div className="flex items-center gap-2 shrink-0">
                <select
                  className="select select-bordered select-sm sm:select-md bg-base-100 rounded-xl min-w-[140px] sm:min-w-[160px] flex-1 sm:flex-initial"
                  value={selectedLibraryId}
                  onChange={(e) => setSelectedLibraryId(e.target.value)}
                >
                  <option value="">{t("admin.all_libraries", "All Libraries")}</option>
                  {libraries.map((lib) => (
                    <option key={lib.id} value={lib.id}>{lib.name}</option>
                  ))}
                </select>

                <div className="join border border-base-200 rounded-xl p-0.5 bg-base-100 shrink-0">
                  <button
                    onClick={() => setViewMode('grid')}
                    className={`join-item btn btn-xs sm:btn-sm ${viewMode === 'grid' ? 'btn-primary !text-white font-bold' : 'btn-ghost text-base-content/70'}`}
                    title={t('admin.grid_view', 'Grid view')}
                  >
                    <LayoutGrid className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => setViewMode('table')}
                    className={`join-item btn btn-xs sm:btn-sm ${viewMode === 'table' ? 'btn-primary !text-white font-bold' : 'btn-ghost text-base-content/70'}`}
                    title={t('admin.table_view', 'Table view')}
                  >
                    <List className="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>

            {/* Action Buttons */}
            <div className="flex items-center gap-2 flex-wrap xl:flex-nowrap shrink-0 justify-start xl:justify-end">
              <button
                onClick={() => setShowLibraryModal(true)}
                className="btn btn-outline btn-sm sm:btn-md gap-2 rounded-xl text-xs sm:text-sm"
              >
                {t("admin.manage_libraries", "Manage Libraries")}
              </button>
              {canImportCalibre && (
                <button
                  onClick={() => setShowCalibreModal(true)}
                  className="btn btn-outline btn-sm sm:btn-md gap-2 rounded-xl text-xs sm:text-sm"
                >
                  <DatabaseBackup className="w-4 h-4" />
                  {t("admin.calibre_import", "Import from Calibre")}
                </button>
              )}
              <button
                onClick={() => setShowUploadModal(true)}
                className="btn btn-primary btn-sm sm:btn-md gap-2 rounded-xl !text-white font-medium text-xs sm:text-sm"
              >
                <Upload className="w-4 h-4" />
                {t("admin.upload", "Upload")}
              </button>
            </div>
          </div>

        {selectedBookIds.length > 0 && (
          <div className="mb-3 px-3 py-2 bg-primary/10 rounded-xl border border-primary/20 flex items-center justify-between gap-2 text-xs">
            <div className="flex items-center gap-2">
              <span className="font-bold text-primary">
                {t("admin.selected_books", "Selected {{count}} books", { count: selectedBookIds.length })}
              </span>
              <button
                onClick={() => setSelectedBookIds([])}
                className="btn btn-ghost btn-xs text-xs opacity-70 hover:opacity-100 h-6 min-h-0"
              >
                {t("common.deselect_all", "Clear selection")}
              </button>
            </div>
            <div className="flex items-center gap-2">
              {canBulkMerge && (
                <button
                  onClick={() => {
                    const targetBook = selectedBooks[0];
                    if (targetBook) {
                      setMergeBook({
                        ...targetBook,
                        files: allSelectedAudioFiles
                      });
                    }
                  }}
                  className="btn btn-outline btn-xs gap-1.5 h-7 min-h-0"
                >
                  <AudioLines className="w-3.5 h-3.5" />
                  {t("audiobook.merge", "Merge into audiobook")}
                </button>
              )}
              <button
                onClick={() => setShowBulkEditModal(true)}
                className="btn btn-outline btn-xs gap-1.5 h-7 min-h-0"
              >
                <Layers className="w-3.5 h-3.5" />
                {t("library.bulk_edit_metadata_title", "Edit Metadata")}
              </button>
              {canBulkConvert && (
                <button
                  onClick={() => setShowBulkConvertModal(true)}
                  className="btn btn-outline btn-xs gap-1.5 h-7 min-h-0"
                >
                  <FileText className="w-3.5 h-3.5" />
                  {t("book.convert_format", "Convert format")}
                </button>
              )}
              <button
                onClick={() => setShowBulkDeleteModal(true)}
                className="btn btn-error btn-xs gap-1.5 text-white h-7 min-h-0"
              >
                <Trash2 className="w-3.5 h-3.5" />
                {t("admin.delete_selected_books", "Delete selected")}
              </button>
            </div>
          </div>
        )}

        {isLoading ? (
          <div className="flex items-center justify-center py-20 opacity-50">
            <Loader2 className="h-8 w-8 animate-spin text-primary mr-3" />
            {t("common.loading")}
          </div>
        ) : books.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-base-300 bg-base-100 p-12 sm:p-16 text-center flex flex-col items-center justify-center gap-3 shadow-xs">
            <div className="grid h-14 w-14 place-items-center rounded-2xl bg-primary/10 text-primary mb-1">
              <BookOpen className="h-7 w-7" />
            </div>
            <div>
              <h3 className="text-base sm:text-lg font-bold text-base-content">{t("common.no_data", "No Data Found")}</h3>
              <p className="text-xs sm:text-sm text-base-content/60 mt-1 max-w-sm">{t("admin.no_books_hint", "Configure a library in the system and upload EPUB files.")}</p>
            </div>
          </div>
        ) : (
          <>
            {viewMode === 'grid' ? (
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 mb-8">
                {books.map((book) => (
                  <BookCard
                    key={book.id}
                    book={book}
                    onClick={() => setActionBook(book)}
                    selected={selectedBookIds.includes(book.id)}
                    onSelectToggle={() => toggleSelectBook(book.id)}
                  />
                ))}
              </div>
            ) : (
              <div className="overflow-x-auto border border-base-200 rounded-2xl bg-base-100 shadow-xs mb-8">
                <table className="table table-zebra w-full">
                  <thead>
                    <tr className="bg-base-200/50 text-base-content/70">
                      <th className="w-12 text-center">
                        <input
                          type="checkbox"
                          className="checkbox checkbox-sm checkbox-primary"
                          checked={isAllSelected}
                          onChange={toggleSelectAll}
                        />
                      </th>
                      <th className="w-[34%] max-w-[340px]">{t("admin.books", "Books")}</th>
                      <th className="w-[16%] max-w-[180px]">{t("admin.author", "Author")}</th>
                      <th className="w-[26%] max-w-[280px]">{t("admin.series", "Series")}</th>
                      <th className="text-right">{t("admin.actions", "Actions")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {books.map((book) => {
                      const isSelected = selectedBookIds.includes(book.id);
                      const meta = book.metadata_json ? parseMetadata(book.metadata_json) : {};
                      const seriesName = meta.series ? (meta.series_index ? `${meta.series} #${meta.series_index}` : meta.series) : "-";
                      const authorName = book.author_name || book.author_id || t('library.unknown_author', 'Unknown');
                      const file_id = book.files?.[0]?.id;

                      return (
                        <tr key={book.id} className={`hover ${isSelected ? "bg-primary/5" : ""}`}>
                          <td className="text-center" onClick={(e) => e.stopPropagation()}>
                            <input
                              type="checkbox"
                              className="checkbox checkbox-sm checkbox-primary"
                              checked={isSelected}
                              onChange={() => toggleSelectBook(book.id)}
                            />
                          </td>
                          <td>
                            <div className="flex items-center gap-3 cursor-pointer" onClick={() => setActionBook(book)}>
                              <div className="w-10 h-14 bg-base-300 rounded shadow-xs overflow-hidden shrink-0 relative flex items-center justify-center">
                                {book.cover_url ? (
                                  <img
                                    src={getMediaUrl(book.cover_url)}
                                    alt={book.title}
                                    className="w-full h-full object-cover"
                                  />
                                ) : (
                                  <span className="text-[9px] font-bold opacity-40 text-center px-1">NOVEL</span>
                                )}
                              </div>
                              <div className="min-w-0 flex-1">
                                <div
                                  className="font-bold text-sm text-base-content line-clamp-2 break-words hover:text-primary transition-colors"
                                  title={book.title}
                                >
                                  {book.title}
                                </div>
                                {book.files && book.files.length > 0 && (
                                  <div className="text-xs opacity-50 flex items-center gap-1.5 mt-0.5">
                                    <span className="badge badge-ghost badge-xs uppercase font-mono">{book.files[0].format}</span>
                                    <span>{formatFileSize(book.files[0].size_bytes)}</span>
                                  </div>
                                )}
                              </div>
                            </div>
                          </td>
                          <td className="text-sm opacity-80">
                            <span className="line-clamp-2 break-words" title={authorName}>
                              {authorName}
                            </span>
                          </td>
                          <td className="text-sm opacity-80">
                            <span
                              className={seriesName === "-" ? "opacity-50" : "line-clamp-2 break-words"}
                              title={seriesName}
                            >
                              {seriesName}
                            </span>
                          </td>
                          <td className="text-right">
                            <div className="flex items-center justify-end gap-1" onClick={(e) => e.stopPropagation()}>
                              <button
                                onClick={() => {
                                  navigate(`/reader/${book.id}${file_id ? `?file_id=${encodeURIComponent(file_id)}` : ""}`);
                                }}
                                className="btn btn-ghost btn-xs btn-square"
                                title={t("reader.read", "Read")}
                              >
                                <Eye className="w-4 h-4 text-primary" />
                              </button>
                              <button
                                onClick={() => openEditModal(book)}
                                className="btn btn-ghost btn-xs btn-square"
                                title={t("admin.edit", "Edit")}
                              >
                                <Edit3 className="w-4 h-4 text-base-content/70" />
                              </button>
                              <button
                                onClick={() => setBookToDelete(book)}
                                className="btn btn-ghost btn-xs btn-square text-error"
                                title={t("admin.delete", "Delete")}
                              >
                                <Trash2 className="w-4 h-4" />
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}

            {hasNextPage && (
              <div className="flex justify-center mt-auto pt-8">
                <button
                  className="btn btn-primary btn-outline"
                  onClick={() => void fetchNextPage()}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage ? (
                    <span className="loading loading-spinner loading-sm"></span>
                  ) : (
                    t("common.load_more")
                  )}
                </button>
              </div>
            )}
          </>
        )}
        </div>
      </div>

      <BookActionModal
        book={actionBook}
        onClose={() => setActionBook(null)}
        onRead={(book) => {
          const file_id = book.files?.[0]?.id;
          setActionBook(null);
          navigate(
            `/reader/${book.id}${file_id ? `?file_id=${encodeURIComponent(file_id)}` : ""}`,
          );
        }}
        onEdit={(book) => {
          setActionBook(null);
          openEditModal(book);
        }}
        onDelete={(book) => {
          setActionBook(null);
          setBookToDelete(book);
        }}
        onArchive={(book) => {
          setActionBook(null);
          void store.archiveBook(book.id, book.status !== "archived");
        }}
        onConvert={(book) => {
          setActionBook(null);
          setConvertBook(book);
        }}
        onMerge={(book) => {
          setActionBook(null);
          setMergeBook(book);
        }}
      />

      {/* ===================== EDIT METADATA MODAL ===================== */}
      <dialog className={`modal ${editingBook ? "modal-open" : ""}`}>
        <div className="modal-box w-11/12 max-w-[96vw] 2xl:max-w-[1700px] h-[95vh] max-h-[96vh] bg-base-100 shadow-2xl p-0 overflow-hidden flex flex-col rounded-2xl">
          {/* Header */}
          <header className="px-6 py-4 border-b border-base-200 bg-base-200/30 flex items-center justify-between shrink-0">
            <div>
              <h3 className="text-xl font-black">{t('admin.metadata', 'Metadata')}</h3>
              <p className="text-xs opacity-50 mt-0.5 font-mono truncate max-w-sm">{editingBook?.id}</p>
            </div>
            <button onClick={() => setEditingBook(null)} className="btn btn-sm btn-circle btn-ghost">✕</button>
          </header>

          {/* Body: 2-column layout */}
          <div className="flex-1 overflow-y-auto">
            <div className="grid grid-cols-1 lg:grid-cols-[340px_1fr] gap-0 min-h-0">

              {/* ========== LEFT COLUMN: Cover Section ========== */}
              <div className="flex flex-col gap-4 p-6 border-r border-base-200 bg-base-200/10">
                <span className="text-xs font-bold text-primary uppercase tracking-wider">{t('admin.cover_image', 'Cover Image')}</span>

                {/* Cover Preview */}
                <div className="flex flex-col items-center gap-2 p-3.5 bg-base-200/30 border border-base-200 rounded-xl">
                  <div className="w-44 aspect-[3/4.12] rounded-xl bg-base-300 border border-base-300 overflow-hidden shadow-md flex items-center justify-center">
                    {coverPreview ? (
                      <img src={getMediaUrl(coverPreview, editingBook?.id)} alt="Cover" loading="lazy" className="w-full h-full object-cover" />
                    ) : (
                      <div className="flex flex-col items-center justify-center text-base-content/30 gap-1.5">
                        <ImageIcon size={28} />
                        <span className="text-[10px] font-bold uppercase">{t('admin.no_cover', 'No Cover')}</span>
                      </div>
                    )}
                  </div>
                </div>

                {/* Cover Tabs */}
                <div className="flex gap-1 flex-nowrap">
                  <button type="button" onClick={() => setCoverTab("book")} className={`btn btn-xs flex-1 gap-1 whitespace-nowrap rounded-lg ${coverTab === "book" ? "btn-primary" : "btn-ghost"}`}>
                    <BookOpen size={12} /> {t('admin.in_book', 'In Book')}
                  </button>
                  <button type="button" onClick={() => setCoverTab("upload")} className={`btn btn-xs flex-1 gap-1 whitespace-nowrap rounded-lg ${coverTab === "upload" ? "btn-primary" : "btn-ghost"}`}>
                    <Upload size={12} /> {t('admin.upload', 'Upload')}
                  </button>
                  <button type="button" onClick={() => setCoverTab("link")} className={`btn btn-xs flex-1 gap-1 whitespace-nowrap rounded-lg ${coverTab === "link" ? "btn-primary" : "btn-ghost"}`}>
                    <LinkIcon size={12} /> {t('admin.url', 'URL')}
                  </button>
                </div>

                {/* Tab Content */}
                <div className="flex-1 min-h-0 overflow-y-auto">
                  {coverTab === "book" && (
                    <div className="flex flex-col gap-1.5 max-h-52 overflow-y-auto">
                      {loadingImages ? (
                        <div className="flex items-center justify-center py-6 opacity-50">
                          <Loader2 className="w-4 h-4 animate-spin mr-2" /> {t('common.loading', 'Loading...')}
                        </div>
                      ) : epubImages.length > 0 ? (
                        epubImages.map((img) => {
                          const fileName = img.split("/").pop() || img;
                          return (
                            <button
                                type="button"
                                key={img}
                                onClick={() => handleSelectEpubImage(img)}
                                className="flex items-center gap-2.5 p-2 rounded-lg hover:bg-primary/10 border border-transparent hover:border-primary/20 cursor-pointer transition-colors text-left"
                            >
                              <img src={getImageAssetUrl(img)} alt={fileName} loading="lazy" className="w-9 h-12 object-cover rounded-lg bg-base-200 border border-base-200 shrink-0" />
                              <span className="text-xs truncate min-w-0 flex-1">{fileName}</span>
                            </button>
                          );
                        })
                      ) : (
                        <div className="text-xs opacity-50 text-center py-4">{t('admin.no_images_in_epub', 'No images found in EPUB')}</div>
                      )}
                    </div>
                  )}

                  {coverTab === "upload" && (
                    <label className="flex flex-col items-center justify-center gap-2 p-4 border-2 border-dashed border-base-300 rounded-xl cursor-pointer hover:border-primary/30 hover:bg-primary/5 transition-colors min-h-25">
                      <input type="file" accept="image/png,image/jpeg,image/jpg,image/gif,image/webp" onChange={handleImageUpload} className="hidden" />
                      <Upload size={22} className="opacity-50" />
                      <span className="text-xs font-medium opacity-70">{t('admin.click_choose_image', 'Click to choose image')}</span>
                      <span className="text-[10px] opacity-40">PNG, JPEG, GIF, WebP</span>
                    </label>
                  )}

                  {coverTab === "link" && (
                    <form onSubmit={(e) => { e.preventDefault(); handleLinkUpload(); }} className="flex flex-col gap-2">
                      <span className="text-[11px] font-bold opacity-60">{t('admin.paste_image_url', 'Paste image URL')}</span>
                      <input
                        type="url"
                        placeholder="https://example.com/cover.jpg"
                        value={linkUrl}
                        onChange={e => setLinkUrl(e.target.value)}
                        className="input input-bordered input-sm w-full text-xs rounded-lg"
                      />
                      <button type="submit" disabled={!linkUrl.trim()} className="btn btn-sm btn-primary w-full rounded-lg">
                        {t('admin.apply', 'Apply')}
                      </button>
                    </form>
                  )}
                </div>

                <div className="border-t border-base-200 pt-4">
                  <div className="mb-2 flex items-center justify-between gap-2">
                    <span className="text-xs font-bold text-primary uppercase tracking-wider">{t('admin.book_files', 'Book Files')}</span>
                    <span className="text-[10px] font-bold text-base-content/40">{bookFiles.length}</span>
                  </div>
                  <label className={`mb-3 flex min-h-20 cursor-pointer flex-col items-center justify-center gap-1.5 rounded-xl border-2 border-dashed border-base-300 p-3 text-center transition-colors hover:border-primary/30 hover:bg-primary/5 ${uploadingBookFiles ? "pointer-events-none opacity-60" : ""}`}>
                    <input
                      type="file"
                      multiple
                      accept={BOOK_FILE_ACCEPT}
                      onChange={handleUploadBookFiles}
                      className="hidden"
                      disabled={uploadingBookFiles}
                    />
                    {uploadingBookFiles ? <Loader2 size={18} className="animate-spin text-primary" /> : <FilePlus2 size={18} className="text-primary" />}
                    <span className="text-xs font-medium opacity-70">{uploadingBookFiles ? t('admin.uploading', 'Uploading...') : t('admin.add_formats', 'Add formats')}</span>
                    <span className="text-[10px] opacity-40">EPUB, MOBI, AZW, PDF, DOCX, TXT</span>
                  </label>
                  <div className="flex max-h-40 flex-col gap-1.5 overflow-y-auto pr-1">
                    {bookFiles.length > 0 ? (
                      bookFiles.map((file: BookFile) => (
                        <div key={file.id} className="flex min-w-0 items-center gap-2 rounded-lg border border-base-200 bg-base-100/70 p-2">
                          <FileText className="h-4 w-4 shrink-0 text-base-content/45" />
                          <div className="min-w-0 flex-1">
                            <div className="truncate text-xs font-semibold">{fileNameFromPath(file.path)}</div>
                            <div className="text-[10px] uppercase text-base-content/45">{file.format || "file"} · {formatFileSize(file.size_bytes)}</div>
                          </div>
                          {bookFiles.length > 1 && (
                            <button
                              type="button"
                              onClick={() => {
                                setDeleteFileId(file.id);
                              }}
                              className="btn btn-ghost btn-square btn-xs text-error hover:bg-error/15 rounded-lg"
                              title={t("admin.delete_file", "Delete file format")}
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          )}
                        </div>
                      ))
                    ) : (
                      <div className="rounded-lg bg-base-200/40 px-3 py-2 text-center text-xs text-base-content/45">No files attached</div>
                    )}
                  </div>
                </div>
              </div>

              {/* ========== RIGHT COLUMN: Metadata Form ========== */}
              <div className="flex flex-col gap-6 p-6 overflow-y-auto">

                {/* Online Search Widget */}
                <div className="bg-primary/5 border border-primary/10 rounded-2xl p-4 sm:p-5">
                  <h4 className="font-bold text-sm text-primary mb-3 flex items-center gap-2">
                    <Globe size={18} />
                    Search Online Metadata & Cover
                  </h4>
                  <div className="flex flex-col sm:flex-row gap-2.5">
                    <select
                      className="select select-bordered select-md bg-base-100 shrink-0 rounded-xl text-sm font-medium"
                      value={searchSource}
                      onChange={e => setSearchSource(e.target.value)}
                    >
                      <option value="fallback">Auto (Fallback)</option>
                      <option value="anilist">AniList (Light Novel)</option>
                      <option value="google">Google Books</option>
                      <option value="openlibrary">Open Library</option>
                    </select>
                    <input
                      type="text"
                      className="input input-bordered input-md flex-1 bg-base-100 min-w-0 rounded-xl text-sm"
                      placeholder="Search title, series or author..."
                      value={onlineSearchQuery}
                      onChange={e => setOnlineSearchQuery(e.target.value)}
                      onKeyDown={e => {
                        if (e.key === "Enter") {
                          e.preventDefault();
                          handleSearchOnline();
                        }
                      }}
                    />
                    <button
                      type="button"
                      onClick={handleSearchOnline}
                      disabled={searching}
                      className="btn btn-md btn-primary gap-1.5 shrink-0 rounded-xl font-bold"
                    >
                      {searching ? <Loader2 className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
                      {searching ? "Searching..." : "Search"}
                    </button>
                  </div>

                  {searchResults.length > 0 && (
                    <div className="mt-3.5 border border-base-200 bg-base-100 rounded-xl max-h-56 overflow-y-auto shadow-inner p-2.5 flex flex-col gap-1.5">
                      <div className="flex justify-between items-center px-2 py-1">
                        <span className="text-xs font-bold text-base-content/60">Select a result to auto-fill:</span>
                        <button type="button" onClick={() => setSearchResults([])} className="text-xs text-error font-bold hover:underline">Close</button>
                      </div>
                      {searchResults.map((res, idx) => (
                        <div
                          key={idx}
                          onClick={() => handleSelectResult(res)}
                          className="flex gap-3 p-2.5 rounded-xl hover:bg-primary/10 border border-transparent hover:border-primary/20 cursor-pointer transition-colors"
                        >
                          {res.cover_image ? (
                            <img src={getMediaUrl(res.cover_image, editingBook?.id)} loading="lazy" className="w-10 h-14 object-cover rounded-lg bg-base-200 border border-base-200 shrink-0" />
                          ) : (
                            <div className="w-10 h-14 rounded-lg bg-base-200 border border-base-200 flex items-center justify-center text-[9px] text-base-content/40 font-bold shrink-0">—</div>
                          )}
                          <div className="flex flex-col justify-center min-w-0">
                            <strong className="text-sm text-primary truncate leading-tight">{res.title}</strong>
                            <span className="text-xs opacity-60 truncate">{res.creator || "Unknown author"}</span>
                            {res.publisher && <span className="text-[11px] opacity-40 truncate">{res.publisher}</span>}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* Form Fields */}
                <form onSubmit={handleEditSubmit} id="metadata-form" className="flex flex-col gap-5">
                  <div className="flex flex-col gap-1.5 w-full">
                    <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 pl-1">{t('book.title', 'Title')}</label>
                    <input type="text" required className="input input-bordered input-md text-sm rounded-xl w-full font-medium" value={formData.title} onChange={e => setFormData({ title: e.target.value })} />
                  </div>

                  <div className="flex flex-col gap-1.5 w-full">
                    <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 pl-1">{t('book.author', 'Author')}</label>
                    <input type="text" className="input input-bordered input-md text-sm rounded-xl w-full font-medium" value={formData.author} onChange={e => setFormData({ author: e.target.value })} />
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 pl-1">{t('book.series', 'Series')}</label>
                      <input type="text" className="input input-bordered input-md text-sm rounded-xl w-full" value={formData.series} onChange={e => setFormData({ series: e.target.value })} />
                    </div>
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 pl-1">{t('book.series_index', 'Series Index')}</label>
                      <input type="text" className="input input-bordered input-md text-sm rounded-xl w-full font-mono" value={formData.series_index} onChange={e => setFormData({ series_index: e.target.value })} />
                    </div>
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 pl-1">{t('book.publisher', 'Publisher')}</label>
                      <input type="text" className="input input-bordered input-md text-sm rounded-xl w-full" value={formData.publisher} onChange={e => setFormData({ publisher: e.target.value })} />
                    </div>
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 pl-1">{t('book.date', 'Date')}</label>
                      <input type="text" className="input input-bordered input-md text-sm rounded-xl w-full" value={formData.date} onChange={e => setFormData({ date: e.target.value })} />
                    </div>
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 pl-1">{t('book.language', 'Language')}</label>
                      <input type="text" className="input input-bordered input-md text-sm rounded-xl w-full font-mono" value={formData.language} onChange={e => setFormData({ language: e.target.value })} />
                    </div>
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 pl-1">{t('book.age_rating', 'Age Rating')}</label>
                      <select
                        value={formData.age_rating || ""}
                        onChange={e => setFormData({ age_rating: e.target.value })}
                        className="select select-bordered select-md text-sm rounded-xl w-full font-medium"
                      >
                        <option value="">{t('common.none', 'None')}</option>
                        <option value="safe">Safe / All Ages</option>
                        <option value="teen">Teen / 13+</option>
                        <option value="mature">Mature / 16+</option>
                        <option value="explicit">Explicit / 18+</option>
                      </select>
                    </div>
                  </div>

                  {/* Tags Chip Editor */}
                  <div className="flex flex-col gap-2 w-full">
                    <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 pl-1">{t('book.tags', 'Tags / Subjects')}</label>
                    <div className="flex flex-wrap gap-2 items-center p-3 bg-base-200/40 border border-base-200 rounded-xl min-h-12">
                      {formData.subjects.map((sub, sIdx) => (
                        <span key={sIdx} className="badge badge-md badge-primary/10 text-primary border border-primary/20 gap-1.5 py-3 px-3 text-xs font-medium rounded-lg">
                          {sub}
                          <button
                            type="button"
                            onClick={() => {
                              const next = formData.subjects.filter((_, i) => i !== sIdx);
                              setFormData({ subjects: next });
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
                          value={tagInput}
                          onChange={(e) => setTagInput(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' || e.key === ',') {
                              e.preventDefault();
                              handleAddTag();
                            }
                          }}
                          placeholder={t('book.add_tag_placeholder', 'Add tag and press Enter...')}
                          className="input input-sm input-bordered bg-base-100 text-xs flex-1 rounded-lg"
                        />
                        <button
                          type="button"
                          onClick={handleAddTag}
                          disabled={!tagInput.trim()}
                          className="btn btn-sm btn-primary rounded-lg gap-1"
                        >
                          <Plus className="w-3.5 h-3.5" />
                          {t('common.add', 'Add')}
                        </button>
                      </div>
                    </div>
                  </div>

                  <div className="flex flex-col gap-1.5 w-full">
                    <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 pl-1">{t('book.description', 'Description')}</label>
                    <textarea rows={5} className="textarea textarea-bordered textarea-md w-full leading-relaxed resize-y text-sm rounded-xl" value={formData.description} onChange={e => setFormData({ description: e.target.value })} />
                  </div>

                  {editingBook?.metadata_json && (
                    <div className="flex flex-col gap-1.5 w-full mt-2">
                      <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 pl-1">{t('book.identifiers', 'Identifiers')}</label>
                      <div className="bg-base-200/50 p-3.5 rounded-xl text-xs font-mono break-all max-h-36 overflow-y-auto">
                        {(() => {
                          try {
                            const meta = JSON.parse(editingBook.metadata_json);
                            return meta.identifier ? JSON.stringify(meta.identifier, null, 2) : "No identifiers found";
                          } catch (e) {
                            return "Invalid JSON";
                          }
                        })()}
                      </div>
                    </div>
                  )}
                </form>
              </div>

            </div>
          </div>

          {/* Footer */}
          <footer className="px-6 py-4 border-t border-base-200 bg-base-200/30 flex justify-end gap-3 shrink-0">
            <button type="button" onClick={() => setEditingBook(null)} className="btn btn-ghost btn-sm sm:btn-md rounded-xl">
              {t("admin.cancel")}
            </button>
            <button type="submit" form="metadata-form" disabled={submitting} className="btn btn-primary btn-sm sm:btn-md px-6 gap-1.5 rounded-xl font-bold !text-white shadow-lg shadow-primary/20">
              {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
              {t("admin.save")}
            </button>
          </footer>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button onClick={() => setEditingBook(null)}>close</button>
        </form>
      </dialog>

      <UploadBooksModal
        open={showUploadModal}
        libraries={libraries}
        uploadLibraryId={uploadLibraryId}
        uploading={uploading}
        uploadProgress={uploadProgress}
        uploadSpeed={uploadSpeed}
        uploadCurrentFile={uploadCurrentFile}
        uploadBytesText={uploadBytesText}
        uploadBatchInfo={uploadBatchInfo}
        accept={BOOK_FILE_ACCEPT}
        onClose={() => setShowUploadModal(false)}
        onLibraryChange={setUploadLibraryId}
        onUploadFiles={handleUploadFiles}
      />

      {canImportCalibre && (
        <CalibreImportModal
          open={showCalibreModal}
          libraries={libraries}
          importing={calibreImportMutation.isPending}
          onClose={() => setShowCalibreModal(false)}
          onImport={(path, library_id) => {
            calibreImportMutation.mutate(
              { path, library_id: library_id || undefined },
              {
                onSuccess: (data) => {
                  toast.success(t("admin.calibre_import_success", { count: data.imported_count }));
                  setShowCalibreModal(false);
                },
                onError: (err) => {
                  toast.error(err instanceof Error ? err.message : String(err));
                },
              }
            );
          }}
        />
      )}

      <ManageLibrariesModal
        open={showLibraryModal}
        libraries={libraries}
        newLibraryName={newLibraryName}
        onClose={() => setShowLibraryModal(false)}
        onNameChange={setNewLibraryName}
        onCreate={handleCreateLibrary}
        onRename={(id, name) =>
          updateLibraryMutation.mutate(
            { id, name },
            { onSuccess: () => toast.success(t("admin.library_renamed", "Library renamed")) }
          )
        }
        onDelete={setLibraryToDelete}
      />

      <DeleteConfirmModal
        open={Boolean(bookToDelete)}
        title="Delete Book"
        message={
          <>
            Are you sure you want to delete book{" "}
            <strong>{bookToDelete?.title}</strong>? This will permanently
            delete the EPUB file and all associated reading data from the
            server.
          </>
        }
        onClose={() => setBookToDelete(null)}
        onConfirm={() => {
          if (bookToDelete) {
            void store.deleteBook(bookToDelete.id);
            setBookToDelete(null);
          }
        }}
      />

      <DeleteConfirmModal
        open={Boolean(libraryToDelete)}
        title="Delete Library"
        message={
          <>
            Are you sure you want to delete library{" "}
            <strong>{libraryToDelete?.name}</strong>? All associated books in
            this library will be detached or deleted.
          </>
        }
        onClose={() => setLibraryToDelete(null)}
        onConfirm={() => {
          if (libraryToDelete) {
            deleteLibraryMutation.mutate(libraryToDelete.id, {
              onSuccess: () => {
                toast.success(t("admin.library_deleted", "Library deleted"));
                // The deleted library may be the one being filtered or uploaded into.
                if (selectedLibraryId === libraryToDelete.id) setSelectedLibraryId("");
                if (uploadLibraryId === libraryToDelete.id) setUploadLibraryId("");
              },
            });
            setLibraryToDelete(null);
          }
        }}
      />

      <DeleteConfirmModal
        open={deleteFileId !== null}
        title={t("admin.delete_file", "Delete file format")}
        message={t("admin.confirm_delete_file", "Are you sure you want to delete this file format?")}
        onClose={() => setDeleteFileId(null)}
        onConfirm={() => {
          if (deleteFileId) {
            void deleteBookFile(deleteFileId);
            setDeleteFileId(null);
          }
        }}
      />

      <BulkDeleteModal
        isOpen={showBulkDeleteModal}
        bookIds={selectedBookIds}
        onClose={() => setShowBulkDeleteModal(false)}
        onSuccess={() => {
          setSelectedBookIds([]);
        }}
      />

      {convertBook && (
        <ConvertBookModal
          open={!!convertBook}
          bookId={convertBook.id}
          files={convertBook.files || []}
          onClose={() => setConvertBook(null)}
        />
      )}

      {mergeBook && (
        <MergeAudiobookModal
          open={!!mergeBook}
          book_id={mergeBook.id!}
          title={mergeBook.title}
          files={mergeBook.files || []}
          onClose={() => setMergeBook(null)}
        />
      )}

      {showBulkConvertModal && (
        <BulkConvertModal
          open={showBulkConvertModal}
          books={selectedBooks}
          onClose={() => {
            setShowBulkConvertModal(false);
            setSelectedBookIds([]);
          }}
        />
      )}

      {showBulkEditModal && (
        <BulkEditMetadataModal
          isOpen={showBulkEditModal}
          books={selectedBooks}
          onClose={() => setShowBulkEditModal(false)}
          onSuccess={() => {
            setSelectedBookIds([]);
            void queryClient.invalidateQueries({ queryKey: ["books"] });
          }}
        />
      )}
    </div>
  );
}
