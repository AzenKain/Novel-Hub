import { queryClient } from "@/config/queryClient";
import i18n from "@/i18n";
import { formatFileSize, formatUploadSpeed, getMetaContent, toStringList } from "@/lib/bookDetail";
import { bookService, metadataService, uploadService } from "@/services";
import { Book, BookFile, Library, MetadataJSON, OnlineMetadataResult } from "@/types";
import { toast } from 'react-toastify';
import { create } from "zustand";

interface BookAdminState {
  search: string;
  selectedLibraryId: string;

  // Editor Modal
  editingBook: Book | null;
  formData: {
    title: string;
    author: string;
    description: string;
    publisher: string;
    language: string;
    date: string;
    subjects: string;
    series: string;
    series_index: string;
  };
  submitting: boolean;
  bookFiles: BookFile[];
  uploadingBookFiles: boolean;

  // Cover preview & tabs
  coverTab: "book" | "upload" | "link";
  epubImages: string[];
  loadingImages: boolean;
  linkUrl: string;
  coverPreview: string | null;
  pendingCover: { type: 'file' | 'url' | 'epub', value: any } | null;

  // Metadata Search
  searchSource: string;
  onlineSearchQuery: string;
  searching: boolean;
  searchResults: OnlineMetadataResult[];

  // Upload Modal
  showUploadModal: boolean;
  uploadLibraryId: string;
  uploading: boolean;
  uploadProgress: number;
  uploadSpeed: string;
  uploadCurrentFile: string;
  uploadBytesText: string;
  uploadBatchInfo: { current: number; total: number } | null;

  // Manage Libraries Modal
  showLibraryModal: boolean;
  newLibraryName: string;

  // Deletion modals state
  bookToDelete: Book | null;
  libraryToDelete: Library | null;

  // Setters
  setSearch: (search: string) => void;
  setSelectedLibraryId: (id: string) => void;
  setSearchSource: (source: string) => void;
  setOnlineSearchQuery: (query: string) => void;
  setCoverTab: (tab: "book" | "upload" | "link") => void;
  setLinkUrl: (url: string) => void;
  setFormData: (data: Partial<{
    title: string;
    author: string;
    description: string;
    publisher: string;
    language: string;
    date: string;
    subjects: string;
    series: string;
    series_index: string;
  }>) => void;
  setShowUploadModal: (show: boolean) => void;
  setUploadLibraryId: (id: string) => void;
  setShowLibraryModal: (show: boolean) => void;
  setNewLibraryName: (name: string) => void;
  setEditingBook: (book: Book | null) => void;
  setSearchResults: (results: OnlineMetadataResult[]) => void;
  setBookToDelete: (book: Book | null) => void;
  setLibraryToDelete: (library: Library | null) => void;

  // Actions
  openEditModal: (book: Book) => void;
  handleSearchOnline: () => Promise<void>;
  handleSelectResult: (result: OnlineMetadataResult) => void;
  handleSelectEpubImage: (imagePath: string) => Promise<void>;
  handleImageUpload: (e: React.ChangeEvent<HTMLInputElement>) => Promise<void>;
  handleLinkUpload: () => Promise<void>;
  handleEditSubmit: (e?: React.SyntheticEvent) => Promise<void>;
  handleUploadBookFiles: (e: React.ChangeEvent<HTMLInputElement>) => Promise<void>;
  handleUploadFiles: (filesOrEvent: FileList | File[] | React.ChangeEvent<HTMLInputElement>) => Promise<void>;
  deleteBook: (id: string) => Promise<void>;
  archiveBook: (id: string, archived: boolean) => Promise<void>;
}

const invalidateBooks = () => {
  void queryClient.invalidateQueries({ queryKey: ["books"] });
  void queryClient.invalidateQueries({ queryKey: ["library"] });
  void queryClient.invalidateQueries({ queryKey: ["metadata"] });
};

export const useBookAdminStore = create<BookAdminState>((set, get) => ({
  search: "",
  selectedLibraryId: "",

  editingBook: null,
  formData: { title: "", author: "", description: "", publisher: "", language: "", date: "", subjects: "", series: "", series_index: "" },
  submitting: false,
  bookFiles: [],
  uploadingBookFiles: false,

  coverTab: "book",
  epubImages: [],
  loadingImages: false,
  linkUrl: "",
  coverPreview: null,
  pendingCover: null,

  searchSource: "fallback",
  onlineSearchQuery: "",
  searching: false,
  searchResults: [],

  showUploadModal: false,
  uploadLibraryId: "",
  uploading: false,
  uploadProgress: 0,
  uploadSpeed: "0 B/s",
  uploadCurrentFile: "",
  uploadBytesText: "",
  uploadBatchInfo: null,

  showLibraryModal: false,
  newLibraryName: "",

  bookToDelete: null,
  libraryToDelete: null,

  setSearch: (search) => set({ search }),
  setSelectedLibraryId: (selectedLibraryId) => set({ selectedLibraryId }),
  setSearchSource: (searchSource) => set({ searchSource }),
  setOnlineSearchQuery: (onlineSearchQuery) => set({ onlineSearchQuery }),
  setCoverTab: (coverTab) => set({ coverTab }),
  setLinkUrl: (linkUrl) => set({ linkUrl }),
  setFormData: (data) => set((state) => ({ formData: { ...state.formData, ...data } })),
  setShowUploadModal: (showUploadModal) => set({ showUploadModal }),
  setUploadLibraryId: (uploadLibraryId) => set({ uploadLibraryId }),
  setShowLibraryModal: (showLibraryModal) => set({ showLibraryModal }),
  setNewLibraryName: (newLibraryName) => set({ newLibraryName }),
  setEditingBook: (editingBook) => set({ editingBook }),
  setSearchResults: (searchResults) => set({ searchResults }),
  setBookToDelete: (bookToDelete) => set({ bookToDelete }),
  setLibraryToDelete: (libraryToDelete) => set({ libraryToDelete }),

  openEditModal: (book) => {
    let publisher = "";
    let language = "";
    let date = "";
    let subjects = "";
    let series = "";
    let series_index = "";

    if (book.metadata_json) {
      try {
        const meta = JSON.parse(book.metadata_json) as MetadataJSON;
        publisher = meta.publisher || meta.publishers?.join(", ") || "";
        language = meta.language || meta.languages?.join(", ") || "";
        date = meta.date || meta.dates?.[0] || "";
        subjects = toStringList(meta.subject).join(", ");
        series = meta.series || getMetaContent(meta, "calibre:series");
        series_index = meta.series_index || getMetaContent(meta, "calibre:series_index");
      } catch (e) {
        console.error("Failed to parse metadata_json", e);
      }
    }

    set({
      editingBook: book,
      formData: {
        title: book.title || "",
        author: book.author_name || book.author_id || "",
        description: book.description || "",
        publisher,
        language,
        date,
        subjects,
        series,
        series_index
      },
      coverTab: "book",
      epubImages: [],
      bookFiles: book.files || [],
      linkUrl: "",
      coverPreview: book.cover_url || null,
      pendingCover: null,
      onlineSearchQuery: book.title || "",
      searchResults: [],
      loadingImages: true
    });

    void Promise.all([
      bookService.listImages(book.id).then(res => {
        if (res.status && res.data) set({ epubImages: res.data });
      }).catch(() => undefined),
      bookService.listFiles(book.id).then(res => {
        if (res.status && res.data) set({ bookFiles: res.data });
      }).catch(() => undefined)
    ]).finally(() => {
      set({ loadingImages: false });
    });
  },

  handleSearchOnline: async () => {
    const { onlineSearchQuery, formData, editingBook, searchSource } = get();
    const query = onlineSearchQuery?.trim() || formData.title?.trim() || editingBook?.title?.trim() || "";
    if (!query) {
      toast.warn("Please enter a book title or keyword to search online");
      return;
    }
    set({ searching: true, searchResults: [] });
    try {
      const results = await metadataService.searchOnline(query, searchSource);
      set({ searchResults: results });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error connecting to server or fetching metadata");
    } finally {
      set({ searching: false });
    }
  },

  handleSelectResult: (result) => {
    const { formData, editingBook } = get();
    set({
      formData: {
        ...formData,
        title: result.title || formData.title,
        author: result.creator || formData.author,
        description: result.description || formData.description,
        publisher: result.publisher || formData.publisher,
        language: result.language || formData.language,
        subjects: result.subject || formData.subjects,
        series: result.series || formData.series,
        series_index: result.series_index || formData.series_index
      },
      searchResults: []
    });
    
    if (result.cover_image && editingBook) {
      set({ 
        coverPreview: result.cover_image, 
        pendingCover: { type: 'url', value: result.cover_image } 
      });
    }
  },

  handleSelectEpubImage: async (imagePath) => {
    const { editingBook } = get();
    if (!editingBook) return;
    set({ 
      coverPreview: `/api/v1/reader/${editingBook.id}/asset/${imagePath}`, 
      pendingCover: { type: 'epub', value: imagePath } 
    });
  },

  handleImageUpload: async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const previewUrl = URL.createObjectURL(file);
    set({ 
      coverPreview: previewUrl, 
      pendingCover: { type: 'file', value: file } 
    });
  },

  handleLinkUpload: async () => {
    const { linkUrl } = get();
    if (!linkUrl) return;
    set({ 
      coverPreview: linkUrl, 
      pendingCover: { type: 'url', value: linkUrl } 
    });
    toast.success("Cover link applied. Click Save at the bottom to download and save.");
  },

  handleEditSubmit: async (e) => {
    if (e) e.preventDefault();
    const { editingBook, formData, pendingCover } = get();
    if (!editingBook) return;

    set({ submitting: true });
    try {
      const submitData = {
        ...formData,
        subjects: formData.subjects.split(',').map(s => s.trim()).filter(Boolean)
      };
      
      await bookService.updateMetadata(editingBook.id, submitData);

      if (pendingCover) {
        if (pendingCover.type === 'file') {
          await bookService.updateCover(editingBook.id, { cover: pendingCover.value });
        } else if (pendingCover.type === 'url') {
          await bookService.updateCover(editingBook.id, { cover_url: pendingCover.value });
        } else if (pendingCover.type === 'epub') {
          await bookService.updateCover(editingBook.id, { epub_image_path: pendingCover.value });
        }
      }

      toast.success("Success!");
      set({ editingBook: null });
      invalidateBooks();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error updating book");
    } finally {
      set({ submitting: false });
    }
  },

  handleUploadBookFiles: async (e) => {
    const files = e.target.files;
    const { editingBook } = get();
    if (!files || files.length === 0 || !editingBook) return;

    set({ uploadingBookFiles: true });
    try {
      let successCount = 0;
      for (const file of Array.from(files)) {
        try {
          const res = await uploadService.uploadFileChunked(file, "book", editingBook.id);
          if (!res.status) throw new Error(res.message || "Upload failed");
          successCount++;
        } catch (fileErr) {
          console.error("Failed to upload book file:", file.name, fileErr);
        }
      }

      if (successCount === 0) throw new Error("All file uploads failed");

      toast.success(`Successfully uploaded ${successCount} files!`);
      const res = await bookService.listFiles(editingBook.id);
      const nextFiles = res.data || [];
      set((state) => ({
        bookFiles: nextFiles,
        editingBook: state.editingBook ? { ...state.editingBook, files: nextFiles } : state.editingBook
      }));
      invalidateBooks();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error uploading files");
    } finally {
      e.target.value = "";
      set({ uploadingBookFiles: false });
    }
  },

  handleUploadFiles: async (filesOrEvent) => {
    let fileArray: File[] = [];
    if (Array.isArray(filesOrEvent)) {
      fileArray = filesOrEvent;
    } else if ("target" in filesOrEvent && filesOrEvent.target && "files" in filesOrEvent.target && filesOrEvent.target.files) {
      fileArray = Array.from(filesOrEvent.target.files);
    } else if (filesOrEvent && "length" in filesOrEvent) {
      fileArray = Array.from(filesOrEvent as FileList);
    }

    const { uploadLibraryId } = get();
    if (!fileArray || fileArray.length === 0 || !uploadLibraryId) return;

    set({
      uploading: true,
      uploadProgress: 0,
      uploadSpeed: "0 B/s",
      uploadCurrentFile: "",
      uploadBytesText: "",
      uploadBatchInfo: { current: 0, total: fileArray.length },
    });

    try {
      let successCount = 0;
      let firstError = "";
      const totalFiles = fileArray.length;

      for (let i = 0; i < totalFiles; i++) {
        const file = fileArray[i];
        set({
          uploadCurrentFile: file.name,
          uploadBatchInfo: { current: i + 1, total: totalFiles },
          uploadProgress: 0,
          uploadSpeed: "0 B/s",
          uploadBytesText: `0 B / ${formatFileSize(file.size)}`,
        });

        try {
          const res = await uploadService.uploadFileChunked(
            file,
            "library",
            uploadLibraryId,
            (stats) => {
              set({
                uploadProgress: stats.progress,
                uploadSpeed: formatUploadSpeed(stats.speedBytesPerSec),
                uploadBytesText: `${formatFileSize(stats.uploadedBytes)} / ${formatFileSize(stats.totalBytes)}`,
              });
            }
          );
          if (!res.status) throw new Error(res.message || "Upload failed");
          successCount++;
        } catch (fileErr) {
          console.error("Failed to upload file:", file.name, fileErr);
          if (!firstError) firstError = fileErr instanceof Error ? fileErr.message : String(fileErr);
        }
      }

      if (successCount === 0) throw new Error(i18n.t("admin.upload_all_failed", { reason: firstError }));

      set({ showUploadModal: false });
      toast.info(successCount === totalFiles
        ? i18n.t("admin.upload_done", { count: successCount })
        : i18n.t("admin.upload_partial", { count: successCount, total: totalFiles }));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      invalidateBooks();
      set({
        uploading: false,
        uploadProgress: 0,
        uploadSpeed: "0 B/s",
        uploadCurrentFile: "",
        uploadBytesText: "",
        uploadBatchInfo: null,
      });
    }
  },

  deleteBook: async (id) => {
    try {
      await bookService.deleteBook(id);
      toast.success("Book deleted successfully!");
      invalidateBooks();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete book");
    }
  },

  archiveBook: async (id, archived) => {
    try {
      const res = await bookService.archiveBook(id, archived);
      if (!res.status) throw new Error(res.message || "Failed to update archive state");
      toast.success(archived ? "Book archived" : "Book unarchived");
      invalidateBooks();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update archive state");
    }
  }
}));
