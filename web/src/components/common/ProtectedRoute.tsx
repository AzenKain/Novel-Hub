import { useAuthStore } from "@/stores";
import { useEffect } from "react";
import { Navigate, Outlet } from "react-router-dom";

interface ProtectedRouteProps {
  requiredRoles?: string[];
  redirectPath?: string;
}

export function ProtectedRoute({ requiredRoles, redirectPath = "/login" }: ProtectedRouteProps) {
  const { user, booted, bootstrap } = useAuthStore();

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

  if (!user) {
    return <Navigate to={redirectPath} replace />;
  }

  if (requiredRoles && requiredRoles.length > 0) {
    const hasRole = user.roles.some((role) => Boolean(role.is_admin) || requiredRoles.includes(role.name.toUpperCase()));
    if (!hasRole) {
      return <Navigate to="/" replace />;
    }
  }

  return <Outlet />;
}
