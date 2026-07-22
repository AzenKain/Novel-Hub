import React from "react";
import { Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuthStore, useLibraryStore } from "@/stores";
import { Menu, Search, LayoutDashboard, BarChart3, User, LogOut } from "lucide-react";
import { ThemeController, LanguageSwitcher } from "@/components/ui";

import { isModOrAdminUser } from "@/utils/permission";

interface TopNavProps {
  showSidebarToggle?: boolean;
}

export const TopNav: React.FC<TopNavProps> = ({ showSidebarToggle = false }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { user, setProfileModalOpen, logout, setLoginModalOpen } = useAuthStore();
  const { search, setSearch } = useLibraryStore();

  const handleSearch = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value);
    // Navigate back to home if we are searching from another page
    if (window.location.pathname !== "/") {
      navigate("/");
    }
  };

  return (
    <div className="navbar flex-wrap gap-2 bg-base-100 shadow-sm border-b border-base-200 z-10 px-3 sm:px-4">
      {showSidebarToggle && (
        <div className="flex-none lg:hidden">
          <label
            htmlFor="main-drawer"
            aria-label="open sidebar"
            className="btn btn-square btn-ghost"
          >
            <Menu className="w-5 h-5" />
          </label>
        </div>
      )}

      <div className="min-w-0 flex-1 basis-56 px-1 sm:px-2">
        <div className="form-control relative w-full max-w-md">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <Search className="w-4 h-4 text-base-content/50" />
          </div>
          <input
            type="text"
            placeholder={t(
              "library.search_placeholder",
              "Search title, author, series, tag...",
            )}
            className="input input-bordered input-sm sm:input-md w-full pl-10 bg-base-200/50 focus:bg-base-100 transition-colors"
            value={search}
            onChange={handleSearch}
          />
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-1 sm:gap-2">
        <ThemeController />
        <LanguageSwitcher />

        {user ? (
          <>
            {isModOrAdminUser(user) && (
              <Link
                to="/admin"
                className="btn btn-ghost btn-sm sm:btn-md gap-2 hidden sm:flex"
              >
                <LayoutDashboard className="w-4 h-4" />
                {t("admin.dashboard", "Admin")}
              </Link>
            )}
            <div className="dropdown dropdown-end">
              <div
                tabIndex={0}
                role="button"
                className="btn btn-ghost btn-circle avatar border border-base-300"
              >
                <div className="w-9 rounded-full bg-primary/10 flex items-center justify-center text-primary font-bold">
                  {user.avatar_url ? (
                    <img
                      src={user.avatar_url}
                      alt="Avatar"
                      loading="lazy"
                    />
                  ) : (
                    <span className="text-lg">
                      {user.full_name
                        ? user.full_name.charAt(0).toUpperCase()
                        : user.email.charAt(0).toUpperCase()}
                    </span>
                  )}
                </div>
              </div>
              <ul
                tabIndex={0}
                className="mt-3 z-[1] p-2 shadow menu menu-sm dropdown-content bg-base-100 rounded-box w-52 border border-base-200"
              >
                <li>
                  <span className="font-semibold opacity-60 px-4 py-2 truncate block">
                    {user.email}
                  </span>
                </li>
                <li>
                  <button onClick={() => setProfileModalOpen(true)} className="flex items-center gap-2">
                    <User className="w-4 h-4 opacity-70" />
                    {t("user.profile", "Profile")}
                  </button>
                </li>
                <li>
                  <Link to="/analytics" className="flex items-center gap-2">
                    <BarChart3 className="w-4 h-4 text-primary opacity-80" />
                    {t("analytics.title", "Reading Analytics")}
                  </Link>
                </li>
                <li>
                  <button className="text-error flex items-center gap-2" onClick={logout}>
                    <LogOut className="w-4 h-4 opacity-80" />
                    {t("auth.logout", "Logout")}
                  </button>
                </li>
              </ul>
            </div>
          </>
        ) : (
          <button
            onClick={() => setLoginModalOpen(true)}
            className="btn btn-primary btn-sm sm:btn-md"
          >
            {t("auth.login", "Login")}
          </button>
        )}
      </div>
    </div>
  );
};
