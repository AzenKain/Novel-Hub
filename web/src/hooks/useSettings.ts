import { settingsService } from "@/services";
import { useSettingsStore } from "@/stores/settingsStore";
import { useAuthStore } from "@/stores/authStore";
import type { PublicSettings } from "@/types";
import { useEffect } from "react";
import { useShallow } from "zustand/react/shallow";

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
  const { publicSettings } = useSettingsStore(
    useShallow((state) => ({ publicSettings: state.publicSettings }))
  );
  const { user } = useAuthStore(
    useShallow((state) => ({ user: state.user }))
  );

  useEffect(() => {
    applySiteSettingsToDOM(publicSettings);

    settingsService.getPublic().then((res) => {
      useSettingsStore.getState().setPublicSettings(res.data || null);
    }).catch(() => {});
  }, [user]);

  useEffect(() => {
    applySiteSettingsToDOM(publicSettings);
  }, [publicSettings]);

  return publicSettings;
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
