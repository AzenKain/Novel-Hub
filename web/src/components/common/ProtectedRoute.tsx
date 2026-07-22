import { useAuthStore } from "@/stores";
import { isAdminUser, isBannedUser } from "@/utils/permission";
import { useEffect } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";

interface ProtectedRouteProps {
  requiredRoles?: string[];
  redirectPath?: string;
}

export function ProtectedRoute({ requiredRoles, redirectPath = "/login" }: ProtectedRouteProps) {
  const { user, booted, bootstrap } = useAuthStore(
    useShallow((state) => ({
      user: state.user,
      booted: state.booted,
      bootstrap: state.bootstrap,
    }))
  );

  useEffect(() => {
    if (!booted) {
      bootstrap();
    }
  }, [booted, bootstrap]);

  if (!booted) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div style={{ color: "var(--muted)" }}>Loading...</div>
      </div>
    );
  }

  if (!user || isBannedUser(user)) {
    return <Navigate to={redirectPath} replace />;
  }

  if (requiredRoles && requiredRoles.length > 0) {
    const hasRole = isAdminUser(user) || user.roles.some((role) => requiredRoles.includes(role.name.toUpperCase()));
    if (!hasRole) {
      return <Navigate to="/" replace />;
    }
  }

  return <Outlet />;
}
