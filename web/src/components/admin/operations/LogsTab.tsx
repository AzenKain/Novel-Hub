import { useLogFilesQuery, useLogTailQuery } from "@/hooks";
import { useDebounce } from "@/hooks/useDebounce";
import { operationsService } from "@/services";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { RefreshCw } from "lucide-react";
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
      <div className="flex flex-wrap gap-2">
        <select
          className="select select-bordered select-sm"
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
          className="select select-bordered select-sm"
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
        <input
          className="input input-bordered input-sm"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t("admin.operations.search_logs")}
        />
        <button
          className="btn btn-sm gap-1.5"
          disabled={tail.isFetching}
          onClick={async () => {
            await tail.refetch();
            toast.info(t("common.refreshed", "Data refreshed"));
          }}
        >
          <RefreshCw className={`w-3.5 h-3.5 ${tail.isFetching ? "animate-spin" : ""}`} />
          {t("admin.operations.refresh")}
        </button>
        {file && (
          <a
            className="btn btn-sm btn-outline"
            href={operationsService.logDownloadUrl(file)}
          >
            {t("common.download")}
          </a>
        )}
      </div>
      <pre className="bg-neutral text-neutral-content rounded-box p-4 h-[60vh] overflow-auto text-xs whitespace-pre-wrap">
        {tail.data?.lines.join("\n") || t("admin.operations.no_logs")}
      </pre>
    </div>
  );
}
