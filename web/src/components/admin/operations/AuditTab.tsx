import { useAuditActionsQuery, useAuditLogsQuery } from "@/hooks";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { RefreshCw } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "react-toastify";

export function AuditTab() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const actions = useAuditActionsQuery();
  const [action, setAction] = useState("");
  const [cursor, setCursor] = useState("");
  const [history, setHistory] = useState<string[]>([]);
  const logs = useAuditLogsQuery(action, cursor);

  const changeAction = (next: string) => {
    setAction(next);
    setCursor("");
    setHistory([]);
  };
  const nextPage = () => {
    if (!logs.data?.nextCursor) return;
    setHistory((prev) => [...prev, cursor]);
    setCursor(logs.data.nextCursor);
  };
  const prevPage = () => {
    setHistory((prev) => {
      setCursor(prev[prev.length - 1] ?? "");
      return prev.slice(0, -1);
    });
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <select
          className="select select-bordered select-sm"
          value={action}
          onChange={(e) => changeAction(e.target.value)}
          aria-label={t("admin.operations.audit_action")}
        >
          <option value="">{t("admin.operations.all_actions")}</option>
          {(actions.data || []).map((item) => (
            <option key={item} value={item}>
              {item}
            </option>
          ))}
        </select>
        <span className="text-sm opacity-60">
          {t("admin.operations.audit_total", { count: logs.data?.total || 0 })}
        </span>
        <button
          className="btn btn-sm btn-ghost gap-1.5 ml-auto"
          disabled={logs.isFetching}
          onClick={async () => {
            await queryClient.invalidateQueries({ queryKey: ["admin", "audit_logs"] });
            await queryClient.invalidateQueries({ queryKey: ["admin", "audit_actions"] });
            await Promise.all([logs.refetch(), actions.refetch()]);
            toast.info(t("common.refreshed", "Đã làm mới dữ liệu"));
          }}
          title={t("admin.operations.refresh")}
        >
          <RefreshCw className={`w-3.5 h-3.5 ${logs.isFetching ? "animate-spin" : ""}`} />
          <span>{t("admin.operations.refresh")}</span>
        </button>
      </div>
      <div className="overflow-x-auto bg-base-100 rounded-box shadow-sm">
        <table className="table table-sm">
          <thead>
            <tr>
              <th>{t("admin.operations.audit_time")}</th>
              <th>{t("admin.operations.audit_actor")}</th>
              <th>{t("admin.operations.audit_action")}</th>
              <th>{t("admin.operations.audit_target")}</th>
              <th>{t("admin.operations.audit_ip")}</th>
            </tr>
          </thead>
          <tbody>
            {(logs.data?.items || []).map((item) => (
              <tr key={item.id}>
                <td className="whitespace-nowrap">
                  {new Date(item.created_at).toLocaleString()}
                </td>
                <td>{item.actor_email || t("admin.operations.audit_system")}</td>
                <td className="font-mono text-xs">{item.action}</td>
                <td className="max-w-md break-words whitespace-normal text-xs">
                  {item.target_label || item.target_id || item.target_type}
                </td>
                <td className="font-mono text-xs">{item.ip}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {!logs.isLoading && !logs.data?.items.length && (
          <div className="p-6 text-center opacity-60">
            {t("admin.operations.audit_empty")}
          </div>
        )}
      </div>
      <div className="flex justify-end gap-2">
        <button
          className="btn btn-sm"
          disabled={!history.length}
          onClick={prevPage}
        >
          {t("common.previous")}
        </button>
        <button
          className="btn btn-sm"
          disabled={!logs.data?.nextCursor}
          onClick={nextPage}
        >
          {t("common.next")}
        </button>
      </div>
    </div>
  );
}
