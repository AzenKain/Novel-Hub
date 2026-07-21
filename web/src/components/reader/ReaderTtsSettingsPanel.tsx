import type { TFunction } from "i18next";
import { Volume2, Search, Sliders, Check, X } from "lucide-react";
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
  ttsVoices,
  ttsSelectedVoice,
  setTtsSelectedVoice,
  ttsRate,
  setTtsRate,
}) => {
  const [voiceSearch, setVoiceSearch] = useState("");

  const filteredVoices = useMemo(() => {
    if (!ttsVoices) return [];
    if (!voiceSearch.trim()) return ttsVoices;
    const lowerSearch = voiceSearch.toLowerCase().trim();
    return ttsVoices.filter(
      (v) =>
        v.name.toLowerCase().includes(lowerSearch) ||
        v.lang.toLowerCase().includes(lowerSearch)
    );
  }, [ttsVoices, voiceSearch]);

  return (
    <div className="reader-settings-panel absolute right-0 top-full z-50 mt-2 w-96 max-w-[calc(100vw-2rem)] rounded-2xl border p-4 shadow-2xl backdrop-blur-md transition-colors duration-300">
      <div className="mb-3 flex items-center justify-between border-b border-current/10 pb-2">
        <div className="flex items-center gap-2">
          <Volume2 className="h-4 w-4 text-primary" />
          <h3 className="text-xs font-bold uppercase tracking-wider opacity-70">
            {t("reader.tts_settings", "Voice & Speed Settings")}
          </h3>
        </div>
        <span className="font-mono text-[10px] opacity-50">
          {ttsVoices?.length || 0} {t("reader.voices", "voices")}
        </span>
      </div>

      <div className="flex flex-col gap-3">
        {/* Currently Selected Voice Banner */}
        <div className="flex items-center justify-between rounded-xl border border-primary/30 bg-primary/10 p-2.5 min-w-0">
          <div className="min-w-0 flex-1 pr-2">
            <div className="text-[10px] font-semibold uppercase tracking-wider text-primary opacity-80">
              {t("reader.current_voice", "Active Voice")}
            </div>
            <div className="truncate text-xs font-medium text-base-content min-w-0">
              {ttsSelectedVoice ? ttsSelectedVoice.name : t("reader.default_voice", "Default System Voice")}
            </div>
          </div>
          {ttsSelectedVoice && (
            <span className="shrink-0 rounded bg-primary/20 px-2 py-0.5 font-mono text-[10px] font-semibold text-primary">
              {ttsSelectedVoice.lang}
            </span>
          )}
        </div>

        {/* Search Input */}
        <div>
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 opacity-40" />
            <input
              type="text"
              placeholder={t("reader.search_voice_placeholder", "Filter voices by name or language...")}
              className="input input-sm input-bordered w-full pl-8 pr-8 text-xs"
              value={voiceSearch}
              onChange={(e) => setVoiceSearch(e.target.value)}
            />
            {voiceSearch && (
              <button
                type="button"
                onClick={() => setVoiceSearch("")}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 opacity-40 hover:opacity-100"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
        </div>

        {/* Custom Scrollable Voice List */}
        <div>
          <div className="mb-1 flex items-center justify-between">
            <span className="text-xs font-medium opacity-80">
              {t("reader.select_voice", "Select Voice")}
            </span>
            {voiceSearch && (
              <span className="text-[10px] font-mono text-primary">
                {filteredVoices.length} {t("reader.found", "found")}
              </span>
            )}
          </div>

          <div className="max-h-52 overflow-y-auto rounded-xl border border-base-300 bg-base-200/50 p-1 divide-y divide-base-300/30">
            {filteredVoices && filteredVoices.length > 0 ? (
              filteredVoices.map((voice) => {
                const isSelected = ttsSelectedVoice?.name === voice.name;
                return (
                  <button
                    key={voice.name}
                    type="button"
                    onClick={() => setTtsSelectedVoice && setTtsSelectedVoice(voice)}
                    className={`flex w-full items-center justify-between gap-2 rounded-lg p-2 text-left text-xs transition-colors min-w-0 ${
                      isSelected
                        ? "bg-primary/20 font-semibold text-primary"
                        : "hover:bg-base-300/60"
                    }`}
                  >
                    <span className="min-w-0 flex-1 truncate">{voice.name}</span>
                    <span className="shrink-0 rounded bg-base-300/80 px-1.5 py-0.5 font-mono text-[10px] opacity-70">
                      {voice.lang}
                    </span>
                    {isSelected && (
                      <Check className="h-4 w-4 shrink-0 text-primary" />
                    )}
                  </button>
                );
              })
            ) : (
              <div className="p-4 text-center text-xs opacity-50">
                {t("reader.no_voices_found", "No voices matching your search")}
              </div>
            )}
          </div>
        </div>

        {/* Speed Controls */}
        <div className="border-t border-current/10 pt-2">
          <div className="mb-1 flex items-center justify-between">
            <span className="flex items-center gap-1.5 text-xs font-medium opacity-80">
              <Sliders className="h-3.5 w-3.5 opacity-60" />
              {t("reader.tts_speed", "Reading Speed")}
            </span>
            <span className="font-mono text-xs font-semibold text-primary">
              {(ttsRate || 1).toFixed(1)}x
            </span>
          </div>
          <input
            type="range"
            min="0.5"
            max="2"
            step="0.1"
            value={ttsRate || 1}
            onChange={(e) => setTtsRate && setTtsRate(parseFloat(e.target.value))}
            className="range range-primary range-xs w-full"
          />
          <div className="mt-1 flex justify-between text-[10px] font-mono opacity-40">
            <span>0.5x</span>
            <span>1.0x</span>
            <span>2.0x</span>
          </div>
        </div>
      </div>
    </div>
  );
};
