import React, { useState, useRef, useMemo } from "react";
import { useTranslation } from "react-i18next";
import html2canvas from "html2canvas-pro";
import { toast } from "react-toastify";
import {
  Download,
  Copy,
  X,
  Sparkles,
  BookOpen,
  Flame,
  Clock,
  Calendar,
  Layers,
  Check,
  Loader2,
  Bookmark,
  RotateCcw,
  TrendingUp,
} from "lucide-react";
import { useAuthStore } from "@/stores";
import { useShallow } from "zustand/react/shallow";
import { getMediaUrl } from "@/config/api";
import { usePublicSettings } from "@/hooks/useSettings";
import type { LibraryBreakdown, ReadingStatsSummary } from "@/types";

export type CardPeriod = "year" | "month" | "all";
export type CardRatio = "story" | "square" | "wide";
export type CardTheme = "aurora" | "cyberpunk" | "sepia" | "eink";

interface ReadingCardModalProps {
  isOpen: boolean;
  onClose: () => void;
  heatmapData?: Record<string, { duration: number; words: number }>;
  summary?: ReadingStatsSummary;
  breakdown?: LibraryBreakdown;
}

export const ReadingCardModal: React.FC<ReadingCardModalProps> = ({
  isOpen,
  onClose,
  heatmapData,
  summary,
  breakdown,
}) => {
  const { t } = useTranslation();
  const { user } = useAuthStore(useShallow((s) => ({ user: s.user })));
  const cardRef = useRef<HTMLDivElement>(null);
  const publicSettings = usePublicSettings();

  const siteLogo = publicSettings?.site?.logo;
  const siteTitle = publicSettings?.site?.title || "NovelHub";

  const defaultQuote = t(
    "analytics.card_quote",
    "A reader lives a thousand lives.",
  );
  const [period, setPeriod] = useState<CardPeriod>("year");
  const [ratio, setRatio] = useState<CardRatio>("story");
  const [theme, setTheme] = useState<CardTheme>("aurora");
  const [customQuote, setCustomQuote] = useState<string>(defaultQuote);
  const [showLogo, setShowLogo] = useState<boolean>(true);
  const [isExporting, setIsExporting] = useState(false);
  const [copied, setCopied] = useState(false);

  const stats = useMemo(() => {
    const now = new Date();
    const currentYear = now.getFullYear();
    const currentMonth = now.getMonth() + 1;
    const monthStr = String(currentMonth).padStart(2, "0");

    let totalWords = 0;
    let totalDuration = 0;
    const activeDateList: string[] = [];

    if (period === "all") {
      totalWords = summary?.total_words ?? 0;
      totalDuration = (summary?.total_minutes ?? 0) * 60;
      const activeDays = summary?.total_active_days ?? 0;
      const longestStreak = summary?.longest_streak_days ?? 0;
      return {
        words: totalWords,
        minutes: Math.round(totalDuration / 60),
        activeDays,
        streak: longestStreak,
        periodLabel: t("analytics.card_period_all", "All-Time Journey"),
      };
    }

    const prefix =
      period === "year" ? `${currentYear}-` : `${currentYear}-${monthStr}-`;

    if (heatmapData && typeof heatmapData === "object") {
      Object.entries(heatmapData).forEach(([date, item]) => {
        if (date.startsWith(prefix) && item && item.words > 0) {
          totalWords += item.words;
          totalDuration += item.duration || 0;
          activeDateList.push(date);
        }
      });
    }

    // Longest streak for period
    let maxStreak = 0;
    if (activeDateList.length > 0) {
      const sorted = [...new Set(activeDateList)].sort();
      maxStreak = 1;
      let curr = 1;
      for (let i = 1; i < sorted.length; i++) {
        const prev = new Date(sorted[i - 1]).getTime();
        const next = new Date(sorted[i]).getTime();
        const diff = Math.round((next - prev) / (1000 * 60 * 60 * 24));
        if (diff === 1) {
          curr++;
          if (curr > maxStreak) maxStreak = curr;
        } else if (diff > 1) {
          curr = 1;
        }
      }
    }

    const monthName = now.toLocaleDateString(undefined, {
      month: "long",
      year: "numeric",
    });
    const periodLabel = period === "year" ? `${currentYear}` : monthName;

    return {
      words: totalWords,
      minutes: Math.round(totalDuration / 60),
      activeDays: activeDateList.length,
      streak: maxStreak,
      periodLabel,
    };
  }, [heatmapData, period, summary, t]);

  // Peak reading day in period
  const peakDay = useMemo(() => {
    if (!heatmapData || typeof heatmapData !== "object") return null;
    const now = new Date();
    const currentYear = now.getFullYear();
    const currentMonth = now.getMonth() + 1;
    const prefix =
      period === "all"
        ? ""
        : period === "year"
          ? `${currentYear}-`
          : `${currentYear}-${String(currentMonth).padStart(2, "0")}-`;

    let maxWords = 0;
    let maxDate = "";
    Object.entries(heatmapData).forEach(([date, item]) => {
      if (
        (prefix === "" || date.startsWith(prefix)) &&
        item &&
        item.words > maxWords
      ) {
        maxWords = item.words;
        maxDate = date;
      }
    });

    if (maxWords <= 0) return null;
    const d = new Date(maxDate);
    const day = String(d.getDate()).padStart(2, "0");
    const month = String(d.getMonth() + 1).padStart(2, "0");
    return { date: `${day}/${month}`, words: maxWords };
  }, [heatmapData, period]);

  // 7-day reading rhythm bars
  const last7Days = useMemo(() => {
    const bars: Array<{ day: string; words: number; date: string }> = [];
    if (!heatmapData || typeof heatmapData !== "object") return bars;
    for (let i = 6; i >= 0; i--) {
      const d = new Date();
      d.setDate(d.getDate() - i);
      const key = d.toLocaleDateString("sv-SE");
      const dayName = d.toLocaleDateString(undefined, { weekday: "narrow" });
      const words = heatmapData[key]?.words ?? 0;
      bars.push({ day: dayName, words, date: key });
    }
    return bars;
  }, [heatmapData]);

  const max7DaysWords = useMemo(() => {
    return Math.max(1, ...last7Days.map((b) => b.words));
  }, [last7Days]);

  // Reader Persona / Level
  const readerLevel = useMemo(() => {
    const w = stats.words;
    if (w >= 500000)
      return {
        title: t("analytics.rank_grandmaster", "Grandmaster Reader"),
        icon: "🏆",
      };
    if (w >= 200000)
      return {
        title: t("analytics.rank_scholar", "Erudite Scholar"),
        icon: "🌟",
      };
    if (w >= 100000)
      return {
        title: t("analytics.rank_bookworm", "Avid Bookworm"),
        icon: "📖",
      };
    if (w >= 30000)
      return {
        title: t("analytics.rank_explorer", "Knowledge Explorer"),
        icon: "🚀",
      };
    return { title: t("analytics.rank_novice", "Novice Reader"), icon: "🌱" };
  }, [stats.words, t]);

  const avgWordsPerDay =
    stats.activeDays > 0 ? Math.round(stats.words / stats.activeDays) : 0;

  const topGenres = useMemo(() => {
    return (breakdown?.tags || []).slice(0, 3).map((tg) => tg.name);
  }, [breakdown?.tags]);

  const topAuthors = useMemo(() => {
    return (breakdown?.authors || []).slice(0, 2).map((a) => a.name);
  }, [breakdown?.authors]);

  const topFormats = useMemo(() => {
    return (breakdown?.formats || []).slice(0, 2).map((f) => f.name);
  }, [breakdown?.formats]);

  const themeStyles = useMemo(() => {
    switch (theme) {
      case "cyberpunk":
        return {
          container:
            "bg-[#090a0f] text-[#f1f5f9] border border-amber-400/40 shadow-2xl",
          heroNumber: "text-amber-400 font-mono",
          badge: "bg-cyan-950/80 text-cyan-300 border border-cyan-500/40",
          statBox: "bg-[#12141e] border border-amber-400/20 text-slate-200",
          accentText: "text-amber-400",
          subText: "text-slate-400",
          borderCol: "border-amber-400/20",
          tagBadge: "bg-amber-400/10 text-amber-300 border border-amber-400/30",
          barActive: "bg-amber-400",
          barInactive: "bg-amber-400/20",
          levelBadge:
            "bg-amber-400/15 text-amber-300 border border-amber-400/30",
        };
      case "sepia":
        return {
          container:
            "bg-[#fcf8f2] text-[#2c1810] border-2 border-[#c5a880] shadow-2xl font-serif",
          heroNumber: "text-[#4a2e1b]",
          badge: "bg-[#ede0d4] text-[#4a2e1b] border border-[#c5a880]",
          statBox: "bg-[#f4ecdf] border border-[#c5a880]/50 text-[#2c1810]",
          accentText: "text-[#8b5a2b]",
          subText: "text-[#7f6a58]",
          borderCol: "border-[#c5a880]/40",
          tagBadge: "bg-[#ede0d4] text-[#4a2e1b] border border-[#c5a880]",
          barActive: "bg-[#8b5a2b]",
          barInactive: "bg-[#8b5a2b]/20",
          levelBadge: "bg-[#ede0d4] text-[#4a2e1b] border border-[#c5a880]",
        };
      case "eink":
        return {
          container: "bg-black text-white border-2 border-white shadow-2xl",
          heroNumber: "text-white font-mono",
          badge: "bg-white text-black font-bold border border-white",
          statBox: "bg-zinc-900 border border-zinc-700 text-white",
          accentText: "text-white",
          subText: "text-zinc-400",
          borderCol: "border-zinc-800",
          tagBadge: "bg-zinc-800 text-white border border-zinc-600",
          barActive: "bg-white",
          barInactive: "bg-zinc-800",
          levelBadge: "bg-zinc-800 text-white border border-zinc-600",
        };
      case "aurora":
      default:
        return {
          container:
            "bg-gradient-to-br from-slate-950 via-indigo-950 to-purple-950 text-slate-100 border border-indigo-500/30 shadow-2xl",
          heroNumber:
            "text-transparent bg-clip-text bg-gradient-to-r from-cyan-300 via-indigo-300 to-fuchsia-300",
          badge: "bg-indigo-500/20 text-indigo-300 border border-indigo-500/40",
          statBox:
            "bg-white/[0.05] border border-white/10 backdrop-blur-sm text-slate-200",
          accentText: "text-indigo-400",
          subText: "text-slate-400",
          borderCol: "border-white/10",
          tagBadge:
            "bg-fuchsia-500/15 text-fuchsia-300 border border-fuchsia-500/30",
          barActive: "bg-gradient-to-t from-indigo-500 to-fuchsia-400",
          barInactive: "bg-white/10",
          levelBadge:
            "bg-indigo-500/20 text-indigo-200 border border-indigo-500/30",
        };
    }
  }, [theme]);

  const generateCanvas = async (): Promise<HTMLCanvasElement | null> => {
    if (!cardRef.current) return null;
    return await html2canvas(cardRef.current, {
      scale: 3,
      useCORS: true,
      allowTaint: true,
      backgroundColor: null,
      logging: false,
    });
  };

  const handleDownload = async () => {
    try {
      setIsExporting(true);
      const canvas = await generateCanvas();
      if (!canvas) return;
      const url = canvas.toDataURL("image/png");
      const a = document.createElement("a");
      a.download = `novelhub-reading-card-${period}-${Date.now()}.png`;
      a.href = url;
      a.click();
    } catch (err) {
      console.error("Download failed", err);
      toast.error(t("analytics.card_error", "Failed to export image"));
    } finally {
      setIsExporting(false);
    }
  };

  const handleCopy = async () => {
    try {
      setIsExporting(true);
      const canvas = await generateCanvas();
      if (!canvas) return;
      canvas.toBlob(async (blob) => {
        if (!blob) return;
        try {
          await navigator.clipboard.write([
            new ClipboardItem({ "image/png": blob }),
          ]);
          setCopied(true);
          toast.success(
            t("analytics.card_copied_toast", "Card copied to clipboard!"),
          );
          setTimeout(() => setCopied(false), 2000);
        } catch {
          const url = canvas.toDataURL("image/png");
          const a = document.createElement("a");
          a.download = `novelhub-reading-card-${period}.png`;
          a.href = url;
          a.click();
        }
      }, "image/png");
    } catch (err) {
      console.error("Copy failed", err);
      toast.error(t("analytics.card_error", "Failed to copy image"));
    } finally {
      setIsExporting(false);
    }
  };

  if (!isOpen) return null;

  const topGenre = breakdown?.tags?.[0]?.name;
  const topAuthor = breakdown?.authors?.[0]?.name;
  const topFormat = breakdown?.formats?.[0]?.name;
  const displayName = user?.full_name || user?.email?.split("@")[0] || "Reader";
  const avatarUrl = user?.avatar_url
    ? getMediaUrl(user.avatar_url, undefined, user.updated_at)
    : null;
  const hoursRead = Math.floor(stats.minutes / 60);
  const remainingMins = stats.minutes % 60;

  return (
    <dialog className="modal modal-open">
      <div className="modal-box max-w-5xl p-4 sm:p-6 bg-base-100 border border-base-300">
        <div className="flex items-center justify-between pb-3 border-b border-base-200">
          <div className="flex items-center gap-2">
            <div className="p-2 bg-primary/10 text-primary rounded-lg">
              <Sparkles className="h-5 w-5" />
            </div>
            <div>
              <h3 className="font-bold text-lg">
                {t("analytics.card_title", "Reading Wrapped")}
              </h3>
              <p className="text-xs text-base-content/60">
                {t(
                  "analytics.card_subtitle",
                  "Share your reading achievements",
                )}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="btn btn-ghost btn-sm btn-square"
            aria-label={t("common.close", "Close")}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="mt-4 grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
          {/* Controls Column */}
          <div className="lg:col-span-5 flex flex-col gap-4">
            {/* Period selector */}
            <div>
              <label className="text-xs font-bold uppercase tracking-wider text-base-content/60 mb-2 block">
                {t("analytics.card_period", "Time Range")}
              </label>
              <div className="join w-full grid grid-cols-3">
                <button
                  type="button"
                  onClick={() => setPeriod("month")}
                  className={`btn btn-sm join-item ${period === "month" ? "btn-primary" : "btn-outline"}`}
                >
                  {t("analytics.card_period_month", "This Month")}
                </button>
                <button
                  type="button"
                  onClick={() => setPeriod("year")}
                  className={`btn btn-sm join-item ${period === "year" ? "btn-primary" : "btn-outline"}`}
                >
                  {t("analytics.card_period_year", "This Year")}
                </button>
                <button
                  type="button"
                  onClick={() => setPeriod("all")}
                  className={`btn btn-sm join-item ${period === "all" ? "btn-primary" : "btn-outline"}`}
                >
                  {t("analytics.card_period_all", "All Time")}
                </button>
              </div>
            </div>

            {/* Ratio selector */}
            <div>
              <label className="text-xs font-bold uppercase tracking-wider text-base-content/60 mb-2 block">
                {t("analytics.card_aspect_ratio", "Format")}
              </label>
              <div className="join w-full grid grid-cols-3">
                <button
                  type="button"
                  onClick={() => setRatio("story")}
                  className={`btn btn-sm join-item ${ratio === "story" ? "btn-primary" : "btn-outline"}`}
                >
                  {t("analytics.card_ratio_story", "Story (9:16)")}
                </button>
                <button
                  type="button"
                  onClick={() => setRatio("square")}
                  className={`btn btn-sm join-item ${ratio === "square" ? "btn-primary" : "btn-outline"}`}
                >
                  {t("analytics.card_ratio_square", "Square (1:1)")}
                </button>
                <button
                  type="button"
                  onClick={() => setRatio("wide")}
                  className={`btn btn-sm join-item ${ratio === "wide" ? "btn-primary" : "btn-outline"}`}
                >
                  {t("analytics.card_ratio_wide", "Wide (16:9)")}
                </button>
              </div>
            </div>

            {/* Theme selector */}
            <div>
              <label className="text-xs font-bold uppercase tracking-wider text-base-content/60 mb-2 block">
                {t("analytics.card_theme", "Visual Theme")}
              </label>
              <div className="grid grid-cols-2 gap-2">
                {(["aurora", "cyberpunk", "sepia", "eink"] as CardTheme[]).map(
                  (thm) => (
                    <button
                      key={thm}
                      type="button"
                      onClick={() => setTheme(thm)}
                      className={`btn btn-sm justify-start gap-2 capitalize ${theme === thm ? "btn-primary" : "btn-outline"}`}
                    >
                      <div
                        className={`w-3 h-3 rounded-full border ${thm === "aurora" ? "bg-indigo-600" : thm === "cyberpunk" ? "bg-amber-400" : thm === "sepia" ? "bg-[#c5a880]" : "bg-black border-white"}`}
                      />
                      {t(`analytics.card_theme_${thm}`, thm)}
                    </button>
                  ),
                )}
              </div>
            </div>

            {/* Custom Quote / Message input */}
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="text-xs font-bold uppercase tracking-wider text-base-content/60 block">
                  {t(
                    "analytics.card_custom_quote_label",
                    "Custom Quote / Message",
                  )}
                </label>
                {customQuote !== defaultQuote && (
                  <button
                    type="button"
                    onClick={() => setCustomQuote(defaultQuote)}
                    className="text-[11px] text-primary hover:underline cursor-pointer flex items-center gap-1 font-medium"
                  >
                    <RotateCcw className="w-3 h-3" />
                    {t("common.reset", "Reset")}
                  </button>
                )}
              </div>
              <input
                type="text"
                value={customQuote}
                onChange={(e) => setCustomQuote(e.target.value)}
                placeholder={defaultQuote}
                maxLength={120}
                className="input input-sm input-bordered w-full text-xs rounded-xl"
              />
            </div>

            {/* Show NovelHub Logo Watermark Toggle */}
            <div className="flex items-center justify-between p-2.5 bg-base-200/50 rounded-xl border border-base-300">
              <div className="flex items-center gap-2">
                <Sparkles className="w-4 h-4 text-primary" />
                <span className="text-xs font-bold text-base-content/80">
                  {t("analytics.card_show_logo", "Show NovelHub Logo")}
                </span>
              </div>
              <input
                type="checkbox"
                checked={showLogo}
                onChange={(e) => setShowLogo(e.target.checked)}
                className="toggle toggle-primary toggle-sm"
              />
            </div>

            {/* Action buttons */}
            <div className="pt-2 flex flex-col sm:flex-row gap-2">
              <button
                type="button"
                onClick={handleDownload}
                disabled={isExporting}
                className="btn btn-primary flex-1 gap-2"
              >
                {isExporting ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Download className="h-4 w-4" />
                )}
                {t("analytics.card_download_png", "Download Image")}
              </button>
              <button
                type="button"
                onClick={handleCopy}
                disabled={isExporting}
                className="btn btn-outline gap-2"
              >
                {copied ? (
                  <Check className="h-4 w-4 text-success" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
                {t("analytics.card_copy_image", "Copy")}
              </button>
            </div>
          </div>

          {/* Card Preview Column */}
          <div className="lg:col-span-7 flex justify-center items-center overflow-auto p-4 bg-base-200/60 rounded-2xl border border-base-300 min-h-[440px]">
            <div
              ref={cardRef}
              className={`p-6 rounded-3xl flex flex-col justify-between transition-all duration-300 ${themeStyles.container}`}
              style={{
                width:
                  ratio === "story"
                    ? "380px"
                    : ratio === "wide"
                      ? "540px"
                      : "400px",
                minHeight:
                  ratio === "story"
                    ? "640px"
                    : ratio === "wide"
                      ? "340px"
                      : "400px",
              }}
            >
              {/* Header */}
              <div>
                <div className="flex items-center justify-between gap-3">
                  <div className="flex items-center gap-2.5">
                    {avatarUrl ? (
                      <img
                        src={avatarUrl}
                        alt={displayName}
                        crossOrigin="anonymous"
                        className="w-10 h-10 rounded-full object-cover border border-white/20"
                      />
                    ) : (
                      <div className="w-10 h-10 rounded-full bg-primary text-primary-content flex items-center justify-center font-black text-sm uppercase">
                        {displayName.slice(0, 2)}
                      </div>
                    )}
                    <div>
                      <div className="font-bold text-sm leading-tight truncate max-w-[160px]">
                        {displayName}
                      </div>
                      <div className={`text-[11px] ${themeStyles.subText}`}>
                        NovelHub Reader
                      </div>
                    </div>
                  </div>
                  <span
                    className={`text-[11px] px-2.5 py-1 rounded-full font-medium tracking-wide uppercase ${themeStyles.badge}`}
                  >
                    {stats.periodLabel}
                  </span>
                </div>

                {/* Wide Mode: 2-Column layout */}
                {ratio === "wide" ? (
                  <div className="grid grid-cols-2 gap-4 mt-4">
                    {/* Left Column */}
                    <div className="flex flex-col justify-between gap-3">
                      <div>
                        <div
                          className={`text-xs uppercase tracking-widest font-bold ${themeStyles.subText}`}
                        >
                          {t("analytics.words_read", "Words Read")}
                        </div>
                        <div
                          className={`text-3xl font-black tracking-tight mt-0.5 ${themeStyles.heroNumber}`}
                        >
                          {stats.words.toLocaleString()}
                        </div>
                        <div className="flex items-center gap-2 mt-1">
                          <span
                            className={`text-[10px] px-2 py-0.5 rounded-full font-bold flex items-center gap-1 ${themeStyles.levelBadge}`}
                          >
                            <span>{readerLevel.icon}</span>
                            <span>{readerLevel.title}</span>
                          </span>
                        </div>
                      </div>

                      <div className="grid grid-cols-3 gap-1.5">
                        <div
                          className={`p-2 rounded-xl flex flex-col items-center text-center ${themeStyles.statBox}`}
                        >
                          <Clock
                            className={`h-3.5 w-3.5 mb-0.5 ${themeStyles.accentText}`}
                          />
                          <div className="font-bold text-xs">
                            {hoursRead > 0
                              ? `${hoursRead}h ${remainingMins}m`
                              : `${stats.minutes}m`}
                          </div>
                          <div className={`text-[9px] ${themeStyles.subText}`}>
                            {t("analytics.card_hours_read", "Time Read")}
                          </div>
                        </div>

                        <div
                          className={`p-2 rounded-xl flex flex-col items-center text-center ${themeStyles.statBox}`}
                        >
                          <Flame
                            className={`h-3.5 w-3.5 mb-0.5 ${themeStyles.accentText}`}
                          />
                          <div className="font-bold text-xs">
                            {stats.streak} {t("analytics.days", "d")}
                          </div>
                          <div className={`text-[9px] ${themeStyles.subText}`}>
                            {t("analytics.card_streak", "Best Streak")}
                          </div>
                        </div>

                        <div
                          className={`p-2 rounded-xl flex flex-col items-center text-center ${themeStyles.statBox}`}
                        >
                          <Calendar
                            className={`h-3.5 w-3.5 mb-0.5 ${themeStyles.accentText}`}
                          />
                          <div className="font-bold text-xs">
                            {stats.activeDays} {t("analytics.days", "d")}
                          </div>
                          <div className={`text-[9px] ${themeStyles.subText}`}>
                            {t("analytics.card_active_days", "Active Days")}
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Right Column */}
                    <div className="flex flex-col justify-between gap-2.5">
                      {/* 7-Day Rhythm */}
                      <div
                        className={`p-2.5 rounded-xl ${themeStyles.statBox} flex flex-col gap-1`}
                      >
                        <div className="flex items-center justify-between">
                          <span
                            className={`text-[9px] uppercase font-bold tracking-wider ${themeStyles.subText} flex items-center gap-1`}
                          >
                            <TrendingUp className="w-3 h-3 text-primary" />
                            {t(
                              "analytics.card_reading_rhythm",
                              "7-Day Reading Rhythm",
                            )}
                          </span>
                          <span
                            className={`text-[9px] font-semibold ${themeStyles.accentText}`}
                          >
                            {last7Days
                              .reduce((acc, b) => acc + b.words, 0)
                              .toLocaleString()}{" "}
                            {t("analytics.words", "words")}
                          </span>
                        </div>
                        <div className="flex items-end justify-between gap-1 h-10 pt-1 px-1">
                          {last7Days.map((b, idx) => {
                            const heightPercent =
                              max7DaysWords > 0
                                ? Math.max(
                                    12,
                                    Math.round((b.words / max7DaysWords) * 100),
                                  )
                                : 12;
                            const isToday = idx === 6;
                            return (
                              <div
                                key={b.date}
                                className="flex-1 flex flex-col items-center gap-0.5 h-full justify-end"
                              >
                                <div
                                  className={`w-full rounded-xs transition-all ${
                                    b.words > 0
                                      ? isToday
                                        ? "bg-primary shadow-xs"
                                        : themeStyles.barActive
                                      : themeStyles.barInactive
                                  }`}
                                  style={{ height: `${heightPercent}%` }}
                                />
                                <span
                                  className={`text-[8px] font-medium leading-none ${isToday ? "text-primary font-bold" : themeStyles.subText}`}
                                >
                                  {b.day}
                                </span>
                              </div>
                            );
                          })}
                        </div>
                      </div>

                      {/* Highlights */}
                      {(topGenres.length > 0 ||
                        topAuthors.length > 0 ||
                        topFormats.length > 0) && (
                        <div className="flex flex-wrap gap-1">
                          {topGenres.slice(0, 2).map((g) => (
                            <span
                              key={g}
                              className={`text-[10px] px-2 py-0.5 rounded-md flex items-center gap-1 ${themeStyles.tagBadge}`}
                            >
                              <Bookmark className="h-2.5 w-2.5" /> {g}
                            </span>
                          ))}
                          {topAuthors.slice(0, 1).map((a) => (
                            <span
                              key={a}
                              className={`text-[10px] px-2 py-0.5 rounded-md flex items-center gap-1 ${themeStyles.tagBadge}`}
                            >
                              <BookOpen className="h-2.5 w-2.5" /> {a}
                            </span>
                          ))}
                          {topFormats.slice(0, 1).map((f) => (
                            <span
                              key={f}
                              className={`text-[10px] px-2 py-0.5 rounded-md flex items-center gap-1 ${themeStyles.tagBadge}`}
                            >
                              <Layers className="h-2.5 w-2.5" /> {f}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                ) : (
                  /* Vertical: Story (9:16) & Square (1:1) */
                  <>
                    {/* Hero Stat & Rank */}
                    <div className="mt-5">
                      <div
                        className={`text-xs uppercase tracking-widest font-bold ${themeStyles.subText}`}
                      >
                        {t("analytics.words_read", "Words Read")}
                      </div>
                      <div
                        className={`text-4xl sm:text-5xl font-black tracking-tight mt-0.5 ${themeStyles.heroNumber}`}
                      >
                        {stats.words.toLocaleString()}
                      </div>
                      <div className="flex items-center gap-2 mt-2">
                        <span
                          className={`text-[11px] px-2.5 py-0.5 rounded-full font-bold flex items-center gap-1.5 ${themeStyles.levelBadge}`}
                        >
                          <span>{readerLevel.icon}</span>
                          <span>{readerLevel.title}</span>
                        </span>
                        {avgWordsPerDay > 0 && (
                          <span
                            className={`text-[10px] ${themeStyles.subText}`}
                          >
                            ~{avgWordsPerDay.toLocaleString()}{" "}
                            {t("analytics.words_per_day", "words/day")}
                          </span>
                        )}
                      </div>
                    </div>

                    {/* KPI Grid */}
                    <div className="grid grid-cols-3 gap-2 mt-4">
                      <div
                        className={`p-2.5 rounded-xl flex flex-col items-center text-center ${themeStyles.statBox}`}
                      >
                        <Clock
                          className={`h-4 w-4 mb-1 ${themeStyles.accentText}`}
                        />
                        <div className="font-bold text-sm">
                          {hoursRead > 0
                            ? `${hoursRead}h ${remainingMins}m`
                            : `${stats.minutes}m`}
                        </div>
                        <div className={`text-[10px] ${themeStyles.subText}`}>
                          {t("analytics.card_hours_read", "Time Read")}
                        </div>
                      </div>

                      <div
                        className={`p-2.5 rounded-xl flex flex-col items-center text-center ${themeStyles.statBox}`}
                      >
                        <Flame
                          className={`h-4 w-4 mb-1 ${themeStyles.accentText}`}
                        />
                        <div className="font-bold text-sm">
                          {stats.streak} {t("analytics.days", "d")}
                        </div>
                        <div className={`text-[10px] ${themeStyles.subText}`}>
                          {t("analytics.card_streak", "Best Streak")}
                        </div>
                      </div>

                      <div
                        className={`p-2.5 rounded-xl flex flex-col items-center text-center ${themeStyles.statBox}`}
                      >
                        <Calendar
                          className={`h-4 w-4 mb-1 ${themeStyles.accentText}`}
                        />
                        <div className="font-bold text-sm">
                          {stats.activeDays} {t("analytics.days", "d")}
                        </div>
                        <div className={`text-[10px] ${themeStyles.subText}`}>
                          {t("analytics.card_active_days", "Active Days")}
                        </div>
                      </div>
                    </div>

                    {/* Habit Insights Row (Story Mode only or when peakDay exists) */}
                    {(ratio === "story" || peakDay) && (
                      <div
                        className={`mt-3 p-2.5 rounded-xl ${themeStyles.statBox} flex items-center justify-around text-center gap-2`}
                      >
                        <div>
                          <div
                            className={`text-[9px] uppercase font-bold tracking-wider ${themeStyles.subText}`}
                          >
                            {t("analytics.card_daily_pace", "Daily Pace")}
                          </div>
                          <div className="font-bold text-xs mt-0.5">
                            {avgWordsPerDay > 0
                              ? `~${avgWordsPerDay.toLocaleString()} ${t("analytics.words_per_day", "words/day")}`
                              : "--"}
                          </div>
                        </div>
                        <div className={`h-6 w-px ${themeStyles.borderCol}`} />
                        <div>
                          <div
                            className={`text-[9px] uppercase font-bold tracking-wider ${themeStyles.subText}`}
                          >
                            {t("analytics.card_peak_day", "Peak Day")}
                          </div>
                          <div className="font-bold text-xs mt-0.5">
                            {peakDay
                              ? `${peakDay.date} (${peakDay.words.toLocaleString()})`
                              : "--"}
                          </div>
                        </div>
                      </div>
                    )}

                    {/* 7-Day Reading Rhythm Bar Chart (Prominently fills Story 9:16) */}
                    {ratio === "story" && (
                      <div
                        className={`mt-3 p-3 rounded-2xl ${themeStyles.statBox} flex flex-col gap-1.5`}
                      >
                        <div className="flex items-center justify-between">
                          <span
                            className={`text-[10px] uppercase font-bold tracking-wider ${themeStyles.subText} flex items-center gap-1.5`}
                          >
                            <TrendingUp className="w-3.5 h-3.5 text-primary" />
                            {t(
                              "analytics.card_reading_rhythm",
                              "7-Day Reading Rhythm",
                            )}
                          </span>
                          <span
                            className={`text-[10px] font-semibold ${themeStyles.accentText}`}
                          >
                            {last7Days
                              .reduce((acc, b) => acc + b.words, 0)
                              .toLocaleString()}{" "}
                            {t("analytics.words", "words")}
                          </span>
                        </div>
                        <div className="flex items-end justify-between gap-1.5 h-12 pt-1 px-1">
                          {last7Days.map((b, idx) => {
                            const heightPercent =
                              max7DaysWords > 0
                                ? Math.max(
                                    12,
                                    Math.round((b.words / max7DaysWords) * 100),
                                  )
                                : 12;
                            const isToday = idx === 6;
                            return (
                              <div
                                key={b.date}
                                className="flex-1 flex flex-col items-center gap-1 h-full justify-end"
                              >
                                <div
                                  className={`w-full rounded-md transition-all ${
                                    b.words > 0
                                      ? isToday
                                        ? "bg-primary shadow-xs"
                                        : themeStyles.barActive
                                      : themeStyles.barInactive
                                  }`}
                                  style={{ height: `${heightPercent}%` }}
                                  title={`${b.date}: ${b.words.toLocaleString()}`}
                                />
                                <span
                                  className={`text-[9px] font-medium leading-none ${isToday ? "text-primary font-bold" : themeStyles.subText}`}
                                >
                                  {b.day}
                                </span>
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    )}

                    {/* Highlights */}
                    {(topGenres.length > 0 ||
                      topAuthors.length > 0 ||
                      topFormats.length > 0) && (
                      <div
                        className={`mt-3 pt-3 border-t ${themeStyles.borderCol} flex flex-col gap-2`}
                      >
                        <div
                          className={`text-[10px] uppercase font-bold tracking-wider ${themeStyles.subText}`}
                        >
                          {t("analytics.card_highlights", "Reading Highlights")}
                        </div>
                        <div className="flex flex-wrap gap-1.5">
                          {topGenres.map((g) => (
                            <span
                              key={g}
                              className={`text-[11px] px-2 py-0.5 rounded-lg flex items-center gap-1 ${themeStyles.tagBadge}`}
                            >
                              <Bookmark className="h-3 w-3" /> {g}
                            </span>
                          ))}
                          {topAuthors.map((a) => (
                            <span
                              key={a}
                              className={`text-[11px] px-2 py-0.5 rounded-lg flex items-center gap-1 ${themeStyles.tagBadge}`}
                            >
                              <BookOpen className="h-3 w-3" /> {a}
                            </span>
                          ))}
                          {topFormats.map((f) => (
                            <span
                              key={f}
                              className={`text-[11px] px-2 py-0.5 rounded-lg flex items-center gap-1 ${themeStyles.tagBadge}`}
                            >
                              <Layers className="h-3 w-3" /> {f}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}
                  </>
                )}
              </div>

              {/* Footer */}
              <div
                className={`mt-5 pt-3.5 border-t ${themeStyles.borderCol} flex items-center ${showLogo ? "justify-between" : "justify-center"} gap-2`}
              >
                {showLogo && (
                  <div className="flex items-center gap-2 shrink-0">
                    {siteLogo ? (
                      <img
                        src={siteLogo}
                        alt="Logo"
                        crossOrigin="anonymous"
                        className="w-5 h-5 rounded-md object-contain shadow-xs"
                      />
                    ) : (
                      <div className="w-5 h-5 rounded-md bg-gradient-to-br from-primary to-secondary text-primary-content flex items-center justify-center text-[9px] font-black shadow-xs">
                        NH
                      </div>
                    )}
                    <span className="font-black text-xs tracking-wider uppercase">
                      {siteTitle}
                    </span>
                  </div>
                )}
                {customQuote.trim() && (
                  <div
                    className={`text-[10px] italic ${themeStyles.subText} ${showLogo ? "text-right" : "text-center"} truncate max-w-[220px]`}
                  >
                    "{customQuote.trim()}"
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
      <form method="dialog" className="modal-backdrop" onClick={onClose}>
        <button type="button">{t("common.close", "Close")}</button>
      </form>
    </dialog>
  );
};
