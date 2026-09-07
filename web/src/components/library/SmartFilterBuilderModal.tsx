import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  X,
  Plus,
  Trash2,
  SlidersHorizontal,
} from "lucide-react";
import { toast } from "react-toastify";
import {
  useCreateSmartFilterMutation,
  useUpdateSmartFilterMutation,
} from "@/hooks";
import { useMetadataFacetQuery } from "@/hooks/useMetadataQueries";
import type { SmartFilter, SmartFilterRuleItem } from "@/types";

let ruleIdSeq = 0;
const nextRuleId = () => `rule-${++ruleIdSeq}`;

interface SmartFilterBuilderModalProps {
  isOpen: boolean;
  onClose: () => void;
  filterToEdit?: SmartFilter | null;
}

export const SmartFilterBuilderModal: React.FC<
  SmartFilterBuilderModalProps
> = ({ isOpen, onClose, filterToEdit }) => {
  const { t } = useTranslation();
  const createMutation = useCreateSmartFilterMutation();
  const updateMutation = useUpdateSmartFilterMutation();

  const [name, setName] = useState("");
  const [rules, setRules] = useState<SmartFilterRuleItem[]>([]);
  const [isPinnedSidebar, setIsPinnedSidebar] = useState(false);
  const [isPinnedHome, setIsPinnedHome] = useState(false);

  useEffect(() => {
    if (filterToEdit) {
      setName(filterToEdit.name);
      setRules(
        (filterToEdit.rules || []).map((r) => ({
          ...r,
          id: r.id || nextRuleId(),
        })),
      );
      setIsPinnedSidebar(filterToEdit.is_pinned_sidebar);
      setIsPinnedHome(filterToEdit.is_pinned_home);
    } else {
      setName("");
      setRules([
        { id: nextRuleId(), field: "status", operator: "eq", value: "unread" },
      ]);
      setIsPinnedSidebar(false);
      setIsPinnedHome(false);
    }
  }, [filterToEdit, isOpen]);

  const handleAddRule = () => {
    setRules((prev) => [
      ...prev,
      { id: nextRuleId(), field: "status", operator: "eq", value: "unread" },
    ]);
  };

  const handleRemoveRule = (index: number) => {
    setRules((prev) => prev.filter((_, i) => i !== index));
  };

  const handleRuleChange = (
    index: number,
    key: keyof SmartFilterRuleItem,
    value: string,
  ) => {
    setRules((prev) => {
      const newRules = [...prev];
      if (!newRules[index]) return prev;
      if (key === "field") {
        const field = value as SmartFilterRuleItem["field"];
        newRules[index].field = field;
        // Set default operator & value when field changes
        if (field === "rating_gte") {
          newRules[index].operator = "gte";
          newRules[index].value = "4";
        } else if (field === "status") {
          newRules[index].operator = "eq";
          newRules[index].value = "unread";
        } else {
          newRules[index].operator = "eq";
          newRules[index].value = "";
        }
      } else {
        newRules[index][key] = value as any;
      }
      return newRules;
    });
  };

  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  const handleSave = () => {
    if (!name.trim()) return;

    // Filter out rules with empty values
    const cleanedRules = rules.filter((r) => r.value.trim() !== "");
    if (cleanedRules.length === 0) {
      toast.error(
        t("library.rules_empty_error", "Please add at least one valid rule."),
      );
      return;
    }

    const payload = {
      name: name.trim(),
      rules: cleanedRules.map(({ id: _id, ...rest }) => rest),
      is_pinned_sidebar: isPinnedSidebar,
      is_pinned_home: isPinnedHome,
    };

    if (filterToEdit) {
      updateMutation.mutate(
        { id: filterToEdit.id, payload },
        {
          onSuccess: () => onClose(),
        },
      );
    } else {
      createMutation.mutate(payload, {
        onSuccess: () => onClose(),
      });
    }
  };

  if (!isOpen) return null;

  return (
    <dialog className="modal modal-open">
      <div className="modal-box max-w-xl w-11/12 sm:w-full max-h-[85dvh] sm:max-h-[85vh] bg-base-100 border border-base-300 shadow-2xl rounded-2xl sm:rounded-3xl p-0 overflow-hidden flex flex-col">
        {/* Pinned Header */}
        <div className="px-5 sm:px-6 py-4 border-b border-base-200 flex items-center justify-between shrink-0 bg-base-100">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-8 h-8 rounded-xl bg-primary/10 text-primary flex items-center justify-center shrink-0">
              <SlidersHorizontal className="w-4 h-4" />
            </div>
            <h3 className="font-bold text-lg text-base-content truncate">
              {filterToEdit
                ? t("library.edit_smart_filter", "Edit Smart Filter")
                : t("library.new_smart_filter", "New Smart Filter")}
            </h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="btn btn-ghost btn-circle btn-sm -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
            aria-label={t("common.close", "Close")}
          >
            <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
          </button>
        </div>

        {/* Scrollable Body */}
        <div className="flex-1 min-h-0 overflow-y-auto px-4 sm:px-6 py-5 space-y-5">
          {/* Name field */}
          <div className="space-y-1.5">
            <label className="block text-xs font-semibold uppercase tracking-wider text-base-content/70">
              {t("library.filter_name", "Filter Name")}
            </label>
            <input
              type="text"
              placeholder={t(
                "library.enter_filter_name",
                "e.g., Unread light novels",
              )}
              className="input input-bordered h-10 min-h-10 text-sm rounded-xl w-full focus:input-primary transition-all"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>

          {/* Rules Builder */}
          <div className="space-y-2.5">
            <div className="flex items-center justify-between">
              <label className="text-xs font-semibold uppercase tracking-wider text-base-content/70">
                {t("library.filter_rules", "Filter Rules")}
              </label>
            </div>

            <div className="space-y-2.5">
              {rules.map((rule, index) => (
                <RuleRow
                  key={rule.id}
                  rule={rule}
                  onChange={(key, val) => handleRuleChange(index, key, val)}
                  onRemove={() => handleRemoveRule(index)}
                  canRemove={rules.length > 1}
                  t={t}
                />
              ))}
            </div>

            <button
              type="button"
              onClick={handleAddRule}
              className="btn btn-ghost border border-dashed border-base-300 hover:border-primary/50 hover:bg-primary/5 text-primary h-10 min-h-10 rounded-xl w-full flex items-center justify-center gap-2 text-sm font-medium transition-all"
            >
              <Plus className="w-4 h-4" />
              <span>{t("library.add_condition", "Add Condition")}</span>
            </button>
          </div>

          {/* Pin Options Card */}
          <div className="rounded-2xl border border-base-200/90 bg-base-200/40 p-3 sm:p-4 space-y-1">
            <label className="flex items-center justify-between gap-3 py-2 px-2 cursor-pointer select-none rounded-xl hover:bg-base-200/60 transition-colors">
              <span className="text-sm font-medium text-base-content leading-snug flex-1 min-w-0 pr-2">
                {t("library.pin_to_sidebar", "Pin to Sidebar")}
              </span>
              <input
                type="checkbox"
                className="checkbox checkbox-primary checkbox-sm shrink-0 rounded-md"
                checked={isPinnedSidebar}
                onChange={(e) => setIsPinnedSidebar(e.target.checked)}
              />
            </label>

            <label className="flex items-center justify-between gap-3 py-2 px-2 cursor-pointer select-none rounded-xl hover:bg-base-200/60 transition-colors">
              <span className="text-sm font-medium text-base-content leading-snug flex-1 min-w-0 pr-2">
                {t("library.pin_to_home", "Pin to Homepage (Dashboard Shelf)")}
              </span>
              <input
                type="checkbox"
                className="checkbox checkbox-primary checkbox-sm shrink-0 rounded-md"
                checked={isPinnedHome}
                onChange={(e) => setIsPinnedHome(e.target.checked)}
              />
            </label>
          </div>
        </div>

        {/* Pinned Footer */}
        <div className="px-5 sm:px-6 py-3.5 border-t border-base-200 bg-base-100 flex items-center justify-end gap-2.5 shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="btn btn-ghost h-10 min-h-10 px-5 rounded-xl text-sm font-medium"
            disabled={isSubmitting}
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={!name.trim() || rules.length === 0 || isSubmitting}
            className="btn btn-primary h-10 min-h-10 px-6 rounded-xl text-sm font-medium shadow-sm hover:shadow-md transition-all"
          >
            {isSubmitting && (
              <span className="loading loading-spinner loading-xs mr-1" />
            )}
            {t("common.save", "Save")}
          </button>
        </div>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose}>close</button>
      </form>
    </dialog>
  );
};

/* --- RuleRow Subcomponent --- */
interface RuleRowProps {
  rule: SmartFilterRuleItem;
  onChange: (key: keyof SmartFilterRuleItem, value: string) => void;
  onRemove: () => void;
  canRemove: boolean;
  t: any;
}

const RuleRow: React.FC<RuleRowProps> = ({
  rule,
  onChange,
  onRemove,
  canRemove,
  t,
}) => {
  const [searchTerm, setSearchTerm] = useState(rule.value || "");
  const [showDropdown, setShowDropdown] = useState(false);

  useEffect(() => {
    setSearchTerm(rule.value || "");
  }, [rule.value]);

  // Map fields to metadata types
  const facetType =
    rule.field === "author_id"
      ? "authors"
      : rule.field === "series_id"
        ? "series"
        : rule.field === "tag_id"
          ? "tags"
          : rule.field === "format"
            ? "formats"
            : null;

  // Use facet query if applicable
  const { items: facetItems = [], isPending } = useMetadataFacetQuery(
    facetType || "authors",
    { search: searchTerm },
  );

  return (
    <div className="bg-base-100 border border-base-200/90 rounded-2xl p-2.5 sm:p-3 shadow-xs hover:border-base-300 transition-colors">
      <div className="flex flex-col sm:flex-row sm:items-center gap-2 w-full">
        {/* Mobile Top Row: Field + Operator + Delete */}
        <div className="flex items-center gap-2 w-full sm:w-auto shrink-0">
          {/* Field Selector */}
          <select
            className="select select-bordered h-10 min-h-10 text-sm rounded-xl flex-1 sm:w-44 shrink-0 focus:select-primary transition-all"
            value={rule.field}
            onChange={(e) => onChange("field", e.target.value)}
          >
            <option value="status">{t("library.rule_status", "Status")}</option>
            <option value="format">{t("library.rule_format", "Format")}</option>
            <option value="rating_gte">{t("library.rule_rating", "Rating")}</option>
            <option value="author_id">{t("library.rule_author", "Author")}</option>
            <option value="series_id">{t("library.rule_series", "Series")}</option>
            <option value="tag_id">{t("library.rule_tag", "Tag")}</option>
          </select>

          {/* Operator */}
          <div className="h-10 min-h-10 px-3 rounded-xl bg-base-200/80 border border-base-300/80 flex items-center justify-center font-bold text-sm text-base-content/80 select-none shrink-0">
            {rule.field === "rating_gte" ? "≥" : "="}
          </div>

          {/* Mobile Delete Button */}
          <button
            type="button"
            onClick={onRemove}
            disabled={!canRemove}
            className="sm:hidden btn btn-ghost h-10 w-10 min-h-10 p-0 rounded-xl text-base-content/50 hover:text-error hover:bg-error/10 disabled:opacity-20 disabled:hover:bg-transparent shrink-0 flex items-center justify-center transition-colors"
            aria-label={t("common.delete", "Delete")}
            title={t("common.delete", "Delete")}
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>

        {/* Value Selector */}
        <div className="flex-1 min-w-0 relative w-full">
          {rule.field === "status" && (
            <select
              className="select select-bordered h-10 min-h-10 text-sm rounded-xl w-full focus:select-primary transition-all"
              value={rule.value}
              onChange={(e) => onChange("value", e.target.value)}
            >
              <option value="unread">{t("library.status_unread", "Unread")}</option>
              <option value="reading">{t("library.status_reading", "Reading")}</option>
              <option value="read">{t("library.status_read", "Read")}</option>
            </select>
          )}

          {rule.field === "rating_gte" && (
            <select
              className="select select-bordered h-10 min-h-10 text-sm rounded-xl w-full focus:select-primary transition-all"
              value={rule.value}
              onChange={(e) => onChange("value", e.target.value)}
            >
              <option value="5">5 ★</option>
              <option value="4">4 ★ & Up</option>
              <option value="3">3 ★ & Up</option>
              <option value="2">2 ★ & Up</option>
              <option value="1">1 ★ & Up</option>
            </select>
          )}

          {facetType && (
            <div className="w-full relative">
              <input
                type="text"
                placeholder={
                  rule.field === "format"
                    ? "EPUB, PDF, CBZ..."
                    : t("library.search_value", "Search or enter value...")
                }
                className="input input-bordered h-10 min-h-10 text-sm rounded-xl w-full focus:input-primary pr-8 transition-all"
                value={searchTerm}
                onFocus={() => setShowDropdown(true)}
                onBlur={() => setTimeout(() => setShowDropdown(false), 200)}
                onChange={(e) => {
                  const val = e.target.value;
                  setSearchTerm(val);
                  onChange("value", val);
                  setShowDropdown(true);
                }}
              />
              {searchTerm ? (
                <button
                  type="button"
                  tabIndex={-1}
                  onClick={() => {
                    setSearchTerm("");
                    onChange("value", "");
                  }}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-base-content/40 hover:text-base-content p-0.5 rounded-full transition-colors"
                  aria-label={t("common.clear", "Clear")}
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              ) : null}

              {showDropdown &&
                (searchTerm.trim() !== "" || facetItems.length > 0) && (
                  <ul className="absolute z-50 left-0 right-0 mt-1.5 max-h-48 overflow-y-auto bg-base-100 border border-base-300 rounded-xl shadow-xl py-1 text-sm divide-y divide-base-200/50">
                    {isPending && (
                      <li className="px-3.5 py-2 text-xs text-base-content/50 flex items-center gap-2">
                        <span className="loading loading-spinner loading-xs" />
                        {t("common.loading", "Loading...")}
                      </li>
                    )}
                    {!isPending &&
                      facetItems.length === 0 &&
                      searchTerm.trim() !== "" && (
                        <li
                          className="px-3.5 py-2 hover:bg-base-200 cursor-pointer text-xs text-base-content/70 italic"
                          onMouseDown={(e) => {
                            e.preventDefault();
                            onChange("value", searchTerm);
                            setShowDropdown(false);
                          }}
                        >
                          {t("library.use_raw_value", "Use value")}: "{searchTerm}"
                        </li>
                      )}
                    {facetItems.map((item) => (
                      <li
                        key={item.id || item.name}
                        className="px-3.5 py-2 hover:bg-base-200 cursor-pointer flex items-center justify-between gap-2 transition-colors"
                        onMouseDown={(e) => {
                          e.preventDefault();
                          const val = item.name || item.id || "";
                          setSearchTerm(val);
                          onChange("value", val);
                          setShowDropdown(false);
                        }}
                      >
                        <span className="font-medium text-sm text-base-content truncate">
                          {item.name}
                        </span>
                        {item.book_count !== undefined && item.book_count > 0 && (
                          <span className="text-xs text-base-content/40 shrink-0">
                            {item.book_count} {t("library.readlist_books", "books")}
                          </span>
                        )}
                      </li>
                    ))}
                  </ul>
                )}
            </div>
          )}
        </div>

        {/* Desktop Delete Button */}
        <button
          type="button"
          onClick={onRemove}
          disabled={!canRemove}
          className="hidden sm:inline-flex btn btn-ghost h-10 w-10 min-h-10 p-0 rounded-xl text-base-content/50 hover:text-error hover:bg-error/10 disabled:opacity-20 disabled:hover:bg-transparent shrink-0 items-center justify-center transition-colors"
          aria-label={t("common.delete", "Delete")}
          title={t("common.delete", "Delete")}
        >
          <Trash2 className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
};
