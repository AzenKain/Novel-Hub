import { useLogFilesQuery, useLogTailQuery } from "@/hooks";
import { useDebounce } from "@/hooks/useDebounce";
import { operationsService } from "@/services";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Download, RefreshCw, Search, X } from "lucide-react";
import { toast } from "react-toastify";

export function LogsTab() {
  const { t } = useTranslation();
  const files = useLogFilesQuery();
  const [file, setFile] = useState("");
  const [level, setLevel] = useState("");
  const [search, setSearch] = useState("");
  const debouncedSearch = useDebounce(search, 400);
  useEffect(() => {
    if (!file && files.data?.[0]) setFile(files.data[0].name);
  }, [file, files.data]);
  const tail = useLogTailQuery(file, level, debouncedSearch);

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row flex-wrap items-stretch sm:items-center justify-between gap-2.5">
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 flex-1 min-w-0">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 w-full sm:w-auto">
            <select
              className="select select-bordered select-sm w-full sm:w-auto"
              value={file}
              onChange={(e) => setFile(e.target.value)}
              aria-label={t("admin.operations.log_file")}
            >
              {(files.data || []).map((item) => (
                <option key={item.name} value={item.name}>
                  {item.name}
                </option>
              ))}
            </select>
            <select
              className="select select-bordered select-sm w-full sm:w-auto"
              value={level}
              onChange={(e) => setLevel(e.target.value)}
              aria-label={t("admin.operations.log_level")}
            >
              <option value="">{t("admin.operations.all_levels")}</option>
              {["debug", "info", "warn", "error", "fatal"].map((item) => (
                <option key={item} value={item}>
                  {item.toUpperCase()}
                </option>
              ))}
            </select>
          </div>

          <div className="relative flex-1 min-w-0 sm:min-w-44">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-base-content/40 pointer-events-none z-10" />
            <input
              className="input input-bordered input-sm w-full pl-8 pr-7"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("admin.operations.search_logs", "Search logs...")}
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearch("")}
                className="absolute right-2 top-1/2 -translate-y-1/2 z-10 p-0.5 text-base-content/40 hover:text-base-content rounded-full cursor-pointer"
                title={t("common.clear", "Clear")}
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
        </div>

        {/* Action buttons: Refresh and Download grouped together */}
        <div className="flex items-center gap-2 shrink-0 w-full sm:w-auto">
          <button
            className="btn btn-sm gap-1.5 flex-1 sm:flex-initial"
            disabled={tail.isFetching}
            onClick={async () => {
              await tail.refetch();
              toast.info(t("common.refreshed", "Data refreshed"));
            }}
            title={t("admin.operations.refresh", "Refresh")}
          >
            <RefreshCw
              className={`w-3.5 h-3.5 ${tail.isFetching ? "animate-spin" : ""}`}
            />
            <span>{t("admin.operations.refresh", "Refresh")}</span>
          </button>
          {file && (
            <a
              className="btn btn-sm btn-outline gap-1.5 flex-1 sm:flex-initial"
              href={operationsService.logDownloadUrl(file)}
            >
              <Download className="w-3.5 h-3.5" />
              <span>{t("common.download", "Download")}</span>
            </a>
          )}
        </div>
      </div>
      <pre className="bg-neutral text-neutral-content rounded-box p-4 h-[60vh] overflow-auto text-xs whitespace-pre-wrap">
        {tail.data?.lines.join("\n") || t("admin.operations.no_logs")}
      </pre>
    </div>
  );
}
