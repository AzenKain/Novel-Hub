import React from "react";
import { useTranslation } from "react-i18next";
import {
  Volume2,
  VolumeX,
  Play,
  Pause,
  CloudRain,
  Wind,
  Coffee,
  Flame,
  Music,
  Plus,
  Sliders,
  Sparkles,
  Waves,
  X,
  Headphones,
  Zap,
  Disc,
  Radio,
  Gamepad2,
} from "lucide-react";
import { useSoundscapeStore, BUILTIN_AMBIENT_PRESETS, type ActiveTrack } from "@/stores/soundscapeStore";
import { useSoundscapesQuery } from "@/hooks/useCustomization";
import { Link } from "react-router-dom";

type ReaderSoundscapePanelProps = {
  onClose?: () => void;
};

const getCategoryIcon = (category: string, iconName?: string) => {
  switch (iconName?.toLowerCase() || category?.toLowerCase()) {
    case "rain":
    case "cloudrain":
      return <CloudRain className="w-4 h-4" />;
    case "wind":
      return <Wind className="w-4 h-4" />;
    case "coffee":
    case "cafe":
      return <Coffee className="w-4 h-4" />;
    case "flame":
    case "fire":
      return <Flame className="w-4 h-4" />;
    case "waves":
    case "water":
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

export const ReaderSoundscapePanel: React.FC<ReaderSoundscapePanelProps> = ({ onClose }) => {
  const { t } = useTranslation();
  const {
    isPlaying,
    masterVolume,
    activeTracks,
    togglePlaying,
    setMasterVolume,
    toggleTrack,
    setTrackVolume,
    stopAll,
    applyPreset,
  } = useSoundscapeStore();

  const { data: serverSoundscapes = [] } = useSoundscapesQuery();

  const allAvailableTracks: ActiveTrack[] = [
    ...BUILTIN_AMBIENT_PRESETS,
    ...serverSoundscapes.map((s) => ({
      id: s.id,
      name: s.name,
      category: s.category || "ambient",
      icon: s.icon || "Music",
      volume: s.volume || 0.5,
      streamUrl: s.stream_url,
    })),
  ];

  const activeCount = Object.keys(activeTracks).length;

  return (
    <div className="reader-settings-panel absolute right-0 top-full z-50 mt-2 max-h-[calc(100vh-5rem)] w-80 md:w-96 overflow-y-auto rounded-2xl border p-4 shadow-2xl backdrop-blur-md transition-colors duration-300">
      {/* Header */}
      <div className="flex items-center justify-between mb-3 border-b border-(--reader-ui-border) pb-2.5">
        <div className="flex items-center gap-2">
          <div className="p-1.5 rounded-lg bg-(--reader-ui-accent-soft) text-(--reader-ui-accent)">
            <Sliders className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-xs font-bold uppercase tracking-wider opacity-90">
              {t("soundscape.title", "Ambient Soundscapes")}
            </h3>
            <p className="text-[11px] opacity-60">
              {activeCount > 0
                ? t("soundscape.active_tracks", "{{count}} active layers", { count: activeCount })
                : t("soundscape.no_active", "No ambient sounds playing")}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-1">
          {activeCount > 0 && (
            <button
              type="button"
              onClick={stopAll}
              className="btn btn-ghost btn-xs text-error"
              title={t("soundscape.stop_all", "Stop all")}
            >
              {t("common.clear", "Clear")}
            </button>
          )}
          {onClose && (
            <button
              type="button"
              onClick={onClose}
              className="reader-control-btn btn btn-ghost btn-circle btn-xs opacity-60 hover:opacity-100"
            >
              <X className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>

      {/* Master Volume & Play/Pause Control Bar */}
      <div className="mb-4 p-3 rounded-xl bg-(--reader-ui-soft) border border-(--reader-ui-border) flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={togglePlaying}
              className={`btn btn-circle btn-sm ${isPlaying && activeCount > 0 ? "reader-action-btn" : "reader-outline-btn"}`}
              title={isPlaying && activeCount > 0 ? t("common.pause", "Pause") : t("common.play", "Play")}
            >
              {isPlaying && activeCount > 0 ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4 fill-current ml-0.5" />}
            </button>
            <span className="text-xs font-bold">
              {activeCount === 0
                ? t("soundscape.no_active", "No ambient sounds playing")
                : isPlaying
                  ? t("soundscape.playing", "Playing")
                  : t("soundscape.paused", "Paused")}
            </span>
          </div>

          <div className="flex items-center gap-1.5 min-w-30">
            {masterVolume === 0 ? (
              <VolumeX className="w-4 h-4 opacity-50 shrink-0" />
            ) : (
              <Volume2 className="w-4 h-4 opacity-70 shrink-0 text-(--reader-ui-accent)" />
            )}
            <input
              type="range"
              min="0"
              max="1"
              step="0.05"
              value={masterVolume}
              onChange={(e) => setMasterVolume(parseFloat(e.target.value))}
              className="range range-xs flex-1"
            />
            <span className="font-mono text-[10px] opacity-60 w-7 text-right">
              {Math.round(masterVolume * 100)}%
            </span>
          </div>
        </div>

        {/* Quick Focus Presets */}
        <div className="flex items-center gap-1.5 pt-1.5 border-t border-(--reader-ui-border) overflow-x-auto custom-scrollbar">
          <span className="text-[10px] font-bold uppercase opacity-50 shrink-0">
            <Sparkles className="w-3 h-3 inline mr-1 text-(--reader-ui-accent)" />
            {t("soundscape.presets", "Presets")}:
          </span>
          <button
            type="button"
            onClick={() => applyPreset("pink_focus")}
            className="btn btn-xs reader-control-btn border border-(--reader-ui-border) rounded-lg text-[11px] shrink-0"
          >
            Pink Focus
          </button>
          <button
            type="button"
            onClick={() => applyPreset("deep_brown")}
            className="btn btn-xs reader-control-btn border border-(--reader-ui-border) rounded-lg text-[11px] shrink-0"
          >
            Brown Cozy
          </button>
        </div>
      </div>

      {/* Available Sound Tracks */}
      <div className="space-y-2 max-h-72 overflow-y-auto pr-1 custom-scrollbar">
        {allAvailableTracks.map((track) => {
          const isActive = Boolean(activeTracks[track.id]);
          const currentVol = activeTracks[track.id]?.volume ?? track.volume;

          return (
            <div
              key={track.id}
              className={`p-2.5 rounded-xl border transition-all ${
                isActive
                  ? "bg-(--reader-ui-accent-soft) border-(--reader-ui-accent)/50 shadow-2xs"
                  : "bg-(--reader-ui-soft) border-(--reader-ui-border) hover:bg-(--reader-ui-hover)"
              }`}
            >
              <div className="flex items-center justify-between gap-2 mb-1.5">
                <button
                  type="button"
                  onClick={() => toggleTrack(track)}
                  className="flex items-center gap-2.5 min-w-0 flex-1 text-left group"
                >
                  <div
                    className={`p-1.5 rounded-lg transition-colors ${
                      isActive
                        ? "bg-(--reader-ui-accent) text-(--reader-ui-accent-text)"
                        : "bg-(--reader-ui-surface-strong) text-(--reader-ui-text) opacity-80"
                    }`}
                  >
                    {getCategoryIcon(track.category, track.icon)}
                  </div>
                  <div className="min-w-0">
                    <p className={`text-xs font-bold truncate ${isActive ? "text-(--reader-ui-accent)" : "text-(--reader-ui-text)"}`}>
                      {track.name}
                    </p>
                    <span className="badge badge-ghost badge-xs text-[10px] opacity-60 uppercase border border-(--reader-ui-border)">
                      {track.category}
                    </span>
                  </div>
                </button>

                <button
                  type="button"
                  onClick={() => toggleTrack(track)}
                  className={`btn btn-circle btn-xs ${isActive ? "reader-action-btn" : "reader-control-btn"}`}
                >
                  {isActive ? <Pause className="w-3 h-3" /> : <Play className="w-3 h-3 fill-current ml-0.5" />}
                </button>
              </div>

              {isActive && (
                <div className="flex items-center gap-2 pt-1.5 border-t border-(--reader-ui-border) animate-in fade-in duration-200">
                  <Volume2 className="w-3 h-3 opacity-60 shrink-0 text-(--reader-ui-accent)" />
                  <input
                    type="range"
                    min="0"
                    max="1"
                    step="0.05"
                    value={currentVol}
                    onChange={(e) => setTrackVolume(track.id, parseFloat(e.target.value))}
                    className="range range-xs flex-1"
                  />
                  <span className="font-mono text-[10px] opacity-60 w-7 text-right">
                    {Math.round(currentVol * 100)}%
                  </span>
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* Footer link to manage / upload personal sounds */}
      <div className="mt-3 pt-2.5 border-t border-(--reader-ui-border) flex items-center justify-between text-xs">
        <span className="text-[11px] opacity-60">
          {t("soundscape.custom_soundscapes_hint", "Want your own sounds?")}
        </span>
        <Link
          to="/profile?tab=customization"
          target="_blank"
          className="link link-hover text-(--reader-ui-accent) font-medium text-[11px] inline-flex items-center gap-1"
        >
          <Plus className="w-3 h-3" />
          {t("soundscape.manage_sounds", "Manage in Profile")}
        </Link>
      </div>
    </div>
  );
};
