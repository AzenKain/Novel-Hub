import { getMediaUrl } from "@/config/api";
import { LanguageSwitcher, ThemeController } from "@/components/ui";
import { useLogoutMutation } from "@/hooks";
import { usePublicSettings } from "@/hooks/useSettings";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";
import {
  BookOpen,
  Copy,
  Key,
  ListTodo,
  LogOut,
  Menu,
  MessageSquareText,
  Settings2,
  Shield,
  Users,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";

export function AdminLayout() {
  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));
  const logoutMutation = useLogoutMutation();
  const navigate = useNavigate();
  const location = useLocation();
  const { t } = useTranslation();
  const settings = usePublicSettings();
  const siteLogo = settings?.site?.logo || "/logo.svg";
  const siteTitle = settings?.site?.title || "NovelHub";

  const handleLogout = () => {
    logoutMutation.mutate(undefined, {
      onSuccess: () => navigate("/"),
    });
  };

  const navItems = [
    {
      name: t("admin.users", "Users"),
      path: "/admin/users",
      icon: Users,
      permissions: ["user.manage"],
    },
    {
      name: t("admin.roles", "Roles"),
      path: "/admin/roles",
      icon: Shield,
      permissions: ["role.manage"],
    },
    {
      name: t("admin.books_libraries", "Books & Libraries"),
      path: "/admin/books",
      icon: BookOpen,
      permissions: [
        "book.upload",
        "book.edit",
        "book.delete",
        "book.bulk.manage",
        "library.manage",
      ],
    },
    {
      name: t("admin.settings", "Settings"),
      path: "/admin/settings",
      icon: Settings2,
      permissions: ["setting.manage"],
    },
    {
      name: t("admin.oauth_settings", "OAuth & OIDC"),
      path: "/admin/oauth",
      icon: Key,
      permissions: ["setting.manage"],
    },
    {
      name: t("admin.duplicates", "Duplicate Files"),
      path: "/admin/duplicates",
      icon: Copy,
      permissions: ["book.duplicate.manage"],
    },
    {
      name: t("admin.operations.title"),
      path: "/admin/operations",
      icon: ListTodo,
      permissions: [
        "job.read",
        "job.manage",
        "system.log.read",
        "system.backup",
      ],
    },
    {
      name: t("admin.reviews", "Reviews"),
      path: "/admin/reviews",
      icon: MessageSquareText,
      permissions: ["book.review.delete"],
    },
  ].filter((item) =>
    item.permissions.some((permission) => hasPermission(user, permission)),
  );

  return (
    <div className="drawer lg:drawer-open bg-base-200 min-h-screen font-sans">
      <input id="admin-drawer" type="checkbox" className="drawer-toggle" />

      <div className="drawer-content flex flex-col min-w-0 h-screen overflow-hidden">
        <div className="navbar bg-base-100 shadow-sm lg:hidden border-b border-base-200 px-3 sm:px-4">
          <div className="flex items-center gap-1.5 flex-none">
            <label
              htmlFor="admin-drawer"
              aria-label={t("common.toggle_sidebar", "Toggle Sidebar")}
              className="btn btn-square btn-ghost btn-sm sm:btn-md"
            >
              <Menu className="w-5 h-5" />
            </label>
            <Link
              to="/"
              className="flex items-center gap-2.5 hover:opacity-85 transition-opacity mr-1"
              title={siteTitle}
            >
              {siteLogo ? (
                <img
                  src={siteLogo}
                  alt={t("common.alt_logo", "Logo")}
                  className="h-9 w-auto max-w-12 object-contain drop-shadow-xs"
                />
              ) : (
                <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-primary/20 bg-linear-to-br from-primary to-secondary font-bold text-primary-content text-xs shadow-xs">
                  NH
                </div>
              )}
              <div className="flex flex-col">
                <span className="font-sans text-lg font-black tracking-tight text-base-content leading-none">
                  {siteTitle}
                </span>
                <span className="mt-0.5 text-[10px] font-semibold uppercase tracking-wider text-base-content/50">
                  {t("admin.panel", "Admin Panel")}
                </span>
              </div>
            </Link>
          </div>
        </div>

        <main className="flex-1 overflow-auto bg-base-200/50">
          <Outlet />
        </main>
      </div>

      <div className="drawer-side z-20 border-r border-base-200 shadow-xl lg:shadow-none">
        <label
          htmlFor="admin-drawer"
          aria-label="close sidebar"
          className="drawer-overlay"
        ></label>
        <aside className="bg-base-100 w-64 min-h-full flex flex-col">
          <div className="h-20 flex flex-col justify-center px-3 border-b border-base-200">
            <Link
              to="/"
              className="flex items-center gap-2.5 px-2 hover:opacity-80 transition-opacity cursor-pointer text-left focus:outline-none"
            >
              {siteLogo ? (
                <img
                  src={siteLogo}
                  alt={t("common.alt_logo", "Logo")}
                  className="h-11 w-auto max-w-14 object-contain shrink-0 drop-shadow-sm"
                />
              ) : (
                <div className="flex h-11 w-11 items-center justify-center rounded-lg border border-primary/20 bg-linear-to-br from-primary to-secondary font-bold text-primary-content shadow-xs shrink-0">
                  NH
                </div>
              )}
              <div>
                <h1 className="font-sans text-lg font-black leading-none tracking-tight text-base-content">
                  {siteTitle}
                </h1>
                <p className="mt-1 text-[11px] font-semibold uppercase tracking-widest text-base-content/50">
                  {t("admin.panel", "Admin Panel")}
                </p>
              </div>
            </Link>
          </div>

          <nav className="flex-1 py-6 px-4">
            <ul className="menu menu-md w-full p-0 gap-1">
              {navItems.map((item) => {
                const isActive = location.pathname.startsWith(item.path);
                const Icon = item.icon;
                return (
                  <li key={item.name}>
                    <Link
                      to={item.path}
                      className={
                        isActive
                          ? "active bg-primary/10 text-primary font-bold"
                          : "text-base-content/80 font-medium"
                      }
                    >
                      <Icon className="w-5 h-5 opacity-70" />
                      {item.name}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </nav>

          <div className="p-4 border-t border-base-200 mt-auto">
            <div className="flex items-center justify-between px-2 pb-4">
              <ThemeController className="dropdown-top" />
              <LanguageSwitcher className="dropdown-top dropdown-end" />
            </div>

            <div className="flex items-center px-2 py-3 mb-2 bg-base-200/50 rounded-xl">
              <div className="avatar mr-3 shrink-0">
                <div className="w-10 h-10 rounded-full ring-1 ring-base-content/10 overflow-hidden bg-primary/10 text-primary flex items-center justify-center font-bold">
                  {user?.avatar_url ? (
                    <img
                      src={getMediaUrl(
                        user.avatar_url,
                        undefined,
                        user.updated_at,
                      )}
                      alt={user.full_name || user.email}
                      className="w-full h-full object-cover"
                      onError={(e) => {
                        (e.target as HTMLElement).style.display = "none";
                      }}
                    />
                  ) : (
                    <span>
                      {user?.full_name?.charAt(0).toUpperCase() ||
                        user?.email?.charAt(0).toUpperCase() ||
                        "U"}
                    </span>
                  )}
                </div>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-bold truncate">
                  {user?.full_name || "Admin"}
                </p>
                <p className="text-xs text-base-content/60 truncate">
                  {user?.email}
                </p>
              </div>
            </div>
            <button
              onClick={handleLogout}
              disabled={logoutMutation.isPending}
              className="btn btn-ghost w-full justify-start text-error hover:bg-error/10 hover:text-error mt-2"
            >
              <LogOut className="h-4 w-4 mr-1" />
              {t("common.logout")}
            </button>
          </div>
        </aside>
      </div>
    </div>
  );
}
