import { queryClient } from "@/config/queryClient";
import { formatFileSize, formatUploadSpeed, getMetaContent, toStringList } from "@/lib/bookDetail";
import { bookService, libraryService, metadataService, uploadService } from "@/services";
import { Book, BookFile, Library, MetadataJSON, OnlineMetadataResult } from "@/types";
import { toast } from 'react-toastify';
import { create } from "zustand";

const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

interface BookAdminState {
  // Books list & Pagination
  books: Book[];
  loading: boolean;
  error: string;
  notice: string;
  page: number;
  search: string;
  selectedLibraryId: string;
  hasMore: boolean;

  // Libraries
  libraries: Library[];

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
  setNotice: (notice: string) => void;
  setError: (error: string) => void;
  setCoverPreview: (url: string | null) => void;
  setPage: (updater: number | ((p: number) => number)) => void;
  setEditingBook: (book: Book | null) => void;
  setSearchResults: (results: OnlineMetadataResult[]) => void;
  setBookToDelete: (book: Book | null) => void;
  setLibraryToDelete: (library: Library | null) => void;
  
  // Actions
  loadData: () => Promise<void>;
  loadLibraries: () => Promise<void>;
  openEditModal: (book: Book) => void;
  closeEditModal: () => void;
  handleSearchOnline: () => Promise<void>;
  handleSelectResult: (result: OnlineMetadataResult) => void;
  handleSelectEpubImage: (imagePath: string) => Promise<void>;
  handleImageUpload: (e: React.ChangeEvent<HTMLInputElement>) => Promise<void>;
  handleLinkUpload: () => Promise<void>;
  handleEditSubmit: (e?: React.SyntheticEvent) => Promise<void>;
  handleUploadBookFiles: (e: React.ChangeEvent<HTMLInputElement>) => Promise<void>;
  handleCreateLibrary: (e: React.SyntheticEvent) => Promise<void>;
  handleRenameLibrary: (id: string, name: string) => Promise<void>;
  handleDeleteLibrary: (id: string) => Promise<void>;
  handleUploadFiles: (filesOrEvent: FileList | File[] | React.ChangeEvent<HTMLInputElement>) => Promise<void>;
  deleteBook: (id: string) => Promise<void>;
  archiveBook: (id: string, archived: boolean) => Promise<void>;
}

export const useBookAdminStore = create<BookAdminState>((set, get) => ({
  books: [],
  loading: true,
  error: "",
  notice: "",
  page: 1,
  search: "",
  selectedLibraryId: "",
  hasMore: true,

  libraries: [],

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

  searchSource: "google",
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

  setSearch: (search) => set({ search, page: 1 }),
  setSelectedLibraryId: (selectedLibraryId) => set({ selectedLibraryId, page: 1 }),
  setSearchSource: (searchSource) => set({ searchSource }),
  setCoverTab: (coverTab) => set({ coverTab }),
  setLinkUrl: (linkUrl) => set({ linkUrl }),
  setFormData: (data) => set((state) => ({ formData: { ...state.formData, ...data } })),
  setShowUploadModal: (showUploadModal) => set({ showUploadModal }),
  setUploadLibraryId: (uploadLibraryId) => set({ uploadLibraryId }),
  setShowLibraryModal: (showLibraryModal) => set({ showLibraryModal }),
  setNewLibraryName: (newLibraryName) => set({ newLibraryName }),
  setNotice: (notice) => set({ notice }),
  setError: (error) => set({ error }),
  setCoverPreview: (coverPreview) => set({ coverPreview }),
  setPage: (updater) => set((state) => ({ page: typeof updater === "function" ? updater(state.page) : updater })),
  setEditingBook: (editingBook) => set({ editingBook }),
  setSearchResults: (searchResults) => set({ searchResults }),
  setBookToDelete: (bookToDelete) => set({ bookToDelete }),
  setLibraryToDelete: (libraryToDelete) => set({ libraryToDelete }),

  loadData: async () => {
    const { page, search, selectedLibraryId } = get();
    set({ loading: true, error: "" });
    try {
      const res = await bookService.getBooks({
        cursor: undefined,
        limit: 24,
        search: search || undefined,
        library_id: selectedLibraryId || undefined
      });

      if (res.status && res.data) {
        set({
          books: res.data,
          hasMore: res.data.length === 24
        });
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      set({ loading: false });
    }
  },

  loadLibraries: async () => {
    try {
      const res = await libraryService.getLibraries();
      if (res.status && res.data) {
        set({ libraries: res.data });
      }
    } catch (err) {
      console.error("Failed to load libraries:", err);
    }
  },

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

  closeEditModal: () => {
    set({ editingBook: null });
  },

  handleSearchOnline: async () => {
    const { editingBook, searchSource } = get();
    if (!editingBook) return;
    set({ searching: true, searchResults: [] });
    try {
      const results = await metadataService.searchOnline(editingBook.title, searchSource);
      set({ searchResults: results });
    } catch (err) {
      toast.error("Error connecting to server or fetching metadata");
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
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["metadata"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
      await get().loadData();
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
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["metadata"] });
      await get().loadData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error uploading files");
    } finally {
      e.target.value = "";
      set({ uploadingBookFiles: false });
    }
  },

  handleCreateLibrary: async (e) => {
    e.preventDefault();
    const { newLibraryName } = get();
    if (!newLibraryName.trim()) return;
    try {
      await libraryService.createLibrary({ name: newLibraryName });
      toast.success("Library created successfully!");
      set({ newLibraryName: "" });
      void queryClient.invalidateQueries({ queryKey: ["libraries"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
      const libRes = await libraryService.getLibraries();
      set({ libraries: libRes.data || [] });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    }
  },

  handleRenameLibrary: async (id, name) => {
    const trimmedName = name.trim();
    if (!trimmedName) return;
    try {
      const res = await libraryService.updateLibrary(id, { name: trimmedName });
      if (!res.status) throw new Error(res.message || "Failed to rename library");
      toast.success("Library renamed successfully!");
      set((state) => ({
        libraries: state.libraries.map((library) =>
          library.id === id ? { ...library, name: trimmedName } : library
        )
      }));
      void queryClient.invalidateQueries({ queryKey: ["libraries"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to rename library");
    }
  },

  handleDeleteLibrary: async (id) => {
    const { selectedLibraryId, uploadLibraryId } = get();
    try {
      await libraryService.deleteLibrary(id);
      toast.success("Library deleted successfully!");
      void queryClient.invalidateQueries({ queryKey: ["libraries"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      const libRes = await libraryService.getLibraries();
      set({
        libraries: libRes.data || [],
        selectedLibraryId: selectedLibraryId === id ? "" : selectedLibraryId,
        uploadLibraryId: uploadLibraryId === id ? "" : uploadLibraryId
      });
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
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
        }
      }

      if (successCount === 0) throw new Error("All uploads failed");

      set({
        showUploadModal: false,
        page: 1,
      });
      toast.info(`Uploaded ${successCount} books. Processing metadata...`);

      for (let attempt = 0; attempt < 12; attempt += 1) {
        await get().loadData();
        void queryClient.invalidateQueries({ queryKey: ["books"] });
        void queryClient.invalidateQueries({ queryKey: ["library"] });
        void queryClient.invalidateQueries({ queryKey: ["metadata"] });
        if (!get().books.some(book => book.status === "processing")) {
          toast.success(`Successfully processed ${successCount} books.`);
          break;
        }
        await sleep(1000);
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
      void queryClient.invalidateQueries({ queryKey: ["metadata"] });
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
    set({ loading: true, error: "" });
    try {
      await bookService.deleteBook(id);
      toast.success("Book deleted successfully!");
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
      void queryClient.invalidateQueries({ queryKey: ["metadata"] });
      await get().loadData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete book");
    } finally {
      set({ loading: false });
    }
  },

  archiveBook: async (id, archived) => {
    set({ loading: true, error: "" });
    try {
      const res = await bookService.archiveBook(id, archived);
      if (!res.status) throw new Error(res.message || "Failed to update archive state");
      toast.success(archived ? "Book archived" : "Book unarchived");
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
      await get().loadData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update archive state");
    } finally {
      set({ loading: false });
    }
  }
}));
