import { offlineStore } from "@/lib/offlineStore";
import { assetKey } from "./useOfflineBook";
import { useCallback, useEffect, useRef } from "react";

export function useOfflineAssets(bookId?: string) {
  const urls = useRef<string[]>([]);

  const revoke = useCallback(() => {
    for (const url of urls.current) URL.revokeObjectURL(url);
    urls.current = [];
  }, []);

  useEffect(() => revoke, [revoke]);

  const track = useCallback((url: string) => {
    urls.current.push(url);
    return url;
  }, []);

  const resolveHTML = useCallback(
    async (html: string) => {
      if (!bookId) return html;
      revoke();
      const matches = [...html.matchAll(/\/api\/v1\/reader\/[^/"']+\/asset\/([^"'?]+)(\?[^"']*)?/g)];
      let resolved = html;
      for (const match of matches) {
        const blob = await offlineStore.getBlob(bookId, assetKey(decodeURIComponent(match[1]))).catch(() => undefined);
        if (blob) resolved = resolved.split(match[0]).join(track(URL.createObjectURL(blob)));
      }
      return resolved;
    },
    [bookId, revoke, track],
  );

  const resolveBlobURL = useCallback(
    async (key: string) => {
      if (!bookId) return undefined;
      const blob = await offlineStore.getBlob(bookId, key).catch(() => undefined);
      return blob ? track(URL.createObjectURL(blob)) : undefined;
    },
    [bookId, track],
  );

  return { resolveHTML, resolveBlobURL };
}
