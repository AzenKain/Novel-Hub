import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Code, Check, RefreshCw } from "lucide-react";
import { useSettingsStore } from "@/stores";
import { toast } from "react-toastify";

export const CustomCSSCard: React.FC = () => {
  const { t } = useTranslation();
  const { customCss, setCustomCss } = useSettingsStore();
  const [cssText, setCssText] = useState(customCss);
  const [saved, setSaved] = useState(false);

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    setCustomCss(cssText);
    setSaved(true);
    toast.success(
      t("profile.custom_css_saved", "Custom CSS updated successfully!"),
    );
    setTimeout(() => setSaved(false), 2500);
  };

  const handleReset = () => {
    setCssText("");
    setCustomCss("");
    toast.info(t("profile.custom_css_reset", "Custom CSS cleared"));
  };

  return (
    <div className="card bg-base-100 shadow-sm border border-base-200">
      <div className="card-body p-5">
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-secondary/10 text-secondary">
              <Code className="w-5 h-5" />
            </div>
            <div>
              <h3 className="card-title text-base">
                {t("profile.custom_css_title", "Custom CSS Injector")}
              </h3>
              <p className="text-xs text-base-content/70">
                {t(
                  "profile.custom_css_desc",
                  "Inject custom CSS styles into NovelHub UI for personalized themes.",
                )}
              </p>
            </div>
          </div>
        </div>

        <form onSubmit={handleSave} className="mt-2 flex flex-col gap-3">
          <textarea
            value={cssText}
            onChange={(e) => setCssText(e.target.value)}
            placeholder={`/* Add your custom CSS rules here */\nbody {\n  font-family: 'Inter', sans-serif;\n}`}
            rows={4}
            className="textarea textarea-bordered font-mono text-xs w-full focus:outline-none"
          />

          <div className="flex items-center justify-between">
            <button
              type="button"
              onClick={handleReset}
              className="btn btn-ghost btn-xs text-base-content/60 gap-1"
            >
              <RefreshCw className="w-3 h-3" />
              {t("common.reset", "Reset")}
            </button>

            <button type="submit" className="btn btn-secondary btn-sm gap-1">
              <Check className="w-4 h-4" />
              {saved
                ? t("common.saved", "Saved!")
                : t("common.save_changes", "Save CSS")}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
