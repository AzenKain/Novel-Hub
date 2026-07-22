import type { Library } from "@/types";
import { Check, ChevronDown, Library as LibraryIcon, Search, SlidersHorizontal, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

interface LibraryScopeSelectorProps {
  selectedLibraryIds: string[];
  onChange: (ids: string[]) => void;
  libraries: Library[];
}

export function LibraryScopeSelector({
  selectedLibraryIds,
  onChange,
  libraries,
}: LibraryScopeSelectorProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState("");
  const dropdownRef = useRef<HTMLDivElement>(null);

  const isAll = selectedLibraryIds.length === 0;

  // Close dropdown on click outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const filteredLibraries = libraries.filter((lib) =>
    lib.name.toLowerCase().includes(search.toLowerCase()) ||
    lib.id.toLowerCase().includes(search.toLowerCase())
  );

  function handleToggleMode(all: boolean) {
    if (all) {
      onChange([]);
    } else {
      if (libraries.length > 0 && selectedLibraryIds.length === 0) {
        // Default to first library if switching to specific mode
        onChange([libraries[0].id]);
      }
    }
  }

  function handleToggleLibrary(id: string) {
    if (selectedLibraryIds.includes(id)) {
      const next = selectedLibraryIds.filter((item) => item !== id);
      onChange(next);
    } else {
      onChange([...selectedLibraryIds, id]);
    }
  }

  function handleRemove(id: string) {
    onChange(selectedLibraryIds.filter((item) => item !== id));
  }

  function handleSelectAll() {
    onChange(libraries.map((l) => l.id));
  }

  function handleClearAll() {
    onChange([]);
  }

  return (
    <div className="flex flex-col gap-2 pt-1">
      {/* Mode Switcher */}
      <div className="flex items-center gap-3">
        <span className="text-xs font-semibold text-base-content/70 shrink-0 flex items-center gap-1.5">
          <SlidersHorizontal className="w-3.5 h-3.5" />
          {t("library_scope", "Library Scope")}:
        </span>

        <div className="flex items-center gap-1 bg-base-200/80 p-0.5 rounded-lg text-xs font-medium border border-base-300/50">
          <button
            type="button"
            onClick={() => handleToggleMode(true)}
            className={`px-3 py-1 rounded-md transition-all ${
              isAll
                ? "bg-primary text-primary-content font-bold shadow-xs"
                : "text-base-content/70 hover:text-base-content"
            }`}
          >
            {t("scope_all_libraries", "All Libraries")}
          </button>
          <button
            type="button"
            onClick={() => handleToggleMode(false)}
            className={`px-3 py-1 rounded-md transition-all ${
              !isAll
                ? "bg-primary text-primary-content font-bold shadow-xs"
                : "text-base-content/70 hover:text-base-content"
            }`}
          >
            {t("scope_specific_libraries", "Specific Libraries")} ({selectedLibraryIds.length})
          </button>
        </div>
      </div>

      {/* Specific Libraries Picker Panel */}
      {!isAll && (
        <div className="bg-base-200/40 border border-base-300/60 rounded-xl p-3 flex flex-col gap-2.5 transition-all animate-fadeIn">
          {/* Selected Library Chips */}
          <div className="flex flex-wrap items-center gap-1.5 min-h-[32px]">
            {selectedLibraryIds.length === 0 ? (
              <span className="text-xs text-warning flex items-center gap-1 italic">
                {t("no_libraries_selected", "No libraries selected (Permission will not match any library)")}
              </span>
            ) : (
              selectedLibraryIds.map((id) => {
                const lib = libraries.find((l) => l.id === id);
                return (
                  <span
                    key={id}
                    className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-base-100 border border-primary/30 text-xs font-medium text-base-content shadow-xs group"
                  >
                    <LibraryIcon className="w-3 h-3 text-primary shrink-0" />
                    <span>{lib ? lib.name : id}</span>
                    <button
                      type="button"
                      onClick={() => handleRemove(id)}
                      className="hover:bg-error/20 hover:text-error rounded-full p-0.5 transition-colors"
                      title={t("remove", "Remove")}
                    >
                      <X className="w-3 h-3" />
                    </button>
                  </span>
                );
              })
            )}

            {/* Dropdown Button */}
            <div className="relative inline-block" ref={dropdownRef}>
              <button
                type="button"
                onClick={() => setIsOpen(!isOpen)}
                className="btn btn-xs btn-outline btn-primary gap-1 font-normal"
              >
                <span>{t("select_libraries", "Select Libraries")}</span>
                <ChevronDown className={`w-3 h-3 transition-transform ${isOpen ? "rotate-180" : ""}`} />
              </button>

              {/* Popover Menu */}
              {isOpen && (
                <div className="absolute left-0 mt-1.5 w-72 bg-base-100 border border-base-300 rounded-xl shadow-xl z-50 p-2.5 flex flex-col gap-2 animate-in fade-in zoom-in-95">
                  {/* Search Input */}
                  <div className="relative">
                    <Search className="w-3.5 h-3.5 text-base-content/40 absolute left-2.5 top-2.5" />
                    <input
                      type="text"
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                      placeholder={t("search_libraries", "Search libraries...")}
                      className="input input-xs input-bordered w-full pl-8"
                      autoFocus
                    />
                  </div>

                  {/* Actions Bar */}
                  <div className="flex items-center justify-between text-[11px] px-1 text-base-content/60 border-b border-base-200 pb-1.5">
                    <button
                      type="button"
                      onClick={handleSelectAll}
                      className="hover:text-primary font-medium transition-colors"
                    >
                      {t("select_all", "Select All")}
                    </button>
                    <button
                      type="button"
                      onClick={handleClearAll}
                      className="hover:text-error font-medium transition-colors"
                    >
                      {t("clear_all", "Clear All")}
                    </button>
                  </div>

                  {/* Libraries Checkbox List */}
                  <div className="max-h-48 overflow-y-auto space-y-0.5 pr-1">
                    {filteredLibraries.length === 0 ? (
                      <div className="text-xs text-base-content/40 text-center py-4">
                        {t("no_libraries_found", "No libraries found")}
                      </div>
                    ) : (
                      filteredLibraries.map((lib) => {
                        const checked = selectedLibraryIds.includes(lib.id);
                        return (
                          <label
                            key={lib.id}
                            className={`flex items-center justify-between p-2 rounded-lg text-xs cursor-pointer transition-colors ${
                              checked
                                ? "bg-primary/10 text-primary font-medium"
                                : "hover:bg-base-200 text-base-content/80"
                            }`}
                          >
                            <div className="flex items-center gap-2 min-w-0">
                              <input
                                type="checkbox"
                                checked={checked}
                                onChange={() => handleToggleLibrary(lib.id)}
                                className="checkbox checkbox-primary checkbox-xs"
                              />
                              <span className="truncate">{lib.name}</span>
                            </div>
                            {checked && <Check className="w-3.5 h-3.5 text-primary shrink-0" />}
                          </label>
                        );
                      })
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
