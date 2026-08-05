import React, { useMemo } from 'react';
import { ActivityCalendar } from 'react-activity-calendar';
import { useReadingHeatmapQuery } from '@/hooks/useReadingStats';
import { useTranslation } from 'react-i18next';
import { useSettingsStore } from '@/stores/settingsStore';

type ReadingHeatmapProps = {
  className?: string;
  showTitle?: boolean;
  compact?: boolean;
};

export const ReadingHeatmap: React.FC<ReadingHeatmapProps> = ({
  className = "rounded-xl border border-base-200 bg-base-100 p-3.5 sm:p-4 shadow-xs",
  showTitle = false,
  compact = false,
}) => {
  const { data: heatmapData, isLoading } = useReadingHeatmapQuery();
  const { t } = useTranslation();
  const theme = useSettingsStore((state) => state.theme);

  const colorScheme = useMemo(() => {
    if (theme === "night" || theme === "coffee") return "dark";
    if (theme === "winter" || theme === "cupcake") return "light";
    const dataTheme = typeof document !== 'undefined' ? document.documentElement.getAttribute("data-theme") : "";
    if (dataTheme === "night" || dataTheme === "coffee" || dataTheme === "dark") return "dark";
    if (typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches && (theme === 'system' || !dataTheme)) {
      return "dark";
    }
    return "light";
  }, [theme]);

  const formattedData = useMemo(() => {
    const dataMap: Record<string, { duration?: number; words?: number }> =
      heatmapData && typeof heatmapData === "object" && !Array.isArray(heatmapData)
        ? heatmapData
        : {};

    const year = new Date().getFullYear();
    const startDate = new Date(year, 0, 1);
    const endDate = new Date(year, 11, 31);

    const result = [];
    const curr = new Date(startDate);

    while (curr <= endDate) {
      const yearStr = curr.getFullYear();
      const monthStr = String(curr.getMonth() + 1).padStart(2, '0');
      const dayStr = String(curr.getDate()).padStart(2, '0');
      const dateStr = `${yearStr}-${monthStr}-${dayStr}`;

      const stats = dataMap[dateStr] || {};
      const words = stats.words || 0;
      let level = 0;
      if (words > 0) level = 1;
      if (words > 1000) level = 2;
      if (words > 5000) level = 3;
      if (words > 10000) level = 4;

      result.push({
        date: dateStr,
        count: words,
        level,
      });

      curr.setDate(curr.getDate() + 1);
    }

    return result;
  }, [heatmapData]);

  const calendarLabels = useMemo(() => ({
    months: [
      t("common.months.jan", "Jan"),
      t("common.months.feb", "Feb"),
      t("common.months.mar", "Mar"),
      t("common.months.apr", "Apr"),
      t("common.months.may", "May"),
      t("common.months.jun", "Jun"),
      t("common.months.jul", "Jul"),
      t("common.months.aug", "Aug"),
      t("common.months.sep", "Sep"),
      t("common.months.oct", "Oct"),
      t("common.months.nov", "Nov"),
      t("common.months.dec", "Dec"),
    ],
    weekdays: [
      t("common.weekdays.sun", "Sun"),
      t("common.weekdays.mon", "Mon"),
      t("common.weekdays.tue", "Tue"),
      t("common.weekdays.wed", "Wed"),
      t("common.weekdays.thu", "Thu"),
      t("common.weekdays.fri", "Fri"),
      t("common.weekdays.sat", "Sat"),
    ],
    totalCount: t("analytics.activities_in_year", "{{count}} activities in {{year}}"),
    legend: {
      less: t("analytics.less", "Less"),
      more: t("analytics.more", "More"),
    },
  }), [t]);

  if (isLoading) {
    return (
      <div className={`${className} flex items-center justify-center p-6`}>
        <span className="loading loading-spinner loading-sm text-primary"></span>
      </div>
    );
  }

  return (
    <div className={className}>
      {showTitle && (
        <h2 className="text-sm sm:text-base font-bold mb-3 text-base-content/90">
          {t("analytics.reading_activity_words", "Reading Activity (Words)")}
        </h2>
      )}
      <div className="w-full overflow-x-auto">
        <ActivityCalendar
          data={formattedData}
          labels={calendarLabels}
          colorScheme={colorScheme}
          blockSize={compact ? 10 : 11}
          blockMargin={compact ? 3 : 3}
          fontSize={11}
          theme={{
            light: ['#ebedf0', '#9be9a8', '#40c463', '#30a14e', '#216e39'],
            dark: ['#22272e', '#0e4429', '#006d32', '#26a641', '#39d353'],
          }}
        />
      </div>
    </div>
  );
};
