import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Sliders,
  Type,
  Palette,
  Upload,
  Link as LinkIcon,
  Trash2,
  Play,
  Pause,
  Sparkles,
  Plus,
  Eye,
  Shield,
  CloudRain,
  Wind,
  Coffee,
  Flame,
  Waves,
  Music,
  Headphones,
  Zap,
  Disc,
  Radio,
  Gamepad2,
  Volume2,
} from "lucide-react";
import { toast } from "react-toastify";
import { useCustomization } from "@/hooks/useCustomization";

export const CustomizationTab: React.FC = () => {
  const { t } = useTranslation();
  const {
    soundscapes,
    isSoundscapesLoading,
    uploadSoundscape,
    isUploadingSoundscape,
    deleteSoundscape,

    customFonts,
    isFontsLoading,
    uploadCustomFont,
    isUploadingFont,
    deleteCustomFont,

    customThemes,
    isThemesLoading,
    createCustomTheme,
    deleteCustomTheme,
  } = useCustomization();

  // Soundscape upload state
  const [soundMode, setSoundMode] = useState<"file" | "url">("file");
  const [soundName, setSoundName] = useState("");
  const [soundCat, setSoundCat] = useState("ambient");
  const [soundUrl, setSoundUrl] = useState("");
  const [soundFile, setSoundFile] = useState<File | null>(null);
  const [playingId, setPlayingId] = useState<string | null>(null);
  const [audioPlayer, setAudioPlayer] = useState<HTMLAudioElement | null>(null);

  // Font upload state
  const [fontMode, setFontMode] = useState<"file" | "url">("file");
  const [fontName, setFontName] = useState("");
  const [fontFamily, setFontFamily] = useState("");
  const [fontUrl, setFontUrl] = useState("");
  const [fontFile, setFontFile] = useState<File | null>(null);

  // Theme creation state
  const [isCreatingTheme, setIsCreatingTheme] = useState(false);
  const [themeName, setThemeName] = useState("");
  const [themeBg, setThemeBg] = useState("#1e1e2e");
  const [themeText, setThemeText] = useState("#cdd6f4");
  const [themeAccent, setThemeAccent] = useState("#89b4fa");
  const [themeCss, setThemeCss] = useState("");

  const handleTogglePlay = (streamUrl: string, id: string) => {
    if (playingId === id) {
      audioPlayer?.pause();
      setPlayingId(null);
      return;
    }
    if (audioPlayer) {
      audioPlayer.pause();
    }
    const audio = new Audio(streamUrl);
    audio.play().catch(() => toast.error(t("soundscape.play_failed", "Failed to play audio")));
    audio.onended = () => setPlayingId(null);
    setAudioPlayer(audio);
    setPlayingId(id);
  };

  const handleUploadSoundscape = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!soundName.trim()) {
      toast.error(t("soundscape.name_required", "Name required"));
      return;
    }
    try {
      await uploadSoundscape({
        name: soundName.trim(),
        category: soundCat,
        icon: soundCat,
        audio_url: soundMode === "url" ? soundUrl.trim() : undefined,
        file: soundMode === "file" && soundFile ? soundFile : undefined,
        is_system: true,
      });
      toast.success(t("admin.soundscape_created", "System default soundscape added"));
      setSoundName("");
      setSoundUrl("");
      setSoundFile(null);
    } catch (err: any) {
      toast.error(err?.response?.data?.message || "Failed to add system soundscape");
    }
  };

  const cleanFontNameFromFile = (filename: string) => {
    const base = filename.replace(/\.(woff2|woff|ttf|otf)$/i, "");
    return base
      .replace(/[_-]+/g, " ")
      .replace(/([a-z])([A-Z])/g, "$1 $2")
      .trim();
  };

  const handleFontFileSelect = (file: File | null) => {
    setFontFile(file);
    if (file) {
      const autoName = cleanFontNameFromFile(file.name);
      if (!fontName) setFontName(autoName);
      if (!fontFamily || fontFamily === fontName) setFontFamily(autoName);
    }
  };

  const handleFontNameChange = (val: string) => {
    setFontName(val);
    if (!fontFamily || fontFamily === fontName) {
      setFontFamily(val);
    }
  };

  const handleUploadFont = async (e: React.FormEvent) => {
    e.preventDefault();
    const effectiveName = fontName.trim();
    const effectiveFamily = fontFamily.trim() || effectiveName;
    if (!effectiveName) {
      toast.error(t("font.fields_required", "Font name is required"));
      return;
    }
    try {
      await uploadCustomFont({
        name: effectiveName,
        font_family: effectiveFamily,
        source_type: fontMode,
        font_url: fontMode === "url" ? fontUrl.trim() : undefined,
        file: fontMode === "file" && fontFile ? fontFile : undefined,
        is_system: true,
      });
      toast.success(t("admin.font_created", "System default font added"));
      setFontName("");
      setFontFamily("");
      setFontUrl("");
      setFontFile(null);
    } catch (err: any) {
      toast.error(err?.response?.data?.message || "Failed to add system font");
    }
  };

  const handleCreateTheme = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!themeName.trim()) {
      toast.error(t("theme.name_required", "Theme name required"));
      return;
    }
    try {
      await createCustomTheme({
        name: themeName.trim(),
        bg_color: themeBg,
        text_color: themeText,
        accent_color: themeAccent,
        custom_css: themeCss.trim(),
        is_system: true,
      });
      toast.success(t("admin.theme_created", "System default theme added"));
      setThemeName("");
      setThemeCss("");
      setIsCreatingTheme(false);
    } catch (err: any) {
      toast.error(err?.response?.data?.message || "Failed to add system theme");
    }
  };

  const getIconEl = (category: string) => {
    switch (category.toLowerCase()) {
      case "rain":
        return <CloudRain className="w-4 h-4" />;
      case "wind":
        return <Wind className="w-4 h-4" />;
      case "coffee":
        return <Coffee className="w-4 h-4" />;
      case "fire":
        return <Flame className="w-4 h-4" />;
      case "waves":
        return <Waves className="w-4 h-4" />;
      case "lofi":
        return <Headphones className="w-4 h-4" />;
      case "edm":
        return <Zap className="w-4 h-4" />;
      case "remix":
      case "tiktok":
        return <Disc className="w-4 h-4" />;
      case "pop":
        return <Radio className="w-4 h-4" />;
      case "game":
      case "anime":
        return <Gamepad2 className="w-4 h-4" />;
      case "white_noise":
      case "asmr":
        return <Volume2 className="w-4 h-4" />;
      default:
        return <Music className="w-4 h-4" />;
    }
  };

  return (
    <div className="space-y-8">
      {/* 1. System Default Ambient Soundscapes */}
      <div className="card bg-base-100 shadow-xl border border-base-content/10">
        <div className="card-body p-5 sm:p-6">
          <div className="flex items-center justify-between border-b border-base-content/10 pb-4">
            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-xl bg-primary/10 text-primary">
                <Sliders className="w-5 h-5" />
              </div>
              <div>
                <h2 className="card-title text-base sm:text-lg flex items-center gap-2">
                  {t("admin.system_soundscapes", "System Default Soundscapes")}
                  <span className="badge badge-primary badge-xs">SYSTEM</span>
                </h2>
                <p className="text-xs opacity-60">
                  {t("admin.system_soundscapes_desc", "Soundscapes managed here are instantly available to all users across the platform.")}
                </p>
              </div>
            </div>
            <span className="badge badge-primary badge-outline text-xs">
              {soundscapes.filter((s) => s.is_system).length} {t("soundscape.tracks", "Tracks")}
            </span>
          </div>

          {/* Admin Upload Form */}
          <form onSubmit={handleUploadSoundscape} className="mt-4 p-4 rounded-2xl bg-base-200/50 border border-base-content/5 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold uppercase tracking-wider opacity-70">
                {t("admin.add_system_soundscape", "Add System Soundscape")}
              </span>
              <div className="join">
                <button
                  type="button"
                  onClick={() => setSoundMode("file")}
                  className={`btn btn-xs join-item ${soundMode === "file" ? "btn-primary" : "btn-ghost"}`}
                >
                  <Upload className="w-3 h-3 mr-1" />
                  {t("common.file", "File")}
                </button>
                <button
                  type="button"
                  onClick={() => setSoundMode("url")}
                  className={`btn btn-xs join-item ${soundMode === "url" ? "btn-primary" : "btn-ghost"}`}
                >
                  <LinkIcon className="w-3 h-3 mr-1" />
                  {t("common.url", "URL")}
                </button>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="label label-text text-xs p-1">{t("soundscape.name", "Name")}</label>
                <input
                  type="text"
                  placeholder="e.g. Cozy Rain, Coffee Shop Ambient"
                  value={soundName}
                  onChange={(e) => setSoundName(e.target.value)}
                  className="input input-bordered input-sm w-full"
                  required
                />
              </div>

              <div>
                <label className="label label-text text-xs p-1">{t("soundscape.category", "Category")}</label>
                <select
                  value={soundCat}
                  onChange={(e) => setSoundCat(e.target.value)}
                  className="select select-bordered select-sm w-full"
                >
                  <option value="rain">{t("soundscape.cat_rain", "Rain & Storm")}</option>
                  <option value="coffee">{t("soundscape.cat_coffee", "Cafe & Library")}</option>
                  <option value="fire">{t("soundscape.cat_fire", "Fireplace & Camp")}</option>
                  <option value="waves">{t("soundscape.cat_waves", "Ocean & River")}</option>
                  <option value="wind">{t("soundscape.cat_wind", "Wind & Nature")}</option>
                  <option value="ambient">{t("soundscape.cat_ambient", "Ambient Music")}</option>
                  <option value="lofi">{t("soundscape.cat_lofi", "Lo-Fi & Chill Beats")}</option>
                  <option value="edm">{t("soundscape.cat_edm", "EDM & Electronic")}</option>
                  <option value="remix">{t("soundscape.cat_remix", "Remix & TikTok Trending")}</option>
                  <option value="pop">{t("soundscape.cat_pop", "Pop & Acoustic")}</option>
                  <option value="game">{t("soundscape.cat_game", "Game & Anime BGM")}</option>
                  <option value="white_noise">{t("soundscape.cat_white_noise", "White Noise & ASMR")}</option>
                  <option value="other">{t("soundscape.cat_other", "Other / Custom Music")}</option>
                </select>
              </div>
            </div>

            {soundMode === "file" ? (
              <div>
                <label className="label label-text text-xs p-1">
                  {t("soundscape.audio_file", "Audio File (MP3, OGG, WAV, M4A, FLAC)")}
                </label>
                <input
                  type="file"
                  accept="audio/mp3,audio/mpeg,audio/ogg,audio/wav,audio/x-m4a,audio/flac"
                  onChange={(e) => setSoundFile(e.target.files?.[0] || null)}
                  className="file-input file-input-bordered file-input-sm w-full"
                  required
                />
              </div>
            ) : (
              <div>
                <label className="label label-text text-xs p-1">{t("soundscape.audio_url", "Direct Audio URL")}</label>
                <input
                  type="url"
                  placeholder="https://example.com/audio.mp3"
                  value={soundUrl}
                  onChange={(e) => setSoundUrl(e.target.value)}
                  className="input input-bordered input-sm w-full"
                  required
                />
              </div>
            )}

            <div className="flex justify-end pt-2">
              <button
                type="submit"
                disabled={isUploadingSoundscape}
                className="btn btn-primary btn-sm rounded-xl gap-2"
              >
                {isUploadingSoundscape ? (
                  <span className="loading loading-spinner loading-xs" />
                ) : (
                  <Sparkles className="w-4 h-4" />
                )}
                {t("admin.publish_system_soundscape", "Publish System Soundscape")}
              </button>
            </div>
          </form>

          {/* System Soundscapes Grid */}
          <div className="mt-4 grid grid-cols-1 sm:grid-cols-2 gap-2.5">
            {soundscapes
              .filter((s) => s.is_system)
              .map((s) => {
                const isPlaying = playingId === s.id;
                return (
                  <div
                    key={s.id}
                    className="flex items-center justify-between p-3 rounded-xl border border-base-content/10 bg-base-200/30"
                  >
                    <div className="flex items-center gap-2.5 min-w-0 flex-1">
                      <button
                        type="button"
                        onClick={() => handleTogglePlay(s.stream_url, s.id)}
                        className={`btn btn-circle btn-sm ${isPlaying ? "btn-primary" : "btn-outline"}`}
                      >
                        {isPlaying ? <Pause className="w-3.5 h-3.5" /> : <Play className="w-3.5 h-3.5 fill-current ml-0.5" />}
                      </button>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5">
                          <span className="opacity-70">{getIconEl(s.category)}</span>
                          <span className="font-semibold text-xs truncate">{s.name}</span>
                        </div>
                        <span className="badge badge-ghost badge-xs text-[9px] uppercase">{s.category}</span>
                      </div>
                    </div>

                    <button
                      type="button"
                      onClick={() => deleteSoundscape(s.id)}
                      className="btn btn-ghost btn-circle btn-xs text-error opacity-60 hover:opacity-100"
                      title={t("common.delete", "Delete")}
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                );
              })}
          </div>
        </div>
      </div>

      {/* 2. System Default Reader Fonts */}
      <div className="card bg-base-100 shadow-xl border border-base-content/10">
        <div className="card-body p-5 sm:p-6">
          <div className="flex items-center justify-between border-b border-base-content/10 pb-4">
            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-xl bg-secondary/10 text-secondary">
                <Type className="w-5 h-5" />
              </div>
              <div>
                <h2 className="card-title text-base sm:text-lg flex items-center gap-2">
                  {t("admin.system_fonts", "System Default Reader Fonts")}
                  <span className="badge badge-secondary badge-xs">SYSTEM</span>
                </h2>
                <p className="text-xs opacity-60">
                  {t("admin.system_fonts_desc", "Fonts configured here appear in the font selector for all readers.")}
                </p>
              </div>
            </div>
            <span className="badge badge-secondary badge-outline text-xs">
              {customFonts.filter((f) => f.is_system).length} {t("font.fonts", "Fonts")}
            </span>
          </div>

          {/* Admin Font Upload Form */}
          <form onSubmit={handleUploadFont} className="mt-4 p-4 rounded-2xl bg-base-200/50 border border-base-content/5 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold uppercase tracking-wider opacity-70">
                {t("admin.add_system_font", "Add System Font")}
              </span>
              <div className="join">
                <button
                  type="button"
                  onClick={() => setFontMode("file")}
                  className={`btn btn-xs join-item ${fontMode === "file" ? "btn-secondary" : "btn-ghost"}`}
                >
                  <Upload className="w-3 h-3 mr-1" />
                  {t("common.file", "File")}
                </button>
                <button
                  type="button"
                  onClick={() => setFontMode("url")}
                  className={`btn btn-xs join-item ${fontMode === "url" ? "btn-secondary" : "btn-ghost"}`}
                >
                  <LinkIcon className="w-3 h-3 mr-1" />
                  {t("font.google_url", "Google Font / Web URL")}
                </button>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="label label-text text-xs p-1">{t("font.display_name", "Display Name")}</label>
                <input
                  type="text"
                  placeholder="e.g. Literata Book, Crimson Pro"
                  value={fontName}
                  onChange={(e) => handleFontNameChange(e.target.value)}
                  className="input input-bordered input-sm w-full"
                  required
                />
              </div>

              <div>
                <label className="label label-text text-xs p-1 flex justify-between">
                  <span>{t("font.css_family", "Font Family (CSS)")}</span>
                  <span className="text-[10px] opacity-50">{t("common.optional_auto", "Auto-detected")}</span>
                </label>
                <input
                  type="text"
                  placeholder={fontName ? fontName : "e.g. 'Literata', serif"}
                  value={fontFamily}
                  onChange={(e) => setFontFamily(e.target.value)}
                  className="input input-bordered input-sm w-full"
                />
              </div>
            </div>

            {fontMode === "file" ? (
              <div>
                <label className="label label-text text-xs p-1">
                  {t("font.font_file", "Font File (.woff2, .woff, .ttf, .otf)")}
                </label>
                <input
                  type="file"
                  accept=".woff2,.woff,.ttf,.otf"
                  onChange={(e) => handleFontFileSelect(e.target.files?.[0] || null)}
                  className="file-input file-input-bordered file-input-sm w-full"
                  required
                />
              </div>
            ) : (
              <div>
                <label className="label label-text text-xs p-1">
                  {t("font.stylesheet_url", "Google Fonts / CSS Stylesheet URL")}
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
                {t("admin.publish_system_font", "Publish System Font")}
              </button>
            </div>
          </form>

          {/* System Fonts List */}
          <div className="mt-4 space-y-3">
            {customFonts
              .filter((f) => f.is_system)
              .map((f) => (
                <div
                  key={f.id}
                  className="p-3.5 rounded-xl border border-base-content/10 bg-base-200/30"
                >
                  <div className="flex items-center justify-between gap-2 mb-2">
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-sm">{f.name}</span>
                      <code className="text-[11px] px-1.5 py-0.5 rounded bg-base-300 opacity-80">
                        {f.font_family}
                      </code>
                      <span className="badge badge-ghost badge-xs text-[9px] uppercase">{f.source_type}</span>
                    </div>

                    <button
                      type="button"
                      onClick={() => deleteCustomFont(f.id)}
                      className="btn btn-ghost btn-circle btn-xs text-error opacity-60 hover:opacity-100"
                      title={t("common.delete", "Delete")}
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>

                  <div
                    className="p-2.5 rounded-lg bg-base-100 border border-base-content/5 text-sm"
                    style={{ fontFamily: `'${f.font_family}', sans-serif` }}
                  >
                    <p className="text-sm line-clamp-1">
                      The quick brown fox jumps over the lazy dog. 0123456789 (Hệ thống chữ mặc định NovelHub)
                    </p>
                  </div>
                </div>
              ))}
          </div>
        </div>
      </div>

      {/* 3. System Default Themes */}
      <div className="card bg-base-100 shadow-xl border border-base-content/10">
        <div className="card-body p-5 sm:p-6">
          <div className="flex items-center justify-between border-b border-base-content/10 pb-4">
            <div className="flex items-center gap-3">
              <div className="p-2.5 rounded-xl bg-accent/10 text-accent">
                <Palette className="w-5 h-5" />
              </div>
              <div>
                <h2 className="card-title text-base sm:text-lg flex items-center gap-2">
                  {t("admin.system_themes", "System Default Themes")}
                  <span className="badge badge-accent badge-xs">SYSTEM</span>
                </h2>
                <p className="text-xs opacity-60">
                  {t("admin.system_themes_desc", "Pre-built color themes available to all readers.")}
                </p>
              </div>
            </div>
            {!isCreatingTheme && (
              <button
                type="button"
                onClick={() => setIsCreatingTheme(true)}
                className="btn btn-primary btn-sm rounded-xl gap-1.5"
              >
                <Plus className="w-4 h-4" />
                {t("theme.new_theme", "New Theme")}
              </button>
            )}
          </div>

          {isCreatingTheme && (
            <form onSubmit={handleCreateTheme} className="mt-4 p-4 rounded-2xl bg-base-200/50 border border-base-content/5 space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold uppercase tracking-wider opacity-70">
                  {t("admin.create_system_theme", "Create System Theme Preset")}
                </span>
                <button
                  type="button"
                  onClick={() => setIsCreatingTheme(false)}
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
                  value={themeName}
                  onChange={(e) => setThemeName(e.target.value)}
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
                      value={themeBg}
                      onChange={(e) => setThemeBg(e.target.value)}
                      className="w-8 h-8 rounded border border-base-content/20 cursor-pointer p-0"
                    />
                    <input
                      type="text"
                      value={themeBg}
                      onChange={(e) => setThemeBg(e.target.value)}
                      className="input input-bordered input-sm font-mono flex-1 uppercase"
                    />
                  </div>
                </div>

                <div>
                  <label className="label label-text text-xs p-1">{t("theme.text_color", "Text Color")}</label>
                  <div className="flex items-center gap-2">
                    <input
                      type="color"
                      value={themeText}
                      onChange={(e) => setThemeText(e.target.value)}
                      className="w-8 h-8 rounded border border-base-content/20 cursor-pointer p-0"
                    />
                    <input
                      type="text"
                      value={themeText}
                      onChange={(e) => setThemeText(e.target.value)}
                      className="input input-bordered input-sm font-mono flex-1 uppercase"
                    />
                  </div>
                </div>

                <div>
                  <label className="label label-text text-xs p-1">{t("theme.accent_color", "Accent Color")}</label>
                  <div className="flex items-center gap-2">
                    <input
                      type="color"
                      value={themeAccent}
                      onChange={(e) => setThemeAccent(e.target.value)}
                      className="w-8 h-8 rounded border border-base-content/20 cursor-pointer p-0"
                    />
                    <input
                      type="text"
                      value={themeAccent}
                      onChange={(e) => setThemeAccent(e.target.value)}
                      className="input input-bordered input-sm font-mono flex-1 uppercase"
                    />
                  </div>
                </div>
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setIsCreatingTheme(false)}
                  className="btn btn-ghost btn-sm"
                >
                  {t("common.cancel", "Cancel")}
                </button>
                <button type="submit" className="btn btn-primary btn-sm rounded-xl gap-1.5">
                  <Sparkles className="w-4 h-4" />
                  {t("admin.publish_system_theme", "Publish System Theme")}
                </button>
              </div>
            </form>
          )}

          <div className="mt-4 grid grid-cols-1 sm:grid-cols-2 gap-3">
            {customThemes
              .filter((th) => th.is_system)
              .map((th) => (
                <div
                  key={th.id}
                  className="p-4 rounded-xl border shadow-2xs transition-all relative"
                  style={{
                    backgroundColor: th.bg_color,
                    color: th.text_color,
                    borderColor: `${th.accent_color}40`,
                  }}
                >
                  <div className="flex items-center justify-between mb-2">
                    <span className="font-bold text-sm">{th.name}</span>
                    <button
                      type="button"
                      onClick={() => deleteCustomTheme(th.id)}
                      className="btn btn-ghost btn-circle btn-xs text-error opacity-60 hover:opacity-100"
                      title={t("common.delete", "Delete")}
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>

                  <div className="flex items-center gap-2 text-[11px] opacity-75 font-mono mb-2">
                    <span>{th.bg_color}</span> / <span>{th.text_color}</span> / <span>{th.accent_color}</span>
                  </div>
                </div>
              ))}
          </div>
        </div>
      </div>
    </div>
  );
};
