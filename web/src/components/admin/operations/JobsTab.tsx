import {
  useCacheStatsQuery,
  useJobTasksQuery,
  useJobsQuery,
  useTriggerJobMutation,
  useLibrariesQuery,
} from "@/hooks";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";
import { libraryService } from "@/services/libraryService";
import { copyText } from "@/utils/clipboard";
import {
  Activity,
  Archive,
  CheckCheck,
  ChevronDown,
  Copy,
  Database,
  FileText,
  FolderMinus,
  FolderSync,
  ListTodo,
  RefreshCw,
  Trash2,
  Wrench,
  Zap,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";
import { useQueryClient } from "@tanstack/react-query";

function getTaskIcon(type: string) {
  switch (type) {
    case "full_maintenance":
      return <Wrench className="w-4 h-4 text-primary shrink-0" />;
    case "scan_library_inbox":
      return <FolderSync className="w-4 h-4 text-info shrink-0" />;
    case "delete_empty_book_dirs":
      return <FolderMinus className="w-4 h-4 text-warning shrink-0" />;
    case "delete_old_uploads":
      return <Trash2 className="w-4 h-4 text-error shrink-0" />;
    case "cleanup_finished_jobs":
      return <CheckCheck className="w-4 h-4 text-success shrink-0" />;
    case "cleanup_audit_logs":
      return <FileText className="w-4 h-4 text-secondary shrink-0" />;
    case "database_health_check":
      return <Activity className="w-4 h-4 text-accent shrink-0" />;
    case "backup_database":
      return <Database className="w-4 h-4 text-primary shrink-0" />;
    case "backup_database_books":
      return <Archive className="w-4 h-4 text-primary shrink-0" />;
    default:
      return <Zap className="w-4 h-4 text-base-content/70 shrink-0" />;
  }
}

export function JobsTab() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [status, setStatus] = useState("");
  const [type, setType] = useState("");
  const jobs = useJobsQuery(status, type);
  const tasks = useJobTasksQuery();
  const cacheStats = useCacheStatsQuery();
  const trigger = useTriggerJobMutation();
  const user = useAuthStore((state) => state.user);
  const canManage = hasPermission(user, "job.manage");

  const [showInboxModal, setShowInboxModal] = useState(false);
  const [selectedInboxLib, setSelectedInboxLib] = useState("");
  const [setupInboxPath, setSetupInboxPath] = useState("");
  const [settingUpInbox, setSettingUpInbox] = useState(false);
  const [copiedInboxPath, setCopiedInboxPath] = useState(false);

  const { data: libsData } = useLibrariesQuery();

  const handleSetupInbox = async () => {
    if (!selectedInboxLib) return;
    setSettingUpInbox(true);
    setSetupInboxPath("");
    try {
      const res = await libraryService.setupLibraryInbox(selectedInboxLib);
      if (res.status && res.data) {
        setSetupInboxPath(res.data);
        toast.success(
          t(
            "admin.operations.inbox_setup_success",
            "Inbox folder setup successfully!",
          ),
        );
      } else {
        toast.error(
          res.message ||
            t(
              "admin.operations.inbox_setup_failed",
              "Failed to setup inbox folder.",
            ),
        );
      }
    } catch (err) {
      toast.error(t("admin.operations.action_failed", "Action failed"));
    } finally {
      setSettingUpInbox(false);
    }
  };

  const handleCopyInboxPath = () => {
    if (!setupInboxPath) return;
    copyText(setupInboxPath).then((success) => {
      if (success) {
        setCopiedInboxPath(true);
        toast.success(t("common.copied", "Copied to clipboard"));
        setTimeout(() => setCopiedInboxPath(false), 2000);
      }
    });
  };

  const run = (taskType: string) =>
    trigger.mutate(taskType, {
      onSuccess: (res) =>
        res.status
          ? toast.success(t("admin.operations.job_queued"))
          : toast.error(res.message),
      onError: () => toast.error(t("admin.operations.action_failed")),
    });

  return (
    <div className="space-y-3">
      {cacheStats.data && (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
          <div className="bg-base-100 border border-base-200 p-3 rounded-2xl shadow-2xs">
            <span className="text-xs text-base-content/60 font-medium">
              {t("admin.operations.cache_hit_rate")}
            </span>
            <div className="text-lg font-bold text-success">
              {(cacheStats.data.hit_rate * 100).toFixed(1)}%
            </div>
            <span className="text-[10px] text-base-content/40 font-mono">
              {t("admin.operations.cache_hits_misses", {
                hits: cacheStats.data.hits.toLocaleString(),
                misses: cacheStats.data.misses.toLocaleString(),
              })}
            </span>
          </div>
          <div className="bg-base-100 border border-base-200 p-3 rounded-2xl shadow-2xs">
            <span className="text-xs text-base-content/60 font-medium">
              {t("admin.operations.cached_entities")}
            </span>
            <div className="text-lg font-bold text-primary">
              {cacheStats.data.entry_count.toLocaleString()}
            </div>
            <span className="text-[10px] text-base-content/40">
              {t("admin.operations.active_ram_entries")}
            </span>
          </div>
          <div className="bg-base-100 border border-base-200 p-3 rounded-2xl shadow-2xs">
            <span className="text-xs text-base-content/60 font-medium">
              {t("admin.operations.ram_budget")}
            </span>
            <div className="text-lg font-bold text-info">
              {(cacheStats.data.max_cost / (1024 * 1024)).toFixed(0)} MB
            </div>
            <span className="text-[10px] text-base-content/40">
              Theine-Go MaxCost
            </span>
          </div>
          <div className="bg-base-100 border border-base-200 p-3 rounded-2xl shadow-2xs">
            <span className="text-xs text-base-content/60 font-medium">
              {t("admin.operations.singleflight_guard")}
            </span>
            <div className="text-lg font-bold text-accent">
              {t("common.active")}
            </div>
            <span className="text-[10px] text-base-content/40">
              {t("admin.operations.stampede_protection")}
            </span>
          </div>
        </div>
      )}
      {canManage && (
        <div className="border border-base-200 bg-base-100 rounded-2xl p-3 sm:p-3.5 shadow-2xs space-y-2">
          <div className="flex items-center justify-between border-b border-base-200/60 pb-2">
            <div className="flex items-center gap-2">
              <Wrench className="w-4 h-4 text-primary" />
              <h3 className="font-bold text-xs uppercase tracking-wider text-base-content/80">
                {t("admin.operations.run_task", "Run Maintenance Task")}
              </h3>
            </div>
            <button
              onClick={() => setShowInboxModal(true)}
              className="btn btn-xs btn-primary btn-outline gap-1 rounded-lg font-bold"
            >
              <FolderSync className="w-3.5 h-3.5" />
              {t("admin.operations.setup_inbox", "Setup Inbox Folder")}
            </button>
          </div>

          <div className="flex flex-wrap items-center gap-1.5 sm:gap-2">
            {(tasks.data || []).map((task) => (
              <button
                key={task.type}
                className="btn btn-sm bg-base-100 border border-base-200 hover:border-primary/40 hover:bg-primary/5 hover:text-primary rounded-xl gap-1.5 font-medium text-xs text-base-content shadow-2xs transition-all"
                disabled={trigger.isPending}
                onClick={() => run(task.type)}
                title={task.description}
              >
                {getTaskIcon(task.type)}
                <span>{t(`admin.operations.tasks.${task.type}`)}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* 1-Row Filter Toolbar: Mọi trạng thái, Mọi tác vụ & Làm mới */}
      <div className="flex items-center justify-between gap-2 w-full">
        <div className="flex items-center gap-2 flex-nowrap min-w-0">
          <select
            className="select select-bordered select-sm bg-base-100 rounded-xl text-xs h-8 min-w-30 max-w-40"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            aria-label={t("admin.operations.status")}
          >
            <option value="">{t("admin.operations.all_statuses")}</option>
            {["pending", "running", "completed", "failed"].map((item) => (
              <option key={item} value={item}>
                {t(`admin.operations.statuses.${item}`)}
              </option>
            ))}
          </select>

          <select
            className="select select-bordered select-sm bg-base-100 rounded-xl text-xs h-8 min-w-32.5 max-w-45"
            value={type}
            onChange={(e) => setType(e.target.value)}
            aria-label={t("admin.operations.task")}
          >
            <option value="">{t("admin.operations.all_tasks")}</option>
            {(tasks.data || []).map((task) => (
              <option key={task.type} value={task.type}>
                {t(`admin.operations.tasks.${task.type}`)}
              </option>
            ))}
          </select>
        </div>

        <button
          className="btn btn-sm btn-ghost gap-1.5 text-xs h-8 shrink-0 px-2 sm:px-3"
          disabled={jobs.isFetching}
          onClick={async () => {
            await queryClient.invalidateQueries({
              queryKey: ["operations", "jobs"],
            });
            await queryClient.invalidateQueries({
              queryKey: ["operations", "tasks"],
            });
            await queryClient.invalidateQueries({
              queryKey: ["operations", "cache-stats"],
            });
            await Promise.all([
              jobs.refetch(),
              tasks.refetch(),
              cacheStats.refetch(),
            ]);
            toast.info(t("common.refreshed", "Data refreshed"));
          }}
          title={t("admin.operations.refresh")}
        >
          <RefreshCw
            className={`w-3.5 h-3.5 ${jobs.isFetching ? "animate-spin" : ""}`}
          />
          <span className="hidden sm:inline">
            {t("admin.operations.refresh")}
          </span>
        </button>
      </div>
      <div className="overflow-x-auto bg-base-100 rounded-box shadow-sm">
        <table className="table table-sm">
          <thead>
            <tr>
              <th>{t("admin.operations.task")}</th>
              <th>{t("admin.operations.status")}</th>
              <th>{t("admin.operations.created")}</th>
              <th>{t("admin.operations.error")}</th>
            </tr>
          </thead>
          <tbody>
            {(jobs.data?.items || []).map((job) => (
              <tr key={job.id}>
                <td>
                  <div className="font-medium">
                    {t(`admin.operations.tasks.${job.type}`)}
                  </div>
                  <div className="text-xs opacity-50 font-mono">{job.id}</div>
                </td>
                <td>
                  <span
                    className={`badge badge-sm ${job.status === "completed" ? "badge-success" : job.status === "failed" ? "badge-error" : job.status === "running" ? "badge-info" : "badge-warning"}`}
                  >
                    {t(`admin.operations.statuses.${job.status || "pending"}`)}
                  </span>
                </td>
                <td>{new Date(job.created_at).toLocaleString()}</td>
                <td className="max-w-xs truncate text-error">
                  {job.errorMsg || "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!jobs.isLoading && !jobs.data?.items.length && (
          <div className="rounded-2xl border border-dashed border-base-300 bg-base-100 p-12 text-center flex flex-col items-center justify-center gap-3 my-4 mx-4 shadow-xs">
            <div className="grid h-12 w-12 place-items-center rounded-2xl bg-primary/10 text-primary">
              <ListTodo className="h-6 w-6" />
            </div>
            <div>
              <p className="font-bold text-base text-base-content/80">
                {t("admin.operations.no_jobs", "No Background Tasks Found")}
              </p>
              <p className="text-xs text-base-content/50 mt-1">
                {t(
                  "admin.operations.no_jobs_hint",
                  "Tasks triggered manually or scheduled automatically will appear here.",
                )}
              </p>
            </div>
          </div>
        )}
      </div>

      {showInboxModal && (
        <div className="modal modal-open">
          <div className="modal-box rounded-3xl border border-base-200 shadow-xl bg-base-100 max-w-lg font-sans">
            <h3 className="font-bold text-lg">
              {t("admin.operations.setup_inbox", "Setup Inbox Folder")}
            </h3>
            <p className="text-xs text-base-content/60 mt-1">
              {t(
                "admin.operations.setup_inbox_desc",
                "Select a library to verify or create its dedicated /inbox/<library_id> folder for background scanning.",
              )}
            </p>

            <div className="mt-4 space-y-4">
              <div className="form-control">
                <label className="label">
                  <span className="label-text font-bold text-xs">
                    {t("admin.operations.select_library", "Select Library")}
                  </span>
                </label>
                <select
                  className="select select-bordered w-full rounded-xl bg-base-100"
                  value={selectedInboxLib}
                  onChange={(e) => {
                    setSelectedInboxLib(e.target.value);
                    setSetupInboxPath("");
                  }}
                >
                  <option value="">
                    -- {t("admin.operations.choose_library", "Choose Library")}{" "}
                    --
                  </option>
                  {(libsData || []).map((lib) => (
                    <option key={lib.id} value={lib.id}>
                      {lib.name} ({lib.id})
                    </option>
                  ))}
                </select>
              </div>

              {setupInboxPath && (
                <div className="bg-base-200/50 p-3 rounded-2xl border border-base-200 space-y-1.5">
                  <span className="text-[10px] font-black uppercase tracking-wider text-base-content/60 block">
                    {t("admin.operations.inbox_path", "Inbox Folder Path")}
                  </span>
                  <div className="flex items-center gap-2 bg-base-100 p-2 rounded-xl border border-base-200">
                    <span className="text-xs font-mono break-all select-all flex-1">
                      {setupInboxPath}
                    </span>
                    <button
                      onClick={handleCopyInboxPath}
                      className="btn btn-ghost btn-xs h-7 w-7 p-0 rounded-lg text-primary shrink-0"
                      title={t("common.copy", "Copy")}
                    >
                      {copiedInboxPath ? (
                        <CheckCheck className="w-3.5 h-3.5" />
                      ) : (
                        <Copy className="w-3.5 h-3.5" />
                      )}
                    </button>
                  </div>
                </div>
              )}
            </div>

            <div className="modal-action gap-2">
              <button
                className="btn btn-ghost rounded-xl btn-sm"
                onClick={() => {
                  setShowInboxModal(false);
                  setSelectedInboxLib("");
                  setSetupInboxPath("");
                }}
              >
                {t("common.close", "Close")}
              </button>
              <button
                className={`btn btn-primary rounded-xl btn-sm ${settingUpInbox ? "loading" : ""}`}
                disabled={!selectedInboxLib || settingUpInbox}
                onClick={handleSetupInbox}
              >
                {t("admin.operations.setup", "Setup")}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
