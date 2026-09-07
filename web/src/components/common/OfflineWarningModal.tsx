import { AlertTriangle, X } from "lucide-react";
import { useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";

const SUPPRESS_KEY = "novelhub:offline-warning-until";

const formatBytes = (bytes: number) => {
  if (bytes <= 0) return "";
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(0)} KB`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
  return `${(bytes / 1024 ** 3).toFixed(2)} GB`;
};

// localStorage rather than a user setting: it is a per-device preference about a per-device
// cache, and a server round trip to decide whether to show a dialog is worse than losing it.
export function offlineWarningSuppressed(): boolean {
  try {
    const until = Number(localStorage.getItem(SUPPRESS_KEY));
    return Number.isFinite(until) && until > Date.now();
  } catch {
    return false;
  }
}

function suppressOfflineWarning() {
  try {
    localStorage.setItem(
      SUPPRESS_KEY,
      String(Date.now() + 24 * 60 * 60 * 1000),
    );
  } catch {
    return;
  }
}

type Props = {
  open: boolean;
  title: string;
  sizeBytes?: number;
  onCancel: () => void;
  onConfirm: () => void;
};

export function OfflineWarningModal({
  open,
  title,
  sizeBytes,
  onCancel,
  onConfirm,
}: Props) {
  const { t } = useTranslation();
  const [dontRemind, setDontRemind] = useState(false);

  if (!open) return null;

  const confirm = () => {
    if (dontRemind) suppressOfflineWarning();
    onConfirm();
  };

  const size = formatBytes(sizeBytes || 0);

  return createPortal(
    <dialog className="modal modal-open z-50">
      <div className="modal-box max-w-md max-h-[80dvh] sm:max-h-[85vh] p-0 overflow-hidden flex flex-col">
        <div className="px-5 py-3.5 border-b border-base-200 flex items-center justify-between gap-3 shrink-0 bg-base-100">
          <h3 className="text-lg font-bold flex items-center gap-2 leading-tight">
            <AlertTriangle className="w-5 h-5 text-warning shrink-0" />
            <span>{t("offline.warning_title")}</span>
          </h3>
          <button
            type="button"
            onClick={onCancel}
            className="btn btn-ghost btn-circle btn-sm -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
            aria-label={t("common.close", "Close")}
          >
            <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
          </button>
        </div>

        <div className="p-5 overflow-y-auto flex-1 min-h-0">
          <p className="text-sm">
            {size
              ? t("offline.warning_body_with_size", { title, size })
              : t("offline.warning_body", { title })}
          </p>
          <p className="mt-2 text-xs opacity-60">
            {t("offline.warning_storage_note")}
          </p>

          <label className="mt-4 flex items-center gap-2 cursor-pointer select-none">
            <input
              type="checkbox"
              className="checkbox checkbox-sm"
              checked={dontRemind}
              onChange={(event) => setDontRemind(event.target.checked)}
            />
            <span className="text-sm">{t("offline.warning_dont_remind")}</span>
          </label>

          <div className="modal-action border-t border-base-200 pt-4 mt-6">
            <button className="btn btn-ghost btn-sm" onClick={onCancel}>
              {t("common.cancel", "Cancel")}
            </button>
            <button className="btn btn-primary btn-sm" onClick={confirm}>
              {t("offline.warning_confirm")}
            </button>
          </div>
        </div>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={onCancel}>close</button>
      </form>
    </dialog>,
    document.body,
  );
}
