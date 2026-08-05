import { AuditTab } from "@/components/admin/operations/AuditTab";
import { BackupsTab } from "@/components/admin/operations/BackupsTab";
import { JobsTab } from "@/components/admin/operations/JobsTab";
import { LogsTab } from "@/components/admin/operations/LogsTab";
import { SchedulesTab } from "@/components/admin/operations/SchedulesTab";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";
import { Archive, CalendarClock, ListTodo, ScrollText, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

export function Operations() {
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.user);
  const canReadJobs = hasPermission(user, "job.read");
  const canManageJobs = hasPermission(user, "job.manage");
  const canReadLogs = hasPermission(user, "system.log.read");
  const canBackup = hasPermission(user, "system.backup");
  const tabs = [
    ...(canReadJobs ? [{ id: "jobs", icon: ListTodo }] : []),
    ...(canReadJobs || canManageJobs ? [{ id: "schedules", icon: CalendarClock }] : []),
    ...(canReadLogs ? [{ id: "logs", icon: ScrollText }] : []),
    ...(canBackup ? [{ id: "backups", icon: Archive }] : []),
    ...(canReadLogs ? [{ id: "audit", icon: ShieldCheck }] : []),
  ];
  const [active, setActive] = useState(tabs[0]?.id || "jobs");

  return <div className="p-4 md:p-8 max-w-7xl mx-auto space-y-6">
    <div><h1 className="text-3xl font-bold">{t("admin.operations.title")}</h1><p className="opacity-60">{t("admin.operations.description")}</p></div>
    <div role="tablist" className="tabs tabs-boxed w-fit">{tabs.map((tab) => { const Icon = tab.icon; return <button key={tab.id} role="tab" className={`tab gap-2 ${active === tab.id ? "tab-active" : ""}`} onClick={() => setActive(tab.id)}><Icon className="w-4 h-4" />{t(`admin.operations.tabs.${tab.id}`)}</button>; })}</div>
    {active === "jobs" && canReadJobs && <JobsTab />}
    {active === "schedules" && (canReadJobs || canManageJobs) && <SchedulesTab />}
    {active === "logs" && canReadLogs && <LogsTab />}
    {active === "backups" && canBackup && <BackupsTab />}
    {active === "audit" && canReadLogs && <AuditTab />}
  </div>;
}
