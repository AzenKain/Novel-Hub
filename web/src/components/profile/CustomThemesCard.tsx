import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Palette,
  Sparkles,
  Trash2,
  Eye,
  Plus,
} from "lucide-react";
import { toast } from "react-toastify";
import { useCustomization } from "@/hooks/useCustomization";
import { hasPermission } from "@/utils/permission";
import { useAuthStore } from "@/stores";

export const CustomThemesCard: React.FC = () => {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const {
    customThemes,
    isThemesLoading,
    createCustomTheme,
    deleteCustomTheme,
  } = useCustomization();

  const [isCreating, setIsCreating] = useState(false);
  const [name, setName] = useState("");
  const [bgColor, setBgColor] = useState("#1e1e2e");
  const [textColor, setTextColor] = useState("#cdd6f4");
  const [accentColor, setAccentColor] = useState("#89b4fa");
  const [customCss, setCustomCss] = useState("");

  const canManage = hasPermission(user, "user.theme.manage") || hasPermission(user, "admin.theme.manage");

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      toast.error(t("theme.name_required", "Theme name is required"));
      return;
    }

    try {
      await createCustomTheme({
        name: name.trim(),
        bg_color: bgColor,
        text_color: textColor,
        accent_color: accentColor,
        custom_css: customCss.trim(),
      });

      toast.success(t("theme.create_success", "Custom theme created successfully"));
      setName("");
      setCustomCss("");
      setIsCreating(false);
    } catch (err: any) {
      toast.error(err?.response?.data?.message || t("theme.create_failed", "Failed to create theme"));
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm(t("theme.delete_confirm", "Delete this custom theme?"))) return;
    try {
      await deleteCustomTheme(id);
      toast.success(t("theme.delete_success", "Custom theme deleted"));
    } catch (err: any) {
      toast.error(err?.response?.data?.message || t("theme.delete_failed", "Failed to delete theme"));
    }
  };

  return (
    <div className="card bg-base-100 shadow-xl border border-base-content/10">
      <div className="card-body p-5 sm:p-6">
        <div className="flex items-center justify-between border-b border-base-content/10 pb-4">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-accent/10 text-accent">
              <Palette className="w-5 h-5" />
            </div>
            <div>
              <h2 className="card-title text-base sm:text-lg">
                {t("theme.personal_themes", "Reader Themes & Custom CSS")}
              </h2>
              <p className="text-xs opacity-60">
                {t("theme.personal_desc", "Design bespoke color palettes and fine-tuned CSS rules for your reading experience.")}
              </p>
            </div>
          </div>
          {canManage && !isCreating && (
            <button
              type="button"
              onClick={() => setIsCreating(true)}
              className="btn btn-primary btn-sm rounded-xl gap-1.5"
            >
              <Plus className="w-4 h-4" />
              {t("theme.new_theme", "New Theme")}
            </button>
          )}
        </div>

        {/* Creation Form */}
        {isCreating && (
          <form onSubmit={handleCreate} className="mt-4 p-4 rounded-2xl bg-base-200/50 border border-base-content/5 space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold uppercase tracking-wider opacity-70">
                {t("theme.create_title", "Create Reader Theme")}
              </span>
              <button
                type="button"
                onClick={() => setIsCreating(false)}
                className="btn btn-ghost btn-xs"
              >
                {t("common.cancel", "Cancel")}
              </button>
            </div>

            <div>
              <label className="label label-text text-xs p-1">{t("theme.name", "Theme Name")}</label>
              <input
                type="text"
                placeholder="e.g. Midnight Cyberpunk, Forest Emerald"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="input input-bordered input-sm w-full"
                required
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <div>
                <label className="label label-text text-xs p-1">{t("theme.bg_color", "Background Color")}</label>
                <div className="flex items-center gap-2">
                  <input
                    type="color"
                    value={bgColor}
                    onChange={(e) => setBgColor(e.target.value)}
                    className="w-8 h-8 rounded border border-base-content/20 cursor-pointer p-0"
                  />
                  <input
                    type="text"
                    value={bgColor}
                    onChange={(e) => setBgColor(e.target.value)}
                    className="input input-bordered input-sm font-mono flex-1 uppercase"
                  />
                </div>
              </div>

              <div>
                <label className="label label-text text-xs p-1">{t("theme.text_color", "Text Color")}</label>
                <div className="flex items-center gap-2">
                  <input
                    type="color"
                    value={textColor}
                    onChange={(e) => setTextColor(e.target.value)}
                    className="w-8 h-8 rounded border border-base-content/20 cursor-pointer p-0"
                  />
                  <input
                    type="text"
                    value={textColor}
                    onChange={(e) => setTextColor(e.target.value)}
                    className="input input-bordered input-sm font-mono flex-1 uppercase"
                  />
                </div>
              </div>

              <div>
                <label className="label label-text text-xs p-1">{t("theme.accent_color", "Accent / Highlight Color")}</label>
                <div className="flex items-center gap-2">
                  <input
                    type="color"
                    value={accentColor}
                    onChange={(e) => setAccentColor(e.target.value)}
                    className="w-8 h-8 rounded border border-base-content/20 cursor-pointer p-0"
                  />
                  <input
                    type="text"
                    value={accentColor}
                    onChange={(e) => setAccentColor(e.target.value)}
                    className="input input-bordered input-sm font-mono flex-1 uppercase"
                  />
                </div>
              </div>
            </div>

            {/* Live Theme Preview */}
            <div
              className="p-4 rounded-xl border transition-all"
              style={{
                backgroundColor: bgColor,
                color: textColor,
                borderColor: accentColor,
              }}
            >
              <div className="flex items-center justify-between text-xs mb-2 opacity-80">
                <span className="font-bold flex items-center gap-1.5">
                  <Eye className="w-3.5 h-3.5" />
                  {t("theme.preview", "Live Preview")}: {name || t("theme.untitled_theme")}
                </span>
                <span
                  className="px-2 py-0.5 rounded text-[10px] font-bold"
                  style={{ backgroundColor: accentColor, color: bgColor }}
                >
                  {t("theme.preview_accent_badge")}
                </span>
              </div>
              <h4 className="text-base font-bold mb-1">{t("theme.preview_title")}</h4>
              <p className="text-xs opacity-90 leading-relaxed">
                {t("theme.preview_text")}
              </p>
            </div>

            <div>
              <label className="label label-text text-xs p-1">
                {t("theme.custom_css", "Extra Custom CSS (Optional)")}
              </label>
              <textarea
                placeholder=".reader-content p { text-indent: 1.5em; }"
                value={customCss}
                onChange={(e) => setCustomCss(e.target.value)}
                rows={2}
                className="textarea textarea-bordered textarea-sm w-full font-mono text-xs"
              />
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setIsCreating(false)}
                className="btn btn-ghost btn-sm"
              >
                {t("common.cancel", "Cancel")}
              </button>
              <button type="submit" className="btn btn-primary btn-sm rounded-xl gap-1.5">
                <Sparkles className="w-4 h-4" />
                {t("theme.save_theme", "Save Theme")}
              </button>
            </div>
          </form>
        )}

        {/* Existing Custom Themes List */}
        <div className="mt-4 space-y-2">
          <span className="text-xs font-bold uppercase tracking-wider opacity-70 block mb-2">
            {t("theme.saved_themes_list", "Saved Custom Themes")}
          </span>

          {isThemesLoading ? (
            <div className="flex justify-center p-6">
              <span className="loading loading-spinner text-primary" />
            </div>
          ) : customThemes.length === 0 ? (
            <div className="text-center p-6 text-xs opacity-50 bg-base-200/40 rounded-xl">
              {t("theme.empty", "No custom themes created yet.")}
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {customThemes.map((th) => {
                const isOwner = user && th.user_id === user.id;

                return (
                  <div
                    key={th.id}
                    className="p-4 rounded-xl border shadow-2xs transition-all relative group"
                    style={{
                      backgroundColor: th.bg_color,
                      color: th.text_color,
                      borderColor: `${th.accent_color}40`,
                    }}
                  >
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <span className="font-bold text-sm">{th.name}</span>
                        {th.is_system && (
                          <span className="badge badge-info badge-xs text-[9px]">{t("common.system", "System")}</span>
                        )}
                      </div>

                      {(isOwner || hasPermission(user, "admin.theme.manage")) && (
                        <button
                          type="button"
                          onClick={() => handleDelete(th.id)}
                          className="btn btn-ghost btn-circle btn-xs text-error opacity-60 hover:opacity-100"
                          title={t("common.delete", "Delete")}
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>

                    <div className="flex items-center gap-2 text-[11px] opacity-75 font-mono mb-2">
                      <span className="flex items-center gap-1">
                        <span className="w-2.5 h-2.5 rounded-full inline-block border" style={{ backgroundColor: th.bg_color }} />
                        {th.bg_color}
                      </span>
                      <span className="flex items-center gap-1">
                        <span className="w-2.5 h-2.5 rounded-full inline-block border" style={{ backgroundColor: th.text_color }} />
                        {th.text_color}
                      </span>
                      <span className="flex items-center gap-1">
                        <span className="w-2.5 h-2.5 rounded-full inline-block border" style={{ backgroundColor: th.accent_color }} />
                        {th.accent_color}
                      </span>
                    </div>

                    <p className="text-xs opacity-85 line-clamp-2 leading-relaxed">
                      {t("theme.preview_sample")}
                    </p>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
