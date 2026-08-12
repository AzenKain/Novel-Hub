import { AudioLines, X } from "lucide-react";
import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

import { useMergeAudiobookMutation } from "@/hooks";
import type { BookFile } from "@/types";

const MERGEABLE_FORMATS = new Set(["m4a", "m4b", "mp3", "flac", "ogg", "wav", "aac"]);

type MergeAudiobookModalProps = {
  open: boolean;
  book_id: string;
  title: string;
  files: BookFile[];
  onClose: () => void;
};

const formatBytes = (n: number) => {
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
};

export const MergeAudiobookModal: React.FC<MergeAudiobookModalProps> = ({ open, book_id, title, files, onClose }) => {
  const { t } = useTranslation();
  const merge = useMergeAudiobookMutation(book_id);
  const [mergedTitle, setMergedTitle] = useState(title);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const mergeable = files.filter((f) => MERGEABLE_FORMATS.has(f.format.toLowerCase()));

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const fileIds = Array.from(selected);
    if (fileIds.length < 2) {
      toast.error(t("audiobook.merge_need_two", "Select at least 2 audio files"));
      return;
    }
    merge.mutate(
      { title: mergedTitle.trim() || title, file_ids: fileIds },
      {
        onSuccess: () => {
          toast.success(t("audiobook.merge_queued", "Merge job started"));
          onClose();
        },
        onError: (err) => toast.error(err.message || t("audiobook.merge_failed", "Failed to start merge")),
      }
    );
  };

  if (!open) return null;

  return (
    <dialog className="modal modal-open">
      <div className="modal-box max-w-lg">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <AudioLines className="w-5 h-5 text-primary" />
            {t("audiobook.merge_into", "Merge into audiobook")}
          </h3>
          <button className="btn btn-square btn-sm btn-ghost" onClick={onClose} aria-label={t("common.close")}>
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="label label-text">{t("audiobook.merged_title", "Merged title")}</label>
            <input
              type="text"
              value={mergedTitle}
              onChange={(e) => setMergedTitle(e.target.value)}
              className="input input-bordered w-full"
            />
          </div>

          <div>
            <label className="label label-text">
              {t("audiobook.select_tracks", "Source files")}
              {mergeable.length > 0 && (
                <span className="badge badge-ghost badge-sm">{selected.size}/{mergeable.length}</span>
              )}
            </label>
            {mergeable.length === 0 ? (
              <p className="text-sm opacity-60">{t("audiobook.no_mergeable_files", "No m4a/m4b/mp3/flac/ogg/wav files on this book")}</p>
            ) : (
              <div className="max-h-56 overflow-y-auto space-y-1 rounded-box border border-base-300 p-2">
                {mergeable.map((f) => (
                  <label key={f.id} className="flex items-center gap-2 p-2 rounded-lg hover:bg-base-200 cursor-pointer">
                    <input type="checkbox" className="checkbox checkbox-sm" checked={selected.has(f.id)} onChange={() => toggle(f.id)} />
                    <span className="text-sm flex-1 truncate">{f.path.split("/").pop()}</span>
                    <span className="text-xs opacity-60">{formatBytes(f.size_bytes)}</span>
                  </label>
                ))}
              </div>
            )}
          </div>

          <div className="modal-action">
            <button type="button" className="btn btn-ghost" onClick={onClose}>
              {t("common.cancel")}
            </button>
            <button type="submit" className="btn btn-primary" disabled={merge.isPending || mergeable.length < 2}>
              {merge.isPending ? t("common.loading") : t("audiobook.start_merge", "Start merge")}
            </button>
          </div>
        </form>
      </div>
    </dialog>
  );
};