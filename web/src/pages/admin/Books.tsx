import { BookActionModal, DeleteConfirmModal, ManageLibrariesModal, UploadBooksModal } from "@/components/admin";
import { BookCard } from "@/components/ui";
import { BOOK_FILE_ACCEPT } from "@/constants";
import { fileNameFromPath, formatFileSize } from "@/lib/bookDetail";
import { useBookAdminStore } from "@/stores";
import type { Book, BookFile } from "@/types";
import { BookOpen, FilePlus2, FileText, Globe, Image as ImageIcon, Link as LinkIcon, Loader2, RefreshCw, Save, Search, Upload } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";

export function Books() {
  const { t } = useTranslation();
  const store = useBookAdminStore(useShallow((state) => ({
    books: state.books, loading: state.loading, error: state.error, notice: state.notice, page: state.page, search: state.search, selectedLibraryId: state.selectedLibraryId, hasMore: state.hasMore,
    libraries: state.libraries, editingBook: state.editingBook, formData: state.formData, submitting: state.submitting,
    bookFiles: state.bookFiles, uploadingBookFiles: state.uploadingBookFiles,
    coverTab: state.coverTab, epubImages: state.epubImages, loadingImages: state.loadingImages, linkUrl: state.linkUrl, coverPreview: state.coverPreview,
    searchSource: state.searchSource, searching: state.searching, searchResults: state.searchResults,
    showUploadModal: state.showUploadModal, uploadLibraryId: state.uploadLibraryId, uploading: state.uploading,
    showLibraryModal: state.showLibraryModal, newLibraryName: state.newLibraryName,
    bookToDelete: state.bookToDelete, libraryToDelete: state.libraryToDelete,
    setSearch: state.setSearch, setSelectedLibraryId: state.setSelectedLibraryId, setSearchSource: state.setSearchSource, setCoverTab: state.setCoverTab, setLinkUrl: state.setLinkUrl,
    setFormData: state.setFormData, setShowUploadModal: state.setShowUploadModal, setUploadLibraryId: state.setUploadLibraryId, setShowLibraryModal: state.setShowLibraryModal,
    setNewLibraryName: state.setNewLibraryName, setNotice: state.setNotice, setError: state.setError, setCoverPreview: state.setCoverPreview, setPage: state.setPage,
    setEditingBook: state.setEditingBook, setSearchResults: state.setSearchResults, setBookToDelete: state.setBookToDelete, setLibraryToDelete: state.setLibraryToDelete,
    openEditModal: state.openEditModal, closeEditModal: state.closeEditModal, handleSearchOnline: state.handleSearchOnline, handleSelectResult: state.handleSelectResult,
    handleSelectEpubImage: state.handleSelectEpubImage, handleImageUpload: state.handleImageUpload, handleLinkUpload: state.handleLinkUpload, handleEditSubmit: state.handleEditSubmit,
    handleUploadBookFiles: state.handleUploadBookFiles, handleCreateLibrary: state.handleCreateLibrary, handleDeleteLibrary: state.handleDeleteLibrary, handleUploadFiles: state.handleUploadFiles, loadData: state.loadData, loadLibraries: state.loadLibraries, deleteBook: state.deleteBook
  })));
  const [actionBook, setActionBook] = useState<Book | null>(null);

  const {
    books, loading, error, notice, page, search, selectedLibraryId, hasMore,
    libraries, editingBook, formData, submitting,
    bookFiles, uploadingBookFiles,
    coverTab, epubImages, loadingImages, linkUrl, coverPreview,
    searchSource, searching, searchResults,
    showUploadModal, uploadLibraryId, uploading,
    showLibraryModal, newLibraryName,
    bookToDelete, libraryToDelete,
    setSearch, setSelectedLibraryId, setSearchSource, setCoverTab, setLinkUrl,
    setFormData, setShowUploadModal, setUploadLibraryId, setShowLibraryModal,
    setNewLibraryName, setNotice, setError, setCoverPreview, setPage,
    setEditingBook, setSearchResults, setBookToDelete, setLibraryToDelete,
    openEditModal, closeEditModal, handleSearchOnline, handleSelectResult,
    handleSelectEpubImage, handleImageUpload, handleLinkUpload, handleEditSubmit,
    handleUploadBookFiles, handleCreateLibrary, handleDeleteLibrary, handleUploadFiles, loadData
  } = store;

  const navigate = useNavigate();

  useEffect(() => {
    void store.loadData();
  }, [page, search, selectedLibraryId]);

  useEffect(() => {
    void store.loadLibraries();
  }, []);

  const getImageAssetUrl = (imagePath: string) => {
    if (!editingBook) return "";
    return `/api/v1/reader/${editingBook.id}/asset/${imagePath}`;
  };

  return (
    <div className="flex flex-col h-full bg-base-100">
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex flex-col gap-4 bg-base-100/50 backdrop-blur-xl sticky top-0 z-10 xl:flex-row xl:items-center xl:justify-between">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold tracking-tight">{t("admin.books")}</h1>
          <p className="text-sm opacity-60 mt-1">Manage EPUB novels</p>
        </div>
        <div className="flex w-full min-w-0 flex-wrap items-center gap-2 xl:w-auto xl:justify-end">
          <div className="flex min-w-0 flex-1 basis-full items-center gap-2 rounded-full bg-base-200 px-4 py-2 sm:basis-64 xl:max-w-72">
            <Search className="w-4 h-4 opacity-50" />
            <input
              type="text"
              placeholder="Search books..."
              className="min-w-0 w-full bg-transparent border-none outline-none text-sm"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <select
            className="select select-sm select-bordered min-w-0 max-w-full flex-1 rounded-full sm:flex-none"
            value={selectedLibraryId}
            onChange={(e) => setSelectedLibraryId(e.target.value)}
          >
            <option value="">All Libraries</option>
            {libraries.map(lib => (
              <option key={lib.id} value={lib.id}>{lib.name}</option>
            ))}
          </select>
          <button
            onClick={() => void loadData()}
            className="btn btn-square btn-ghost btn-sm sm:btn-md"
            title="Refresh list"
            disabled={loading}
          >
            <RefreshCw className={`h-5 w-5 ${loading ? "animate-spin" : ""}`} />
          </button>
          <button
            onClick={() => setShowLibraryModal(true)}
            className="btn btn-secondary btn-sm sm:btn-md gap-2"
          >
            Manage Libraries
          </button>
          <button
            onClick={() => setShowUploadModal(true)}
            className="btn btn-primary btn-sm sm:btn-md gap-2"
          >
            <Upload className="w-4 h-4" />
            Upload
          </button>
        </div>
      </header>



      <div className="flex-1 overflow-auto p-8">
        {loading && books.length === 0 ? (
          <div className="flex items-center justify-center py-20 opacity-50">
            <Loader2 className="h-8 w-8 animate-spin text-primary mr-3" />
            {t("common.loading")}
          </div>
        ) : books.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 opacity-50 border-2 border-dashed border-base-300 rounded-2xl bg-base-200/50">
            <BookOpen className="h-12 w-12 mb-4" />
            <p className="text-lg font-medium">{t("common.no_data")}</p>
            <p className="text-sm mt-1">Config a library in the system and upload EPUB files.</p>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 mb-8">
              {books.map((book) => (
                <BookCard key={book.id} book={book} onClick={() => setActionBook(book)} />
              ))}
            </div>

            {/* Pagination */}
            {books.length > 0 && (
              <div className="flex justify-center mt-auto pt-8">
                <div className="join">
                  <button
                    className="join-item btn btn-sm"
                    disabled={page === 1}
                    onClick={() => setPage(p => Math.max(1, p - 1))}
                  >
                    «
                  </button>
                  <button className="join-item btn btn-sm pointer-events-none">Page {page}</button>
                  <button
                    className="join-item btn btn-sm"
                    disabled={!hasMore}
                    onClick={() => setPage(p => p + 1)}
                  >
                    »
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      <BookActionModal
        book={actionBook}
        onClose={() => setActionBook(null)}
        onRead={(book) => {
          const fileId = book.files?.[0]?.id;
          setActionBook(null);
          navigate(
            `/reader/${book.id}${fileId ? `?file_id=${encodeURIComponent(fileId)}` : ""}`,
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
      />

      {/* ===================== EDIT METADATA MODAL ===================== */}
      <dialog className={`modal ${editingBook ? "modal-open" : ""}`}>
        <div className="modal-box w-11/12 max-w-5xl bg-base-100 shadow-2xl p-0 overflow-hidden flex flex-col max-h-[90vh]">
          {/* Header */}
          <header className="px-6 py-4 border-b border-base-200 bg-base-200/30 flex items-center justify-between shrink-0">
            <div>
              <h3 className="text-lg font-black">{t('admin.metadata', 'Metadata')}</h3>
              <p className="text-xs opacity-50 mt-0.5 font-mono truncate max-w-sm">{editingBook?.id}</p>
            </div>
            <button onClick={() => setEditingBook(null)} className="btn btn-sm btn-circle btn-ghost">✕</button>
          </header>

          {/* Body: 2-column layout */}
          <div className="flex-1 overflow-y-auto">
            <div className="grid grid-cols-1 md:grid-cols-[260px_1fr] gap-0 min-h-0">

              {/* ========== LEFT COLUMN: Cover Section ========== */}
              <div className="flex flex-col gap-3 p-5 border-r border-base-200 bg-base-200/10">
                <span className="text-xs font-bold text-primary uppercase tracking-wider">{t('admin.cover_image', 'Cover Image')}</span>

                {/* Cover Preview */}
                <div className="flex flex-col items-center gap-2 p-3 bg-base-200/30 border border-base-200 rounded-lg">
                  <div className="w-36 aspect-[3/4.12] rounded-md bg-base-300 border border-base-300 overflow-hidden shadow-md flex items-center justify-center">
                    {coverPreview ? (
                      <img src={coverPreview} alt="Cover" loading="lazy" className="w-full h-full object-cover" />
                    ) : (
                      <div className="flex flex-col items-center justify-center text-base-content/30 gap-1">
                        <ImageIcon size={24} />
                        <span className="text-[10px] font-bold uppercase">{t('admin.no_cover', 'No Cover')}</span>
                      </div>
                    )}
                  </div>
                </div>

                {/* Cover Tabs */}
                <div className="flex gap-1 flex-nowrap">
                  <button type="button" onClick={() => setCoverTab("book")} className={`btn btn-xs flex-1 gap-1 whitespace-nowrap ${coverTab === "book" ? "btn-primary" : "btn-ghost"}`}>
                    <BookOpen size={12} /> {t('admin.in_book', 'In Book')}
                  </button>
                  <button type="button" onClick={() => setCoverTab("upload")} className={`btn btn-xs flex-1 gap-1 whitespace-nowrap ${coverTab === "upload" ? "btn-primary" : "btn-ghost"}`}>
                    <Upload size={12} /> {t('admin.upload', 'Upload')}
                  </button>
                  <button type="button" onClick={() => setCoverTab("link")} className={`btn btn-xs flex-1 gap-1 whitespace-nowrap ${coverTab === "link" ? "btn-primary" : "btn-ghost"}`}>
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
                                className="flex items-center gap-2 p-1.5 rounded-md hover:bg-primary/10 border border-transparent hover:border-primary/20 cursor-pointer transition-colors text-left"
                            >
                              <img src={getImageAssetUrl(img)} alt={fileName} loading="lazy" className="w-8 h-10 object-cover rounded bg-base-200 border border-base-200 shrink-0" />
                              <span className="text-[11px] truncate min-w-0 flex-1">{fileName}</span>
                            </button>
                          );
                        })
                      ) : (
                        <div className="text-xs opacity-50 text-center py-4">{t('admin.no_images_in_epub', 'No images found in EPUB')}</div>
                      )}
                    </div>
                  )}

                  {coverTab === "upload" && (
                    <label className="flex flex-col items-center justify-center gap-2 p-4 border-2 border-dashed border-base-300 rounded-lg cursor-pointer hover:border-primary/30 hover:bg-primary/5 transition-colors min-h-25">
                      <input type="file" accept="image/png,image/jpeg,image/jpg,image/gif,image/webp" onChange={handleImageUpload} className="hidden" />
                      <Upload size={20} className="opacity-50" />
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
                        className="input input-bordered input-sm w-full text-xs"
                      />
                      <button type="submit" disabled={!linkUrl.trim()} className="btn btn-sm btn-primary w-full">
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
                  <label className={`mb-3 flex min-h-20 cursor-pointer flex-col items-center justify-center gap-1.5 rounded-lg border-2 border-dashed border-base-300 p-3 text-center transition-colors hover:border-primary/30 hover:bg-primary/5 ${uploadingBookFiles ? "pointer-events-none opacity-60" : ""}`}>
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
                        <div key={file.id} className="flex min-w-0 items-center gap-2 rounded-md border border-base-200 bg-base-100/70 p-2">
                          <FileText className="h-4 w-4 shrink-0 text-base-content/45" />
                          <div className="min-w-0 flex-1">
                            <div className="truncate text-[11px] font-semibold">{fileNameFromPath(file.path)}</div>
                            <div className="text-[10px] uppercase text-base-content/45">{file.format || "file"} · {formatFileSize(file.sizeBytes)}</div>
                          </div>
                        </div>
                      ))
                    ) : (
                      <div className="rounded-md bg-base-200/40 px-3 py-2 text-center text-xs text-base-content/45">No files attached</div>
                    )}
                  </div>
                </div>
              </div>

              {/* ========== RIGHT COLUMN: Metadata Form ========== */}
              <div className="flex flex-col gap-5 p-5 overflow-y-auto">

                {/* Online Search Widget */}
                <div className="bg-primary/5 border border-primary/10 rounded-xl p-4">
                  <h4 className="font-bold text-sm text-primary mb-3 flex items-center gap-2">
                    <Globe size={16} />
                    Search Online Metadata & Cover
                  </h4>
                  <div className="flex flex-col sm:flex-row gap-2">
                    <select
                      className="select select-bordered select-sm bg-base-100 shrink-0"
                      value={searchSource}
                      onChange={e => setSearchSource(e.target.value)}
                    >
                      <option value="fallback">Auto (Fallback)</option>
                      <option value="anilist">AniList (Light Novel)</option>
                      <option value="google">Google Books</option>
                      <option value="openlibrary">Open Library</option>
                    </select>
                    <button
                      type="button"
                      onClick={handleSearchOnline}
                      disabled={searching}
                      className="btn btn-sm btn-primary gap-1"
                    >
                      {searching ? <Loader2 className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
                      {searching ? "Searching..." : "Search"}
                    </button>
                  </div>

                  {searchResults.length > 0 && (
                    <div className="mt-3 border border-base-200 bg-base-100 rounded-lg max-h-52 overflow-y-auto shadow-inner p-2 flex flex-col gap-1.5">
                      <div className="flex justify-between items-center px-2 py-1">
                        <span className="text-[11px] font-bold text-base-content/60">Select a result to auto-fill:</span>
                        <button type="button" onClick={() => setSearchResults([])} className="text-[11px] text-error font-bold hover:underline">Close</button>
                      </div>
                      {searchResults.map((res, idx) => (
                        <div
                          key={idx}
                          onClick={() => handleSelectResult(res)}
                          className="flex gap-3 p-2 rounded-md hover:bg-primary/10 border border-transparent hover:border-primary/20 cursor-pointer transition-colors"
                        >
                          {res.coverImage ? (
                            <img src={res.coverImage} loading="lazy" className="w-9 h-12 object-cover rounded bg-base-200 border border-base-200 shrink-0" referrerPolicy="no-referrer" />
                          ) : (
                            <div className="w-9 h-12 rounded bg-base-200 border border-base-200 flex items-center justify-center text-[8px] text-base-content/40 font-bold shrink-0">—</div>
                          )}
                          <div className="flex flex-col justify-center min-w-0">
                            <strong className="text-sm text-primary truncate leading-tight">{res.title}</strong>
                            <span className="text-xs opacity-60 truncate">{res.creator || "Unknown author"}</span>
                            {res.publisher && <span className="text-[10px] opacity-40 truncate">{res.publisher}</span>}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* Form Fields */}
                <form onSubmit={handleEditSubmit} id="metadata-form" className="flex flex-col gap-5">
                  <div className="flex flex-col gap-1.5 w-full">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Tên sách (Title)</label>
                    <input type="text" required className="input input-bordered w-full" value={formData.title} onChange={e => setFormData({ ...formData, title: e.target.value })} />
                  </div>

                  <div className="flex flex-col gap-1.5 w-full">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Tác giả (Author)</label>
                    <input type="text" className="input input-bordered w-full" value={formData.author} onChange={e => setFormData({ ...formData, author: e.target.value })} />
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Series</label>
                      <input type="text" className="input input-bordered w-full" value={formData.series} onChange={e => setFormData({ ...formData, series: e.target.value })} />
                    </div>
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Series ID</label>
                      <input type="text" className="input input-bordered w-full" value={formData.seriesIndex} onChange={e => setFormData({ ...formData, seriesIndex: e.target.value })} />
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Ngày xuất bản (Date)</label>
                      <input type="text" className="input input-bordered w-full" value={formData.date} onChange={e => setFormData({ ...formData, date: e.target.value })} />
                    </div>
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Nhà xuất bản (Publisher)</label>
                      <input type="text" className="input input-bordered w-full" value={formData.publisher} onChange={e => setFormData({ ...formData, publisher: e.target.value })} />
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Ngôn ngữ (Language)</label>
                      <input type="text" className="input input-bordered w-full" value={formData.language} onChange={e => setFormData({ ...formData, language: e.target.value })} />
                    </div>
                    <div className="flex flex-col gap-1.5 w-full">
                      <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Đánh dấu (Tags, comma separated)</label>
                      <input type="text" className="input input-bordered w-full" value={formData.subjects} onChange={e => setFormData({ ...formData, subjects: e.target.value })} />
                    </div>
                  </div>

                  <div className="flex flex-col gap-1.5 w-full">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Mô tả (Description)</label>
                    <textarea rows={6} className="textarea textarea-bordered w-full leading-relaxed resize-y" value={formData.description} onChange={e => setFormData({ ...formData, description: e.target.value })} />
                  </div>

                  {editingBook?.metadataJson && (
                    <div className="flex flex-col gap-1.5 w-full mt-4">
                      <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Nhận diện (Identifiers)</label>
                      <div className="bg-base-200/50 p-3 rounded-lg text-xs font-mono break-all max-h-32 overflow-y-auto">
                        {(() => {
                          try {
                            const meta = JSON.parse(editingBook.metadataJson);
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
          <footer className="px-6 py-3 border-t border-base-200 bg-base-200/30 flex justify-end gap-3 shrink-0">
            <button type="button" onClick={() => setEditingBook(null)} className="btn btn-ghost btn-sm">
              {t("admin.cancel")}
            </button>
            <button type="submit" form="metadata-form" disabled={submitting} className="btn btn-primary btn-sm px-6 gap-1">
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
        accept={BOOK_FILE_ACCEPT}
        onClose={() => setShowUploadModal(false)}
        onLibraryChange={setUploadLibraryId}
        onUploadFiles={handleUploadFiles}
      />

      <ManageLibrariesModal
        open={showLibraryModal}
        libraries={libraries}
        newLibraryName={newLibraryName}
        onClose={() => setShowLibraryModal(false)}
        onNameChange={setNewLibraryName}
        onCreate={handleCreateLibrary}
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
            void handleDeleteLibrary(libraryToDelete.id);
            setLibraryToDelete(null);
          }
        }}
      />
    </div>
  );
}
