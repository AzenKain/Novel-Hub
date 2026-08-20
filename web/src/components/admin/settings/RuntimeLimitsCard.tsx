import { useUpdateAdminSettingsMutation } from "@/hooks";
import { useSettingsAdminStore } from "@/stores";
import type { RuntimeLimits } from "@/types";
import { Gauge, Loader2, Save } from "lucide-react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";
import { useShallow } from "zustand/react/shallow";

const MEBIBYTE = 1024 * 1024;
const BYTE_FIELDS: (keyof RuntimeLimits)[] = [
  "upload_chunk_bytes",
  "upload_bytes",
  "cover_bytes",
  "site_asset_bytes",
  "soundscape_bytes",
  "font_bytes",
];
const SECOND_FIELDS: (keyof RuntimeLimits)[] = [
  "rate_limit_auth_window_seconds",
];
const LIMIT_FIELDS: (keyof RuntimeLimits)[] = [
  "upload_chunk_bytes",
  "upload_chunks",
  "upload_sessions",
  "upload_bytes",
  "upload_session_ttl_seconds",
  "cover_bytes",
  "site_asset_bytes",
  "soundscape_bytes",
  "font_bytes",
  "rate_limit_auth",
  "rate_limit_auth_window_seconds",
];
const SETTING_KEYS: Record<keyof RuntimeLimits, string> = {
  upload_chunk_bytes: "limits.upload_chunk_bytes",
  upload_chunks: "limits.upload_chunks",
  upload_sessions: "limits.upload_sessions",
  upload_bytes: "limits.upload_bytes",
  upload_session_ttl_seconds: "limits.upload_session_ttl_seconds",
  cover_bytes: "limits.cover_bytes",
  site_asset_bytes: "limits.site_asset_bytes",
  soundscape_bytes: "limits.soundscape_bytes",
  font_bytes: "limits.font_bytes",
  rate_limit_auth: "limits.rate_limit_auth",
  rate_limit_auth_window_seconds: "limits.rate_limit_auth_window_seconds",
};

export function RuntimeLimitsCard() {
  const { t } = useTranslation();
  const mutation = useUpdateAdminSettingsMutation();
  const { limits, limitBounds, setLimits } = useSettingsAdminStore(useShallow((state) => ({
    limits: state.limits,
    limitBounds: state.limitBounds,
    setLimits: state.setLimits,
  })));

  if (!limits || !limitBounds) return null;

  const isBytes = (key: keyof RuntimeLimits) => BYTE_FIELDS.includes(key);
  const isTTL = (key: keyof RuntimeLimits) => key === "upload_session_ttl_seconds";
  const isSeconds = (key: keyof RuntimeLimits) => SECOND_FIELDS.includes(key);
  const displayValue = (key: keyof RuntimeLimits, value: number) => isBytes(key) ? value / MEBIBYTE : isTTL(key) ? value / 60 : value;
  const backendValue = (key: keyof RuntimeLimits, value: number) => Math.round(isBytes(key) ? value * MEBIBYTE : isTTL(key) ? value * 60 : value);
  const valid = LIMIT_FIELDS.every((key) => limits[key] >= limitBounds.min[key] && limits[key] <= limitBounds.max[key]);

  function save() {
    if (!limits) return;
    const payload = Object.fromEntries(LIMIT_FIELDS.map((key) => [SETTING_KEYS[key], limits[key]]));
    mutation.mutate(payload, {
      onSuccess: () => toast.success(t("settings.saved_success")),
      onError: (error) => toast.error(error instanceof Error ? error.message : String(error)),
    });
  }

  return (
    <div className="card bg-base-100 border border-base-200 shadow-sm">
      <div className="card-body p-4 sm:p-5">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-3">
          <div>
            <div className="flex items-center gap-2 mb-0.5">
              <Gauge className="h-5 w-5 text-primary" />
              <h2 className="card-title text-lg">{t("settings.runtime_limits")}</h2>
            </div>
            <p className="text-xs text-base-content/50">{t("settings.runtime_limits_desc")}</p>
          </div>
          <button type="button" onClick={save} disabled={mutation.isPending || !valid} className="btn btn-primary btn-sm gap-1 shrink-0 self-start sm:self-center">
            {mutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            {t("settings.save_runtime_limits")}
          </button>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {LIMIT_FIELDS.filter(
            (key) => key !== "rate_limit_auth" && key !== "rate_limit_auth_window_seconds"
          ).map((key) => {
            const byteField = isBytes(key);
            return (
              <label key={key} className="flex flex-col gap-1.5">
                <span className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">{t(`settings.limit_${key}`)}</span>
                <div className="join w-full">
                  <input
                    type="number"
                    className="input input-bordered join-item w-full"
                    min={displayValue(key, limitBounds.min[key])}
                    max={displayValue(key, limitBounds.max[key])}
                    step={byteField ? 0.01 : 1}
                    value={displayValue(key, limits[key])}
                    onChange={(event) => {
                      const value = event.currentTarget.valueAsNumber;
                      if (Number.isFinite(value)) setLimits({ ...limits, [key]: backendValue(key, value) });
                    }}
                  />
                  <span className="join-item flex items-center border border-base-300 bg-base-200 px-3 text-sm">
                    {t(byteField ? "settings.unit_mib" : isTTL(key) ? "settings.unit_minutes" : isSeconds(key) ? "settings.unit_seconds" : "settings.unit_count")}
                  </span>
                </div>
                <span className="text-xs text-base-content/50 pl-1">
                  {t("settings.limit_range", {
                    min: displayValue(key, limitBounds.min[key]).toLocaleString(),
                    max: displayValue(key, limitBounds.max[key]).toLocaleString(),
                  })}
                </span>
              </label>
            );
          })}

          <div className="col-span-full border-t border-base-200 mt-2 pt-4">
            <div className="flex flex-col gap-2">
              <span className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                {t("settings.rate_limiting_title")}
              </span>
              <div className="flex flex-wrap items-center gap-2 p-3 bg-base-200/40 border border-base-300/50 rounded-xl text-sm font-medium text-base-content/85">
                <span>{t("settings.rate_limit_allow_max")}</span>
                <input
                  type="number"
                  className="input input-bordered input-sm w-20 text-center focus:outline-none"
                  min={limitBounds.min.rate_limit_auth}
                  max={limitBounds.max.rate_limit_auth}
                  value={limits.rate_limit_auth}
                  onChange={(event) => {
                    const value = event.currentTarget.valueAsNumber;
                    if (Number.isFinite(value)) setLimits({ ...limits, rate_limit_auth: value });
                  }}
                />
                <span>{t("settings.rate_limit_every")}</span>
                <input
                  type="number"
                  className="input input-bordered input-sm w-24 text-center focus:outline-none"
                  min={limitBounds.min.rate_limit_auth_window_seconds}
                  max={limitBounds.max.rate_limit_auth_window_seconds}
                  value={limits.rate_limit_auth_window_seconds}
                  onChange={(event) => {
                    const value = event.currentTarget.valueAsNumber;
                    if (Number.isFinite(value)) setLimits({ ...limits, rate_limit_auth_window_seconds: value });
                  }}
                />
                <span>{t("settings.rate_limit_per_ip")}</span>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-1.5 pl-1 mt-1 text-[11px] text-base-content/50">
                <div>
                  • {t("settings.limit_rate_limit_auth")}: {t("settings.limit_range", {
                    min: limitBounds.min.rate_limit_auth.toLocaleString(),
                    max: limitBounds.max.rate_limit_auth.toLocaleString(),
                  })}
                </div>
                <div>
                  • {t("settings.limit_rate_limit_auth_window_seconds")}: {t("settings.limit_range", {
                    min: limitBounds.min.rate_limit_auth_window_seconds.toLocaleString(),
                    max: limitBounds.max.rate_limit_auth_window_seconds.toLocaleString(),
                  })}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
