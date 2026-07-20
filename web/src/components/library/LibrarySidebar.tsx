import { Plus } from "lucide-react";
import React from "react";

import { usePublicSettings } from "@/hooks/useSettings";
import type { Collection, User } from "@/types";
import type { MetadataFacetSection } from "./MetadataIndexView";

export type LibraryNavItem = {
  id: string;
  label: string;
  icon: React.ReactNode;
};

type LibrarySidebarProps = {
  t: (key: string, fallback: string) => string;
  user: User | null;
  primaryNavItems: LibraryNavItem[];
  facetSections: MetadataFacetSection[];
  secondaryNavItems: LibraryNavItem[];
  collections: Collection[];
  activeNav: string;
  activeFacet: unknown;
  activeCollection: string;
  onNavClick: (nav: string) => void;
  onCollectionClick: (collection: string) => void;
  onNewCollection: () => void;
};

export const LibrarySidebar: React.FC<LibrarySidebarProps> = ({
  t,
  user,
  primaryNavItems,
  facetSections,
  secondaryNavItems,
  collections,
  activeNav,
  activeFacet,
  activeCollection,
  onNavClick,
  onCollectionClick,
  onNewCollection,
}) => {
  const settings = usePublicSettings();
  const siteTitle = settings?.site?.title || "NovelHub";
  const siteDesc = settings?.site?.description || "Local library manager";
  const siteLogo = settings?.site?.logo;

  const renderNavButton = (item: LibraryNavItem) => (
    <li key={item.id}>
      <button
        className={`${activeNav === item.id && !activeFacet ? "active bg-primary/10 text-primary font-bold" : ""}`}
        onClick={() => onNavClick(item.id)}
      >
        {item.icon} {item.label}
      </button>
    </li>
  );

  return (
    <div className="drawer-side z-20 border-r border-base-200 shadow-xl">
      <label
        htmlFor="main-drawer"
        aria-label="close sidebar"
        className="drawer-overlay"
      />
      <div className="menu flex min-h-full w-64 flex-col gap-5 bg-base-100 p-3 text-base-content">
        <div className="mt-2 mb-3 flex items-center gap-2.5 px-2">
          {siteLogo ? (
            <img src={siteLogo} alt="Logo" className="h-9 w-9 rounded-lg object-contain shadow-md" />
          ) : (
            <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-primary/20 bg-gradient-to-br from-primary to-secondary font-bold text-primary-content shadow-md shadow-primary/20">
              NH
            </div>
          )}
          <div>
            <h1 className="text-lg font-black leading-none tracking-tight">
              {siteTitle}
            </h1>
            <p className="mt-1 text-[11px] font-semibold uppercase tracking-widest text-base-content/50">
              {siteDesc}
            </p>
          </div>
        </div>

        <div>
          <li className="menu-title px-2 pb-2 text-xs font-bold uppercase tracking-wider text-base-content/40">
            {t("admin.library", "Library")}
          </li>
          <ul className="menu menu-md w-full gap-1 p-0">
            {primaryNavItems.map(renderNavButton)}
            {facetSections.map((section) => (
              <li key={section.nav}>
                <button
                  className={`${activeNav === section.nav ? "active bg-primary/10 text-primary font-bold" : ""}`}
                  onClick={() => onNavClick(section.nav)}
                >
                  {section.icon} {section.label}
                </button>
              </li>
            ))}
            {secondaryNavItems.map(renderNavButton)}
          </ul>
        </div>

        <div>
          <div className="flex items-center justify-between px-2 pb-2">
            <span className="menu-title !p-0 text-xs font-bold uppercase tracking-wider text-base-content/40">
              {t("library.collections", "Collections")}
            </span>
            {user && (
              <button
                onClick={onNewCollection}
                className="btn btn-ghost btn-circle btn-xs text-base-content/50 hover:text-primary"
                title={t("library.new_collection", "New Collection")}
              >
                <Plus className="h-4 w-4" />
              </button>
            )}
          </div>

          <ul className="menu menu-md w-full gap-1 p-0">
            {collections.length > 0 ? (
              collections.map((collection) => (
                <li key={collection.id}>
                  <button
                    className={`${activeCollection === collection.name ? "active bg-primary/10 text-primary font-bold" : ""}`}
                    onClick={() => onCollectionClick(collection.name)}
                  >
                    <span className="flex h-4 w-4 items-center justify-center rounded bg-base-200 text-[10px] font-bold uppercase">
                      {collection.name.charAt(0)}
                    </span>
                    {collection.name}
                  </button>
                </li>
              ))
            ) : (
              <div className="px-2 py-3 text-xs font-medium text-base-content/40">
                {user
                  ? t("library.no_collections", "No collections yet")
                  : t("library.login_collections", "Login to create collections")}
              </div>
            )}
          </ul>
        </div>
      </div>
    </div>
  );
};
