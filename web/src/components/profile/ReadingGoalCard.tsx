import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Loader2, Pencil, Target } from 'lucide-react';
import { useReadingGoalQuery, useUpsertReadingGoalMutation } from '@/hooks/useReadingStats';

export const ReadingGoalCard: React.FC<{ todayWords: number }> = ({ todayWords }) => {
  const { t } = useTranslation();
  const { data: goal } = useReadingGoalQuery();
  const mutation = useUpsertReadingGoalMutation();
  const [editing, setEditing] = useState(false);
  const [wordsPerDay, setWordsPerDay] = useState(0);
  const [booksPerYear, setBooksPerYear] = useState(0);

  const target = goal?.target_words_per_day ?? 0;
  const percent = target > 0 ? Math.min(100, Math.round((todayWords / target) * 100)) : 0;

  function openEditor() {
    setWordsPerDay(goal?.target_words_per_day ?? 1000);
    setBooksPerYear(goal?.target_books_per_year ?? 12);
    setEditing(true);
  }

  function save() {
    mutation.mutate({ wordsPerDay, booksPerYear }, { onSuccess: () => setEditing(false) });
  }

  return (
    <>
      <div className="bg-base-100 border border-base-200 p-4 rounded-2xl shadow-sm flex items-center gap-3">
        <div className="p-3 bg-success/10 text-success rounded-xl">
          <Target className="h-6 w-6" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="text-2xl font-black">{percent}%</div>
          <div className="text-xs text-base-content/60 truncate">{t('analytics.goal_progress', 'Daily Goal Progress')}</div>
          {target > 0 && (
            <div className="text-xs text-base-content/40">
              {t('analytics.goal_today', '{{read}} / {{target}} words', { read: todayWords.toLocaleString(), target: target.toLocaleString() })}
            </div>
          )}
        </div>
        <button type="button" onClick={openEditor} className="btn btn-ghost btn-xs btn-square" aria-label={t('analytics.goal_edit', 'Edit reading goal')}>
          <Pencil className="h-4 w-4" />
        </button>
      </div>

      {editing && (
        <dialog className="modal modal-open">
          <div className="modal-box max-w-sm">
            <h3 className="font-bold text-lg mb-4">{t('analytics.goal_edit', 'Edit reading goal')}</h3>
            <label className="flex flex-col gap-1.5 mb-3">
              <span className="text-xs font-bold uppercase tracking-wider opacity-60">{t('analytics.goal_words_per_day', 'Words per day')}</span>
              <input
                type="number"
                min={1}
                max={1000000}
                className="input input-bordered w-full"
                value={wordsPerDay}
                onChange={(event) => setWordsPerDay(event.currentTarget.valueAsNumber || 0)}
              />
            </label>
            <label className="flex flex-col gap-1.5">
              <span className="text-xs font-bold uppercase tracking-wider opacity-60">{t('analytics.goal_books_per_year', 'Books per year')}</span>
              <input
                type="number"
                min={1}
                max={10000}
                className="input input-bordered w-full"
                value={booksPerYear}
                onChange={(event) => setBooksPerYear(event.currentTarget.valueAsNumber || 0)}
              />
            </label>
            <div className="modal-action">
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setEditing(false)}>{t('common.cancel', 'Cancel')}</button>
              <button
                type="button"
                className="btn btn-primary btn-sm gap-1"
                disabled={mutation.isPending || wordsPerDay < 1 || booksPerYear < 1}
                onClick={save}
              >
                {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {t('common.save', 'Save')}
              </button>
            </div>
          </div>
          <form method="dialog" className="modal-backdrop" onClick={() => setEditing(false)}>
            <button type="button">{t('common.close', 'Close')}</button>
          </form>
        </dialog>
      )}
    </>
  );
};
