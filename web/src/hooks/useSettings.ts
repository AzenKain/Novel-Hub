import { settingsService } from "@/services";
import type { PublicSettings } from "@/types";
import { useEffect, useState } from "react";

let cached: PublicSettings | null = null;
let fetching = false;
let listeners: Array<(s: PublicSettings | null) => void> = [];

function notify() {
  listeners.forEach((fn) => fn(cached));
}

export function usePublicSettings(): PublicSettings | null {
  const [settings, setSettings] = useState<PublicSettings | null>(cached);

  useEffect(() => {
    if (cached) {
      setSettings(cached);
      return;
    }
    listeners.push(setSettings);
    if (!fetching) {
      fetching = true;
      settingsService.getPublic().then((res) => {
        cached = res.data || null;
        fetching = false;
        notify();
      });
    }
    return () => {
      listeners = listeners.filter((fn) => fn !== setSettings);
    };
  }, []);

  return settings;
}

// Forces a re-fetch of the public settings on the next render. Call this after
// an action that changes setup state (e.g. completing the setup wizard) so the
// SetupGuard sees the fresh `setup_completed` value instead of the stale cache.
export async function invalidatePublicSettings(): Promise<void> {
  cached = null;
  fetching = false;
  try {
    const res = await settingsService.getPublic();
    cached = res.data || null;
  } catch {
    cached = null;
  }
  notify();
}
