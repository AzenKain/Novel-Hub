import React from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";
import {
  Download, X, Pause, Play, RotateCcw, Trash2,
  CheckCircle2, AlertCircle, Clock, Loader2, Info
} from "lucide-react";
import { useDownloadManagerStore } from "@/stores/downloadManagerStore";
import { getMediaUrl } from "@/config/api";
import { formatFileSize } from "@/lib/bookDetail";

export const DownloadManagerPanel: React.FC = () => {
  const { t } = useTranslation();
  const location = useLocation();

  const {
    items,
    isOpen,
    activeTab,
    isPaused,
    open,
    close,
    setActiveTab,
    removeDownload,
    retryDownload,
    clearCompleted,
    clearFailed,
    pauseQueue,
    resumeQueue,
    cancelAll,
  } = useDownloadManagerStore(useShallow((state) => ({
    items: state.items,
    isOpen: state.isOpen,
    activeTab: state.activeTab,
    isPaused: state.isPaused,
    open: state.open,
    close: state.close,
    setActiveTab: state.setActiveTab,
    removeDownload: state.removeDownload,
    retryDownload: state.retryDownload,
    clearCompleted: state.clearCompleted,
    clearFailed: state.clearFailed,
    pauseQueue: state.pauseQueue,
    resumeQueue: state.resumeQueue,
    cancelAll: state.cancelAll,
  })));

  const isExcludedRoute =
    location.pathname.startsWith('/reader') ||
    location.pathname.startsWith('/offline') ||
    ['/login', '/register', '/forgot-password', '/setup', '/activate'].includes(location.pathname);

  if (isExcludedRoute) return null;

  const activeItems = items.filter(i => i.status === 'downloading' || i.status === 'queued' || i.status === 'failed');
  const completedItems = items.filter(i => i.status === 'completed');
  
  const downloadingItems = activeItems.filter(i => i.status === 'downloading');
  const queuedItems = activeItems.filter(i => i.status === 'queued');
  const failedItems = activeItems.filter(i => i.status === 'failed');

  const formatSpeed = (bytesPerSec?: number) => {
    if (!bytesPerSec) return "";
    return t("download_manager.speed", { speed: formatFileSize(bytesPerSec) });
  };

  // Render floating bubble when panel is closed but downloads exist
  if (!isOpen) {
    if (items.length === 0) return null;

    const isDownloading = downloadingItems.length > 0;
    const hasFailed = failedItems.length > 0;

    return (
      <aside 
        aria-label={t("download_manager.title", "Downloads")}
        className="fixed bottom-4 right-4 sm:bottom-6 sm:right-6 z-95 flex items-center gap-2 group animate-in fade-in slide-in-from-bottom-4 duration-300 max-w-[calc(100vw-2rem)]"
      >
        <button
          type="button"
          onClick={open}
          className={`btn shadow-2xl rounded-full px-3.5 sm:px-4 h-11 sm:h-12 min-h-11 border gap-2 sm:gap-2.5 transition-all duration-300 hover:scale-105 ${
            hasFailed
              ? "btn-error text-error-content shadow-error/30"
              : isDownloading
              ? "btn-primary text-primary-content shadow-primary/30 ring-2 ring-primary/40 ring-offset-2 ring-offset-base-100"
              : "bg-base-200/95 border-base-300 backdrop-blur-md text-base-content hover:bg-base-300 shadow-base-content/10"
          }`}
          title={t("download_manager.open_manager", "Open Download Manager")}
          aria-label={t("download_manager.open_manager", "Open Download Manager")}
        >
          {isDownloading ? (
            <Loader2 className="w-4 h-4 sm:w-5 sm:h-5 animate-spin shrink-0" />
          ) : hasFailed ? (
            <AlertCircle className="w-4 h-4 sm:w-5 sm:h-5 shrink-0" />
          ) : completedItems.length > 0 && activeItems.length === 0 ? (
            <CheckCircle2 className="w-4 h-4 sm:w-5 sm:h-5 text-success shrink-0" />
          ) : (
            <Download className="w-4 h-4 sm:w-5 sm:h-5 shrink-0" />
          )}

          <div className="flex flex-col items-start text-left leading-none min-w-0">
            <span className="text-[11px] sm:text-xs font-bold tracking-tight truncate max-w-30 sm:max-w-none">
              {isDownloading
                ? t("download_manager.downloading")
                : hasFailed
                ? t("download_manager.failed")
                : t("download_manager.title")}
            </span>
            <span className="text-[9px] sm:text-[10px] opacity-80 mt-0.5">
              {activeItems.length > 0
                ? `${activeItems.length} ${t("download_manager.items", "items")}`
                : `${completedItems.length} ${t("download_manager.completed_short", "completed")}`}
            </span>
          </div>

          {activeItems.length > 0 && (
            <span className="badge badge-xs sm:badge-sm badge-neutral font-black ml-0.5 sm:ml-1">
              {activeItems.length}
            </span>
          )}
        </button>
      </aside>
    );
  }

  return (
    <>
      <div 
        className="fixed inset-0 bg-black/50 z-100 transition-opacity" 
        onClick={close}
      />
      <div className="fixed top-0 right-0 h-full w-full sm:max-w-md md:max-w-lg bg-base-100 shadow-2xl z-101 flex flex-col transform transition-transform duration-300">
        <div className="flex items-center justify-between p-3.5 sm:p-4 border-b border-base-300">
          <div className="flex items-center gap-2 font-bold text-base sm:text-lg">
            <Download className="w-5 h-5 text-primary" />
            {t("download_manager.title")}
          </div>
          <button onClick={close} className="btn btn-ghost btn-sm btn-square" aria-label={t("common.close", "Close")}>
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="flex px-4 border-b border-base-300">
          <button
            className={`flex-1 py-3 font-medium text-sm flex items-center justify-center gap-2 border-b-2 transition-colors ${
              activeTab === "active" ? "border-primary text-primary" : "border-transparent text-base-content/70 hover:text-base-content"
            }`}
            onClick={() => setActiveTab("active")}
          >
            {t("download_manager.downloading")}
            <div className="badge badge-sm badge-neutral">{activeItems.length}</div>
          </button>
          <button
            className={`flex-1 py-3 font-medium text-sm flex items-center justify-center gap-2 border-b-2 transition-colors ${
              activeTab === "completed" ? "border-primary text-primary" : "border-transparent text-base-content/70 hover:text-base-content"
            }`}
            onClick={() => setActiveTab("completed")}
          >
            {t("download_manager.completed")}
            <div className="badge badge-sm badge-neutral">{completedItems.length}</div>
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-6">
          {/* Multi-download permission notice */}
          <div className="rounded-xl bg-info/10 border border-info/20 p-3 flex items-start gap-2.5 text-xs text-base-content/80 leading-relaxed">
            <Info className="w-4 h-4 text-info shrink-0 mt-0.5" />
            <p>
              {t("download_manager.permission_notice", "💡 Note: If your browser prompts for permission to download multiple files (Automatic Downloads), please click \"Allow\".")}
            </p>
          </div>

          {activeTab === "active" ? (
            <>
              {activeItems.length === 0 ? (
                <div className="flex flex-col items-center justify-center h-48 text-base-content/50 gap-2">
                  <Download className="w-8 h-8 opacity-50" />
                  <p>{t("download_manager.no_downloads")}</p>
                </div>
              ) : (
                <>
                  {downloadingItems.length > 0 && (
                    <div className="flex flex-col gap-3">
                      <div className="flex items-center justify-between">
                        <h3 className="font-semibold text-sm text-base-content/70 uppercase tracking-wider">
                          {t("download_manager.downloading")} ({downloadingItems.length})
                        </h3>
                        {isPaused ? (
                          <button onClick={resumeQueue} className="btn btn-xs btn-ghost gap-1">
                            <Play className="w-3 h-3" /> {t("download_manager.resume")}
                          </button>
                        ) : (
                          <button onClick={pauseQueue} className="btn btn-xs btn-ghost gap-1">
                            <Pause className="w-3 h-3" /> {t("download_manager.pause")}
                          </button>
                        )}
                      </div>
                      {downloadingItems.map(item => (
                        <div key={item.id} className="bg-base-200 rounded-lg p-3 flex gap-3 items-center">
                          {item.coverUrl ? (
                            <img src={getMediaUrl(item.coverUrl, item.bookId)} alt="" className="w-10 h-14 object-cover rounded shadow-sm" />
                          ) : (
                            <div className="w-10 h-14 bg-base-300 rounded flex items-center justify-center">
                              <Download className="w-4 h-4 opacity-50" />
                            </div>
                          )}
                          <div className="flex-1 min-w-0">
                            <h4 className="font-medium text-sm truncate">{item.title}</h4>
                            <div className="flex items-center gap-2 mt-1 mb-2">
                              <span className="text-xs font-semibold text-primary uppercase">{item.format}</span>
                              <span className="text-xs text-base-content/70">{formatFileSize(item.sizeBytes || 0)}</span>
                              {item.speed && (
                                <span className="text-xs text-base-content/70 ml-auto">{formatSpeed(item.speed)}</span>
                              )}
                            </div>
                            <progress className="progress progress-primary w-full h-1.5" value={item.progress} max="100"></progress>
                          </div>
                          <button onClick={() => removeDownload(item.id)} className="btn btn-ghost btn-square btn-sm text-base-content/50 hover:text-error" title={t("common.cancel", "Cancel")}>
                            <X className="w-4 h-4" />
                          </button>
                        </div>
                      ))}
                    </div>
                  )}

                  {failedItems.length > 0 && (
                    <div className="flex flex-col gap-3">
                      <div className="flex items-center justify-between">
                        <h3 className="font-semibold text-sm text-error uppercase tracking-wider">
                          {t("download_manager.failed")} ({failedItems.length})
                        </h3>
                        <div className="flex gap-1">
                          <button onClick={() => {
                            failedItems.forEach(i => retryDownload(i.id));
                          }} className="btn btn-xs btn-ghost gap-1">
                            <RotateCcw className="w-3 h-3" /> {t("download_manager.retry_all")}
                          </button>
                          <button onClick={clearFailed} className="btn btn-xs btn-ghost gap-1 text-error hover:bg-error/20 hover:text-error">
                            <Trash2 className="w-3 h-3" /> {t("download_manager.clear_all")}
                          </button>
                        </div>
                      </div>
                      {failedItems.map(item => (
                        <div key={item.id} className="bg-error/10 border border-error/20 rounded-lg p-3 flex gap-3 items-center">
                          {item.coverUrl ? (
                            <img src={getMediaUrl(item.coverUrl, item.bookId)} alt="" className="w-10 h-14 object-cover rounded shadow-sm opacity-80" />
                          ) : (
                            <div className="w-10 h-14 bg-base-300 rounded flex items-center justify-center opacity-80">
                              <AlertCircle className="w-4 h-4 text-error" />
                            </div>
                          )}
                          <div className="flex-1 min-w-0">
                            <h4 className="font-medium text-sm truncate">{item.title}</h4>
                            <div className="flex items-center gap-2 mt-1">
                              <AlertCircle className="w-3 h-3 text-error" />
                              <span className="text-xs text-error truncate">
                                {item.error === 'download_manager.interrupted' ? t(item.error) : item.error}
                              </span>
                            </div>
                          </div>
                          <div className="flex flex-col gap-1">
                            <button onClick={() => retryDownload(item.id)} className="btn btn-ghost btn-square btn-sm hover:text-primary" title={t("download_manager.redownload", "Re-download")}>
                              <RotateCcw className="w-4 h-4" />
                            </button>
                            <button onClick={() => removeDownload(item.id)} className="btn btn-ghost btn-square btn-sm text-base-content/50 hover:text-error" title={t("common.delete", "Delete")}>
                              <X className="w-4 h-4" />
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}

                  {queuedItems.length > 0 && (
                    <div className="flex flex-col gap-3">
                      <div className="flex items-center justify-between">
                        <h3 className="font-semibold text-sm text-base-content/70 uppercase tracking-wider">
                          {t("download_manager.queued")} ({queuedItems.length})
                        </h3>
                        <div className="flex items-center gap-1.5">
                          {(isPaused || downloadingItems.length === 0) && (
                            <button
                              onClick={resumeQueue}
                              className="btn btn-xs btn-primary gap-1 font-bold shadow-xs"
                            >
                              <Play className="w-3 h-3" />
                              {t("download_manager.start_downloads", "Start Downloads")}
                            </button>
                          )}
                          <button onClick={cancelAll} className="btn btn-xs btn-ghost gap-1">
                            <X className="w-3 h-3" /> {t("download_manager.cancel_all")}
                          </button>
                        </div>
                      </div>
                      {queuedItems.map(item => (
                        <div key={item.id} className="bg-base-200/50 rounded-lg p-3 flex gap-3 items-center opacity-85 hover:opacity-100 transition-opacity">
                          {item.coverUrl ? (
                            <img src={getMediaUrl(item.coverUrl, item.bookId)} alt="" className="w-10 h-14 object-cover rounded shadow-sm" />
                          ) : (
                            <div className="w-10 h-14 bg-base-300 rounded flex items-center justify-center">
                              <Clock className="w-4 h-4 opacity-50" />
                            </div>
                          )}
                          <div className="flex-1 min-w-0">
                            <h4 className="font-medium text-sm truncate">{item.title}</h4>
                            <div className="flex items-center gap-2 mt-1">
                              <span className="text-xs font-semibold text-base-content/70 uppercase">{item.format}</span>
                              <span className="text-xs text-base-content/50">{formatFileSize(item.sizeBytes || 0)}</span>
                            </div>
                          </div>
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => resumeQueue()}
                              className="btn btn-ghost btn-square btn-sm text-primary hover:bg-primary/10"
                              title={t("download_manager.start_downloads", "Start Downloads")}
                            >
                              <Play className="w-4 h-4" />
                            </button>
                            <button
                              onClick={() => removeDownload(item.id)}
                              className="btn btn-ghost btn-square btn-sm text-base-content/50 hover:text-error"
                              title={t("common.cancel", "Cancel")}
                            >
                              <X className="w-4 h-4" />
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </>
              )}
            </>
          ) : (
            <>
              {completedItems.length === 0 ? (
                <div className="flex flex-col items-center justify-center h-48 text-base-content/50 gap-2">
                  <CheckCircle2 className="w-8 h-8 opacity-50" />
                  <p>{t("download_manager.no_downloads")}</p>
                </div>
              ) : (
                <div className="flex flex-col gap-3">
                  <div className="flex items-center justify-between">
                    <h3 className="font-semibold text-sm text-success uppercase tracking-wider">
                      {t("download_manager.completed")} ({completedItems.length})
                    </h3>
                    <button onClick={clearCompleted} className="btn btn-xs btn-ghost gap-1 hover:text-error">
                      <Trash2 className="w-3 h-3" /> {t("download_manager.clear_all")}
                    </button>
                  </div>
                  {completedItems.map(item => (
                    <div key={item.id} className="bg-success/5 border border-success/20 rounded-lg p-3 flex gap-3 items-center">
                      {item.coverUrl ? (
                        <img src={getMediaUrl(item.coverUrl, item.bookId)} alt="" className="w-10 h-14 object-cover rounded shadow-sm" />
                      ) : (
                        <div className="w-10 h-14 bg-base-300 rounded flex items-center justify-center">
                          <CheckCircle2 className="w-4 h-4 text-success" />
                        </div>
                      )}
                      <div className="flex-1 min-w-0">
                        <h4 className="font-medium text-sm truncate">{item.title}</h4>
                        <div className="flex items-center gap-2 mt-1">
                          <span className="text-xs font-semibold text-success uppercase">{item.format}</span>
                          <span className="text-xs text-base-content/70">{formatFileSize(item.sizeBytes || 0)}</span>
                          {item.completedAt && (
                            <span className="text-xs text-base-content/50 ml-auto">
                              {new Date(item.completedAt).toLocaleDateString()}
                            </span>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-1">
                        <button 
                          onClick={() => retryDownload(item.id)} 
                          className="btn btn-ghost btn-square btn-sm hover:text-primary"
                          title={t("download_manager.redownload", "Re-download")}
                          aria-label={t("download_manager.redownload", "Re-download")}
                        >
                          <RotateCcw className="w-4 h-4" />
                        </button>
                        <button 
                          onClick={() => removeDownload(item.id)} 
                          className="btn btn-ghost btn-square btn-sm text-base-content/50 hover:text-error"
                          title={t("common.delete", "Delete")}
                          aria-label={t("common.delete", "Delete")}
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </>
  );
};
