import React, { useMemo } from 'react';
import { ActivityCalendar } from 'react-activity-calendar';
import { useReadingHeatmapQuery } from '@/hooks/useReadingStats';
import { useTranslation } from 'react-i18next';

export const ReadingHeatmap: React.FC = () => {
  const { data: heatmapData, isLoading } = useReadingHeatmapQuery();
  const { t } = useTranslation();

  const formattedData = useMemo(() => {
    if (!heatmapData || typeof heatmapData !== "object" || Array.isArray(heatmapData)) return [];
    const isoDateRegex = /^\d{4}-\d{2}-\d{2}$/;
    const result = Object.keys(heatmapData)
      .filter((dateStr) => isoDateRegex.test(dateStr))
      .map((dateStr) => {
        const stats = heatmapData[dateStr] || {};
        const words = stats.words || 0;
        let level = 0;
        if (words > 0) level = 1;
        if (words > 1000) level = 2;
        if (words > 5000) level = 3;
        if (words > 10000) level = 4;
        return {
          date: dateStr,
          count: words,
          level,
        };
      });

    if (result.length === 0) {
      const today = new Date().toLocaleDateString('sv-SE');
      result.push({ date: today, count: 0, level: 0 });
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

  if (isLoading || formattedData.length === 0) return null;

  return (
    <div className="bg-base-200 p-4 rounded-xl">
      <h2 className="text-lg font-bold mb-4">{t("analytics.reading_activity_words", "Reading Activity (Words)")}</h2>
      <ActivityCalendar
        data={formattedData}
        labels={calendarLabels}
        theme={{
          light: ['#ebedf0', '#9be9a8', '#40c463', '#30a14e', '#216e39'],
          dark: ['#161b22', '#0e4429', '#006d32', '#26a641', '#39d353'],
        }}
      />
    </div>
  );
};
