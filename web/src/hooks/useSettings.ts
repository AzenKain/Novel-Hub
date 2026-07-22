import { settingsService } from "@/services";
import { useSettingsStore } from "@/stores/settingsStore";
import type { PublicSettings } from "@/types";
import { useEffect } from "react";

let fetching = false;

function applySiteSettingsToDOM(settings: PublicSettings | null) {
  if (!settings?.site) return;
  const site = settings.site;
  
  if (site.title) {
    document.title = site.title;
  }
  
  if (site.favicon) {
    let link = document.querySelector("link[rel~='icon']") as HTMLLinkElement;
    if (!link) {
      link = document.createElement('link');
      link.rel = 'icon';
      document.head.appendChild(link);
    }
    link.href = site.favicon;
  }
}

export function usePublicSettings(): PublicSettings | null {
  const publicSettings = useSettingsStore(state => state.publicSettings);
  const setPublicSettings = useSettingsStore(state => state.setPublicSettings);

  useEffect(() => {
    applySiteSettingsToDOM(publicSettings);

    if (!fetching) {
      fetching = true;
      settingsService.getPublic().then((res) => {
        setPublicSettings(res.data || null);
        fetching = false;
      }).catch(() => {
        fetching = false;
      });
    }
  }, []);

  useEffect(() => {
    applySiteSettingsToDOM(publicSettings);
  }, [publicSettings]);

  return publicSettings;
}

export function isPolicyAllowed(policy: import("@/types").LibraryPolicy | undefined, libraryId?: string): boolean {
  if (!policy) return true;
  if (policy.mode === "disabled") return false;
  if (policy.mode === "selected_libraries") {
    return !!libraryId && (policy.library_ids || []).includes(libraryId);
  }
  return true;
}

export async function invalidatePublicSettings(): Promise<void> {
  fetching = false;
  try {
    const res = await settingsService.getPublic();
    useSettingsStore.getState().setPublicSettings(res.data || null);
    applySiteSettingsToDOM(res.data || null);
  } catch {
    useSettingsStore.getState().setPublicSettings(null);
  }
}
