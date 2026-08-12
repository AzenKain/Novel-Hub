import { BookOpen, ListMusic, Plus, Search, Trash2 } from "lucide-react";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

import {
  useAudiobookChaptersQuery,
  useDeleteAudiobookChapterMutation,
  useLookupAudiobookChaptersMutation,
  useUpsertAudiobookChapterMutation,
} from "@/hooks";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";
import type { AudiobookChapter } from "@/types";

type AudiobookChaptersCardProps = {
  book_id: string;
};

type DraftRow = {
  id?: string;
  chapter_index: number;
  title: string;
  start_sec: number;
  end_sec: number | null;
};

const fmtSec = (sec: number | null | undefined) => (sec == null ? "" : String(sec));

const parseSec = (raw: string, fallback: number | null): number | null => {
  const v = parseFloat(raw);
  return raw.trim() === "" ? fallback : Number.isFinite(v) ? v : fallback;
};

export const AudiobookChaptersCard: React.FC<AudiobookChaptersCardProps> = ({ book_id }) => {
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.user);
  const { data: chapters, isLoading } = useAudiobookChaptersQuery(book_id);
  const upsert = useUpsertAudiobookChapterMutation(book_id);
  const remove = useDeleteAudiobookChapterMutation(book_id);
  const lookup = useLookupAudiobookChaptersMutation(book_id);
  const [asin, setAsin] = useState("");
  const [drafts, setDrafts] = useState<Record<string, DraftRow>>({});

  const canEdit = hasPermission(user, "book.edit");
  if (!hasPermission(user, "book.read")) return null;

  useEffect(() => {
    if (!chapters) return;
    const next: Record<string, DraftRow> = {};
    for (const ch of chapters) {
      next[ch.id] = {
        id: ch.id,
        chapter_index: ch.chapter_index,
        title: ch.title,
        start_sec: ch.start_sec,
        end_sec: ch.end_sec ?? null,
      };
    }
    setDrafts(next);
  }, [chapters]);

  const setDraft = (id: string, patch: Partial<DraftRow>) => {
    setDrafts((prev) => ({ ...prev, [id]: { ...prev[id], ...patch } }));
  };

  const persist = (id: string) => {
    const d = drafts[id];
    if (!d) return;
    if (!d.id && !d.title.trim()) return; // abandon blank new rows
    const end = d.end_sec == null ? undefined : d.end_sec;
    upsert.mutate(
      { id: d.id, chapter: { chapter_index: d.chapter_index, title: d.title || "—", start_sec: d.start_sec, end_sec: end } },
      {
        onError: (err) => toast.error(err.message || t("audiobook.chapter_save_failed", "Failed to save chapter")),
      }
    );
  };

  const handleAdd = () => {
    const maxIndex = chapters?.reduce((m, c) => Math.max(m, c.chapter_index), -1) ?? -1;
    const key = `new-${maxIndex + 1}`;
    setDrafts((prev) => ({
      ...prev,
      [key]: { chapter_index: maxIndex + 1, title: "", start_sec: 0, end_sec: null },
    }));
  };

  const handleDelete = (ch: AudiobookChapter) => {
    remove.mutate(ch.id, { onError: (err) => toast.error(err.message || t("audiobook.chapter_delete_failed", "Failed to delete chapter")) });
  };

  const handleLookup = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = asin.trim();
    if (!trimmed) {
      toast.error(t("audiobook.enter_asin", "Enter an ASIN"));
      return;
    }
    lookup.mutate(
      { asin: trimmed },
      {
        onSuccess: () => {
          toast.success(t("audiobook.lookup_success", "Chapters imported from Audnexus"));
          setAsin("");
        },
        onError: (err) => toast.error(err.message || t("audiobook.lookup_failed", "Audnexus lookup failed")),
      }
    );
  };

  const rows = Object.entries(drafts).sort((a, b) => a[1].chapter_index - b[1].chapter_index);

  return (
    <div>
      <h3 className="text-lg font-semibold flex items-center gap-2 mb-3">
        <ListMusic className="w-5 h-5 text-primary" />
        {t("audiobook.chapters", "Audiobook chapters")}
        <span className="badge badge-ghost badge-sm">{isLoading ? "…" : rows.length}</span>
      </h3>

      {canEdit && (
        <form onSubmit={handleLookup} className="flex items-center gap-2 mb-3">
          <input
            type="text"
            value={asin}
            onChange={(e) => setAsin(e.target.value)}
            placeholder={t("audiobook.asin_placeholder", "ASIN (e.g. B002V0F37U)")}
            className="input input-sm input-bordered flex-1"
          />
          <button className="btn btn-sm btn-ghost" type="submit" disabled={lookup.isPending}>
            <Search className="w-4 h-4" />
            {t("audiobook.lookup_audnexus", "Lookup (Audnexus)")}
          </button>
        </form>
      )}

      {rows.length === 0 && !isLoading ? (
        <p className="text-sm opacity-60 flex items-center gap-2">
          <BookOpen className="w-4 h-4" />
          {t("audiobook.no_chapters", "No chapters yet")}
        </p>
      ) : (
        <div className="space-y-2">
          {rows.map(([key, row]) => (
            <div key={key} className="flex items-center gap-2 text-sm">
              <span className="badge badge-outline badge-sm w-10 justify-center shrink-0">{row.chapter_index + 1}</span>
              <input
                type="text"
                value={row.title}
                onChange={(e) => setDraft(key, { title: e.target.value })}
                onBlur={() => persist(key)}
                disabled={!canEdit}
                className="input input-sm input-bordered flex-1 min-w-0"
              />
              <input
                type="number"
                step="0.1"
                value={fmtSec(row.start_sec)}
                onChange={(e) => setDraft(key, { start_sec: parseSec(e.target.value, 0) ?? 0 })}
                onBlur={() => persist(key)}
                disabled={!canEdit}
                className="input input-sm input-bordered w-24 shrink-0 text-right"
                title={t("audiobook.start_sec", "Start (s)")}
              />
              <input
                type="number"
                step="0.1"
                value={fmtSec(row.end_sec)}
                onChange={(e) => setDraft(key, { end_sec: parseSec(e.target.value, null) })}
                onBlur={() => persist(key)}
                disabled={!canEdit}
                className="input input-sm input-bordered w-24 shrink-0 text-right"
                title={t("audiobook.end_sec", "End (s)")}
              />
              {canEdit && row.id && (
                <button
                  className="btn btn-square btn-ghost btn-sm shrink-0"
                  onClick={() => {
                    const ch = chapters?.find((c) => c.id === row.id);
                    if (ch) handleDelete(ch);
                  }}
                  aria-label={t("common.delete")}
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      {canEdit && (
        <button className="btn btn-sm btn-ghost mt-3" onClick={handleAdd}>
          <Plus className="w-4 h-4" />
          {t("audiobook.add_chapter", "Add chapter")}
        </button>
      )}
    </div>
  );
};