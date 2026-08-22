import { create } from "zustand";
import { persist } from "zustand/middleware";
import { bookService } from "@/services/bookService";

export interface DownloadItem {
  id: string;          // unique download ID (bookId + fileId or just bookId)
  bookId: string;
  fileId?: string;
  title: string;
  coverUrl?: string;
  format: string;
  sizeBytes?: number;
  status: 'queued' | 'downloading' | 'completed' | 'failed';
  progress: number;    // 0-100
  speed?: number;      // bytes per second  
  error?: string;
  addedAt: number;     // timestamp
  completedAt?: number;
}

export interface DownloadManagerState {
  items: DownloadItem[];
  isOpen: boolean;     // panel visibility
  activeTab: 'active' | 'completed'; // active drawer tab
  isPaused: boolean;   // queue paused
  maxConcurrent: number; // max simultaneous downloads (default 3)
  
  // Actions
  open: () => void;
  close: () => void;
  toggle: () => void;
  setActiveTab: (tab: 'active' | 'completed') => void;
  addDownload: (item: Omit<DownloadItem, 'id' | 'status' | 'progress' | 'addedAt'>, autoStart?: boolean) => void;
  addBulkDownloads: (items: Array<Omit<DownloadItem, 'id' | 'status' | 'progress' | 'addedAt'>>, autoStart?: boolean) => void;
  removeDownload: (id: string) => void;
  retryDownload: (id: string) => void;
  clearCompleted: () => void;
  clearFailed: () => void;
  pauseQueue: () => void;
  resumeQueue: () => void;
  cancelAll: () => void;
  processQueue: () => void;
}

export const useDownloadManagerStore = create<DownloadManagerState>()(
  persist(
    (set, get) => ({
      items: [],
      isOpen: false,
      activeTab: 'active',
      isPaused: false,
      maxConcurrent: 3,

      open: () => set({ isOpen: true, activeTab: 'active' }),
      close: () => set({ isOpen: false }),
      toggle: () => set((state) => ({ isOpen: !state.isOpen, ...(state.isOpen ? {} : { activeTab: 'active' }) })),
      setActiveTab: (tab) => set({ activeTab: tab }),
      
      addDownload: (item, autoStart = false) => {
        set((state) => {
          const id = item.fileId ? `${item.bookId}-${item.fileId}` : item.bookId;
          if (state.items.some((i) => i.id === id)) return state;
          
          const isAnyDownloading = state.items.some((i) => i.status === 'downloading');
          return {
            items: [
              ...state.items,
              {
                ...item,
                id,
                status: 'queued',
                progress: 0,
                addedAt: Date.now(),
              },
            ],
            isOpen: false,
            isPaused: autoStart ? state.isPaused : (isAnyDownloading ? state.isPaused : true),
          };
        });
        if (autoStart && !get().isPaused) {
          get().processQueue();
        }
      },

      addBulkDownloads: (items, autoStart = false) => {
        set((state) => {
          const newItems: DownloadItem[] = [];
          for (const item of items) {
            const id = item.fileId ? `${item.bookId}-${item.fileId}` : item.bookId;
            if (!state.items.some((i) => i.id === id) && !newItems.some((i) => i.id === id)) {
              newItems.push({
                ...item,
                id,
                status: 'queued',
                progress: 0,
                addedAt: Date.now(),
              });
            }
          }
          if (newItems.length === 0) return state;
          const isAnyDownloading = state.items.some((i) => i.status === 'downloading');
          return {
            items: [...state.items, ...newItems],
            isOpen: true,
            activeTab: 'active',
            isPaused: autoStart ? state.isPaused : (isAnyDownloading ? state.isPaused : true),
          };
        });
        if (autoStart && !get().isPaused) {
          get().processQueue();
        }
      },

      removeDownload: (id: string) => {
        set((state) => ({
          items: state.items.filter((i) => i.id !== id),
        }));
        get().processQueue();
      },

      retryDownload: (id: string) => {
        set((state) => ({
          items: state.items.map((i) => 
            i.id === id ? { ...i, status: 'queued', error: undefined, progress: 0 } : i
          ),
        }));
        get().processQueue();
      },

      clearCompleted: () => {
        set((state) => ({
          items: state.items.filter((i) => i.status !== 'completed'),
        }));
      },

      clearFailed: () => {
        set((state) => ({
          items: state.items.filter((i) => i.status !== 'failed'),
        }));
      },

      pauseQueue: () => {
        set({ isPaused: true });
      },

      resumeQueue: () => {
        set({ isPaused: false });
        get().processQueue();
      },

      cancelAll: () => {
        set((state) => ({
          items: state.items.filter((i) => i.status !== 'queued' && i.status !== 'downloading'),
        }));
      },

      processQueue: async () => {
        const state = get();
        if (state.isPaused) return;

        const downloading = state.items.filter((i) => i.status === 'downloading');
        if (downloading.length >= state.maxConcurrent) return;

        const queued = state.items.find((i) => i.status === 'queued');
        if (!queued) return;

        // Mark as downloading
        set((state) => ({
          items: state.items.map((i) =>
            i.id === queued.id ? { ...i, status: 'downloading', progress: 0 } : i
          ),
        }));

        try {
          const url = bookService.getDownloadUrl(queued.bookId, queued.fileId);
          const response = await fetch(url);
          
          if (!response.ok) {
            throw new Error(`Download failed with status ${response.status}`);
          }

          const contentLength = response.headers.get('content-length');
          const totalBytes = contentLength ? parseInt(contentLength, 10) : queued.sizeBytes || 0;
          
          if (!response.body) throw new Error("No response body");

          const reader = response.body.getReader();
          const chunks: BlobPart[] = [];
          let receivedBytes = 0;
          const startTime = Date.now();
          let lastUpdateTime = startTime;
          let lastReceivedBytes = 0;

          while (true) {
            const { done, value } = await reader.read();
            
            // Check if cancelled/paused midway
            const currentItem = get().items.find((i) => i.id === queued.id);
            if (!currentItem || currentItem.status !== 'downloading') {
              // Was removed or cancelled
              reader.cancel();
              return;
            }

            if (done) break;

            chunks.push(value);
            receivedBytes += value.length;

            const now = Date.now();
            if (now - lastUpdateTime > 500) {
              const timeDiffSeconds = (now - lastUpdateTime) / 1000;
              const bytesDiff = receivedBytes - lastReceivedBytes;
              const speed = bytesDiff / timeDiffSeconds;
              
              set((state) => ({
                items: state.items.map((i) =>
                  i.id === queued.id ? { 
                    ...i, 
                    progress: totalBytes ? Math.round((receivedBytes / totalBytes) * 100) : 0,
                    speed 
                  } : i
                ),
              }));
              
              lastUpdateTime = now;
              lastReceivedBytes = receivedBytes;
            }
          }

          const blob = new Blob(chunks, { type: response.headers.get('content-type') || 'application/octet-stream' });
          const blobUrl = URL.createObjectURL(blob);
          
          const a = document.createElement('a');
          a.href = blobUrl;
          
          const contentDisposition = response.headers.get('content-disposition');
          let filename = `${queued.title}.${queued.format}`;
          if (contentDisposition) {
            const filenameMatch = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
            if (filenameMatch && filenameMatch[1]) {
              filename = filenameMatch[1].replace(/['"]/g, '');
            }
          }
          
          a.download = filename;
          document.body.appendChild(a);
          a.click();
          document.body.removeChild(a);
          URL.revokeObjectURL(blobUrl);

          set((state) => ({
            items: state.items.map((i) =>
              i.id === queued.id ? { ...i, status: 'completed', progress: 100, completedAt: Date.now(), speed: undefined } : i
            ),
          }));

        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : "Unknown error";
          set((state) => ({
            items: state.items.map((i) =>
              i.id === queued.id ? { ...i, status: 'failed', error: errorMessage, speed: undefined } : i
            ),
          }));
        } finally {
          setTimeout(() => {
            get().processQueue();
          }, 400);
        }
      }
    }),
    {
      name: "novelhub-downloads",
      partialize: (state) => ({ 
        items: state.items.map(item => 
          item.status === 'downloading' 
            ? { ...item, status: 'failed', error: 'download_manager.interrupted' } 
            : item
        )
      })
    }
  )
);
