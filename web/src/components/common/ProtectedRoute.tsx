import { useAuthStore } from "@/stores";
import type { ProtectedRouteProps } from "@/types";
import { hasPermission, isAdminUser, isBannedUser } from "@/utils/permission";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Navigate, Outlet } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";

export function ProtectedRoute({
  requiredRoles,
  requiredPermission,
  requiredAnyPermissions,
  redirectPath = "/login",
}: ProtectedRouteProps) {
  const { user, booted, bootstrap } = useAuthStore(
    useShallow((state) => ({
      user: state.user,
      booted: state.booted,
      bootstrap: state.bootstrap,
    }))
  );

  const { t } = useTranslation();

  useEffect(() => {
    if (!booted) {
      bootstrap();
    }
  }, [booted, bootstrap]);

  if (!booted) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div style={{ color: "var(--muted)" }}>{t("common.loading")}</div>
      </div>
    );
  }

  if (!user || isBannedUser(user)) {
    return <Navigate to={redirectPath} replace />;
  }

  const hasRequiredRole = !requiredRoles?.length || isAdminUser(user) || user.roles.some(
    (role) => typeof role?.name === "string" && requiredRoles.includes(role.name.toUpperCase()),
  );
  const hasRequiredPermission = !requiredPermission || hasPermission(user, requiredPermission);
  const hasAnyRequiredPermission = !requiredAnyPermissions?.length || requiredAnyPermissions.some(
    (permission) => hasPermission(user, permission),
  );

  if (!hasRequiredRole || !hasRequiredPermission || !hasAnyRequiredPermission) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
