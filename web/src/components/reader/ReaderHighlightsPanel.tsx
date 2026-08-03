import type { TFunction } from "i18next";
import { Check, Pencil, Trash2, X } from "lucide-react";
import React, { useState } from "react";

import type { Highlight } from "@/types";

export const HIGHLIGHT_COLORS = ["#fef08a", "#bbf7d0", "#bfdbfe", "#e9d5ff"];

type ReaderHighlightsPanelProps = {
  t: TFunction;
  highlights: Highlight[];
  onSelect?: (highlight: Highlight) => void;
  onUpdate: (id: string, color: string, note?: string) => void;
  onDelete: (id: string) => void;
};

export const ReaderHighlightsPanel: React.FC<ReaderHighlightsPanelProps> = ({
  t,
  highlights,
  onSelect,
  onUpdate,
  onDelete,
}) => {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftNote, setDraftNote] = useState("");
  const [draftColor, setDraftColor] = useState(HIGHLIGHT_COLORS[0]);

  const startEditing = (highlight: Highlight) => {
    setEditingId(highlight.id);
    setDraftNote(highlight.note || "");
    setDraftColor(highlight.color || HIGHLIGHT_COLORS[0]);
  };

  const stopEditing = () => {
    setEditingId(null);
    setDraftNote("");
  };

  const saveEditing = (id: string) => {
    onUpdate(id, draftColor, draftNote.trim() || undefined);
    stopEditing();
  };

  return (
    <>
      <div className="reader-sidebar-list-title">
        <span>{t("reader.highlights", "Highlights")}</span>
        {highlights.length > 0 && (
          <span className="reader-sidebar-count">{highlights.length}</span>
        )}
      </div>

      {highlights.length === 0 ? (
        <p className="px-4 pb-4 text-xs opacity-60">
          {t("reader.no_highlights", "No highlights in this chapter yet.")}
        </p>
      ) : (
        <ul className="flex flex-col gap-2 px-4 pb-4">
          {highlights.map((highlight) => {
            const isEditing = editingId === highlight.id;

            return (
              <li
                key={highlight.id}
                className="rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) p-3"
              >
                <div className="flex items-start gap-2">
                  <span
                    className="mt-1 h-3 w-3 shrink-0 rounded-full border border-current/20"
                    style={{ backgroundColor: highlight.color }}
                    aria-hidden="true"
                  />
                  <button
                    type="button"
                    onClick={() => onSelect?.(highlight)}
                    className="line-clamp-3 flex-1 text-left text-xs leading-snug hover:underline"
                  >
                    {highlight.text_content}
                  </button>
                </div>

                {isEditing ? (
                  <div className="mt-2 flex flex-col gap-2">
                    <div className="flex items-center gap-1.5">
                      {HIGHLIGHT_COLORS.map((color) => (
                        <button
                          key={color}
                          type="button"
                          onClick={() => setDraftColor(color)}
                          style={{ backgroundColor: color }}
                          className={`h-4 w-4 rounded-full border transition-transform hover:scale-125 ${
                            draftColor === color
                              ? "border-current"
                              : "border-current/20"
                          }`}
                          aria-label={t("reader.highlight_color", "Highlight color")}
                        />
                      ))}
                    </div>
                    <textarea
                      value={draftNote}
                      onChange={(e) => setDraftNote(e.target.value)}
                      rows={2}
                      className="textarea textarea-bordered textarea-xs w-full"
                      placeholder={t("reader.highlight_note_placeholder", "Add a note")}
                    />
                    <div className="flex justify-end gap-1">
                      <button
                        type="button"
                        onClick={stopEditing}
                        className="reader-control-btn btn btn-ghost btn-xs"
                        title={t("common.cancel", "Cancel")}
                        aria-label={t("common.cancel", "Cancel")}
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                      <button
                        type="button"
                        onClick={() => saveEditing(highlight.id)}
                        className="reader-action-btn btn btn-xs px-2.5"
                        title={t("common.save", "Save")}
                        aria-label={t("common.save", "Save")}
                      >
                        <Check className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>
                ) : (
                  <>
                    {highlight.note && (
                      <p className="mt-1.5 text-xs italic opacity-70">{highlight.note}</p>
                    )}
                    <div className="mt-1.5 flex justify-end gap-1">
                      <button
                        type="button"
                        onClick={() => startEditing(highlight)}
                        className="btn btn-ghost btn-xs"
                        title={t("reader.edit_highlight", "Edit note")}
                        aria-label={t("reader.edit_highlight", "Edit note")}
                      >
                        <Pencil className="h-3.5 w-3.5" />
                      </button>
                      <button
                        type="button"
                        onClick={() => onDelete(highlight.id)}
                        className="btn btn-ghost btn-xs text-error"
                        title={t("common.delete", "Delete")}
                        aria-label={t("common.delete", "Delete")}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </>
  );
};
