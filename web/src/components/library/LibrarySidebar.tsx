import { Plus, MoreVertical, Edit2, Filter, Trash2 } from "lucide-react";
import React, { useState } from "react";
import { Link } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { featureService } from "@/services";
import { toast } from "react-toastify";
import { DeleteConfirmModal } from "@/components/admin/books/DeleteConfirmModal";
import { usePublicSettings } from "@/hooks/useSettings";
import type { Collection, LibraryNavItem, MetadataFacetSection, SmartCollection, SmartCollectionRule, User, SmartFilter } from "@/types";

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
  hasMoreCollections?: boolean;
  onLoadMoreCollections?: () => void;
  isFetchingMoreCollections?: boolean;
  smartCollections?: SmartCollection[];
  onSmartCollectionClick?: (rule: SmartCollectionRule) => void;
  onRenameSmartCollection?: (id: string, name: string) => void;
  onDeleteSmartCollection?: (id: string) => void;
  smartFilters?: SmartFilter[];
  onSmartFilterClick?: (id: string) => void;
  onEditSmartFilter?: (filter: SmartFilter) => void;
  onDeleteSmartFilter?: (id: string) => void;
  onNewSmartFilter?: () => void;
  activeSmartFilterId?: string;
};

import { useLibraryStore } from "@/stores";
import { useShallow } from "zustand/react/shallow";

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
  hasMoreCollections,
  onLoadMoreCollections,
  isFetchingMoreCollections,
  smartCollections = [],
  onSmartCollectionClick,
  onRenameSmartCollection,
  onDeleteSmartCollection,
  smartFilters = [],
  onSmartFilterClick,
  onEditSmartFilter,
  onDeleteSmartFilter,
  onNewSmartFilter,
  activeSmartFilterId,
}) => {
  const queryClient = useQueryClient();
  const settings = usePublicSettings();
  const siteTitle = settings?.site?.title || "NovelHub";
  const siteDesc = settings?.site?.description || "Local library manager";
  const siteLogo = settings?.site?.logo;

  const { setSearch, setActiveFacet } = useLibraryStore(
    useShallow((state) => ({
      setSearch: state.setSearch,
      setActiveFacet: state.setActiveFacet,
    }))
  );

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

  const [editingCollection, setEditingCollection] = useState<{id: string, name: string} | null>(null);
  const [editingName, setEditingName] = useState("");
  const [editingSmartId, setEditingSmartId] = useState<string | null>(null);
  const [editingSmartName, setEditingSmartName] = useState("");
  const [deletingCollectionTarget, setDeletingCollectionTarget] = useState<{ id: string; name: string } | null>(null);
  const [isDeleting, setIsDeleting] = useState<string | null>(null);
  
  const handleEditSmartCollection = () => {
    if (!editingSmartId || !editingSmartName.trim() || !onRenameSmartCollection) {
      setEditingSmartId(null);
      return;
    }
    onRenameSmartCollection(editingSmartId, editingSmartName.trim());
    setEditingSmartId(null);
  };

  const handleEditCollection = async () => {
    if (!editingCollection || !editingName.trim()) return;
    try {
      const res = await featureService.updateCollection(editingCollection.id, editingName);
      if (res.status) {
        await queryClient.invalidateQueries({ queryKey: ["collections"] });
        if (activeCollection === editingCollection.name) {
          onCollectionClick(res.data!.name);
        }
      }
    } catch (e) {
      console.error(e);
    } finally {
      setEditingCollection(null);
    }
  };

  const confirmDeleteCollection = async () => {
    if (!deletingCollectionTarget) return;
    const { id, name } = deletingCollectionTarget;
    setIsDeleting(id);
    try {
      const res = await featureService.deleteCollection(id);
      if (res.status) {
        await queryClient.invalidateQueries({ queryKey: ["collections"] });
        if (activeCollection === name) {
          onCollectionClick("");
        }
        toast.success(t("library.collection_deleted", "Collection deleted successfully"));
      }
    } catch (e) {
      console.error(e);
    } finally {
      setIsDeleting(null);
      setDeletingCollectionTarget(null);
    }
  };

  return (
    <div className="drawer-side z-20 border-r border-base-200 shadow-xl">
      <label
        htmlFor="main-drawer"
        aria-label="close sidebar"
        className="drawer-overlay"
      />
      <div className="menu flex min-h-full w-64 flex-col gap-5 bg-base-100 p-3 text-base-content">
        <Link
          to="/"
          onClick={() => {
            setSearch("");
            setActiveFacet(null);
            onNavClick("books");
            onCollectionClick("");
          }}
          className="mt-2 mb-3 flex items-center gap-2.5 px-2 hover:opacity-80 transition-opacity cursor-pointer text-left focus:outline-none"
        >
          {siteLogo ? (
            <img src={siteLogo} alt={t("common.alt_logo", "Logo")} className="h-9 w-9 rounded-lg object-contain shadow-md" />
          ) : (
            <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-primary/20 bg-gradient-to-br from-primary to-secondary font-bold text-primary-content shadow-md shadow-primary/20">
              NH
            </div>
          )}
          <div>
            <h1 className="text-lg font-black leading-none tracking-tight text-base-content">
              {siteTitle}
            </h1>
            <p className="mt-1 text-[11px] font-semibold uppercase tracking-widest text-base-content/50">
              {siteDesc}
            </p>
          </div>
        </Link>

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
              <div className="tooltip tooltip-left" data-tip={t("library.new_collection", "New Collection")}>
                <button
                  onClick={onNewCollection}
                  className="btn btn-ghost btn-circle btn-xs text-base-content/50 hover:text-primary"
                  aria-label={t("library.new_collection", "New Collection")}
                >
                  <Plus className="h-4 w-4" />
                </button>
              </div>
            )}
          </div>

          <ul className="menu menu-md w-full gap-1 p-0">
            {collections.length > 0 ? (
              collections.map((collection) => (
                <li key={collection.id}>
                  <div className={`group flex items-center justify-between !p-0 ${activeCollection === collection.name ? "active bg-primary/10 text-primary font-bold rounded-lg" : ""}`}>
                    <button
                      className="flex-1 flex items-center gap-2 p-2 px-3 text-left bg-transparent border-none min-w-0"
                      onClick={() => onCollectionClick(collection.name)}
                    >
                      <span className="flex h-4 w-4 shrink-0 items-center justify-center rounded bg-base-200 text-[10px] font-bold uppercase">
                        {collection.name.charAt(0)}
                      </span>
                      <span className="truncate">{collection.name}</span>
                    </button>
                    {user && (
                      <div className="dropdown dropdown-top dropdown-end">
                        <button
                          tabIndex={0}
                          className="btn btn-ghost btn-xs btn-square opacity-0 group-hover:opacity-100 focus:opacity-100 transition-opacity mr-1"
                          aria-label={t("library.collection_options", "Collection options")}
                        >
                          {isDeleting === collection.id ? (
                            <span className="loading loading-spinner loading-xs"></span>
                          ) : (
                            <MoreVertical className="w-4 h-4" />
                          )}
                        </button>
                        <ul
                          tabIndex={0}
                          className="dropdown-content z-30 menu p-1.5 shadow-xl bg-base-100 rounded-xl w-36 border border-base-200 mb-1"
                        >
                          <li>
                            <button
                              type="button"
                              onClick={() => {
                                setEditingCollection(collection);
                                setEditingName(collection.name);
                                if (document.activeElement instanceof HTMLElement) {
                                  document.activeElement.blur();
                                }
                              }}
                              className="flex items-center gap-2 px-3 py-2 text-xs rounded-lg hover:bg-base-200/60"
                            >
                              <Edit2 className="w-3.5 h-3.5 text-base-content/70" />
                              <span>{t("common.edit", "Edit")}</span>
                            </button>
                          </li>
                          <li>
                            <button
                              type="button"
                              className="flex items-center gap-2 px-3 py-2 text-xs rounded-lg text-error hover:bg-error/10"
                              onClick={() => {
                                setDeletingCollectionTarget({ id: collection.id, name: collection.name });
                                if (document.activeElement instanceof HTMLElement) {
                                  document.activeElement.blur();
                                }
                              }}
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                              <span>{t("common.delete", "Delete")}</span>
                            </button>
                          </li>
                        </ul>
                      </div>
                    )}
                  </div>
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
          
          {hasMoreCollections && (
            <div className="px-2 mt-2">
              <button
                className="btn btn-ghost btn-sm w-full text-xs"
                onClick={onLoadMoreCollections}
                disabled={isFetchingMoreCollections}
              >
                {isFetchingMoreCollections ? <span className="loading loading-spinner loading-xs"></span> : t("common.load_more", "Load more")}
              </button>
            </div>
          )}
        </div>

        {user && smartCollections.length > 0 && (
          <div>
            <div className="flex items-center gap-1.5 px-2 pb-2">
              <Filter className="h-3.5 w-3.5 text-base-content/40" />
              <span className="menu-title !p-0 text-xs font-bold uppercase tracking-wider text-base-content/40">
                {t("library.smart_collections", "Smart Collections")}
              </span>
            </div>
            <ul className="menu menu-md w-full gap-1 p-0">
              {smartCollections.map((smart) => (
                <li key={smart.id}>
                  {editingSmartId === smart.id ? (
                    <div className="flex items-center gap-1 px-2 py-1">
                      <input
                        autoFocus
                        className="input input-ghost input-sm min-h-0 h-7 flex-1 px-2 py-0 text-sm focus:outline-none"
                        value={editingSmartName}
                        onChange={(e) => setEditingSmartName(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") handleEditSmartCollection();
                          if (e.key === "Escape") setEditingSmartId(null);
                        }}
                        aria-label={t("library.rename_smart_collection", "Rename smart collection")}
                      />
                      <button
                        className="btn btn-ghost btn-xs btn-square text-success"
                        onClick={handleEditSmartCollection}
                        title={t("common.save", "Save")}
                      >
                        <Edit2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  ) : (
                  <div className="group flex items-center justify-between !p-0">
                    <button
                      className="flex-1 flex items-center gap-2 p-2 px-3 text-left bg-transparent border-none"
                      onClick={() => onSmartCollectionClick?.(smart.rule)}
                    >
                      <span className="truncate">{smart.name}</span>
                    </button>
                    {onRenameSmartCollection && (
                      <button
                        className="btn btn-ghost btn-xs btn-square opacity-0 group-hover:opacity-100 transition-opacity text-base-content/60 mr-0.5"
                        onClick={() => {
                          setEditingSmartId(smart.id);
                          setEditingSmartName(smart.name);
                        }}
                        title={t("library.rename_smart_collection", "Rename smart collection")}
                      >
                        <Edit2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                    {onDeleteSmartCollection && (
                      <button
                        className="btn btn-ghost btn-xs btn-square opacity-0 group-hover:opacity-100 transition-opacity text-error mr-1"
                        onClick={() => onDeleteSmartCollection(smart.id)}
                        title={t("common.delete", "Delete")}
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )}

        {user && (
          <div className="mt-4">
            <div className="flex items-center justify-between px-2 pb-2">
              <div className="flex items-center gap-1.5">
                <Filter className="h-3.5 w-3.5 text-base-content/40" />
                <span className="menu-title !p-0 text-xs font-bold uppercase tracking-wider text-base-content/40">
                  {t("library.smart_filters", "Smart Filters")}
                </span>
              </div>
              {onNewSmartFilter && (
                <button
                  onClick={onNewSmartFilter}
                  className="btn btn-ghost btn-xs btn-square hover:bg-base-350"
                  title={t("library.new_smart_filter", "New Smart Filter")}
                >
                  <Plus className="w-3.5 h-3.5 text-base-content/60" />
                </button>
              )}
            </div>
            <ul className="menu menu-md w-full gap-1 p-0">
              {smartFilters
                .filter((sf) => sf.is_pinned_sidebar)
                .map((sf) => (
                  <li key={sf.id}>
                    <div className={`group flex items-center justify-between !p-0 ${activeSmartFilterId === sf.id ? "bg-primary/10 text-primary font-semibold" : ""}`}>
                      <button
                        className="flex-1 flex items-center gap-2 p-2 px-3 text-left bg-transparent border-none"
                        onClick={() => onSmartFilterClick?.(sf.id)}
                      >
                        <span className="truncate">{sf.name}</span>
                      </button>
                      <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity mr-1">
                        {onEditSmartFilter && (
                          <button
                            className="btn btn-ghost btn-xs btn-square text-base-content/60"
                            onClick={() => onEditSmartFilter(sf)}
                            title={t("common.edit", "Edit")}
                          >
                            <Edit2 className="w-3 h-3" />
                          </button>
                        )}
                        {onDeleteSmartFilter && (
                          <button
                            className="btn btn-ghost btn-xs btn-square text-error"
                            onClick={() => onDeleteSmartFilter(sf.id)}
                            title={t("common.delete", "Delete")}
                          >
                            <Trash2 className="w-3 h-3" />
                          </button>
                        )}
                      </div>
                    </div>
                  </li>
                ))}
              {smartFilters.filter((sf) => sf.is_pinned_sidebar).length === 0 && (
                <li className="px-3 py-1.5 text-xs text-base-content/40 italic">
                  {t("library.no_pinned_filters", "No pinned filters")}
                </li>
              )}
            </ul>
          </div>
        )}
      </div>

      {/* Edit Collection Modal */}
      {editingCollection && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <button
            className="absolute inset-0 bg-black/50 backdrop-blur-xs"
            onClick={() => setEditingCollection(null)}
          />
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void handleEditCollection();
            }}
            className="relative z-10 w-full max-w-sm rounded-2xl border border-base-200 bg-base-100 p-6 shadow-2xl space-y-4"
          >
            <h3 className="text-lg font-bold text-base-content">
              {t("library.edit_collection", "Edit collection")}
            </h3>
            <div>
              <label className="text-xs font-medium text-base-content/70 mb-1 block">
                {t("library.collection_name", "Collection name")}
              </label>
              <input
                type="text"
                className="input input-bordered w-full rounded-xl"
                value={editingName}
                onChange={(e) => setEditingName(e.target.value)}
                autoFocus
              />
            </div>
            <div className="flex items-center justify-end gap-2 pt-2">
              <button
                type="button"
                className="btn btn-ghost rounded-xl"
                onClick={() => setEditingCollection(null)}
              >
                {t("common.cancel", "Cancel")}
              </button>
              <button
                type="submit"
                className="btn btn-primary rounded-xl"
                disabled={!editingName.trim()}
              >
                {t("common.save", "Save")}
              </button>
            </div>
          </form>
        </div>
      )}
      {/* Delete Confirmation Modal */}
      {deletingCollectionTarget && (
        <DeleteConfirmModal
          open={!!deletingCollectionTarget}
          title={t("library.delete_collection", "Delete Collection")}
          message={t("library.confirm_delete_collection", "Are you sure you want to delete this collection?")}
          onClose={() => setDeletingCollectionTarget(null)}
          onConfirm={confirmDeleteCollection}
          loading={isDeleting === deletingCollectionTarget.id}
        />
      )}
    </div>
  );
};
