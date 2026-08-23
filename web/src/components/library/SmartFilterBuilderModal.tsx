import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { X, Plus, Trash2, Tag, User, BookOpen, Star, FileText } from "lucide-react";
import { toast } from "react-toastify";
import {
  useCreateSmartFilterMutation,
  useUpdateSmartFilterMutation,
  useSmartFiltersQuery,
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

export const SmartFilterBuilderModal: React.FC<SmartFilterBuilderModalProps> = ({
  isOpen,
  onClose,
  filterToEdit,
}) => {
  const { t } = useTranslation();
  const createMutation = useCreateSmartFilterMutation();
  const updateMutation = useUpdateSmartFilterMutation();
  const { data: smartFilters = [] } = useSmartFiltersQuery();

  const [name, setName] = useState("");
  const [rules, setRules] = useState<SmartFilterRuleItem[]>([]);
  const [isPinnedSidebar, setIsPinnedSidebar] = useState(false);
  const [isPinnedHome, setIsPinnedHome] = useState(false);

  useEffect(() => {
    if (filterToEdit) {
      setName(filterToEdit.name);
      setRules((filterToEdit.rules || []).map((r) => ({ ...r, id: r.id || nextRuleId() })));
      setIsPinnedSidebar(filterToEdit.is_pinned_sidebar);
      setIsPinnedHome(filterToEdit.is_pinned_home);
    } else {
      setName("");
      setRules([{ id: nextRuleId(), field: "status", operator: "eq", value: "unread" }]);
      setIsPinnedSidebar(false);
      setIsPinnedHome(false);
    }
  }, [filterToEdit, isOpen]);

  const handleAddRule = () => {
    setRules([...rules, { id: nextRuleId(), field: "status", operator: "eq", value: "unread" }]);
  };

  const handleRemoveRule = (index: number) => {
    const newRules = rules.filter((_, i) => i !== index);
    setRules(newRules);
  };

  const handleRuleChange = (
    index: number,
    key: keyof SmartFilterRuleItem,
    value: string
  ) => {
    const newRules = [...rules];
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
    setRules(newRules);
  };

  const handleSave = () => {
    if (!name.trim()) return;

    // Filter out rules with empty values
    const cleanedRules = rules.filter((r) => r.value.trim() !== "");
    if (cleanedRules.length === 0) {
      toast.error(t("library.rules_empty_error", "Please add at least one valid rule."));
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
        }
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
      <div className="modal-box max-w-2xl bg-base-100 border border-base-300 shadow-xl rounded-2xl p-6">
        <div className="flex justify-between items-center border-b border-base-200 pb-3 mb-4">
          <h3 className="font-extrabold text-xl flex items-center gap-2">
            <span>
              {filterToEdit
                ? t("library.edit_smart_filter", "Edit Smart Filter")
                : t("library.new_smart_filter", "New Smart Filter")}
            </span>
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="btn btn-ghost btn-sm btn-circle text-base-content hover:bg-base-200 border border-base-300/70 shadow-xs"
            aria-label={t("common.close", "Close")}
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="flex flex-col gap-4">
          {/* Name field */}
          <div className="form-control w-full">
            <label className="label font-semibold text-sm">
              {t("library.filter_name", "Filter Name")}
            </label>
            <input
              type="text"
              placeholder={t("library.enter_filter_name", "e.g., Unread light novels")}
              className="input input-bordered w-full rounded-xl"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>

          {/* Rules Builder */}
          <div>
            <label className="label font-semibold text-sm pb-1">
              {t("library.filter_rules", "Filter Rules")}
            </label>
            <div className="flex flex-col gap-3">
              {rules.map((rule, index) => (
                <RuleRow
                  key={rule.id}
                  rule={rule}
                  onChange={(key, val) => handleRuleChange(index, key, val)}
                  onRemove={() => handleRemoveRule(index)}
                  t={t}
                />
              ))}
            </div>

            <button
              type="button"
              onClick={handleAddRule}
              className="btn btn-sm btn-outline btn-primary rounded-xl mt-3 flex items-center gap-1.5"
            >
              <Plus className="w-4 h-4" />
              {t("library.add_condition", "Add Condition")}
            </button>
          </div>

          <div className="divider my-1"></div>

          {/* Pin Options */}
          <div className="flex flex-col gap-2.5 bg-base-200/50 p-4 rounded-xl border border-base-300">
            <label className="label cursor-pointer justify-between py-1">
              <span className="font-semibold text-sm">
                {t("library.pin_to_sidebar", "Pin to Sidebar")}
              </span>
              <input
                type="checkbox"
                className="checkbox checkbox-primary rounded-md"
                checked={isPinnedSidebar}
                onChange={(e) => setIsPinnedSidebar(e.target.checked)}
              />
            </label>

            <label className="label cursor-pointer justify-between py-1">
              <span className="font-semibold text-sm">
                {t("library.pin_to_home", "Pin to Homepage (Dashboard Shelf)")}
              </span>
              <input
                type="checkbox"
                className="checkbox checkbox-primary rounded-md"
                checked={isPinnedHome}
                onChange={(e) => setIsPinnedHome(e.target.checked)}
              />
            </label>
          </div>
        </div>

        <div className="modal-action border-t border-base-200 pt-4 mt-5">
          <button
            onClick={onClose}
            className="btn btn-outline rounded-xl btn-sm"
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            onClick={handleSave}
            disabled={!name.trim() || rules.length === 0}
            className="btn btn-primary rounded-xl btn-sm px-5"
          >
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
  t: any;
}

const RuleRow: React.FC<RuleRowProps> = ({ rule, onChange, onRemove, t }) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [showDropdown, setShowDropdown] = useState(false);

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
    { search: searchTerm }
  );

  return (
    <div className="flex gap-2 items-center bg-base-100 border border-base-200 p-2.5 rounded-xl shadow-xs">
      {/* Field Selector */}
      <select
        className="select select-bordered select-sm rounded-lg shrink-0 w-32"
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
      <span className="text-xs font-semibold text-base-content/60 px-1">
        {rule.field === "rating_gte" ? "≥" : "="}
      </span>

      {/* Value Selector */}
      <div className="flex-1 min-w-0 relative">
        {rule.field === "status" && (
          <select
            className="select select-bordered select-sm rounded-lg w-full"
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
            className="select select-bordered select-sm rounded-lg w-full"
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
          <div className="w-full">
            <input
              type="text"
              placeholder={
                rule.value
                  ? rule.value
                  : t("library.search_value", "Search or enter value...")
              }
              className="input input-bordered input-sm rounded-lg w-full"
              value={searchTerm}
              onFocus={() => setShowDropdown(true)}
              onBlur={() => setTimeout(() => setShowDropdown(false), 200)}
              onChange={(e) => {
                setSearchTerm(e.target.value);
                onChange("value", e.target.value);
              }}
            />
            {showDropdown && (searchTerm.trim() !== "" || facetItems.length > 0) && (
              <ul className="absolute z-50 left-0 right-0 mt-1 max-h-48 overflow-y-auto bg-base-100 border border-base-200 rounded-lg shadow-lg py-1 text-sm">
                {isPending && <li className="px-3 py-1.5 text-xs text-base-content/40">Loading...</li>}
                {!isPending && facetItems.length === 0 && (
                  <li
                    className="px-3 py-1.5 hover:bg-base-200 cursor-pointer text-xs"
                    onClick={() => {
                      onChange("value", searchTerm);
                      setShowDropdown(false);
                    }}
                  >
                    Use raw value: "{searchTerm}"
                  </li>
                )}
                {facetItems.map((item) => (
                  <li
                    key={item.id || item.name}
                    className="px-3 py-1.5 hover:bg-base-200 cursor-pointer flex justify-between gap-2"
                    onMouseDown={() => {
                      onChange("value", item.id || item.name);
                      setSearchTerm(item.name);
                      setShowDropdown(false);
                    }}
                  >
                    <span className="font-semibold truncate">{item.name}</span>
                    <span className="text-xs text-base-content/40 shrink-0">
                      ({item.book_count} books)
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>

      {/* Delete Rule Button */}
      <button
        type="button"
        onClick={onRemove}
        className="btn btn-ghost btn-sm btn-circle hover:text-error shrink-0"
      >
        <Trash2 className="w-4 h-4" />
      </button>
    </div>
  );
};
