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

  return (
    <div className="flex flex-col h-full bg-base-100 font-sans">
      {/* Sticky Header */}
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("admin.operations.title")}</h1>
          <p className="text-sm text-base-content/60 mt-1">{t("admin.operations.description")}</p>
        </div>
      </header>

      {/* Main Content Area */}
      <div className="flex-1 overflow-auto p-4 sm:p-6 lg:p-8">
        <div className="max-w-7xl mx-auto w-full space-y-3.5">
          {/* Sub-tabs Navigation */}
          <div role="tablist" className="flex items-center gap-1.5 bg-base-200/50 p-1.5 rounded-2xl border border-base-200 w-fit flex-wrap">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              const isActive = active === tab.id;
              return (
                <button
                  key={tab.id}
                  role="tab"
                  className={`flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-bold transition-all ${
                    isActive
                      ? "bg-base-100 text-primary shadow-sm border border-base-200"
                      : "text-base-content/70 hover:text-base-content hover:bg-base-100/50"
                  }`}
                  onClick={() => setActive(tab.id)}
                >
                  <Icon className="w-4 h-4" />
                  {t(`admin.operations.tabs.${tab.id}`)}
                </button>
              );
            })}
          </div>

          {/* Active Tab Content */}
          {active === "jobs" && canReadJobs && <JobsTab />}
          {active === "schedules" && (canReadJobs || canManageJobs) && <SchedulesTab />}
          {active === "logs" && canReadLogs && <LogsTab />}
          {active === "backups" && canBackup && <BackupsTab />}
          {active === "audit" && canReadLogs && <AuditTab />}
        </div>
      </div>
    </div>
  );
}
