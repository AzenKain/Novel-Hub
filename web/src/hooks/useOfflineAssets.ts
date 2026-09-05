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
      let resolved = html;

      const apiMatches = [
        ...html.matchAll(
          /\/api\/v1\/reader\/[^/"']+\/asset\/([^"'?#]+)(\?[^"']*)?/g,
        ),
      ];
      for (const match of apiMatches) {
        const fullUrl = match[0];
        const rawPath = decodeURIComponent(match[1]);
        const fileName = rawPath.split("/").pop() || rawPath;

        let blob = await offlineStore
          .getBlob(bookId, assetKey(rawPath))
          .catch(() => undefined);
        if (!blob && fileName !== rawPath) {
          blob = await offlineStore
            .getBlob(bookId, assetKey(fileName))
            .catch(() => undefined);
        }

        if (blob) {
          const blobUrl = track(URL.createObjectURL(blob));
          resolved = resolved.replaceAll(fullUrl, blobUrl);
        }
      }

      const coverBlob = await offlineStore
        .getBlob(bookId, "cover")
        .catch(() => undefined);
      if (coverBlob) {
        const coverBlobUrl = track(URL.createObjectURL(coverBlob));
        resolved = resolved.replace(
          /<img\b(?![^>]*\bsrc=)([^>]*\b(?:class|alt)=["'][^"']*\bcover\b[^"']*["'][^>]*)>/gi,
          (_m, attrs) => {
            return `<img src="${coverBlobUrl}" ${attrs}>`;
          },
        );
        resolved = resolved.replace(
          /<img\b([^>]*\bsrc=["']\s*["'][^>]*)>/gi,
          (_m, attrs) => {
            return `<img src="${coverBlobUrl}" ${attrs.replace(/src=["']\s*["']/, "")}>`;
          },
        );
      }

      return resolved;
    },
    [bookId, revoke, track],
  );

  const resolveBlobURL = useCallback(
    async (key: string) => {
      if (!bookId) return undefined;
      const blob = await offlineStore
        .getBlob(bookId, key)
        .catch(() => undefined);
      return blob ? track(URL.createObjectURL(blob)) : undefined;
    },
    [bookId, track],
  );

  return { resolveHTML, resolveBlobURL };
}
