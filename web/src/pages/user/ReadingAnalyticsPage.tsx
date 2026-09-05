import React, { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import {
  Clock,
  BookOpen,
  Flame,
  ArrowLeft,
  Headphones,
  Sparkles,
} from "lucide-react";
import { TopNav } from "@/components/common/TopNav";
import { ReadingHeatmap } from "@/components/profile/ReadingHeatmap";
import { ReadingGoalCard } from "@/components/profile/ReadingGoalCard";
import { ReadingCardModal } from "@/components/profile/ReadingCardModal";
import {
  useReadingHeatmapQuery,
  useReadingStatsSummaryQuery,
  useLibraryBreakdownQuery,
} from "@/hooks/useReadingStats";
import type { NameCount } from "@/types";

function TopList({ items, label }: { items?: NameCount[]; label: string }) {
  const { t } = useTranslation();
  if (!items || items.length === 0) {
    return (
      <div className="text-xs text-base-content/50 py-4 text-center">
        {t("analytics.no_data", "No data yet")}
      </div>
    );
  }
  const max = items[0]?.count || 1;
  return (
    <div>
      <h3 className="text-xs font-bold uppercase tracking-wider text-base-content/50 mb-2">
        {label}
      </h3>
      <div className="flex flex-col gap-1.5">
        {items.slice(0, 6).map((item) => (
          <div key={item.name} className="flex items-center gap-2 text-sm">
            <span className="w-32 truncate shrink-0 text-base-content/80">
              {item.name}
            </span>
            <div className="flex-1 h-2 bg-base-200 rounded-full overflow-hidden">
              <div
                className="h-full bg-primary rounded-full"
                style={{
                  width: `${Math.max(4, Math.round((item.count / max) * 100))}%`,
                }}
              />
            </div>
            <span className="text-xs text-base-content/50 w-8 text-right shrink-0">
              {item.count}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

export const ReadingAnalyticsPage: React.FC = () => {
  const { t } = useTranslation();
  const { data: heatmapData } = useReadingHeatmapQuery();
  const { data: summary } = useReadingStatsSummaryQuery();
  const { data: breakdown } = useLibraryBreakdownQuery();
  const [isCardModalOpen, setIsCardModalOpen] = useState(false);

  const { totalWords, activeDays, todayWords, weekBars } = useMemo(() => {
    let words = 0;
    let days = 0;
    let today = 0;
    const bars: { label: string; words: number }[] = [];
    if (
      heatmapData &&
      typeof heatmapData === "object" &&
      !Array.isArray(heatmapData)
    ) {
      const todayKey = new Date().toLocaleDateString("sv-SE");
      for (let i = 6; i >= 0; i--) {
        const date = new Date();
        date.setDate(date.getDate() - i);
        const key = date.toLocaleDateString("sv-SE");
        const wordsOnDay = heatmapData[key]?.words ?? 0;
        bars.push({ label: key.slice(5), words: wordsOnDay });
        if (i === 0) today = wordsOnDay;
      }
      Object.entries(heatmapData).forEach(([date, item]: [string, any]) => {
        if (item && item.words > 0) {
          words += item.words;
          days += 1;
        }
      });
    }
    return {
      totalWords: words,
      activeDays: days,
      todayWords: today,
      weekBars: bars,
    };
  }, [heatmapData]);

  const weekMax = Math.max(1, ...weekBars.map((b) => b.words));

  return (
    <div className="bg-base-200 min-h-screen flex flex-col">
      <TopNav showSidebarToggle={false} />

      <div className="flex-1 container mx-auto p-4 sm:p-6 max-w-5xl flex flex-col gap-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl sm:text-3xl font-black">
              {t("analytics.title", "Reading Analytics & Stats")}
            </h1>
            <p className="text-xs sm:text-sm text-base-content/60">
              {t(
                "analytics.subtitle",
                "Track your reading habits, streaks, and activity over time.",
              )}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setIsCardModalOpen(true)}
              className="btn btn-primary btn-sm gap-1.5 shadow-sm"
            >
              <Sparkles className="h-4 w-4" />
              {t("analytics.export_card", "Export Card")}
            </button>
            <Link to="/" className="btn btn-ghost btn-sm gap-1 text-primary">
              <ArrowLeft className="h-4 w-4" />
              {t("library.back_to_library", "Back to Library")}
            </Link>
          </div>
        </div>

        {/* KPI Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-base-100 border border-base-200 p-4 rounded-2xl shadow-sm flex items-center gap-3">
            <div className="p-3 bg-primary/10 text-primary rounded-xl">
              <BookOpen className="h-6 w-6" />
            </div>
            <div>
              <div className="text-2xl font-black">
                {totalWords.toLocaleString()}
              </div>
              <div className="text-xs text-base-content/60">
                {t("analytics.words_read", "Words Read")}
              </div>
            </div>
          </div>

          <div className="bg-base-100 border border-base-200 p-4 rounded-2xl shadow-sm flex items-center gap-3">
            <div className="p-3 bg-secondary/10 text-secondary rounded-xl">
              <Flame className="h-6 w-6" />
            </div>
            <div>
              <div className="text-2xl font-black">
                {summary?.current_streak_days ?? 0}{" "}
                {t("analytics.days", "days")}
              </div>
              <div className="text-xs text-base-content/60">
                {t("analytics.current_streak", "Day Streak")} ·{" "}
                {t("analytics.longest_streak", "Best: {{count}} days", {
                  count: summary?.longest_streak_days ?? 0,
                })}
              </div>
            </div>
          </div>

          <div className="bg-base-100 border border-base-200 p-4 rounded-2xl shadow-sm flex items-center gap-3">
            <div className="p-3 bg-accent/10 text-accent rounded-xl">
              <Clock className="h-6 w-6" />
            </div>
            <div>
              <div className="text-2xl font-black">
                {summary
                  ? summary.total_minutes.toLocaleString()
                  : `~${Math.round(totalWords / 250)}`}
              </div>
              <div className="text-xs text-base-content/60">
                {t("analytics.total_minutes", "Minutes Read")}
              </div>
            </div>
          </div>

          <ReadingGoalCard todayWords={todayWords} />
        </div>

        {/* This Week strip */}
        <div className="bg-base-100 border border-base-200 p-6 rounded-2xl shadow-sm">
          <h2 className="text-lg font-bold mb-4">
            {t("analytics.this_week", "This Week")}
          </h2>
          <div className="flex items-end gap-2 h-24">
            {weekBars.map((bar, i) => (
              <div
                key={bar.label}
                className="flex-1 flex flex-col items-center gap-1"
              >
                <div
                  className="w-full rounded-t-lg bg-primary/20"
                  style={{
                    height: `${Math.max(3, Math.round((bar.words / weekMax) * 100))}%`,
                    minHeight: 4,
                  }}
                />
                <span className="text-[10px] text-base-content/50">
                  {bar.label}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Activity Heatmap */}
        <div className="bg-base-100 border border-base-200 p-6 rounded-2xl shadow-sm">
          <h2 className="text-lg font-bold mb-4">
            {t("analytics.activity_grid", "Annual Reading Activity")}
          </h2>
          <ReadingHeatmap showTitle={false} />
        </div>

        {/* Library Breakdown */}
        <div className="bg-base-100 border border-base-200 p-6 rounded-2xl shadow-sm">
          <h2 className="text-lg font-bold mb-4">
            {t("analytics.library_breakdown", "Library Breakdown")}
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <TopList
              items={breakdown?.formats}
              label={t("analytics.top_formats", "Top Formats")}
            />
            <TopList
              items={breakdown?.authors}
              label={t("analytics.top_authors", "Top Authors")}
            />
            <TopList
              items={breakdown?.tags}
              label={t("analytics.top_tags", "Top Genres")}
            />
            <TopList
              items={breakdown?.languages}
              label={t("analytics.top_languages", "Top Languages")}
            />
          </div>
          {breakdown && breakdown.avg_speed_wpm > 0 && (
            <p className="mt-3 text-xs text-base-content/60">
              {t("analytics.avg_speed", "Avg listening speed: {{wpm}} wpm", {
                wpm: Math.round(breakdown.avg_speed_wpm),
              })}
            </p>
          )}
          {breakdown?.listening && breakdown.listening.length > 0 && (
            <div className="mt-6">
              <div className="flex items-center gap-2 mb-2">
                <Headphones className="h-4 w-4 text-primary" />
                <h3 className="text-xs font-bold uppercase tracking-wider text-base-content/50">
                  {t("analytics.listening_history", "Listening by Month")}
                </h3>
              </div>
              <div className="flex items-end gap-2 h-24">
                {breakdown.listening.slice(-12).map((m) => (
                  <div
                    key={m.month}
                    className="flex-1 flex flex-col items-center gap-1"
                  >
                    <div
                      className="w-full rounded-t-lg bg-accent/30"
                      style={{
                        height: `${Math.max(4, Math.round((m.hours / Math.max(1, ...breakdown.listening.map((l) => l.hours))) * 100))}%`,
                        minHeight: 4,
                      }}
                    />
                    <span className="text-[10px] text-base-content/50">
                      {m.month.slice(5)}-{m.month.slice(2, 4)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
          {breakdown &&
            breakdown.formats.length === 0 &&
            breakdown.tags.length === 0 && (
              <div className="text-xs text-base-content/50 py-4 text-center">
                {t("analytics.no_data", "No data yet")}
              </div>
            )}
        </div>
      </div>

      <ReadingCardModal
        isOpen={isCardModalOpen}
        onClose={() => setIsCardModalOpen(false)}
        heatmapData={heatmapData}
        summary={summary}
        breakdown={breakdown}
      />
    </div>
  );
};
