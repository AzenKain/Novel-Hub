import { RefreshCw } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { registerSW } from "virtual:pwa-register";

export function UpdatePrompt() {
  const { t } = useTranslation();
  const [needRefresh, setNeedRefresh] = useState(false);
  const [applyUpdate, setApplyUpdate] = useState<(() => void) | null>(null);

  useEffect(() => {
    const update = registerSW({
      onNeedRefresh: () => setNeedRefresh(true),
    });
    setApplyUpdate(() => () => void update(true));
  }, []);

  if (!needRefresh) return null;

  return (
    <div className="toast toast-end toast-bottom z-50">
      <div className="alert alert-info shadow-lg flex items-center gap-3">
        <span className="text-sm">{t("pwa.update_available")}</span>
        <button
          className="btn btn-sm btn-primary gap-1"
          onClick={() => applyUpdate?.()}
        >
          <RefreshCw size={14} />
          {t("pwa.reload")}
        </button>
        <button className="btn btn-sm btn-ghost" onClick={() => setNeedRefresh(false)}>
          {t("pwa.later")}
        </button>
      </div>
    </div>
  );
}
