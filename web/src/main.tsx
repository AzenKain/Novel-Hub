import { ProtectedRoute } from "@/components/common";
import "@/i18n";
import { AdminLayout, Books, Duplicates, Reviews, Roles, Settings, Users } from "@/pages/admin";
import { RegisterPage, SetupWizard } from "@/pages/auth";
import { LibraryWorkspace } from "@/pages/library";
import { ReaderWorkspace } from "@/pages/reader";
import { ReadingAnalyticsPage } from "@/pages/user/ReadingAnalyticsPage";
import { useSettingsStore } from "@/stores";
import React, { useEffect } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Navigate, Route, Routes, useLocation } from "react-router-dom";
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

import { useCurrentUserQuery } from "@/hooks";

function App() {
  const { isLoading: isAuthLoading } = useCurrentUserQuery();
  const booted = useAuthStore((state) => state.booted);

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
            <Route path="/" element={<LibraryWorkspace />} />
            <Route path="/books/:bookId" element={<LibraryWorkspace />} />
            <Route path="/reader/:bookId" element={<ReaderWorkspace />} />
            <Route path="/analytics" element={<ReadingAnalyticsPage />} />
            <Route path="/setup" element={<SetupWizard />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route
              path="/admin"
              element={<ProtectedRoute requiredRoles={["ADMIN", "MOD"]} />}
            >
              <Route element={<AdminLayout />}>
                <Route index element={<Navigate to="books" replace />} />
                <Route path="users" element={<Users />} />
                <Route path="roles" element={<Roles />} />
                <Route path="books" element={<Books />} />
                <Route path="settings" element={<Settings />} />
                <Route path="duplicates" element={<Duplicates />} />
                <Route path="reviews" element={<Reviews />} />
              </Route>
            </Route>

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </SetupGuard>
        <ToastContainer
          position="top-right"
          autoClose={3000}
          hideProgressBar
          theme="colored"
        />
      </ThemeInitializer>
    </BrowserRouter>
  );
}

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5,
      refetchOnWindowFocus: false,
    },
  },
});

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

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.getRegistrations().then(function (registrations) {
    for (let registration of registrations) {
      registration.unregister();
    }
  }).catch(function (err) {
    console.log('Service Worker unregistration failed: ', err);
  });
}
