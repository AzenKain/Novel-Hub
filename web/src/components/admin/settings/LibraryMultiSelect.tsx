import type { Library as LibraryType } from "@/types";
import { useTranslation } from "react-i18next";

export function LibraryMultiSelect({
  ids,
  libraries,
  onChange,
}: {
  ids: string[];
  libraries: LibraryType[];
  onChange: (ids: string[]) => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-wrap gap-2 p-3 bg-base-200/50 rounded-lg border border-base-200 max-h-40 overflow-y-auto">
      {libraries.length === 0 ? (
        <span className="text-xs opacity-50">{t("admin.no_libraries_available")}</span>
      ) : (
        libraries.map((lib) => {
          const checked = ids.includes(lib.id);
          return (
            <label
              key={lib.id}
              className={`cursor-pointer badge text-xs px-3 py-3 transition-colors ${
                checked ? "badge-primary" : "badge-outline hover:badge-ghost"
              }`}
            >
              <input
                type="checkbox"
                className="sr-only"
                checked={checked}
                onChange={() =>
                  onChange(
                    checked ? ids.filter((i) => i !== lib.id) : [...ids, lib.id]
                  )
                }
              />
              {lib.name}
            </label>
          );
        })
      )}
    </div>
  );
}
