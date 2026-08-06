import { ProtectedRoute, UpdatePrompt } from "@/components/common";
import "@/i18n";
import { AdminLayout, Books, Duplicates, Operations, Reviews, Roles, Settings, Users } from "@/pages/admin";
import { ForgotPasswordPage, LoginPage, RegisterPage, SetupWizard } from "@/pages/auth";
import { LibraryWorkspace } from "@/pages/library";
import { ReaderWorkspace } from "@/pages/reader";
import { ReadingAnalyticsPage } from "@/pages/user/ReadingAnalyticsPage";
import { OfflineBooksPage } from "@/pages/user/OfflineBooksPage";
import { useSettingsStore } from "@/stores";
import React, { useEffect } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Navigate, Outlet, Route, Routes, useLocation } from "react-router-dom";
import "./styles.css";

function ThemeInitializer({ children }: { children: React.ReactNode }) {
  const theme = useSettingsStore((state) => state.theme);

  useEffect(() => {
    const root = window.document.documentElement;

    if (theme === "system") {
      const systemTheme = window.matchMedia("(prefers-color-scheme: dark)")
        .matches
        ? "night"
        : "winter";
      root.setAttribute("data-theme", systemTheme);

      const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
      const handleChange = (e: MediaQueryListEvent) => {
        root.setAttribute("data-theme", e.matches ? "night" : "winter");
      };

      mediaQuery.addEventListener("change", handleChange);
      return () => mediaQuery.removeEventListener("change", handleChange);
    } else {
      root.setAttribute("data-theme", theme);
    }
  }, [theme]);

  return <>{children}</>;
}

import { usePublicSettings } from "@/hooks/useSettings";
import { useAuthStore } from "@/stores";

import { ToastContainer } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";

function SetupGuard({ children }: { children: React.ReactNode }) {
  const settings = usePublicSettings();
  const location = useLocation();

  if (settings) {
    if (settings.setup_completed && location.pathname === "/setup") {
      return <Navigate to="/" replace />;
    }
    if (!settings.setup_completed && location.pathname !== "/setup") {
      return <Navigate to="/setup" replace />;
    }
  }
  return <>{children}</>;
}

function GuestGuard() {
  const settings = usePublicSettings();
  const user = useAuthStore((state) => state.user);

  if (settings && settings.guest_login_required && !user) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}

import { useCurrentUserQuery } from "@/hooks";
import { initOfflineSyncManager } from "@/lib/offlineSyncManager";

function App() {
  const { isLoading: isAuthLoading } = useCurrentUserQuery();
  const booted = useAuthStore((state) => state.booted);

  useEffect(() => {
    return initOfflineSyncManager();
  }, []);

  if (!booted || isAuthLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-base-100 text-base-content">
        <span className="loading loading-spinner loading-lg text-primary"></span>
      </div>
    );
  }

  return (
    <BrowserRouter>
      <ThemeInitializer>
        <SetupGuard>
          <Routes>
            <Route path="/offline" element={<OfflineBooksPage />} />
            <Route path="/offline/reader/:book_id" element={<ReaderWorkspace />} />
            <Route element={<GuestGuard />}>
              <Route path="/" element={<LibraryWorkspace />} />
              <Route path="/books/:book_id" element={<LibraryWorkspace />} />
              <Route path="/reader/:book_id" element={<ReaderWorkspace />} />
              <Route element={<ProtectedRoute requiredPermission="user.stats.read" />}>
                <Route path="/analytics" element={<ReadingAnalyticsPage />} />
              </Route>
              <Route
                path="/admin"
                element={<ProtectedRoute requiredAnyPermissions={["admin.access", "job.read", "job.manage", "system.log.read", "system.backup"]} />}
              >
                <Route element={<AdminLayout />}>
                  <Route index element={<Navigate to="books" replace />} />
                  <Route element={<ProtectedRoute requiredPermission="user.manage" />}>
                    <Route path="users" element={<Users />} />
                  </Route>
                  <Route element={<ProtectedRoute requiredPermission="role.manage" />}>
                    <Route path="roles" element={<Roles />} />
                  </Route>
                  <Route element={<ProtectedRoute requiredAnyPermissions={["book.upload", "book.edit", "book.delete", "book.bulk.manage", "library.manage"]} />}>
                    <Route path="books" element={<Books />} />
                  </Route>
                  <Route element={<ProtectedRoute requiredPermission="setting.manage" />}>
                    <Route path="settings" element={<Settings />} />
                  </Route>
                  <Route element={<ProtectedRoute requiredPermission="book.duplicate.manage" />}>
                    <Route path="duplicates" element={<Duplicates />} />
                  </Route>
                  <Route element={<ProtectedRoute requiredAnyPermissions={["job.read", "job.manage", "system.log.read", "system.backup"]} />}>
                    <Route path="operations" element={<Operations />} />
                  </Route>
                  <Route element={<ProtectedRoute requiredPermission="book.review.delete" />}>
                    <Route path="reviews" element={<Reviews />} />
                  </Route>
                </Route>
              </Route>
            </Route>

            <Route path="/login" element={<LoginPage />} />
            <Route path="/setup" element={<SetupWizard />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/forgot-password" element={<ForgotPasswordPage />} />

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </SetupGuard>
        <ToastContainer
          position="top-right"
          autoClose={3000}
          hideProgressBar
          theme="colored"
        />
        <UpdatePrompt />
      </ThemeInitializer>
    </BrowserRouter>
  );
}

import { queryClient } from "@/config/queryClient";
import { QueryClientProvider } from "@tanstack/react-query";

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <React.Suspense
        fallback={
          <div className="flex h-screen items-center justify-center bg-gray-50 dark:bg-gray-950 text-gray-500">
            Loading translations...
          </div>
        }
      >
        <App />
      </React.Suspense>
    </QueryClientProvider>
  </React.StrictMode>,
);


