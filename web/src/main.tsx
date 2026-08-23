import { DownloadManagerPanel, ProtectedRoute, UpdatePrompt } from "@/components/common";
import "@/i18n";
import { queryClient } from "@/config/queryClient";
import { QueryClientProvider } from "@tanstack/react-query";
import { useSettingsStore } from "@/stores";
import React, { Suspense, lazy, useEffect } from "react";
import { useShallow } from "zustand/react/shallow";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Navigate, Outlet, Route, Routes, useLocation } from "react-router-dom";
import "./styles.css";

const AdminLayout = lazy(() => import("@/pages/admin").then(m => ({ default: m.AdminLayout })));
const Books = lazy(() => import("@/pages/admin").then(m => ({ default: m.Books })));
const Duplicates = lazy(() => import("@/pages/admin").then(m => ({ default: m.Duplicates })));
const OAuthSettings = lazy(() => import("@/pages/admin").then(m => ({ default: m.OAuthSettings })));
const Operations = lazy(() => import("@/pages/admin").then(m => ({ default: m.Operations })));
const Reviews = lazy(() => import("@/pages/admin").then(m => ({ default: m.Reviews })));
const Roles = lazy(() => import("@/pages/admin").then(m => ({ default: m.Roles })));
const Settings = lazy(() => import("@/pages/admin").then(m => ({ default: m.Settings })));
const Users = lazy(() => import("@/pages/admin").then(m => ({ default: m.Users })));
const ForgotPasswordPage = lazy(() => import("@/pages/auth").then(m => ({ default: m.ForgotPasswordPage })));
const LoginPage = lazy(() => import("@/pages/auth").then(m => ({ default: m.LoginPage })));
const RegisterPage = lazy(() => import("@/pages/auth").then(m => ({ default: m.RegisterPage })));
const SetupWizard = lazy(() => import("@/pages/auth").then(m => ({ default: m.SetupWizard })));
const ActivateMagicCodePage = lazy(() => import("@/pages/auth/ActivateMagicCodePage").then(m => ({ default: m.ActivateMagicCodePage })));
const LibraryWorkspace = lazy(() => import("@/pages/library").then(m => ({ default: m.LibraryWorkspace })));
const ReadListPage = lazy(() => import("@/pages/library").then(m => ({ default: m.ReadListPage })));
const AdvancedSearchPage = lazy(() => import("@/pages/library/AdvancedSearchPage").then(m => ({ default: m.AdvancedSearchPage })));
const PodcastsPage = lazy(() => import("@/pages/podcasts/PodcastsPage").then(m => ({ default: m.PodcastsPage })));
const ReaderWorkspace = lazy(() => import("@/pages/reader").then(m => ({ default: m.ReaderWorkspace })));
const ReadingAnalyticsPage = lazy(() => import("@/pages/user/ReadingAnalyticsPage").then(m => ({ default: m.ReadingAnalyticsPage })));
const OfflineBooksPage = lazy(() => import("@/pages/user/OfflineBooksPage").then(m => ({ default: m.OfflineBooksPage })));
const ProfilePage = lazy(() => import("@/pages/user/ProfilePage").then(m => ({ default: m.ProfilePage })));

function ThemeInitializer({ children }: { children: React.ReactNode }) {
  const { theme, customCss } = useSettingsStore(useShallow((state) => ({
    theme: state.theme,
    customCss: state.customCss,
  })));

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

  useEffect(() => {
    let styleTag = document.getElementById("novelhub-custom-css");
    if (customCss && customCss.trim()) {
      if (!styleTag) {
        styleTag = document.createElement("style");
        styleTag.id = "novelhub-custom-css";
        document.head.appendChild(styleTag);
      }
      styleTag.textContent = customCss;
    } else if (styleTag) {
      styleTag.remove();
    }
  }, [customCss]);

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

import { useCurrentUserQuery, usePodcastDownloadWatcher } from "@/hooks";
import { initOfflineSyncManager } from "@/lib/offlineSyncManager";

function App() {
  const { isLoading: isAuthLoading } = useCurrentUserQuery();
  const booted = useAuthStore((state) => state.booted);
  usePodcastDownloadWatcher();

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
          <Suspense fallback={<div className="flex h-screen items-center justify-center bg-base-100 text-base-content"><span className="loading loading-spinner loading-lg text-primary"></span></div>}>
          <Routes>
            <Route path="/offline" element={<OfflineBooksPage />} />
            <Route path="/offline/reader/:book_id" element={<ReaderWorkspace />} />
            <Route element={<GuestGuard />}>
              <Route path="/" element={<LibraryWorkspace />} />
              <Route path="/search" element={<AdvancedSearchPage />} />
              <Route path="/books/:book_id" element={<LibraryWorkspace />} />
              <Route path="/reader/:book_id" element={<ReaderWorkspace />} />
              <Route path="/read-lists" element={<ReadListPage />} />
              <Route element={<ProtectedRoute />}>
                <Route path="/profile" element={<ProfilePage />} />
              </Route>
              <Route element={<ProtectedRoute requiredPermission="podcast.manage" />}>
                <Route path="/podcasts" element={<PodcastsPage />} />
              </Route>
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
                    <Route path="oauth" element={<OAuthSettings />} />
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
            <Route path="/activate" element={<ActivateMagicCodePage />} />
            <Route path="/setup" element={<SetupWizard />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/forgot-password" element={<ForgotPasswordPage />} />

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
          </Suspense>
        </SetupGuard>
        <ToastContainer
          position="top-right"
          autoClose={3000}
          hideProgressBar
          theme="colored"
        />
        <UpdatePrompt />
        <DownloadManagerPanel />
      </ThemeInitializer>
    </BrowserRouter>
  );
}

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <React.Suspense
        fallback={
          <div className="flex h-screen items-center justify-center bg-base-100 text-base-content/60">
            Loading translations...
          </div>
        }
      >
        <App />
      </React.Suspense>
    </QueryClientProvider>
  </React.StrictMode>,
);


