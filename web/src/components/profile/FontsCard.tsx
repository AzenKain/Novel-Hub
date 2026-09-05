import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Type,
  Upload,
  Link as LinkIcon,
  Trash2,
  Sparkles,
  Eye,
} from "lucide-react";
import { toast } from "react-toastify";
import { useCustomization } from "@/hooks/useCustomization";
import { hasPermission } from "@/utils/permission";
import { useAuthStore } from "@/stores";

const cleanFontNameFromFile = (filename: string) => {
  const base = filename.replace(/\.(woff2|woff|ttf|otf)$/i, "");
  return base
    .replace(/[_-]+/g, " ")
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .trim();
};

export const FontsCard: React.FC = () => {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const {
    customFonts,
    isFontsLoading,
    uploadCustomFont,
    isUploadingFont,
    deleteCustomFont,
  } = useCustomization();

  const [sourceType, setSourceType] = useState<"file" | "url">("file");
  const [name, setName] = useState("");
  const [fontFamily, setFontFamily] = useState("");
  const [fontUrl, setFontUrl] = useState("");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  const canManage =
    hasPermission(user, "user.font.manage") ||
    hasPermission(user, "admin.font.manage");

  const handleFileSelect = (file: File | null) => {
    setSelectedFile(file);
    if (file) {
      const autoName = cleanFontNameFromFile(file.name);
      if (!name) setName(autoName);
      if (!fontFamily || fontFamily === name) setFontFamily(autoName);
    }
  };

  const handleNameChange = (val: string) => {
    setName(val);
    if (!fontFamily || fontFamily === name) {
      setFontFamily(val);
    }
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    const effectiveName = name.trim();
    const effectiveFamily = fontFamily.trim() || effectiveName;
    if (!effectiveName) {
      toast.error(t("font.fields_required", "Font name is required"));
      return;
    }

    if (sourceType === "file" && !selectedFile) {
      toast.error(t("font.file_required", "Please select a font file"));
      return;
    }

    if (sourceType === "url" && !fontUrl.trim()) {
      toast.error(t("font.url_required", "Please enter a font stylesheet URL"));
      return;
    }

    try {
      await uploadCustomFont({
        name: effectiveName,
        font_family: effectiveFamily,
        source_type: sourceType,
        font_url: sourceType === "url" ? fontUrl.trim() : undefined,
        file: sourceType === "file" && selectedFile ? selectedFile : undefined,
      });

      toast.success(t("font.upload_success", "Custom font added successfully"));
      setName("");
      setFontFamily("");
      setFontUrl("");
      setSelectedFile(null);
    } catch (err: any) {
      toast.error(
        err?.response?.data?.message ||
          t("font.upload_failed", "Failed to add font"),
      );
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm(t("font.delete_confirm", "Delete this custom font?")))
      return;
    try {
      await deleteCustomFont(id);
      toast.success(t("font.delete_success", "Font deleted"));
    } catch (err: any) {
      toast.error(
        err?.response?.data?.message ||
          t("font.delete_failed", "Failed to delete font"),
      );
    }
  };

  return (
    <div className="card bg-base-100 shadow-xl border border-base-content/10">
      <div className="card-body p-5 sm:p-6">
        <div className="flex items-center justify-between border-b border-base-content/10 pb-4">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-secondary/10 text-secondary">
              <Type className="w-5 h-5" />
            </div>
            <div>
              <h2 className="card-title text-base sm:text-lg">
                {t("font.personal_fonts", "Personal Reader Fonts")}
              </h2>
              <p className="text-xs opacity-60">
                {t(
                  "font.personal_desc",
                  "Upload custom typography files (WOFF2, TTF, OTF) or import Google Fonts.",
                )}
              </p>
            </div>
          </div>
          <span className="badge badge-secondary badge-outline text-xs">
            {customFonts.length} {t("font.fonts", "Fonts")}
          </span>
        </div>

        {canManage ? (
          <form
            onSubmit={handleUpload}
            className="mt-4 p-4 rounded-2xl bg-base-200/50 border border-base-content/5 space-y-3"
          >
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold uppercase tracking-wider opacity-70">
                {t("font.add_font", "Add Custom Font")}
              </span>
              <div className="join">
                <button
                  type="button"
                  onClick={() => setSourceType("file")}
                  className={`btn btn-xs join-item ${sourceType === "file" ? "btn-secondary" : "btn-ghost"}`}
                >
                  <Upload className="w-3 h-3 mr-1" />
                  {t("common.file", "File")}
                </button>
                <button
                  type="button"
                  onClick={() => setSourceType("url")}
                  className={`btn btn-xs join-item ${sourceType === "url" ? "btn-secondary" : "btn-ghost"}`}
                >
                  <LinkIcon className="w-3 h-3 mr-1" />
                  {t("font.google_url", "Google Font / Web URL")}
                </button>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="label label-text text-xs p-1">
                  {t("font.display_name", "Display Name")}
                </label>
                <input
                  type="text"
                  placeholder="e.g. Bookerly, Literata"
                  value={name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  className="input input-bordered input-sm w-full"
                  required
                />
              </div>

              <div>
                <label className="label label-text text-xs p-1 flex justify-between">
                  <span>{t("font.css_family", "Font Family (CSS)")}</span>
                  <span className="text-[10px] text-base-content/50">
                    {t("common.optional_auto", "Auto-detected")}
                  </span>
                </label>
                <input
                  type="text"
                  placeholder={name ? name : "e.g. 'Bookerly', serif"}
                  value={fontFamily}
                  onChange={(e) => setFontFamily(e.target.value)}
                  className="input input-bordered input-sm w-full"
                />
              </div>
            </div>

            {sourceType === "file" ? (
              <div>
                <label className="label label-text text-xs p-1">
                  {t("font.font_file", "Font File (.woff2, .woff, .ttf, .otf)")}
                </label>
                <input
                  type="file"
                  accept=".woff2,.woff,.ttf,.otf"
                  onChange={(e) =>
                    handleFileSelect(e.target.files?.[0] || null)
                  }
                  className="file-input file-input-bordered file-input-sm w-full"
                  required
                />
              </div>
            ) : (
              <div>
                <label className="label label-text text-xs p-1">
                  {t(
                    "font.stylesheet_url",
                    "Google Fonts / CSS Stylesheet URL",
                  )}
                </label>
                <input
                  type="url"
                  placeholder="https://fonts.googleapis.com/css2?family=Playfair+Display&display=swap"
                  value={fontUrl}
                  onChange={(e) => setFontUrl(e.target.value)}
                  className="input input-bordered input-sm w-full"
                  required
                />
              </div>
            )}

            <div className="flex justify-end pt-2">
              <button
                type="submit"
                disabled={isUploadingFont}
                className="btn btn-secondary btn-sm rounded-xl gap-2"
              >
                {isUploadingFont ? (
                  <span className="loading loading-spinner loading-xs" />
                ) : (
                  <Sparkles className="w-4 h-4" />
                )}
                {t("font.add_btn", "Add Font")}
              </button>
            </div>
          </form>
        ) : (
          <div className="alert alert-warning text-xs mt-3">
            {t(
              "font.no_permission",
              "You do not have permission to upload personal fonts.",
            )}
          </div>
        )}

        {/* Custom Fonts List */}
        <div className="mt-4 space-y-2">
          <span className="text-xs font-bold uppercase tracking-wider opacity-70 block mb-2">
            {t("font.installed_fonts", "Installed Custom Fonts")}
          </span>

          {isFontsLoading ? (
            <div className="flex justify-center p-6">
              <span className="loading loading-spinner text-secondary" />
            </div>
          ) : customFonts.length === 0 ? (
            <div className="text-center p-6 text-xs opacity-50 bg-base-200/40 rounded-xl">
              {t("font.empty", "No custom fonts added yet.")}
            </div>
          ) : (
            <div className="space-y-3">
              {customFonts.map((f) => {
                const isOwner = user && f.user_id === user.id;

                return (
                  <div
                    key={f.id}
                    className="p-3.5 rounded-xl border border-base-content/10 bg-base-200/30 hover:bg-base-200/60 transition-colors"
                  >
                    <div className="flex items-center justify-between gap-2 mb-2">
                      <div className="flex items-center gap-2">
                        <span className="font-bold text-sm">{f.name}</span>
                        <code className="text-[11px] px-1.5 py-0.5 rounded bg-base-300 opacity-80">
                          {f.font_family}
                        </code>
                        {f.is_system && (
                          <span className="badge badge-info badge-xs text-[9px]">
                            {t("common.system", "System")}
                          </span>
                        )}
                        <span className="badge badge-ghost badge-xs text-[9px] uppercase">
                          {f.source_type}
                        </span>
                      </div>

                      {(isOwner ||
                        hasPermission(user, "admin.font.manage")) && (
                        <button
                          type="button"
                          onClick={() => handleDelete(f.id)}
                          className="btn btn-ghost btn-circle btn-xs text-error opacity-60 hover:opacity-100"
                          title={t("common.delete", "Delete")}
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>

                    {/* Live Preview Line */}
                    <div
                      className="p-2.5 rounded-lg bg-base-100 border border-base-content/5 text-sm"
                      style={{ fontFamily: `'${f.font_family}', sans-serif` }}
                    >
                      <div className="flex items-center gap-1.5 text-[11px] opacity-50 mb-1">
                        <Eye className="w-3 h-3" />
                        <span>
                          {t("font.preview", "Sample Typography Preview")}:
                        </span>
                      </div>
                      <p className="text-base line-clamp-1">
                        The quick brown fox jumps over the lazy dog. 0123456789
                      </p>
                      <p className="text-sm opacity-80 line-clamp-1 mt-0.5">
                        Trăm năm trong cõi người ta, chữ tài chữ mệnh khéo là
                        ghét nhau.
                      </p>
                    </div>
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
