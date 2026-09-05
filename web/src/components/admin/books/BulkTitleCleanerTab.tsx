import React, { useState, useMemo, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  Wand2,
  Sparkles,
  ArrowRight,
  Check,
  RotateCcw,
  CheckSquare,
  Square,
  Search,
  Sliders,
} from "lucide-react";
import { toast } from "react-toastify";
import type { BulkEditItem } from "./BulkEditMetadataModal";

interface BulkTitleCleanerTabProps {
  items: BulkEditItem[];
  onApplyChanges: (
    changes: Array<{ id: string; title: string; author: string }>,
  ) => void;
}

import {
  toTitleCase,
  stripBrackets,
  stripParentheses,
  replaceUnderscores,
  splitDashAuthorTitle,
  splitDashTitleAuthor,
  splitTitleByAuthor,
  cleanWhitespace,
  applyCustomRegex,
} from "@/utils/metadataCleaner";

export const BulkTitleCleanerTab: React.FC<BulkTitleCleanerTabProps> = ({
  items,
  onApplyChanges,
}) => {
  const { t } = useTranslation();

  const [cleanedState, setCleanedState] = useState<
    Record<string, { title: string; author: string; enabled: boolean }>
  >(() => {
    const init: Record<
      string,
      { title: string; author: string; enabled: boolean }
    > = {};
    items.forEach((it) => {
      init[it.id] = {
        title: it.title,
        author: it.author,
        enabled: true,
      };
    });
    return init;
  });

  useEffect(() => {
    setCleanedState((prev) => {
      const next: Record<
        string,
        { title: string; author: string; enabled: boolean }
      > = {};
      items.forEach((it) => {
        next[it.id] = {
          title: prev[it.id]?.title ?? it.title,
          author: prev[it.id]?.author ?? it.author,
          enabled: prev[it.id]?.enabled ?? true,
        };
      });
      return next;
    });
  }, [items]);

  const [searchFilter, setSearchFilter] = useState("");
  const [regexPattern, setRegexPattern] = useState("");
  const [regexReplace, setRegexReplace] = useState("");
  const [regexTarget, setRegexTarget] = useState<"title" | "author" | "both">(
    "title",
  );

  const applyPreset = (
    cleanerFn: (
      title: string,
      author: string,
    ) => { title: string; author: string },
  ) => {
    setCleanedState((prev) => {
      const next = { ...prev };
      let changedCount = 0;
      Object.keys(next).forEach((id) => {
        if (!next[id].enabled) return;
        const res = cleanerFn(next[id].title, next[id].author);
        if (res.title !== next[id].title || res.author !== next[id].author) {
          next[id] = { ...next[id], title: res.title, author: res.author };
          changedCount++;
        }
      });
      toast.info(
        t("library.cleaner_preset_applied", "Updated {{count}} books", {
          count: changedCount,
        }),
      );
      return next;
    });
  };

  const handleStripBrackets = () => {
    applyPreset((title, author) => ({
      title: stripBrackets(title),
      author,
    }));
  };

  const handleStripParentheses = () => {
    applyPreset((title, author) => ({
      title: stripParentheses(title),
      author,
    }));
  };

  const handleReplaceUnderscores = () => {
    applyPreset((title, author) => ({
      title: replaceUnderscores(title),
      author: replaceUnderscores(author),
    }));
  };

  const handleTitleCase = () => {
    applyPreset((title, author) => ({
      title: toTitleCase(title),
      author: toTitleCase(author),
    }));
  };

  const handleSplitAuthorTitle = () => {
    applyPreset((title, author) => {
      const res = splitDashAuthorTitle(title);
      return res ? { title: res.title, author: res.author } : { title, author };
    });
  };

  const handleSplitTitleAuthor = () => {
    applyPreset((title, author) => {
      const res = splitDashTitleAuthor(title);
      return res ? { title: res.title, author: res.author } : { title, author };
    });
  };

  const handleSplitByAuthor = () => {
    applyPreset((title, author) => {
      const res = splitTitleByAuthor(title);
      return res ? { title: res.title, author: res.author } : { title, author };
    });
  };

  const handleTrimWhitespace = () => {
    applyPreset((title, author) => ({
      title: cleanWhitespace(title),
      author: cleanWhitespace(author),
    }));
  };

  const handleApplyRegex = () => {
    if (!regexPattern) return;
    try {
      const reg = new RegExp(regexPattern, "g");
      setCleanedState((prev) => {
        const next = { ...prev };
        let count = 0;
        Object.keys(next).forEach((id) => {
          if (!next[id].enabled) return;
          let newTitle = next[id].title;
          let newAuthor = next[id].author;

          if (regexTarget === "title" || regexTarget === "both") {
            newTitle = newTitle
              .replace(reg, regexReplace)
              .replace(/\s+/g, " ")
              .trim();
          }
          if (regexTarget === "author" || regexTarget === "both") {
            newAuthor = newAuthor
              .replace(reg, regexReplace)
              .replace(/\s+/g, " ")
              .trim();
          }

          if (newTitle !== next[id].title || newAuthor !== next[id].author) {
            next[id] = { ...next[id], title: newTitle, author: newAuthor };
            count++;
          }
        });
        toast.info(
          t("library.cleaner_regex_applied", "Regex updated {{count}} books", {
            count,
          }),
        );
        return next;
      });
    } catch (err) {
      toast.error(
        t("library.cleaner_regex_invalid", "Invalid Regular Expression"),
      );
    }
  };

  const handleResetAll = () => {
    const next: Record<
      string,
      { title: string; author: string; enabled: boolean }
    > = {};
    items.forEach((it) => {
      next[it.id] = {
        title: it.original.title,
        author: it.original.author_name || "",
        enabled: true,
      };
    });
    setCleanedState(next);
  };

  const handleApplyToBooks = () => {
    const changes: Array<{ id: string; title: string; author: string }> = [];
    items.forEach((it) => {
      const current = cleanedState[it.id];
      if (current && current.enabled) {
        changes.push({
          id: it.id,
          title: current.title,
          author: current.author,
        });
      }
    });
    onApplyChanges(changes);
    toast.success(
      t(
        "library.cleaner_applied_success",
        "Applied cleaned titles and authors to {{count}} books",
        {
          count: changes.length,
        },
      ),
    );
  };

  const toggleAll = (checked: boolean) => {
    setCleanedState((prev) => {
      const next = { ...prev };
      Object.keys(next).forEach((id) => {
        next[id] = { ...next[id], enabled: checked };
      });
      return next;
    });
  };

  const filteredItems = useMemo(() => {
    if (!searchFilter.trim()) return items;
    const q = searchFilter.toLowerCase();
    return items.filter(
      (it) =>
        it.title.toLowerCase().includes(q) ||
        it.author.toLowerCase().includes(q) ||
        (cleanedState[it.id]?.title || "").toLowerCase().includes(q),
    );
  }, [items, searchFilter, cleanedState]);

  const allSelected = useMemo(() => {
    return items.every((it) => cleanedState[it.id]?.enabled);
  }, [items, cleanedState]);

  return (
    <div className="flex flex-col gap-4 p-4">
      <div className="bg-base-200/50 p-3.5 rounded-2xl border border-base-300">
        <div className="flex items-center gap-2 mb-2.5">
          <Sparkles className="h-4 w-4 text-primary" />
          <span className="text-xs font-bold uppercase tracking-wider text-base-content/70">
            {t("library.cleaner_presets", "1-Click Presets")}
          </span>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={handleStripBrackets}
            className="btn btn-xs btn-outline rounded-lg"
          >
            {t("library.cleaner_strip_brackets", "Strip [Brackets]")}
          </button>
          <button
            type="button"
            onClick={handleStripParentheses}
            className="btn btn-xs btn-outline rounded-lg"
          >
            {t("library.cleaner_strip_parens", "Strip (Parentheses)")}
          </button>
          <button
            type="button"
            onClick={handleReplaceUnderscores}
            className="btn btn-xs btn-outline rounded-lg"
          >
            {t("library.cleaner_underscores", "Underscore to Space")}
          </button>
          <button
            type="button"
            onClick={handleTitleCase}
            className="btn btn-xs btn-outline rounded-lg"
          >
            {t("library.cleaner_title_case", "Title Case")}
          </button>
          <button
            type="button"
            onClick={handleSplitAuthorTitle}
            className="btn btn-xs btn-outline rounded-lg"
          >
            {t("library.cleaner_split_author_title", 'Split "Author - Title"')}
          </button>
          <button
            type="button"
            onClick={handleSplitTitleAuthor}
            className="btn btn-xs btn-outline rounded-lg"
          >
            {t("library.cleaner_split_title_author", 'Split "Title - Author"')}
          </button>
          <button
            type="button"
            onClick={handleSplitByAuthor}
            className="btn btn-xs btn-outline rounded-lg"
          >
            {t("library.cleaner_split_by", 'Split "Title by Author"')}
          </button>
          <button
            type="button"
            onClick={handleTrimWhitespace}
            className="btn btn-xs btn-outline rounded-lg"
          >
            {t("library.cleaner_trim", "Trim Spaces")}
          </button>
          <button
            type="button"
            onClick={handleResetAll}
            className="btn btn-xs btn-ghost gap-1 ml-auto text-base-content/60"
          >
            <RotateCcw className="h-3 w-3" />
            {t("common.reset", "Reset")}
          </button>
        </div>

        <div className="mt-3 pt-3 border-t border-base-300 flex flex-wrap items-center gap-2 text-xs">
          <Sliders className="h-3.5 w-3.5 text-base-content/50" />
          <span className="font-semibold text-base-content/70">
            {t("library.cleaner_regex", "Custom Regex")}:
          </span>
          <input
            type="text"
            placeholder={t(
              "library.cleaner_regex_pattern",
              "Pattern (e.g. ^Chương \\d+:\\s*)",
            )}
            value={regexPattern}
            onChange={(e) => setRegexPattern(e.target.value)}
            className="input input-xs input-bordered w-48 font-mono"
          />
          <input
            type="text"
            placeholder={t("library.cleaner_regex_replace", "Replacement")}
            value={regexReplace}
            onChange={(e) => setRegexReplace(e.target.value)}
            className="input input-xs input-bordered w-32 font-mono"
          />
          <select
            value={regexTarget}
            onChange={(e) => setRegexTarget(e.target.value as any)}
            className="select select-xs select-bordered"
          >
            <option value="title">
              {t("library.title_only", "Title Only")}
            </option>
            <option value="author">
              {t("library.author_only", "Author Only")}
            </option>
            <option value="both">{t("library.both_fields", "Both")}</option>
          </select>
          <button
            type="button"
            onClick={handleApplyRegex}
            disabled={!regexPattern}
            className="btn btn-xs btn-primary gap-1"
          >
            <Wand2 className="h-3 w-3" />
            {t("common.apply", "Apply")}
          </button>
        </div>
      </div>

      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => toggleAll(!allSelected)}
            className="btn btn-xs btn-ghost gap-1"
          >
            {allSelected ? (
              <CheckSquare className="h-4 w-4 text-primary" />
            ) : (
              <Square className="h-4 w-4" />
            )}
            {t("common.select_all", "Select All")}
          </button>
          <span className="text-xs text-base-content/60">
            {t(
              "library.cleaner_selected_count",
              "{{count}} books enabled for cleanup",
              {
                count: Object.values(cleanedState).filter((v) => v.enabled)
                  .length,
              },
            )}
          </span>
        </div>

        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="h-3.5 w-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-base-content/40 pointer-events-none z-10" />
            <input
              type="text"
              placeholder={t("common.search", "Filter...")}
              value={searchFilter}
              onChange={(e) => setSearchFilter(e.target.value)}
              className="input input-xs input-bordered pl-8 w-44"
            />
          </div>
          <button
            type="button"
            onClick={handleApplyToBooks}
            className="btn btn-xs btn-primary gap-1 shadow-sm"
          >
            <Check className="h-3.5 w-3.5" />
            {t("library.cleaner_apply_all", "Apply Cleaned Data")}
          </button>
        </div>
      </div>

      <div className="border border-base-200 rounded-xl overflow-hidden bg-base-100 max-h-[50vh] overflow-y-auto">
        <table className="table table-xs table-pin-rows">
          <thead>
            <tr className="bg-base-200/80">
              <th className="w-8"></th>
              <th className="w-5/12">
                {t("library.original_metadata", "Original Title & Author")}
              </th>
              <th className="w-8 text-center"></th>
              <th className="w-6/12">
                {t("library.cleaned_metadata", "Cleaned Preview (Editable)")}
              </th>
            </tr>
          </thead>
          <tbody>
            {filteredItems.map((item) => {
              const current = cleanedState[item.id] || {
                title: item.title,
                author: item.author,
                enabled: true,
              };
              const isModified =
                current.title !== item.original.title ||
                current.author !== (item.original.author_name || "");

              return (
                <tr
                  key={item.id}
                  className={
                    current.enabled ? "hover:bg-base-200/40" : "opacity-40"
                  }
                >
                  <td>
                    <input
                      type="checkbox"
                      checked={current.enabled}
                      onChange={(e) =>
                        setCleanedState((p) => ({
                          ...p,
                          [item.id]: {
                            ...p[item.id],
                            enabled: e.target.checked,
                          },
                        }))
                      }
                      className="checkbox checkbox-xs checkbox-primary"
                    />
                  </td>
                  <td>
                    <div className="font-semibold text-xs leading-snug line-clamp-1">
                      {item.original.title}
                    </div>
                    <div className="text-[11px] text-base-content/50">
                      {item.original.author_name ||
                        t("common.unknown_author", "Unknown")}
                    </div>
                  </td>
                  <td className="text-center">
                    <ArrowRight
                      className={`h-3.5 w-3.5 mx-auto ${isModified ? "text-primary" : "text-base-content/20"}`}
                    />
                  </td>
                  <td>
                    <div className="flex flex-col gap-1">
                      <input
                        type="text"
                        value={current.title}
                        onChange={(e) =>
                          setCleanedState((p) => ({
                            ...p,
                            [item.id]: { ...p[item.id], title: e.target.value },
                          }))
                        }
                        className={`input input-xs input-bordered w-full font-medium ${isModified ? "border-primary/40 bg-primary/[0.02]" : ""}`}
                      />
                      <input
                        type="text"
                        placeholder={t("book.author", "Author")}
                        value={current.author}
                        onChange={(e) =>
                          setCleanedState((p) => ({
                            ...p,
                            [item.id]: {
                              ...p[item.id],
                              author: e.target.value,
                            },
                          }))
                        }
                        className="input input-xs input-bordered w-full text-[11px] text-base-content/70"
                      />
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};
