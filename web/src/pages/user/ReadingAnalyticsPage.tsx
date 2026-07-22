import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { Clock, BookOpen, Flame, Target, ArrowLeft } from 'lucide-react';
import { TopNav } from '@/components/common/TopNav';
import { ReadingHeatmap } from '@/components/profile/ReadingHeatmap';
import { useReadingHeatmapQuery } from '@/hooks/useReadingStats';

export const ReadingAnalyticsPage: React.FC = () => {
  const { t } = useTranslation();
  const { data: heatmapData } = useReadingHeatmapQuery();

  const { totalWords, activeDays } = useMemo(() => {
    let words = 0;
    let days = 0;
    if (heatmapData && typeof heatmapData === 'object' && !Array.isArray(heatmapData)) {
      Object.values(heatmapData).forEach((item: any) => {
        if (item && item.words > 0) {
          words += item.words;
          days += 1;
        }
      });
    }
    return { totalWords: words, activeDays: days };
  }, [heatmapData]);

  return (
    <div className="bg-base-200 min-h-screen flex flex-col">
      <TopNav showSidebarToggle={false} />

      <div className="flex-1 container mx-auto p-4 sm:p-6 max-w-5xl flex flex-col gap-6">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl sm:text-3xl font-black">{t('analytics.title', 'Reading Analytics & Stats')}</h1>
            <p className="text-xs sm:text-sm text-base-content/60">{t('analytics.subtitle', 'Track your reading habits, streaks, and activity over time.')}</p>
          </div>
          <Link to="/" className="btn btn-ghost btn-sm gap-1 text-primary">
            <ArrowLeft className="h-4 w-4" />
            {t('library.back_to_library', 'Back to Library')}
          </Link>
        </div>

        {/* KPI Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-base-100 border border-base-200 p-4 rounded-2xl shadow-sm flex items-center gap-3">
            <div className="p-3 bg-primary/10 text-primary rounded-xl">
              <BookOpen className="h-6 w-6" />
            </div>
            <div>
              <div className="text-2xl font-black">{totalWords.toLocaleString()}</div>
              <div className="text-xs text-base-content/60">{t('analytics.words_read', 'Words Read')}</div>
            </div>
          </div>

          <div className="bg-base-100 border border-base-200 p-4 rounded-2xl shadow-sm flex items-center gap-3">
            <div className="p-3 bg-secondary/10 text-secondary rounded-xl">
              <Flame className="h-6 w-6" />
            </div>
            <div>
              <div className="text-2xl font-black">{activeDays} {t('analytics.days', 'days')}</div>
              <div className="text-xs text-base-content/60">{t('analytics.active_days', 'Active Days')}</div>
            </div>
          </div>

          <div className="bg-base-100 border border-base-200 p-4 rounded-2xl shadow-sm flex items-center gap-3">
            <div className="p-3 bg-accent/10 text-accent rounded-xl">
              <Clock className="h-6 w-6" />
            </div>
            <div>
              <div className="text-2xl font-black">~{Math.round(totalWords / 250)} {t('analytics.mins', 'mins')}</div>
              <div className="text-xs text-base-content/60">{t('analytics.est_time', 'Estimated Reading Time')}</div>
            </div>
          </div>

          <div className="bg-base-100 border border-base-200 p-4 rounded-2xl shadow-sm flex items-center gap-3">
            <div className="p-3 bg-success/10 text-success rounded-xl">
              <Target className="h-6 w-6" />
            </div>
            <div>
              <div className="text-2xl font-black">85%</div>
              <div className="text-xs text-base-content/60">{t('analytics.goal_progress', 'Daily Goal Progress')}</div>
            </div>
          </div>
        </div>

        {/* Activity Heatmap */}
        <div className="bg-base-100 border border-base-200 p-6 rounded-2xl shadow-sm">
          <h2 className="text-lg font-bold mb-4">{t('analytics.activity_grid', 'Annual Reading Activity')}</h2>
          <ReadingHeatmap />
        </div>
      </div>
    </div>
  );
};
