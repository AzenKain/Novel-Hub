import type { TFunction } from "i18next";
import { Volume2, Sliders } from "lucide-react";
import React, { useMemo, useState } from "react";

type ReaderTtsSettingsPanelProps = {
  t: TFunction;
  ttsVoices?: SpeechSynthesisVoice[];
  ttsSelectedVoice?: SpeechSynthesisVoice | null;
  setTtsSelectedVoice?: (voice: SpeechSynthesisVoice | null) => void;
  ttsRate?: number;
  setTtsRate?: (rate: number) => void;
};

export const ReaderTtsSettingsPanel: React.FC<ReaderTtsSettingsPanelProps> = ({
  t,
  ttsVoices = [],
  ttsSelectedVoice,
  setTtsSelectedVoice,
  ttsRate = 1.0,
  setTtsRate,
}) => {
  const [voiceSearch, setVoiceSearch] = useState("");

  const filteredVoices = useMemo(() => {
    if (!voiceSearch.trim()) return ttsVoices;
    const q = voiceSearch.toLowerCase().trim();
    return ttsVoices.filter(
      (v) =>
        v.name.toLowerCase().includes(q) ||
        v.lang.toLowerCase().includes(q)
    );
  }, [ttsVoices, voiceSearch]);

  return (
    <div className="reader-settings-panel absolute right-0 top-full z-50 mt-2 w-80 sm:w-96 rounded-2xl border p-4 shadow-2xl space-y-4 animate-in fade-in duration-150 backdrop-blur-md">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-current/10 pb-2.5">
        <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-current">
          <Volume2 className="size-4 opacity-80" />
          <span>{t("reader.tts_settings_header", "Voice & Speed Settings")}</span>
        </div>
        <span className="text-xs opacity-60 font-mono">
          {ttsVoices.length} {t("reader.voices_count", "voices")}
        </span>
      </div>

      {/* Current Voice Banner */}
      <div className="rounded-xl border border-current/15 bg-current/5 p-3">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <div className="text-[10px] font-bold uppercase tracking-wider opacity-60">
              {t("reader.current_voice", "Current Voice")}
            </div>
            <div className="truncate text-xs font-semibold text-current mt-0.5">
              {ttsSelectedVoice?.name || t("reader.no_voice_selected", "No voice selected")}
            </div>
          </div>
          {ttsSelectedVoice && (
            <span className="text-[10px] font-mono font-medium px-2 py-0.5 rounded-md border border-current/20 bg-current/10 text-current shrink-0">
              {ttsSelectedVoice.lang}
            </span>
          )}
        </div>
      </div>

      {/* Search Input */}
      <div>
        <input
          type="text"
          placeholder={t("reader.search_voice_or_lang", "Search by name or language...")}
          value={voiceSearch}
          onChange={(e) => setVoiceSearch(e.target.value)}
          className="input input-bordered input-sm w-full text-xs rounded-xl border border-current/20 bg-current/5 text-current placeholder:opacity-40 focus:border-primary focus:outline-hidden"
        />
      </div>

      {/* Select Voice List */}
      <div className="space-y-1.5">
        <div className="text-xs font-medium opacity-80">
          {t("reader.select_voice", "Select Voice")}
        </div>
        <div className="max-h-48 space-y-1.5 overflow-y-auto rounded-xl border border-current/15 bg-current/5 p-2">
          {filteredVoices.length === 0 ? (
            <div className="py-6 text-center text-xs opacity-50">
              {t("reader.no_voices_found", "No voices matching your search")}
            </div>
          ) : (
            filteredVoices.map((voice) => {
              const isSelected = ttsSelectedVoice?.name === voice.name;
              return (
                <button
                  key={`${voice.name}-${voice.lang}`}
                  type="button"
                  onClick={() => setTtsSelectedVoice?.(voice)}
                  className={`flex w-full items-center justify-between rounded-lg p-2.5 text-left border transition-all ${
                    isSelected
                      ? "border-primary bg-primary/15 text-primary shadow-xs font-semibold"
                      : "border-current/10 bg-current/5 hover:bg-current/10 text-current font-medium"
                  }`}
                >
                  <span className="truncate text-xs pr-2">{voice.name}</span>
                  <span className="text-[10px] font-mono px-1.5 py-0.5 rounded border border-current/20 bg-current/10 text-current shrink-0">
                    {voice.lang}
                  </span>
                </button>
              );
            })
          )}
        </div>
      </div>

      {/* Playback Speed Slider */}
      <div className="space-y-2 pt-1">
        <div className="flex items-center justify-between text-xs font-medium text-current">
          <span className="flex items-center gap-1.5 opacity-90">
            <Sliders className="size-3.5" />
            {t("reader.tts_speed", "Reading Speed")}
          </span>
          <span className="font-bold text-xs font-mono text-current">{ttsRate.toFixed(1)}x</span>
        </div>
        <input
          type="range"
          min="0.5"
          max="2.0"
          step="0.1"
          value={ttsRate}
          onChange={(e) => setTtsRate?.(parseFloat(e.target.value))}
          className="range range-xs range-primary w-full"
        />
        <div className="flex justify-between px-1 text-[10px] opacity-50 font-mono">
          <span>0.5x</span>
          <span>1.0x</span>
          <span>2.0x</span>
        </div>
      </div>
    </div>
  );
};
