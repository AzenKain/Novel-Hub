import type { TFunction } from "i18next";
import i18n from "@/i18n";
import { Volume2, Play, Copy, MessageSquarePlus, Languages, Sparkles, Loader2, Check } from "lucide-react";
import React, { useState, useEffect, useLayoutEffect } from "react";
import {
  SUPPORTED_LANGUAGES,
  getDefaultTargetLanguage,
  saveTargetLanguagePreference,
  translateText,
} from "@/utils/translation";

type ReaderSelectionToolbarProps = {
  t: TFunction;
  toolbarPos: { top: number; left: number; placement?: "above" | "below" };
  isSupported: boolean;
  selectedText?: string;
  onReadSelection: () => void;
  onReadFromHere: () => void;
  onCopyText?: () => void;
  onHighlight?: (color: string, note?: string) => void;
  onOpenQuoteCard?: (text?: string, imageUrl?: string) => void;
};

const HIGHLIGHT_COLOR_OPTIONS = [
  { color: "#fef08a", bgClass: "bg-[#facc15]", labelKey: "reader.highlight_yellow", defaultName: "Yellow" },
  { color: "#bbf7d0", bgClass: "bg-[#4ade80]", labelKey: "reader.highlight_green", defaultName: "Green" },
  { color: "#bfdbfe", bgClass: "bg-[#60a5fa]", labelKey: "reader.highlight_blue", defaultName: "Blue" },
  { color: "#fbcfe8", bgClass: "bg-[#f472b6]", labelKey: "reader.highlight_pink", defaultName: "Pink" },
  { color: "#fed7aa", bgClass: "bg-[#fb923c]", labelKey: "reader.highlight_orange", defaultName: "Orange" },
  { color: "#e9d5ff", bgClass: "bg-[#c084fc]", labelKey: "reader.highlight_purple", defaultName: "Purple" },
  { color: "#fecaca", bgClass: "bg-[#f87171]", labelKey: "reader.highlight_red", defaultName: "Red" },
  { color: "#99f6e4", bgClass: "bg-[#2dd4bf]", labelKey: "reader.highlight_cyan", defaultName: "Cyan" },
];

export const ReaderSelectionToolbar: React.FC<ReaderSelectionToolbarProps> = ({
  t,
  toolbarPos,
  isSupported,
  selectedText,
  onReadSelection,
  onReadFromHere,
  onCopyText,
  onHighlight,
  onOpenQuoteCard,
}) => {
  const [note, setNote] = useState("");
  const [showTranslate, setShowTranslate] = useState(false);
  const [translatedText, setTranslatedText] = useState("");
  const [translating, setTranslating] = useState(false);
  const [targetLang, setTargetLang] = useState(() =>
    getDefaultTargetLanguage(typeof i18n !== "undefined" ? i18n.language : undefined)
  );
  const [copiedTrans, setCopiedTrans] = useState(false);
  const toolbarRef = React.useRef<HTMLDivElement>(null);
  const [computedTop, setComputedTop] = useState(toolbarPos.top);

  // Measure actual toolbar height after render and clamp within viewport bounds
  // (between header bar and bottom viewport edge) so toolbar is always completely visible.
  useLayoutEffect(() => {
    const el = toolbarRef.current;
    if (!el) return;
    const height = el.offsetHeight;
    const topBarHeight = 56;
    const margin = 8;
    const vh = window.innerHeight;

    let top = toolbarPos.top;
    // If expanding downwards would overflow the bottom of viewport, shift up just enough
    if (top + height > vh - margin) {
      top = vh - height - margin;
    }
    // Always keep top below reader header bar
    top = Math.max(topBarHeight + margin, top);
    setComputedTop(top);
  }, [toolbarPos.top, showTranslate, translatedText, translating]);

  useEffect(() => {
    setComputedTop(toolbarPos.top);
  }, [toolbarPos.top]);

  const handleColorClick = (color: string) => {
    onHighlight?.(color, note);
  };

  const handleTranslate = async (lang = targetLang) => {
    const text = (selectedText || window.getSelection()?.toString() || "").trim();
    if (!text) return;
    setShowTranslate(true);
    setTranslating(true);
    try {
      const res = await translateText(text, lang);
      setTranslatedText(res.text);
      saveTargetLanguagePreference(lang);
    } catch {
      setTranslatedText(t("common.translate_error", "Could not translate this passage"));
    } finally {
      setTranslating(false);
    }
  };

  return (
    <div
      ref={toolbarRef}
      data-reader-toolbar="true"
      onMouseDown={(e) => {
        e.stopPropagation();
      }}
      onMouseUp={(e) => {
        e.stopPropagation();
      }}
      className="reader-selection-toolbar fixed z-50 flex w-[calc(100vw-1rem)] max-w-92 sm:max-w-96 max-h-[calc(100vh-5rem)] flex-col gap-2 rounded-2xl border border-(--reader-ui-border) bg-(--reader-ui-surface-strong) p-2.5 shadow-2xl backdrop-blur-md animate-in fade-in duration-100"
      style={{ top: `${computedTop}px`, left: `${toolbarPos.left}px` }}
    >
      {/* Top Row: 8 Highlight Colors */}
      {onHighlight && (
        <div data-reader-toolbar="true" className="flex items-center justify-between gap-1 w-full px-0.5">
          {HIGHLIGHT_COLOR_OPTIONS.map((item) => (
            <button
              key={item.color}
              type="button"
              data-reader-toolbar="true"
              onMouseDown={(e) => {
                e.preventDefault();
                e.stopPropagation();
              }}
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                handleColorClick(item.color);
              }}
              className={`h-5 w-5 sm:h-5.5 sm:w-5.5 shrink-0 rounded-full border border-black/15 shadow-xs transition-all hover:scale-125 active:scale-95 focus:outline-hidden ${item.bgClass}`}
              title={t(item.labelKey, item.defaultName)}
              aria-label={t(item.labelKey, item.defaultName)}
            />
          ))}
        </div>
      )}

      {/* Action Row 1: Text-to-Speech (TTS) if supported */}
      {isSupported && (
        <div data-reader-toolbar="true" className="flex items-center gap-1.5 pt-0.5 w-full">
          <button
            type="button"
            data-reader-toolbar="true"
            onMouseDown={(e) => {
              e.preventDefault();
              e.stopPropagation();
            }}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onReadSelection();
            }}
            className="btn btn-ghost btn-xs h-7.5 flex-1 min-w-0 px-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) hover:bg-(--reader-ui-hover) gap-1.5 text-[11px] sm:text-xs font-medium transition-colors"
            title={t("reader.read_selection", "Read Selection")}
            aria-label={t("reader.read_selection", "Read Selection")}
          >
            <Volume2 className="h-3.5 w-3.5 text-sky-500 shrink-0" />
            <span className="truncate">{t("reader.read_selection", "Read Selection")}</span>
          </button>
          <button
            type="button"
            data-reader-toolbar="true"
            onMouseDown={(e) => {
              e.preventDefault();
              e.stopPropagation();
            }}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onReadFromHere();
            }}
            className="btn btn-ghost btn-xs h-7.5 flex-1 min-w-0 px-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) hover:bg-(--reader-ui-hover) gap-1.5 text-[11px] sm:text-xs font-medium transition-colors"
            title={t("reader.read_from_here", "Read From Here")}
            aria-label={t("reader.read_from_here", "Read From Here")}
          >
            <Play className="h-3.5 w-3.5 text-emerald-500 shrink-0" />
            <span className="truncate">{t("reader.read_from_here", "Read From Here")}</span>
          </button>
        </div>
      )}

      {/* Action Row 2: Copy, Translate & Quote Tools */}
      <div data-reader-toolbar="true" className="flex items-center gap-1.5 w-full">
        {onCopyText && (
          <button
            type="button"
            data-reader-toolbar="true"
            onMouseDown={(e) => {
              e.preventDefault();
              e.stopPropagation();
            }}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onCopyText();
            }}
            className="btn btn-ghost btn-xs h-7 flex-1 min-w-0 px-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) hover:bg-(--reader-ui-hover) gap-1 text-[11px] sm:text-xs font-medium transition-colors"
            title={t("reader.copy_text", "Copy Text")}
            aria-label={t("reader.copy_text", "Copy Text")}
          >
            <Copy className="h-3.5 w-3.5 text-amber-500 shrink-0" />
            <span>{t("common.copy", "Copy")}</span>
          </button>
        )}

        {/* Quick Translate Button */}
        <button
          type="button"
          data-reader-toolbar="true"
          onMouseDown={(e) => {
            e.preventDefault();
            e.stopPropagation();
          }}
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            if (showTranslate) {
              setShowTranslate(false);
            } else {
              handleTranslate(targetLang);
            }
          }}
          className={`btn btn-ghost btn-xs h-7 flex-1 min-w-0 px-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) hover:bg-(--reader-ui-hover) gap-1 text-[11px] sm:text-xs font-medium transition-colors ${
            showTranslate ? "border-primary text-primary font-semibold" : ""
          }`}
          title={t("reader.translate", "Translate")}
        >
          <Languages className="h-3.5 w-3.5 text-indigo-400 shrink-0" />
          <span>{t("reader.translate", "Translate")}</span>
        </button>

        {/* Quote Card Button */}
        {onOpenQuoteCard && (
          <button
            type="button"
            data-reader-toolbar="true"
            onMouseDown={(e) => {
              e.preventDefault();
              e.stopPropagation();
            }}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              const textToQuote = (selectedText || window.getSelection()?.toString() || "").trim();
              onOpenQuoteCard(textToQuote);
            }}
            className="btn btn-ghost btn-xs h-7 flex-1 min-w-0 px-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) hover:bg-(--reader-ui-hover) gap-1 text-[11px] sm:text-xs font-medium transition-colors"
            title={t("reader.quote_card", "Create quote image")}
          >
            <Sparkles className="h-3.5 w-3.5 text-purple-400 shrink-0" />
            <span>{t("reader.quote", "Quote")}</span>
          </button>
        )}
      </div>

      {/* Expandable Translation Result Box */}
      {showTranslate && (
        <div
          data-reader-toolbar="true"
          className="w-full min-h-0 flex flex-col rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) text-xs animate-in fade-in zoom-in-95 duration-150 overflow-hidden"
        >
          <div className="flex items-center justify-between gap-2 border-b border-(--reader-ui-border)/60 p-2.5 pb-1.5 shrink-0">
            <div className="flex items-center gap-1.5 font-bold text-[11px] text-primary shrink-0">
              <Languages className="w-3.5 h-3.5" />
              <span>{t("reader.translation_result", "Translation")}</span>
            </div>
            <select
              value={targetLang}
              onChange={(e) => {
                const newLang = e.target.value;
                setTargetLang(newLang);
                saveTargetLanguagePreference(newLang);
                handleTranslate(newLang);
              }}
              className="select select-bordered select-xs h-6 min-h-0 text-[10px] sm:text-[11px] rounded-lg bg-(--reader-ui-surface-strong) text-(--reader-ui-text) font-medium max-w-42.5 sm:max-w-50"
            >
              {SUPPORTED_LANGUAGES.map((lang) => (
                <option key={lang.code} value={lang.code}>
                  {lang.flag} {lang.nativeName} ({lang.name})
                </option>
              ))}
            </select>
          </div>

          {translating ? (
            <div className="py-2 flex items-center justify-center gap-2 text-xs text-(--reader-ui-muted) shrink-0">
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
              <span>{t("common.loading", "Loading...")}</span>
            </div>
          ) : (
            <div className="flex items-start justify-between gap-2 p-2.5 max-h-48 sm:max-h-56 min-h-0 overflow-y-auto overscroll-contain">
              <p className="leading-relaxed select-text flex-1 font-medium whitespace-pre-wrap">{translatedText}</p>
              {translatedText && (
                <button
                  type="button"
                  onClick={() => {
                    navigator.clipboard.writeText(translatedText);
                    setCopiedTrans(true);
                    setTimeout(() => setCopiedTrans(false), 2000);
                  }}
                  className="btn btn-ghost btn-circle btn-xs shrink-0 sticky top-0"
                  title={t("common.copy", "Copy")}
                >
                  {copiedTrans ? <Check className="w-3 h-3 text-success" /> : <Copy className="w-3 h-3" />}
                </button>
              )}
            </div>
          )}
        </div>
      )}

      {/* Bottom Row: Optional Note Input */}
      {onHighlight && (
        <div data-reader-toolbar="true" className="w-full pt-1 border-t border-(--reader-ui-border)/60">
          <div className="relative flex items-start">
            <MessageSquarePlus className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-(--reader-ui-muted) pointer-events-none" />
            <textarea
              rows={2}
              data-reader-toolbar="true"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              onMouseDown={(e) => {
                e.stopPropagation();
              }}
              onClick={(e) => {
                e.stopPropagation();
              }}
              onKeyDown={(e) => {
                e.stopPropagation();
                if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                  e.preventDefault();
                  handleColorClick(HIGHLIGHT_COLOR_OPTIONS[0].color);
                }
              }}
              placeholder={t("reader.add_note_placeholder", "Add a note (optional)...")}
              className="reader-input w-full rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) pl-8 pr-2.5 py-1.5 text-xs text-(--reader-ui-text) placeholder:text-(--reader-ui-muted)/70 focus:border-(--reader-ui-accent) focus:outline-hidden transition-colors resize-none leading-relaxed"
            />
          </div>
        </div>
      )}
    </div>
  );
};
