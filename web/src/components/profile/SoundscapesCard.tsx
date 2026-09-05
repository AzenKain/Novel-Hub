import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Music,
  Upload,
  Link as LinkIcon,
  Trash2,
  Play,
  Pause,
  Sliders,
  Sparkles,
  CloudRain,
  Wind,
  Coffee,
  Flame,
  Waves,
  Headphones,
  Zap,
  Disc,
  Radio,
  Gamepad2,
  Volume2,
} from "lucide-react";
import { toast } from "react-toastify";
import { useCustomization } from "@/hooks/useCustomization";
import { hasPermission } from "@/utils/permission";
import { useAuthStore } from "@/stores";

export const SoundscapesCard: React.FC = () => {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const {
    soundscapes,
    isSoundscapesLoading,
    uploadSoundscape,
    isUploadingSoundscape,
    deleteSoundscape,
  } = useCustomization();

  const [mode, setMode] = useState<"file" | "url">("file");
  const [name, setName] = useState("");
  const [category, setCategory] = useState("ambient");
  const [icon, setIcon] = useState("Music");
  const [audioUrl, setAudioUrl] = useState("");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [playingId, setPlayingId] = useState<string | null>(null);
  const [audioPlayer, setAudioPlayer] = useState<HTMLAudioElement | null>(null);

  const canManage =
    hasPermission(user, "user.soundscape.manage") ||
    hasPermission(user, "admin.soundscape.manage");

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
    audio
      .play()
      .catch(() =>
        toast.error(t("soundscape.play_failed", "Failed to play audio")),
      );
    audio.onended = () => setPlayingId(null);
    setAudioPlayer(audio);
    setPlayingId(id);
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      toast.error(t("soundscape.name_required", "Soundscape name is required"));
      return;
    }

    if (mode === "file" && !selectedFile) {
      toast.error(t("soundscape.file_required", "Please select an audio file"));
      return;
    }

    if (mode === "url" && !audioUrl.trim()) {
      toast.error(t("soundscape.url_required", "Please enter an audio URL"));
      return;
    }

    try {
      await uploadSoundscape({
        name: name.trim(),
        category,
        icon,
        audio_url: mode === "url" ? audioUrl.trim() : undefined,
        file: mode === "file" && selectedFile ? selectedFile : undefined,
      });

      toast.success(
        t("soundscape.upload_success", "Soundscape uploaded successfully"),
      );
      setName("");
      setAudioUrl("");
      setSelectedFile(null);
    } catch (err: any) {
      toast.error(
        err?.response?.data?.message ||
          t("soundscape.upload_failed", "Failed to upload soundscape"),
      );
    }
  };

  const handleDelete = async (id: string) => {
    if (
      !window.confirm(t("soundscape.delete_confirm", "Delete this soundscape?"))
    )
      return;
    try {
      if (playingId === id) {
        audioPlayer?.pause();
        setPlayingId(null);
      }
      await deleteSoundscape(id);
      toast.success(t("soundscape.delete_success", "Soundscape deleted"));
    } catch (err: any) {
      toast.error(
        err?.response?.data?.message ||
          t("soundscape.delete_failed", "Failed to delete"),
      );
    }
  };

  const getIconEl = (iconName: string) => {
    switch (iconName.toLowerCase()) {
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
    <div className="card bg-base-100 shadow-xl border border-base-content/10">
      <div className="card-body p-5 sm:p-6">
        <div className="flex items-center justify-between border-b border-base-content/10 pb-4">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-primary/10 text-primary">
              <Sliders className="w-5 h-5" />
            </div>
            <div>
              <h2 className="card-title text-base sm:text-lg">
                {t(
                  "soundscape.personal_soundscapes",
                  "Personal Ambient Soundscapes",
                )}
              </h2>
              <p className="text-xs opacity-60">
                {t(
                  "soundscape.personal_desc",
                  "Upload background audio & sound effects to listen while reading novels.",
                )}
              </p>
            </div>
          </div>
          <span className="badge badge-primary badge-outline text-xs">
            {soundscapes.length} {t("soundscape.tracks", "Tracks")}
          </span>
        </div>

        {canManage ? (
          <form
            onSubmit={handleUpload}
            className="mt-4 p-4 rounded-2xl bg-base-200/50 border border-base-content/5 space-y-3"
          >
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold uppercase tracking-wider opacity-70">
                {t("soundscape.add_soundscape", "Add New Soundscape")}
              </span>
              <div className="join">
                <button
                  type="button"
                  onClick={() => setMode("file")}
                  className={`btn btn-xs join-item ${mode === "file" ? "btn-primary" : "btn-ghost"}`}
                >
                  <Upload className="w-3 h-3 mr-1" />
                  {t("common.file", "File")}
                </button>
                <button
                  type="button"
                  onClick={() => setMode("url")}
                  className={`btn btn-xs join-item ${mode === "url" ? "btn-primary" : "btn-ghost"}`}
                >
                  <LinkIcon className="w-3 h-3 mr-1" />
                  {t("common.url", "URL")}
                </button>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="label label-text text-xs p-1">
                  {t("soundscape.name", "Name")}
                </label>
                <input
                  type="text"
                  placeholder="e.g. Japanese Rain Garden"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="input input-bordered input-sm w-full"
                  required
                />
              </div>

              <div>
                <label className="label label-text text-xs p-1">
                  {t("soundscape.category", "Category / Tag")}
                </label>
                <select
                  value={category}
                  onChange={(e) => {
                    setCategory(e.target.value);
                    setIcon(e.target.value);
                  }}
                  className="select select-bordered select-sm w-full"
                >
                  <option value="rain">
                    {t("soundscape.cat_rain", "Rain & Storm")}
                  </option>
                  <option value="coffee">
                    {t("soundscape.cat_coffee", "Cafe & Library")}
                  </option>
                  <option value="fire">
                    {t("soundscape.cat_fire", "Fireplace & Camp")}
                  </option>
                  <option value="waves">
                    {t("soundscape.cat_waves", "Ocean & River")}
                  </option>
                  <option value="wind">
                    {t("soundscape.cat_wind", "Wind & Nature")}
                  </option>
                  <option value="ambient">
                    {t("soundscape.cat_ambient", "Ambient Music")}
                  </option>
                  <option value="lofi">
                    {t("soundscape.cat_lofi", "Lo-Fi & Chill Beats")}
                  </option>
                  <option value="edm">
                    {t("soundscape.cat_edm", "EDM & Electronic")}
                  </option>
                  <option value="remix">
                    {t("soundscape.cat_remix", "Remix & TikTok Trending")}
                  </option>
                  <option value="pop">
                    {t("soundscape.cat_pop", "Pop & Acoustic")}
                  </option>
                  <option value="game">
                    {t("soundscape.cat_game", "Game & Anime BGM")}
                  </option>
                  <option value="white_noise">
                    {t("soundscape.cat_white_noise", "White Noise & ASMR")}
                  </option>
                  <option value="other">
                    {t("soundscape.cat_other", "Other / Custom Music")}
                  </option>
                </select>
              </div>
            </div>

            {mode === "file" ? (
              <div>
                <label className="label label-text text-xs p-1">
                  {t(
                    "soundscape.audio_file",
                    "Audio File (MP3, OGG, WAV, M4A, FLAC)",
                  )}
                </label>
                <input
                  type="file"
                  accept="audio/mp3,audio/mpeg,audio/ogg,audio/wav,audio/x-m4a,audio/flac"
                  onChange={(e) => setSelectedFile(e.target.files?.[0] || null)}
                  className="file-input file-input-bordered file-input-sm w-full"
                  required
                />
              </div>
            ) : (
              <div>
                <label className="label label-text text-xs p-1">
                  {t("soundscape.audio_url", "Direct Audio URL")}
                </label>
                <input
                  type="url"
                  placeholder="https://example.com/ambient.mp3"
                  value={audioUrl}
                  onChange={(e) => setAudioUrl(e.target.value)}
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
                {t("soundscape.upload_btn", "Save Soundscape")}
              </button>
            </div>
          </form>
        ) : (
          <div className="alert alert-warning text-xs mt-3">
            {t(
              "soundscape.no_permission",
              "You do not have permission to upload personal soundscapes.",
            )}
          </div>
        )}

        {/* Soundscapes List */}
        <div className="mt-4 space-y-2">
          <span className="text-xs font-bold uppercase tracking-wider opacity-70 block mb-2">
            {t("soundscape.available_tracks", "Available Soundscapes")}
          </span>

          {isSoundscapesLoading ? (
            <div className="flex justify-center p-6">
              <span className="loading loading-spinner text-primary" />
            </div>
          ) : soundscapes.length === 0 ? (
            <div className="text-center p-6 text-xs opacity-50 bg-base-200/40 rounded-xl">
              {t("soundscape.empty", "No soundscapes uploaded yet.")}
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
              {soundscapes.map((s) => {
                const isPlaying = playingId === s.id;
                const isOwner = user && s.user_id === user.id;

                return (
                  <div
                    key={s.id}
                    className="flex items-center justify-between p-3 rounded-xl border border-base-content/10 bg-base-200/30 hover:bg-base-200/60 transition-colors"
                  >
                    <div className="flex items-center gap-2.5 min-w-0 flex-1">
                      <button
                        type="button"
                        onClick={() => handleTogglePlay(s.stream_url, s.id)}
                        className={`btn btn-circle btn-sm ${isPlaying ? "btn-primary" : "btn-outline"}`}
                      >
                        {isPlaying ? (
                          <Pause className="w-3.5 h-3.5" />
                        ) : (
                          <Play className="w-3.5 h-3.5 fill-current ml-0.5" />
                        )}
                      </button>

                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5">
                          <span className="opacity-70">
                            {getIconEl(s.icon || s.category)}
                          </span>
                          <span className="font-semibold text-xs truncate">
                            {s.name}
                          </span>
                        </div>
                        <div className="flex items-center gap-1 mt-0.5">
                          <span className="badge badge-ghost badge-xs text-[9px] uppercase">
                            {s.category}
                          </span>
                          {s.is_system && (
                            <span className="badge badge-info badge-xs text-[9px]">
                              {t("common.system", "System")}
                            </span>
                          )}
                        </div>
                      </div>
                    </div>

                    {(isOwner ||
                      hasPermission(user, "admin.soundscape.manage")) && (
                      <button
                        type="button"
                        onClick={() => handleDelete(s.id)}
                        className="btn btn-ghost btn-circle btn-xs text-error opacity-60 hover:opacity-100 ml-2"
                        title={t("common.delete", "Delete")}
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
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
