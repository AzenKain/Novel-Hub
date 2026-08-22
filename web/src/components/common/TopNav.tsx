import React, { useState, useEffect, useRef } from "react";
import { Link, useNavigate, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuthStore, useLibraryStore } from "@/stores";
import { Menu, Search, LayoutDashboard, BarChart3, User, LogOut, CloudDownload, ListOrdered, X, BookOpen, ChevronRight, Loader2, Podcast } from "lucide-react";
import { ThemeController, LanguageSwitcher } from "@/components/ui";
import { useShallow } from "zustand/react/shallow";
import { hasPermission } from "@/utils/permission";
import { metadataNavIds } from "@/lib/libraryMetadata";
import { LoginView } from "./LoginView";
import { RegisterView } from "./RegisterView";
import { useDebounce } from "@/hooks";
import { bookService } from "@/services";
import { getMediaUrl } from "@/config/api";
import { useQuery } from "@tanstack/react-query";
import type { Book } from "@/types";

interface TopNavProps {
  showSidebarToggle?: boolean;
  hideAuthButtons?: boolean;
}

export const TopNav: React.FC<TopNavProps> = ({ showSidebarToggle = false, hideAuthButtons = false }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const isAuthPage = hideAuthButtons || ["/login", "/register", "/forgot-password"].includes(location.pathname);
  const { user, setProfileModalOpen, logout, setLoginModalOpen } = useAuthStore(
    useShallow((state) => ({
      user: state.user,
      setProfileModalOpen: state.setProfileModalOpen,
      logout: state.logout,
      setLoginModalOpen: state.setLoginModalOpen,
    }))
  );
  const { search, setSearch, activeNav, setActiveNav, activeFacet, setActiveFacet } = useLibraryStore(
    useShallow((state) => ({
      search: state.search,
      setSearch: state.setSearch,
      activeNav: state.activeNav,
      setActiveNav: state.setActiveNav,
      activeFacet: state.activeFacet,
      setActiveFacet: state.setActiveFacet,
    }))
  );

  const [localQuery, setLocalQuery] = useState(search || "");
  const [popoverOpen, setPopoverOpen] = useState(false);
  const searchContainerRef = useRef<HTMLDivElement>(null);

  const debouncedQuery = useDebounce(localQuery, 300);

  useEffect(() => {
    setLocalQuery(search || "");
  }, [search]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (searchContainerRef.current && !searchContainerRef.current.contains(e.target as Node)) {
        setPopoverOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const isOffline = location.pathname.startsWith("/offline");

  const { data: quickResultsData, isLoading: isQuickLoading } = useQuery<Book[]>({
    queryKey: ["quickSearch", debouncedQuery],
    queryFn: async () => {
      if (!debouncedQuery.trim()) return [];
      const res = await bookService.getBooks({ search: debouncedQuery.trim(), limit: 5 });
      return res.data || [];
    },
    enabled: debouncedQuery.trim().length > 0 && !isOffline,
    staleTime: 30000,
  });

  const submitSearch = (queryToSubmit?: string) => {
    const q = (queryToSubmit !== undefined ? queryToSubmit : localQuery).trim();
    setSearch(q);
    if (activeFacet) {
      setActiveFacet(null);
    }
    setPopoverOpen(false);

    if (!isOffline) {
      if (q) {
        navigate(`/search?q=${encodeURIComponent(q)}`);
      } else {
        navigate(`/search`);
      }
    }
  };

  const clearFacet = () => {
    setActiveFacet(null);
    const params = new URLSearchParams(location.search);
    params.delete("facet");
    params.delete("facet_id");
    params.delete("name");
    params.delete("facet_name");
    const newQuery = params.toString();
    navigate(`/${newQuery ? `?${newQuery}` : ""}`, { replace: true });
  };

  const clearSearch = () => {
    setLocalQuery("");
    setSearch("");
    setPopoverOpen(false);
    const params = new URLSearchParams(location.search);
    params.delete("search");
    params.delete("q");
    const newQuery = params.toString();
    if (!isOffline) {
      navigate(`/${newQuery ? `?${newQuery}` : ""}`, { replace: true });
    }
  };

  return (
    <div className="navbar flex-wrap gap-2 bg-base-100 shadow-sm border-b border-base-200 z-10 px-3 sm:px-4">
      {showSidebarToggle && (
        <div className="flex-none lg:hidden">
          <div className="tooltip tooltip-bottom" data-tip={t("common.toggle_sidebar", "Toggle Sidebar")}>
            <label
              htmlFor="main-drawer"
              aria-label="open sidebar"
              className="btn btn-square btn-ghost"
            >
              <Menu className="w-5 h-5" />
            </label>
          </div>
        </div>
      )}

      <div className="min-w-0 flex-1 basis-56 px-1 sm:px-2">
        <div ref={searchContainerRef} className="form-control relative w-full max-w-md">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none z-10">
            <Search className="w-4 h-4 text-base-content/50" />
          </div>
          {activeFacet ? (
            <div className="input input-bordered input-sm sm:input-md w-full pl-10 pr-3 flex items-center justify-between bg-primary/10 border-primary/40">
              <div className="flex items-center gap-1.5 min-w-0 overflow-hidden">
                <span className="badge badge-primary badge-sm font-bold uppercase shrink-0">
                  {t(`library.facets.${activeFacet.type}`, activeFacet.type)}
                </span>
                <span className="text-xs sm:text-sm font-bold truncate text-base-content" title={activeFacet.name}>
                  {activeFacet.name}
                </span>
              </div>
              <button
                type="button"
                onClick={clearFacet}
                className="btn btn-ghost btn-circle btn-xs text-base-content/60 hover:text-error shrink-0"
                aria-label={t("common.clear", "Clear filter")}
                title={t("common.clear", "Clear filter")}
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          ) : (
            <form
              onSubmit={(e) => {
                e.preventDefault();
                submitSearch();
              }}
              className="w-full relative"
            >
              <input
                type="text"
                placeholder={t(
                  "library.search_placeholder",
                  "Search title, author, series, tag...",
                )}
                className="input input-bordered input-sm sm:input-md w-full pl-10 pr-9 bg-base-200/50 focus:bg-base-100 transition-colors"
                value={localQuery}
                onFocus={() => {
                  if (localQuery.trim() && !isOffline) setPopoverOpen(true);
                }}
                onChange={(e) => {
                  const val = e.target.value;
                  setLocalQuery(val);
                  if (isOffline) {
                    setSearch(val);
                  } else {
                    setPopoverOpen(val.trim().length > 0);
                  }
                }}
                onKeyDown={(e) => {
                  if (e.key === "Escape") {
                    clearSearch();
                    (e.target as HTMLInputElement).blur();
                  }
                }}
              />
              {!activeFacet && localQuery && (
                <button
                  type="button"
                  onClick={clearSearch}
                  className="absolute inset-y-0 right-0 pr-3 flex items-center text-base-content/40 hover:text-base-content transition-colors"
                  aria-label={t("common.clear", "Clear")}
                >
                  <X className="w-4 h-4" />
                </button>
              )}
            </form>
          )}

          {/* Quick Search Popover Dropdown */}
          {!isOffline && !activeFacet && popoverOpen && localQuery.trim().length > 0 && (
            <div className="absolute top-full left-0 right-0 mt-1.5 bg-base-100 border border-base-200 shadow-2xl rounded-2xl p-2 z-50 overflow-hidden animate-in fade-in slide-in-from-top-2">
              <div className="px-3 py-1.5 text-xs font-bold text-base-content/50 uppercase tracking-wider flex items-center justify-between">
                <span>{t("library.quick_search_results", "Quick Results")}</span>
                {isQuickLoading && <Loader2 className="w-3.5 h-3.5 animate-spin text-primary" />}
              </div>

              {isQuickLoading ? (
                <div className="py-6 text-center text-xs text-base-content/60 flex items-center justify-center gap-2">
                  <Loader2 className="w-4 h-4 animate-spin text-primary" />
                  <span>{t("common.loading", "Searching...")}</span>
                </div>
              ) : quickResultsData && quickResultsData.length > 0 ? (
                <div className="flex flex-col gap-1 py-1">
                  {quickResultsData.map((book) => (
                    <button
                      key={book.id}
                      type="button"
                      onClick={() => {
                        setPopoverOpen(false);
                        navigate(`/books/${book.id}`);
                      }}
                      className="flex items-center gap-3 p-2 rounded-xl hover:bg-base-200/70 text-left transition-colors w-full group"
                    >
                      {book.cover_url ? (
                        <img
                          src={getMediaUrl(book.cover_url)}
                          alt={book.title}
                          className="w-10 h-14 object-cover rounded-md bg-base-200 shrink-0 shadow-2xs group-hover:scale-105 transition-transform"
                        />
                      ) : (
                        <div className="w-10 h-14 rounded-md bg-primary/10 grid place-items-center text-primary shrink-0">
                          <BookOpen className="w-5 h-5" />
                        </div>
                      )}
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-bold text-base-content truncate group-hover:text-primary transition-colors">
                          {book.title}
                        </p>
                        <p className="text-xs text-base-content/60 truncate mt-0.5">
                          {book.author_name || t("book.author_unknown", "Unknown Author")}
                        </p>
                      </div>
                    </button>
                  ))}
                </div>
              ) : (
                <div className="py-5 text-center text-xs text-base-content/50">
                  {t("library.no_quick_results", "No matching books found")}
                </div>
              )}

              <div className="border-t border-base-200 pt-1.5 mt-1">
                <button
                  type="button"
                  onClick={() => submitSearch()}
                  className="btn btn-ghost btn-sm w-full justify-between rounded-xl font-medium text-primary text-xs hover:bg-primary/10"
                >
                  <span className="truncate pr-2">
                    {t("library.view_all_results", "View all results for \"{{query}}\"", { query: localQuery.trim() })}
                  </span>
                  <ChevronRight className="w-4 h-4 shrink-0" />
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-1 sm:gap-2">
        <div className="tooltip tooltip-bottom" data-tip={t("common.theme", "Theme")}>
          <ThemeController />
        </div>
        <div className="tooltip tooltip-bottom" data-tip={t("common.language", "Language")}>
          <LanguageSwitcher />
        </div>

        {user && (
          <div className="tooltip tooltip-bottom" data-tip={t("library.readlists", "Read Lists")}>
            <Link
              to="/read-lists"
              className="btn btn-ghost btn-circle btn-sm sm:btn-md text-base-content/70 hover:text-primary"
              aria-label={t("library.readlists", "Read Lists")}
            >
              <ListOrdered className="w-5 h-5" />
            </Link>
          </div>
        )}

        {user && hasPermission(user, "podcast.manage") && (
          <div className="tooltip tooltip-bottom" data-tip={t("podcasts.title", "Podcasts")}>
            <Link
              to="/podcasts"
              className="btn btn-ghost btn-circle btn-sm sm:btn-md text-base-content/70 hover:text-primary"
              aria-label={t("podcasts.title", "Podcasts")}
            >
              <Podcast className="w-5 h-5" />
            </Link>
          </div>
        )}

        <div className="tooltip tooltip-bottom" data-tip={t("offline.title", "Offline Books")}>
          <Link
            to="/offline"
            className="btn btn-ghost btn-circle btn-sm sm:btn-md text-base-content/70 hover:text-primary"
            aria-label={t("offline.title", "Offline Books")}
          >
            <CloudDownload className="w-5 h-5" />
          </Link>
        </div>

        {user ? (
          <>
            {hasPermission(user, "admin.access") && (
              <div className="tooltip tooltip-bottom" data-tip={t("admin.dashboard", "Admin")}>
                <Link
                  to="/admin"
                  className="btn btn-ghost btn-sm sm:btn-md gap-2 hidden sm:flex"
                >
                  <LayoutDashboard className="w-4 h-4" />
                  {t("admin.dashboard", "Admin")}
                </Link>
              </div>
            )}
            <div className="dropdown dropdown-end">
              <div className="tooltip tooltip-bottom" data-tip={t("user.profile", "Profile")}>
                <div
                  tabIndex={0}
                  role="button"
                  className="btn btn-ghost btn-circle avatar border border-base-300"
                >
                  <div className="w-9 rounded-full bg-primary/10 flex items-center justify-center text-primary font-bold">
                    {user.avatar_url && (
                      <img
                        src={getMediaUrl(user.avatar_url, undefined, user.updated_at)}
                        alt="Avatar"
                        loading="lazy"
                        onError={(e) => {
                          e.currentTarget.style.display = "none";
                          const fallback = e.currentTarget.nextElementSibling;
                          if (fallback) {
                            (fallback as HTMLElement).style.display = "flex";
                          }
                        }}
                      />
                    )}
                    <span 
                      className="text-lg"
                      style={{ display: user.avatar_url ? "none" : "flex" }}
                    >
                      {user.full_name
                        ? user.full_name.charAt(0).toUpperCase()
                        : user.email.charAt(0).toUpperCase()}
                    </span>
                  </div>
                </div>
              </div>
              <ul
                tabIndex={0}
                className="mt-3 z-1 p-2 shadow menu menu-sm dropdown-content bg-base-100 rounded-box w-52 border border-base-200"
              >
                <li>
                  <span className="font-semibold opacity-60 px-4 py-2 truncate block">
                    {user.email}
                  </span>
                </li>
                <li>
                  <Link to="/profile" className="flex items-center gap-2">
                    <User className="w-4 h-4 opacity-70" />
                    {t("user.profile", "Profile")}
                  </Link>
                </li>
                {hasPermission(user, "user.stats.read") && (
                  <li>
                    <Link to="/analytics" className="flex items-center gap-2">
                      <BarChart3 className="w-4 h-4 text-primary opacity-80" />
                      {t("analytics.title", "Reading Analytics")}
                    </Link>
                  </li>
                )}
                <li>
                  <Link to="/offline" className="flex items-center gap-2">
                    <CloudDownload className="w-4 h-4 opacity-70" />
                    {t("offline.title")}
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
        ) : !isAuthPage ? (
          <button
            onClick={() => setLoginModalOpen(true)}
            className="btn btn-primary btn-sm sm:btn-md"
          >
            {t("auth.login", "Login")}
          </button>
        ) : null}
      </div>
      <LoginView />
      <RegisterView />
    </div>
  );
};
