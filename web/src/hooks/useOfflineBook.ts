import { API_BASE, getMediaUrl } from "@/config/api";
import { offlineStore } from "@/lib/offlineStore";
import { readerService } from "@/services";
import type { Book } from "@/types";
import { useCallback, useEffect, useState } from "react";

type Status = "unknown" | "absent" | "downloading" | "ready";

const RAW_FILE_FORMATS = /^(pdf|mp3|m4a|m4b|flac)$/i;

export const rawFileKey = (fileId: string) => `file:${fileId}`;

export const assetKey = (path: string) => `asset:${path}`;

function assetPathsFrom(html: string): string[] {
  const paths = new Set<string>();
  for (const match of html.matchAll(
    /\/api\/v1\/reader\/[^/"']+\/asset\/([^"'?#]+)/g,
  )) {
    paths.add(decodeURIComponent(match[1]));
  }
  return [...paths];
}

async function fetchBlob(url: string): Promise<Blob> {
  const res = await fetch(url, { credentials: "include" });
  if (!res.ok) throw new Error(`fetch_failed_${res.status}`);
  return res.blob();
}

export function useOfflineBook(bookId?: string, fileId?: string) {
  const [status, setStatus] = useState<Status>("unknown");
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!bookId) return;
    let active = true;
    void offlineStore
      .getBook(bookId)
      .then((entry) => active && setStatus(entry ? "ready" : "absent"))
      .catch(() => active && setStatus("absent"));
    return () => {
      active = false;
    };
  }, [bookId]);

  const download = useCallback(async () => {
    if (!bookId) return;
    setStatus("downloading");
    setProgress(0);
    setError(null);
    try {
      const res = await readerService.getBootstrap(bookId, fileId);
      if (!res.status || !res.data)
        throw new Error(res.message || "bootstrap_failed");

      const book: Book = res.data.book;
      const activeFile =
        (book.files || []).find((file) => file.id === fileId) ||
        book.files?.[0];
      const rawFile =
        activeFile && RAW_FILE_FORMATS.test(activeFile.format)
          ? activeFile
          : undefined;
      const chapters = [...res.data.chapters].sort(
        (a, b) => a.chapter_index - b.chapter_index,
      );

      if (rawFile) {
        const query = `?file_id=${encodeURIComponent(rawFile.id)}`;
        const blob = await fetchBlob(
          `${API_BASE}/reader/${encodeURIComponent(bookId)}/file${query}`,
        );
        await offlineStore.saveBlob(bookId, rawFileKey(rawFile.id), blob);
        setProgress(100);
      } else {
        const query = fileId ? `?file_id=${encodeURIComponent(fileId)}` : "";
        const pending: string[] = [];
        for (let i = 0; i < chapters.length; i++) {
          const html = await readerService.getChapterHtml(
            bookId,
            chapters[i].id,
            fileId,
          );
          await offlineStore.saveChapter(bookId, chapters[i].id, html);
          for (const path of assetPathsFrom(html)) {
            if (!pending.includes(path)) pending.push(path);
          }
          setProgress(
            Math.round(
              ((i + 1) / (chapters.length + pending.length || 1)) * 100,
            ),
          );
        }
        for (let i = 0; i < pending.length; i++) {
          const rawPath = pending[i];
          const url = `${API_BASE}/reader/${encodeURIComponent(bookId)}/asset/${rawPath
            .split("/")
            .map(encodeURIComponent)
            .join("/")}${query}`;
          const blob = await fetchBlob(url).catch(() => undefined);
          if (blob) {
            await offlineStore.saveBlob(bookId, assetKey(rawPath), blob);
            const fileName = rawPath.split("/").pop();
            if (fileName && fileName !== rawPath) {
              await offlineStore.saveBlob(bookId, assetKey(fileName), blob);
            }
          }
          setProgress(
            Math.round(
              ((chapters.length + i + 1) / (chapters.length + pending.length)) *
                100,
            ),
          );
        }
      }

      if (book.cover_url) {
        const coverBlob = await fetchBlob(getMediaUrl(book.cover_url)).catch(
          () => undefined,
        );
        if (coverBlob) await offlineStore.saveBlob(bookId, "cover", coverBlob);
      }

      await offlineStore.saveBook({ book, chapters, savedAt: Date.now() });
      setStatus("ready");
    } catch (err) {
      await offlineStore.deleteBook(bookId).catch(() => undefined);
      setError(err instanceof Error ? err.message : String(err));
      setStatus("absent");
    }
  }, [bookId, fileId]);

  const remove = useCallback(async () => {
    if (!bookId) return;
    await offlineStore.deleteBook(bookId);
    setStatus("absent");
    setProgress(0);
  }, [bookId]);

  return { status, progress, error, download, remove };
}
