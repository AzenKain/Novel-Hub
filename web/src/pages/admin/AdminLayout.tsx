import { LanguageSwitcher, ThemeController } from "@/components/ui";
import { useLogoutMutation } from "@/hooks";
import { usePublicSettings } from "@/hooks/useSettings";
import { useAuthStore } from "@/stores";
import { BookOpen, Copy, LogOut, Menu, MessageSquareText, Settings2, Shield, Users } from "lucide-react";
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

  const handleLogout = () => {
    logoutMutation.mutate(undefined, {
      onSuccess: () => navigate("/"),
    });
  };

  const navItems = [
    { name: t("admin.users", "Users"), path: "/admin/users", icon: Users },
    { name: t("admin.roles", "Roles"), path: "/admin/roles", icon: Shield },
    { name: t("admin.books_libraries", "Books & Libraries"), path: "/admin/books", icon: BookOpen },
    { name: t("admin.settings", "Settings"), path: "/admin/settings", icon: Settings2 },
    { name: t("admin.duplicates", "Duplicate Files"), path: "/admin/duplicates", icon: Copy },
    { name: t("admin.reviews", "Reviews"), path: "/admin/reviews", icon: MessageSquareText }
  ];

  return (
    <div className="drawer lg:drawer-open bg-base-200 min-h-screen font-sans">
      <input id="admin-drawer" type="checkbox" className="drawer-toggle" />
      
      <div className="drawer-content flex flex-col min-w-0 h-screen overflow-hidden">
        {/* Mobile Navbar */}
        <div className="navbar bg-base-100 shadow-sm lg:hidden border-b border-base-200 px-4">
          <div className="flex-none">
            <label htmlFor="admin-drawer" aria-label="open sidebar" className="btn btn-square btn-ghost">
              <Menu className="w-5 h-5" />
            </label>
          </div>
          <div className="flex-1 px-2 mx-2 font-bold">{t('admin.panel', 'Admin Panel')}</div>
        </div>

        <main className="flex-1 overflow-auto bg-base-200/50">
          <Outlet />
        </main>
      </div>

      <div className="drawer-side z-20 border-r border-base-200 shadow-xl lg:shadow-none">
        <label htmlFor="admin-drawer" aria-label="close sidebar" className="drawer-overlay"></label> 
        <aside className="bg-base-100 w-64 min-h-full flex flex-col">
          <div className="h-20 flex flex-col justify-center px-6 border-b border-base-200">
            <Link to="/" className="flex items-center hover:opacity-80 transition-opacity">
              {settings?.site?.logo ? (
                <img src={settings.site.logo} alt="Logo" className="w-8 h-8 rounded bg-base-100 object-contain mr-3" />
              ) : (
                <div className="w-8 h-8 rounded bg-gradient-to-br from-primary to-secondary text-primary-content flex items-center justify-center font-bold mr-3">NH</div>
              )}
              <div className="flex flex-col">
                <span className="text-xl font-bold leading-tight">{settings?.site?.title || "NovelHub"}</span>
                <span className="text-xs text-base-content/60 font-medium uppercase tracking-wider">{t('admin.panel', 'Admin Panel')}</span>
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
                      className={isActive ? "active bg-primary/10 text-primary font-bold" : "text-base-content/80 font-medium"}
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
              <div className="w-10 h-10 rounded-full bg-primary/10 border border-primary/20 text-primary flex items-center justify-center font-bold mr-3 shrink-0">
                {user?.full_name?.charAt(0).toUpperCase() || user?.email?.charAt(0).toUpperCase() || "U"}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-bold truncate">
                  {user?.full_name || "Admin"}
                </p>
                <p className="text-xs text-base-content/60 truncate">{user?.email}</p>
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
