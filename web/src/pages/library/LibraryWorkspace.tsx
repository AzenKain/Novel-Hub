import { toast } from "react-toastify";
import { RecentlyReadPanel } from "@/components/book-detail";
import { ReadingHeatmap } from "@/components/profile/ReadingHeatmap";
import { BookDetailPage } from "./BookDetailPage";
import { LoginView, TopNav } from "@/components/common";
import { LibrarySidebar, MetadataIndexView, type LibraryNavItem, type MetadataFacetSection } from "@/components/library";
import { BulkActionToolbar, BulkDeleteModal, BulkMoveModal, BulkTagModal } from "@/components/library";
import { BookCard, BookGrid } from "@/components/ui";
import { UserProfile } from "@/pages/user";
import { featureService } from "@/services";
import type { Book, MetadataCount, SmartCollectionRule, SmartFilter } from "@/types";
import { useQueryClient } from "@tanstack/react-query";
import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";

import { SmartFilterBuilderModal } from "@/components/library/SmartFilterBuilderModal";
import { SmartFilterShelf } from "@/components/library/SmartFilterShelf";

import { settingsKeyToNavId } from "@/constants";
import {
  useBookmarkedBooksQuery,
  useBooksQuery,
  useCollectionsQuery,
  useCreateSmartCollectionMutation,
  useDebounce,
  useDeleteSmartCollectionMutation,
  useHotBooksQuery,
  useLibraryStatsQuery,
  useMetadataFacetQuery,
  usePublicSettings,
  useRandomBooksQuery,
  useReadingHistoryQuery,
  useSmartCollectionsQuery,
  useSmartFiltersQuery,
  useSmartFilterBooksInfiniteQuery,
  useDeleteSmartFilterMutation,
  useReorderSmartFiltersHomeMutation,
} from "@/hooks";
import {
  alphabetFilters,
  sortMetadataItems,
  metadataNavIds,
} from "@/lib/libraryMetadata";
import { useAuthStore, useLibraryStore } from "@/stores";
import { hasPermission } from "@/utils/permission";
import {
  Activity,
  AlertCircle,
  Archive,
  ArrowDownAZ,
  ArrowUpAZ,
  Bookmark,
  BookmarkPlus,
  BookOpen,
  Building2,
  Download,
  Eye,
  EyeOff,
  FileType,
  Flame,
  Languages,
  Layers,
  LayoutGrid,
  Search,
  Shuffle,
  Star,
  Users
} from "lucide-react";


function isItemVisible(visibleKeys: string[] | undefined, id: string): boolean {
  if (!visibleKeys || visibleKeys.length === 0) return true;
  return visibleKeys.some((key) => (settingsKeyToNavId[key] || key) === id);
}

const EMPTY_ARRAY: any[] = [];

export const LibraryWorkspace = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { book_id } = useParams<{ book_id: string }>();
  const { t } = useTranslation();
  const { user, setLoginModalOpen, setProfileModalOpen, logout } = useAuthStore(useShallow((state) => ({ user: state.user, setLoginModalOpen: state.setLoginModalOpen, setProfileModalOpen: state.setProfileModalOpen, logout: state.logout })));
  const {
    books,
    setBooks,
    loading,
    setLoading,
    search,
    setSearch,
    selectedBook,
    setSelectedBook,
    activeNav,
    setActiveNav,
    activeCollection,
    setActiveCollection,
    activeChip,
    setActiveChip,
    activeFacet,
    setActiveFacet,
    stats,
    setStats,
    metadataFacets,
    setMetadataFacets,
    collections,
    setCollections,
    addCollection,
    recentHistory,
    setRecentHistory,
    showNewCollectionModal,
    setShowNewCollectionModal,
    newCollectionName,
    setNewCollectionName,
    collectionError,
    setCollectionError,
    randomSeed,
    setRandomSeed,
    metadataQuery,
    setMetadataQuery,
    metadataAlpha,
    setMetadataAlpha,
    metadataSort,
    setMetadataSort,
    activeSmartFilterId,
    setActiveSmartFilterId,
    sort,
    setSort,
  } = useLibraryStore(useShallow((state) => ({
    books: state.books,
    setBooks: state.setBooks,
    loading: state.loading,
    setLoading: state.setLoading,
    search: state.search,
    setSearch: state.setSearch,
    selectedBook: state.selectedBook,
    setSelectedBook: state.setSelectedBook,
    activeNav: state.activeNav,
    setActiveNav: state.setActiveNav,
    activeCollection: state.activeCollection,
    setActiveCollection: state.setActiveCollection,
    activeChip: state.activeChip,
    setActiveChip: state.setActiveChip,
    activeFacet: state.activeFacet,
    setActiveFacet: state.setActiveFacet,
    stats: state.stats,
    setStats: state.setStats,
    metadataFacets: state.metadataFacets,
    setMetadataFacets: state.setMetadataFacets,
    collections: state.collections,
    setCollections: state.setCollections,
    addCollection: state.addCollection,
    recentHistory: state.recentHistory,
    setRecentHistory: state.setRecentHistory,
    showNewCollectionModal: state.showNewCollectionModal,
    setShowNewCollectionModal: state.setShowNewCollectionModal,
    newCollectionName: state.newCollectionName,
    setNewCollectionName: state.setNewCollectionName,
    collectionError: state.collectionError,
    setCollectionError: state.setCollectionError,
    randomSeed: state.randomSeed,
    setRandomSeed: state.setRandomSeed,
    metadataQuery: state.metadataQuery,
    setMetadataQuery: state.setMetadataQuery,
    metadataAlpha: state.metadataAlpha,
    setMetadataAlpha: state.setMetadataAlpha,
    metadataSort: state.metadataSort,
    setMetadataSort: state.setMetadataSort,
    activeSmartFilterId: state.activeSmartFilterId,
    setActiveSmartFilterId: state.setActiveSmartFilterId,
    sort: state.sort,
    setSort: state.setSort,
  })));
  
  const publicSettings = usePublicSettings();
  const [selectedBookIds, setSelectedBookIds] = React.useState<string[]>([]);
  const [showBulkDeleteModal, setShowBulkDeleteModal] = React.useState(false);
  const [showBulkMoveModal, setShowBulkMoveModal] = React.useState(false);
  const [showBulkTagModal, setShowBulkTagModal] = React.useState(false);
  const debouncedSearch = useDebounce(search, 500);

  const { data: hotBooksData } = useHotBooksQuery(8);
  const topBooks = hotBooksData || [];

  const { data: randomBooksData } = useRandomBooksQuery(8);
  const randomBooks = randomBooksData || [];

  const recentReading = recentHistory.slice(0, 4);

  const openBookDetail = (book: Book) => {
    setSelectedBook(book);
    navigate(`/books/${encodeURIComponent(book.id)}`);
  };

  const handleNavClick = (nav: string) => {
    setActiveSmartFilterId(null);
    setActiveNav(nav);
    setActiveCollection("");
    setActiveFacet(null);
    setActiveChip("All");
    setMetadataQuery("");
    setMetadataAlpha("All");
    
    const params = new URLSearchParams();
    if (nav && nav !== "books") params.set("nav", nav);
    if (search) params.set("search", search);
    if (sort && sort !== "recently_added") params.set("sort", sort);
    navigate(`/${params.toString() ? `?${params.toString()}` : ""}`, { replace: true });
  };

  const handleCollectionClick = (collection: string) => {
    setActiveSmartFilterId(null);
    setActiveCollection(collection);
    setActiveNav("");
    setActiveFacet(null);
    setActiveChip("All");
    setMetadataQuery("");
    setMetadataAlpha("All");

    const params = new URLSearchParams();
    if (collection) params.set("collection", collection);
    if (search) params.set("search", search);
    if (sort && sort !== "recently_added") params.set("sort", sort);
    navigate(`/${params.toString() ? `?${params.toString()}` : ""}`, { replace: true });
  };

  // Filters live in the Zustand store and sync with URL search params for SEO & shareable links.
  const handleSmartCollectionClick = (rule: SmartCollectionRule) => {
    setActiveSmartFilterId(null);
    setSearch(rule.search || "");
    setActiveNav(rule.nav || "");
    setActiveCollection(rule.collection || "");
    setActiveChip(rule.chip || "All");
    setActiveFacet(rule.facet && rule.facet_id ? { type: rule.facet, id: rule.facet_id, name: rule.facet_id } : null);
    setMetadataQuery("");
    setMetadataAlpha("All");
    if (book_id) navigate("/");
  };

  const handleSmartFilterClick = (id: string) => {
    setActiveSmartFilterId(id);
    setActiveNav("");
    setActiveCollection("");
    setActiveFacet(null);
    setActiveChip("All");
    setMetadataQuery("");
    setMetadataAlpha("All");
    if (book_id) navigate("/");
  };

  const handleFacetClick = (type: string, item: MetadataCount, nav: string) => {
    setActiveSmartFilterId(null);
    setActiveNav(nav);
    setActiveCollection("");
    setActiveFacet({ type, id: item.id, name: item.name });
    setActiveChip("All");
    
    const singularFacet = type.endsWith("s") && type !== "series" ? type.slice(0, -1) : type;
    const params = new URLSearchParams();
    params.set("nav", nav);
    params.set("facet", singularFacet);
    if (item.name) params.set("name", item.name);
    if (item.id) params.set("facet_id", item.id);
    if (search) params.set("search", search);
    if (sort && sort !== "recently_added") params.set("sort", sort);

    navigate(`/${params.toString() ? `?${params.toString()}` : ""}`, { replace: true });
  };

  const handleDragStart = (e: React.DragEvent, id: string) => {
    e.dataTransfer.setData("text/plain", id);
    setDraggedShelfId(id);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleDrop = (e: React.DragEvent, targetId: string) => {
    e.preventDefault();
    const sourceId = e.dataTransfer.getData("text/plain") || draggedShelfId;
    if (!sourceId || sourceId === targetId) return;

    const pinnedFilters = smartFilters
      .filter((sf) => sf.is_pinned_home)
      .sort((a, b) => a.home_position - b.home_position);

    const sourceIndex = pinnedFilters.findIndex((sf) => sf.id === sourceId);
    const targetIndex = pinnedFilters.findIndex((sf) => sf.id === targetId);
    if (sourceIndex === -1 || targetIndex === -1) return;

    const updatedFilters = [...pinnedFilters];
    const [draggedItem] = updatedFilters.splice(sourceIndex, 1);
    updatedFilters.splice(targetIndex, 0, draggedItem);

    const reorderPayload = updatedFilters.map((sf, idx) => ({
      id: sf.id,
      position: idx,
    }));

    reorderHomeMutation.mutate(reorderPayload, {
      onSuccess: () => {
        toast.success(t("library.shelves_reordered", "Homepage shelves reordered"));
      },
    });

    setDraggedShelfId(null);
  };

  const queryClient = useQueryClient();
  const isMetadataNav = metadataNavIds.includes(activeNav) && !activeFacet;

  const [showSaveSearchModal, setShowSaveSearchModal] = useState(false);
  const [smartCollectionName, setSmartCollectionName] = useState("");
  const { data: smartCollections = [] } = useSmartCollectionsQuery(!!user);
  const createSmartCollection = useCreateSmartCollectionMutation();
  const deleteSmartCollection = useDeleteSmartCollectionMutation();

  const [showSmartFilterModal, setShowSmartFilterModal] = useState(false);
  const [editingSmartFilter, setEditingSmartFilter] = useState<SmartFilter | null>(null);
  const [draggedShelfId, setDraggedShelfId] = useState<string | null>(null);

  const { data: smartFilters = [] } = useSmartFiltersQuery();
  const deleteSmartFilter = useDeleteSmartFilterMutation();
  const reorderHomeMutation = useReorderSmartFiltersHomeMutation();

  const { data: smartFilterBooksRaw, isLoading: sfLoading, fetchNextPage: fetchNextSfBooks, hasNextPage: hasMoreSfBooks, isFetchingNextPage: isFetchingMoreSfBooks } = useSmartFilterBooksInfiniteQuery(
    activeSmartFilterId || "",
    undefined,
    20,
    !!activeSmartFilterId
  );

  const smartFilterBooksData = useMemo(() => {
    if (!smartFilterBooksRaw) return EMPTY_ARRAY;
    const all = smartFilterBooksRaw.pages.flatMap((p) => p.data || []);
    const seen = new Set<string>();
    return all.filter((b) => {
      if (!b || !b.id || seen.has(b.id)) return false;
      seen.add(b.id);
      return true;
    });
  }, [smartFilterBooksRaw]);

  const searchParams = useMemo(() => ({
    search: debouncedSearch,
    nav: activeNav,
    collection: activeCollection,
    chip: activeChip,
    facet: activeFacet?.type,
    facet_id: activeFacet?.id,
    sort,
  }), [debouncedSearch, activeNav, activeCollection, activeChip, activeFacet, sort]);

  const { data: booksDataRaw, isLoading: normalLoading, fetchNextPage: fetchNextBooks, hasNextPage: hasMoreBooks, isFetchingNextPage: isFetchingMoreBooks } = useBooksQuery(
    searchParams,
    !isMetadataNav && activeNav !== "bookmarks" && !activeSmartFilterId
  );
  const booksData = useMemo(() => {
    if (!booksDataRaw) return EMPTY_ARRAY;
    const all = booksDataRaw.pages.flatMap((p) => p.data || []);
    const seen = new Set<string>();
    return all.filter((b) => {
      if (!b || !b.id || seen.has(b.id)) return false;
      seen.add(b.id);
      return true;
    });
  }, [booksDataRaw]);

  const { data: bookmarkedBooksRaw, isLoading: bookmarksLoading, fetchNextPage: fetchNextBookmarks, hasNextPage: hasMoreBookmarks, isFetchingNextPage: isFetchingMoreBookmarks } = useBookmarkedBooksQuery(
    activeNav === "bookmarks" && !!user
  );
  const bookmarkedBooksData = useMemo(() => {
    if (!bookmarkedBooksRaw) return EMPTY_ARRAY;
    const all = bookmarkedBooksRaw.pages.flatMap((p) => p.data || []);
    const seen = new Set<string>();
    return all.filter((b) => {
      if (!b || !b.id || seen.has(b.id)) return false;
      seen.add(b.id);
      return true;
    });
  }, [bookmarkedBooksRaw]);

  const { data: statsData } = useLibraryStatsQuery();
  const { data: collectionsData, fetchNextPage: fetchNextCollections, hasNextPage: hasMoreCollections, isFetchingNextPage: isFetchingMoreCollections } = useCollectionsQuery(!!user);
  const { data: historyRaw, fetchNextPage: fetchNextHistory, hasNextPage: hasMoreHistory, isFetchingNextPage: isFetchingMoreHistory } = useReadingHistoryQuery(!!user);
  const historyData = useMemo(() => (historyRaw?.pages.flatMap(p => p.data || []) || EMPTY_ARRAY) as import("@/types").ReadingHistory[], [historyRaw]);

  const activeFacetNav = (["authors", "series", "tags", "publishers", "languages", "formats"] as const)
    .find((nav) => nav === activeNav);
  const {
    items: activeFacetItems,
    fetchNextPage: fetchNextFacetPage,
    hasNextPage: hasMoreFacetItems,
    isFetchingNextPage: isFetchingMoreFacetItems,
  } = useMetadataFacetQuery(activeFacetNav ?? "authors", {
    search: metadataQuery,
    alpha: metadataAlpha,
  });

  useEffect(() => {
    if (isMetadataNav) {
      setBooks([]);
    } else if (activeNav === "bookmarks") {
      if (bookmarkedBooksData) {
        setBooks(bookmarkedBooksData);
        if (bookmarkedBooksData.length > 0 && !selectedBook) {
          setSelectedBook(bookmarkedBooksData[0]);
        }
      }
    } else if (activeSmartFilterId) {
      if (smartFilterBooksData) {
        setBooks(smartFilterBooksData);
        if (smartFilterBooksData.length > 0 && !selectedBook) {
          setSelectedBook(smartFilterBooksData[0]);
        }
      }
    } else {
      if (booksData) {
        setBooks(booksData);
        if (booksData.length > 0 && !selectedBook) {
          setSelectedBook(booksData[0]);
        }
      }
    }
  }, [isMetadataNav, activeNav, activeSmartFilterId, booksData, bookmarkedBooksData, smartFilterBooksData, setBooks, setSelectedBook]);

  useEffect(() => {
    if (activeNav === "bookmarks") {
      setLoading(bookmarksLoading);
    } else if (activeSmartFilterId) {
      setLoading(sfLoading);
    } else {
      setLoading(normalLoading);
    }
  }, [activeNav, bookmarksLoading, normalLoading, setLoading]);

  useEffect(() => {
    if (statsData) setStats(statsData);
  }, [statsData, setStats]);

  useEffect(() => {
    if (collectionsData) {
      const allCollections = collectionsData.pages.flatMap((page) => page.data || []);
      setCollections(allCollections);
    }
  }, [collectionsData, setCollections]);

  useEffect(() => {
    if (historyData) setRecentHistory(historyData);
  }, [historyData, setRecentHistory]);

  useEffect(() => {
    if (!activeFacetNav) return;
    setMetadataFacets({ [activeFacetNav]: activeFacetItems });
  }, [activeFacetNav, activeFacetItems, setMetadataFacets]);

  const handleCreateCollection = async (e?: React.SyntheticEvent) => {
    if (e) e.preventDefault();
    if (!newCollectionName.trim()) return;
    setCollectionError("");
    try {
      const res = await featureService.createCollection(
        newCollectionName.trim(),
      );
      if (res.status && res.data) {
        addCollection(res.data);
        setNewCollectionName("");
        setShowNewCollectionModal(false);
        toast.success(t("library.collection_created", "Collection created successfully"));
        await queryClient.invalidateQueries({ queryKey: ["collections"] });
      } else {
        setCollectionError(
          res.message || t("library.fail_create_collection", "Failed to create collection"),
        );
      }
    } catch (err: any) {
      setCollectionError(
        err?.message || t("library.fail_create_collection", "Failed to create collection"),
      );
    }
  };

  const visibleItems = publicSettings?.sidebar_visible_items;

  const primaryNavItems: LibraryNavItem[] = useMemo(() => {
    const all: LibraryNavItem[] = [
      {
        id: "books",
        label: t("library.books", "Books"),
        icon: <BookOpen className="w-4 h-4 opacity-70" />,
      },
      {
        id: "hot",
        label: t("library.hot_books", "Hot books"),
        icon: <Flame className="w-4 h-4 opacity-70" />,
      },
      {
        id: "downloaded",
        label: t("library.downloaded_books", "Downloaded books"),
        icon: <Download className="w-4 h-4 opacity-70" />,
      },
      {
        id: "top_rated",
        label: t("library.top_rated_books", "Top rated books"),
        icon: <Star className="w-4 h-4 opacity-70" />,
      },
      {
        id: "bookmarks",
        label: t("library.bookmarked_books", "Bookmarked books"),
        icon: <Bookmark className="w-4 h-4 opacity-70" />,
      },
      {
        id: "read",
        label: t("library.read_books", "Read books"),
        icon: <Eye className="w-4 h-4 opacity-70" />,
      },
      {
        id: "unread",
        label: t("library.unread_books", "Unread books"),
        icon: <EyeOff className="w-4 h-4 opacity-70" />,
      },
    ];
    return visibleItems ? all.filter((item) => isItemVisible(visibleItems, item.id)) : all;
  }, [visibleItems, t]);

  const facetSections: MetadataFacetSection[] = useMemo(() => {
    const all: MetadataFacetSection[] = [
      {
        nav: "tags",
        type: "tag",
        label: t("library.subjects", "Subjects"),
        icon: <Bookmark className="w-4 h-4 opacity-70" />,
        items: metadataFacets.tags,
      },
      {
        nav: "series",
        type: "series",
        label: t("library.series", "Series"),
        icon: <Layers className="w-4 h-4 opacity-70" />,
        items: metadataFacets.series,
      },
      {
        nav: "authors",
        type: "author",
        label: t("library.authors", "Authors"),
        icon: <Users className="w-4 h-4 opacity-70" />,
        items: metadataFacets.authors,
      },
      {
        nav: "publishers",
        type: "publisher",
        label: t("library.publishers", "Publishers"),
        icon: <Building2 className="w-4 h-4 opacity-70" />,
        items: metadataFacets.publishers,
      },
      {
        nav: "languages",
        type: "language",
        label: t("library.languages", "Languages"),
        icon: <Languages className="w-4 h-4 opacity-70" />,
        items: metadataFacets.languages,
      },
      {
        nav: "formats",
        type: "format",
        label: t("library.file_formats", "File formats"),
        icon: <FileType className="w-4 h-4 opacity-70" />,
        items: metadataFacets.formats,
      },
    ];
    return visibleItems ? all.filter((item) => isItemVisible(visibleItems, item.nav)) : all;
  }, [visibleItems, metadataFacets, t]);

  const secondaryNavItems: LibraryNavItem[] = useMemo(() => {
    const all: LibraryNavItem[] = [
      {
        id: "ratings",
        label: t("library.ratings", "Ratings"),
        icon: <Star className="w-4 h-4 opacity-70" />,
      },
      {
        id: "archived",
        label: t("library.archived_books", "Archived books"),
        icon: <Archive className="w-4 h-4 opacity-70" />,
      },
    ];
    return visibleItems ? all.filter((item) => isItemVisible(visibleItems, item.id)) : all;
  }, [visibleItems, t]);

  const currentFacetSection = facetSections.find(
    (section) => section.nav === activeNav,
  );
  const isMetadataIndex = !!currentFacetSection && !activeFacet;
  const isCatalogPage = !!currentFacetSection || !!activeSmartFilterId || activeNav === "bookmarks";
  const activeNavLabel =
    primaryNavItems.find((item) => item.id === activeNav)?.label ||
    currentFacetSection?.label ||
    secondaryNavItems.find((item) => item.id === activeNav)?.label;
  const activeSmartFilter = smartFilters.find((sf) => sf.id === activeSmartFilterId);
  const bookListTitle = activeSmartFilter
    ? activeSmartFilter.name
    : activeFacet
      ? `${currentFacetSection?.label || activeNavLabel}: ${activeFacet.name}`
      : activeNavLabel || t("library.all_books", "All books");
  const filteredMetadataItems = useMemo(
    () => sortMetadataItems(currentFacetSection?.items || [], metadataSort),
    [currentFacetSection, metadataSort],
  );

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const nav = params.get("nav") || "";
    const rawFacet = params.get("facet") || "";
    const facetId = params.get("facet_id") || "";
    const name = params.get("name") || params.get("facet_name") || "";
    const searchParam = params.get("search") || params.get("q") || "";
    const collectionParam = params.get("collection") || "";
    const chipParam = params.get("chip") || "";
    const sortParam = params.get("sort") || "recently_added";

    if (searchParam !== search) {
      setSearch(searchParam);
    }
    if (collectionParam && collectionParam !== activeCollection) {
      setActiveCollection(collectionParam);
      setActiveNav("");
      setActiveFacet(null);
    }
    if (chipParam && chipParam !== activeChip) {
      setActiveChip(chipParam);
    }
    if (sortParam !== sort) {
      setSort(sortParam as any);
    }

    if (!nav && !rawFacet) return;

    // Support both singular and plural facet types (e.g. "publisher" -> "publishers", "author" -> "authors")
    const facetType = rawFacet ? (rawFacet.endsWith("s") || rawFacet === "series" ? rawFacet : `${rawFacet}s`) : nav;
    const targetNav = nav || facetType;

    if (!metadataNavIds.includes(targetNav)) {
      if (nav) setActiveNav(nav);
      return;
    }

    const section = facetSections.find(
      (item) => item.nav === targetNav || item.type === facetType || item.type === rawFacet,
    );

    const normalizedName = name.trim().toLowerCase();
    const matched = section?.items.find(
      (item) => item.id === facetId || item.name.trim().toLowerCase() === normalizedName,
    );

    if (rawFacet || name || facetId) {
      const isAlreadyActive = activeFacet &&
        activeFacet.type === (section?.type || facetType) &&
        (activeFacet.id === facetId || activeFacet.name.trim().toLowerCase() === normalizedName);

      if (isAlreadyActive) {
        return;
      }

      if (!matched && !facetId && name) {
        if (activeNav !== targetNav) {
          setActiveNav(targetNav);
          setActiveCollection("");
          setActiveChip(chipParam || "All");
        }
        if (metadataQuery !== name) {
          setMetadataQuery(name);
        }
        if (metadataAlpha !== "All") {
          setMetadataAlpha("All");
        }
        return;
      }

      setActiveNav(targetNav);
      setActiveCollection("");
      setActiveChip(chipParam || "All");
      setMetadataQuery("");
      setMetadataAlpha("All");
      setActiveFacet({
        type: section?.type || facetType,
        id: matched?.id || facetId || name,
        name: matched?.name || name || facetId,
      });
    } else if (nav) {
      setActiveNav(nav);
      setActiveFacet(null);
      setMetadataQuery("");
      setMetadataAlpha("All");
    }
  }, [
    location.search,
    facetSections,
    search,
    activeCollection,
    activeChip,
    activeNav,
    activeFacet,
    metadataQuery,
    metadataAlpha,
  ]);

  const metadataControls = (
    <div className="flex flex-col gap-3">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div className="relative w-full max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-base-content/45" />
          <input
            value={metadataQuery}
            onChange={(event) => setMetadataQuery(event.target.value)}
            className="input input-bordered input-sm w-full bg-base-100 pl-9"
            placeholder={t("library.filter_by_name", "Filter by name...")}
          />
        </div>
        <div className="join">
          <button
            className={`btn btn-sm join-item ${metadataSort === "name-asc" ? "btn-primary" : "btn-outline"}`}
            onClick={() => setMetadataSort("name-asc")}
            title={t("library.sort_name_asc", "Name A-Z")}
          >
            <ArrowDownAZ className="h-4 w-4" />
          </button>
          <button
            className={`btn btn-sm join-item ${metadataSort === "name-desc" ? "btn-primary" : "btn-outline"}`}
            onClick={() => setMetadataSort("name-desc")}
            title={t("library.sort_name_desc", "Name Z-A")}
          >
            <ArrowUpAZ className="h-4 w-4" />
          </button>
          <button
            className={`btn btn-sm join-item ${metadataSort === "count-desc" ? "btn-primary" : "btn-outline"}`}
            onClick={() => setMetadataSort("count-desc")}
            title={t("library.sort_count_desc", "Most books")}
          >
            {t("library.count", "Count")}
          </button>
        </div>
      </div>
      <div className="flex flex-wrap gap-1">
        {alphabetFilters.map((letter) => (
          <button
            key={letter}
            className={`btn btn-xs min-w-9 ${metadataAlpha === letter ? "btn-primary" : "btn-outline"}`}
            onClick={() => setMetadataAlpha(letter)}
          >
            {letter}
          </button>
        ))}
      </div>
    </div>
  );

  const renderMetadataIndex = () => {
    if (!currentFacetSection) return null;
    return (
      <MetadataIndexView
        section={currentFacetSection}
        filteredItems={filteredMetadataItems}
        controls={metadataControls}
        t={t}
        onFacetClick={handleFacetClick}
        hasMore={hasMoreFacetItems}
        loadingMore={isFetchingMoreFacetItems}
        onLoadMore={() => void fetchNextFacetPage()}
      />
    );
  };

  const renderBookList = () => (
    <>
      {isCatalogPage && activeFacet && (
        <section className="rounded-2xl bg-base-100 shadow-sm border border-base-200 p-4 sm:p-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-2xl font-black tracking-tight sm:text-3xl">
                {bookListTitle}
              </h2>
              <p className="mt-1 text-sm text-base-content/50">
                {new Intl.NumberFormat().format(books.length)}{" "}
                {t("library.books_indexed", "books indexed")}
              </p>
            </div>
            <button
              className="btn btn-outline btn-sm w-full sm:w-auto"
              onClick={() => setActiveFacet(null)}
            >
              {t("common.back", "Back")}
            </button>
          </div>
        </section>
      )}

      {!isCatalogPage && (
        <>
          <section className="rounded-2xl bg-base-100 shadow-sm border border-base-200 overflow-hidden">
            <div className="p-5 sm:p-6">
              <div className="flex flex-col gap-2">
                <span className="text-[11px] font-bold uppercase tracking-[0.22em] text-primary">
                  {t("library.local_library", "Local library")}
                </span>
                <h2 className="text-2xl sm:text-3xl font-black leading-tight">
                  {t("library.hero_title", "Welcome to NovelHub")}
                </h2>
                <p className="text-sm sm:text-base text-base-content/70 max-w-3xl">
                  {t(
                    "library.hero_desc",
                    "Your personal, fast, and highly customizable local light novel library.",
                  )}
                </p>
              </div>

              <div className="mt-5 grid grid-cols-1 overflow-hidden rounded-xl border border-base-200 bg-base-100 sm:grid-cols-3 shadow-2xs">
                <div className="p-3.5 sm:p-4">
                  <div className="text-[11px] font-medium uppercase tracking-wider text-base-content/50">
                    {t("library.books_indexed", "books indexed")}
                  </div>
                  <div className="mt-1.5 text-2xl font-black">
                    {stats.total_books || 0}
                  </div>
                </div>
                <div className="border-t border-base-200 p-3.5 sm:border-l sm:border-t-0 sm:p-4">
                  <div className="text-[11px] font-medium uppercase tracking-wider text-base-content/50">
                    {t("library.series_tracked", "series tracked")}
                  </div>
                  <div className="mt-1.5 text-2xl font-black">
                    {stats.series_tracked || 0}
                  </div>
                </div>
                <div className="border-t border-base-200 p-3.5 sm:border-l sm:border-t-0 sm:p-4">
                  <div className="text-[11px] font-medium uppercase tracking-wider text-base-content/50">
                    {t("library.need_review", "need review")}
                  </div>
                  <div className="mt-1.5 text-2xl font-black text-secondary">
                    {stats.need_review || 0}
                  </div>
                </div>
              </div>
            </div>
          </section>

          {(!publicSettings || publicSettings.home_sections.random_books !== false) && (
            <section className="rounded-2xl bg-base-100 shadow-sm border border-base-200 p-4 sm:p-5">
              <div className="mb-4 flex items-center gap-3">
                <span className="grid h-10 w-10 place-items-center rounded-xl bg-secondary/10 text-secondary">
                  <Shuffle className="h-5 w-5" />
                </span>
                <div>
                  <h3 className="text-lg font-black">
                    {t("library.random_books", "Random books")}
                  </h3>
                  <p className="text-sm text-base-content/50">
                    {t(
                      "library.random_books_hint",
                      "Refresh the shelf when you want something unexpected.",
                    )}
                  </p>
                </div>
              </div>

              {randomBooks.length > 0 ? (
                <div className="flex gap-4 overflow-x-auto pb-2 scrollbar-thin scroll-smooth items-stretch">
                  {randomBooks.map((book) => (
                    <div key={book.id} className="w-36 sm:w-44 shrink-0 flex flex-col">
                      <BookCard
                        book={book}
                        onClick={openBookDetail}
                      />
                    </div>
                  ))}
                </div>
              ) : (
                <div className="rounded-xl border border-dashed border-base-300 bg-base-100 p-8 text-center text-sm text-base-content/45 shadow-2xs">
                  {t(
                    "library.no_random_books",
                    "Add books to get random suggestions.",
                  )}
                </div>
              )}
            </section>
          )}

          {(!publicSettings || publicSettings.home_sections.top_books !== false) && (
            <section className="rounded-2xl bg-base-100 shadow-sm border border-base-200 p-4 sm:p-5">
              <div className="mb-4 flex items-center gap-3">
                <span className="grid h-10 w-10 place-items-center rounded-xl bg-primary/10 text-primary">
                  <Flame className="h-5 w-5" />
                </span>
                <div>
                  <h3 className="text-lg font-black">
                    {t("library.top_books", "Top books")}
                  </h3>
                  <p className="text-sm text-base-content/50">
                    {t("library.top_books_hint", "Most read books in your library.")}
                  </p>
                </div>
              </div>
              {topBooks.length > 0 ? (
                <div className="flex gap-4 overflow-x-auto pb-2 scrollbar-thin scroll-smooth items-stretch">
                  {topBooks.map((book) => (
                    <div key={book.id} className="w-36 sm:w-44 shrink-0 flex flex-col">
                      <BookCard key={book.id} book={book} onClick={openBookDetail} />
                    </div>
                  ))}
                </div>
              ) : (
                <div className="rounded-xl border border-dashed border-base-300 bg-base-100 p-8 text-center text-sm text-base-content/45 shadow-2xs">
                  {t(
                    "library.no_top_books",
                    "Books will appear here once they start getting reads.",
                  )}
                </div>
              )}
            </section>
          )}
          {smartFilters
            .filter((sf) => sf.is_pinned_home)
            .sort((a, b) => a.home_position - b.home_position)
            .map((sf) => (
              <SmartFilterShelf
                key={sf.id}
                filter={sf}
                onEdit={(filter) => {
                  setEditingSmartFilter(filter);
                  setShowSmartFilterModal(true);
                }}
                onDelete={(id) => deleteSmartFilter.mutate(id)}
                onBookClick={openBookDetail}
                onDragStart={(e) => handleDragStart(e, sf.id)}
                onDragOver={handleDragOver}
                onDrop={(e) => handleDrop(e, sf.id)}
              />
            ))}
        </>
      )}

      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div className="flex flex-wrap gap-2">
          {["All", "Reading", "Unread", "No cover"].map((chip) => (
            <button
              key={chip}
              onClick={() => setActiveChip(chip)}
              className={`btn btn-sm rounded-full border-base-300 ${activeChip === chip ? "btn-primary" : "btn-ghost bg-base-100 hover:bg-base-200"}`}
            >
              {t(`library.${chip.toLowerCase().replace(" ", "_")}`, chip)}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-2 w-full sm:w-auto">
          {user && (
            <button
              type="button"
              className="btn btn-sm btn-outline gap-1 shrink-0"
              onClick={() => {
                setSmartCollectionName(search || activeCollection || activeNav || "");
                setShowSaveSearchModal(true);
              }}
              title={t("library.save_search", "Save current search")}
            >
              <BookmarkPlus className="h-4 w-4" />
              <span className="hidden sm:inline">{t("library.save_search", "Save current search")}</span>
            </button>
          )}
          <select
            className="select select-bordered select-sm w-full sm:w-auto bg-base-100"
            value={sort}
            onChange={(e) => {
              const newSort = e.target.value as any;
              setSort(newSort);
              const params = new URLSearchParams(location.search);
              if (newSort && newSort !== "recently_added") {
                params.set("sort", newSort);
              } else {
                params.delete("sort");
              }
              navigate(`/${params.toString() ? `?${params.toString()}` : ""}`, { replace: true });
            }}
          >
            <option value="recently_added">{t("library.recently_added", "Recently added")}</option>
            <option value="title_az">{t("library.title_az", "Title A-Z")}</option>
            <option value="series_order">{t("library.series_order", "Series order")}</option>
          </select>
        </div>
      </div>

      <section className="rounded-2xl bg-base-100 shadow-sm border border-base-200 p-4 sm:p-5">
        {!isCatalogPage && (
          <div className="mb-4 flex items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <span className="grid h-10 w-10 place-items-center rounded-xl bg-primary/10 text-primary">
                <LayoutGrid className="h-5 w-5" />
              </span>
              <div>
                <h3 className="text-lg font-black">{bookListTitle}</h3>
                <p className="text-sm text-base-content/50">
                  {new Intl.NumberFormat().format(books.length)}{" "}
                  {t("library.books_indexed", "books indexed")}
                </p>
              </div>
            </div>
          </div>
        )}

        {loading && books.length === 0 ? (
          <div className="flex justify-center items-center py-20">
            <span className="loading loading-spinner loading-lg text-primary"></span>
          </div>
        ) : books.length > 0 ? (
          <>
            <BookGrid books={books} onBookClick={openBookDetail} />
            {((activeNav === "bookmarks" && hasMoreBookmarks) || (activeNav !== "bookmarks" && hasMoreBooks)) && (
              <div className="mt-8 flex justify-center">
                <button
                  className="btn btn-primary btn-outline"
                  onClick={() => activeNav === "bookmarks" ? fetchNextBookmarks() : fetchNextBooks()}
                  disabled={isFetchingMoreBookmarks || isFetchingMoreBooks}
                >
                  {(isFetchingMoreBookmarks || isFetchingMoreBooks) ? (
                    <span className="loading loading-spinner loading-sm"></span>
                  ) : (
                    t("common.load_more", "Load more")
                  )}
                </button>
              </div>
            )}
          </>
        ) : (
          <div className="rounded-xl border border-dashed border-base-300 bg-base-100 p-12 text-center shadow-2xs">
            <BookOpen className="mx-auto mb-3 h-10 w-10 text-base-content/25" />
            <p className="font-bold text-base-content/70">
              {t("library.no_books_found", "No books found")}
            </p>
            <p className="mt-1 text-sm text-base-content/45">
              {t(
                "library.no_books_found_hint",
                "Try another filter or add books to your library.",
              )}
            </p>
          </div>
        )}
      </section>
    </>
  );

  return (
    <div className="drawer lg:drawer-open bg-base-100 min-h-screen font-sans">
      <input id="main-drawer" type="checkbox" className="drawer-toggle" />

      {/* Main Content */}
      <div className="drawer-content flex flex-col h-screen overflow-hidden">
        <TopNav showSidebarToggle={true} />

        {/* Scrollable Main Area */}
        <div className="flex-1 overflow-y-auto p-4 sm:p-5 lg:p-6">
          <div
            className={`mx-auto grid w-full max-w-[1700px] grid-cols-1 gap-5 ${isCatalogPage || book_id ? "" : "xl:grid-cols-[minmax(0,1fr)_300px] 2xl:grid-cols-[minmax(0,1fr)_320px]"}`}
          >
            <main className="min-w-0 flex flex-col gap-5">
              {book_id ? (
                <BookDetailPage />
              ) : (
                isMetadataIndex ? renderMetadataIndex() : renderBookList()
              )}
            </main>

            {!isCatalogPage && !book_id && (
              <aside className="min-w-0 xl:sticky xl:top-0 xl:self-start flex flex-col gap-5">
                <RecentlyReadPanel
                  className="mt-0"
                  items={recentReading}
                  onOpen={(item) =>
                    navigate(
                      `/reader/${item.book_id}${item.file_id ? `?file_id=${encodeURIComponent(item.file_id)}` : ""}`,
                    )
                  }
                  t={t}
                />
                {hasPermission(user, "user.stats.read") && (
                  <div className="rounded-xl border border-base-300 bg-base-100 p-4 shadow-sm">
                    <div className="mb-3 flex items-center justify-between gap-2">
                      <h4 className="flex items-center gap-1.5 text-xs font-bold text-base-content/80 whitespace-nowrap overflow-hidden text-ellipsis">
                        <Activity className="h-4 w-4 shrink-0 text-primary" />
                        <span>{t("analytics.short_title", "Activity")}</span>
                      </h4>
                      <Link
                        to="/analytics"
                        className="text-xs font-semibold text-primary hover:underline shrink-0"
                      >
                        {t("common.view_analytics", "Analytics →")}
                      </Link>
                    </div>
                    <ReadingHeatmap />
                  </div>
                )}
              </aside>
            )}
          </div>
        </div>
      </div>
      {/* End Main Content */}

      <LibrarySidebar
        t={t}
        user={user}
        primaryNavItems={primaryNavItems}
        facetSections={facetSections}
        secondaryNavItems={secondaryNavItems}
        collections={collections}
        activeNav={activeNav}
        activeFacet={activeFacet}
        activeCollection={activeCollection}
        onNavClick={handleNavClick}
        onCollectionClick={handleCollectionClick}
        onNewCollection={() => {
          setShowNewCollectionModal(true);
          setCollectionError("");
        }}
        hasMoreCollections={hasMoreCollections}
        onLoadMoreCollections={() => fetchNextCollections()}
        isFetchingMoreCollections={isFetchingMoreCollections}
        smartCollections={smartCollections}
        onSmartCollectionClick={handleSmartCollectionClick}
        onDeleteSmartCollection={(id) => deleteSmartCollection.mutate(id)}
        smartFilters={smartFilters}
        onSmartFilterClick={handleSmartFilterClick}
        onEditSmartFilter={(sf) => {
          setEditingSmartFilter(sf);
          setShowSmartFilterModal(true);
        }}
        onDeleteSmartFilter={(id) => deleteSmartFilter.mutate(id)}
        onNewSmartFilter={() => {
          setEditingSmartFilter(null);
          setShowSmartFilterModal(true);
        }}
        activeSmartFilterId={activeSmartFilterId || undefined}
      />

      <SmartFilterBuilderModal
        isOpen={showSmartFilterModal}
        onClose={() => {
          setShowSmartFilterModal(false);
          setEditingSmartFilter(null);
        }}
        filterToEdit={editingSmartFilter}
      />

      <LoginView />
      <UserProfile />
      {/* New Collection Modal */}
      <dialog className={`modal ${showNewCollectionModal ? "modal-open" : ""}`}>
        <div className="modal-box">
          <h3 className="font-bold text-lg border-b border-base-200 pb-4 mb-4">
            {t("library.new_collection", "New Collection")}
          </h3>
          {collectionError && (
            <div className="alert alert-error mb-4 py-2 rounded-lg text-sm flex items-center gap-2">
              <AlertCircle className="w-4 h-4" />
              {collectionError}
            </div>
          )}
          <form
            onSubmit={handleCreateCollection}
            className="flex flex-col gap-4"
          >
            <div className="flex flex-col gap-1.5 w-full">
              <label className="text-sm font-medium pl-1">
                {t("library.enter_collection_name", "Enter collection name")}
              </label>
              <input
                type="text"
                placeholder={t(
                  "library.collection_name_placeholder",
                  "Collection name...",
                )}
                className="input input-bordered w-full"
                value={newCollectionName}
                onChange={(e) => setNewCollectionName(e.target.value)}
                required
                autoFocus
              />
            </div>
            <div className="modal-action">
              <button
                type="button"
                onClick={() => setShowNewCollectionModal(false)}
                className="btn btn-ghost"
              >
                {t("common.cancel", "Cancel")}
              </button>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={!newCollectionName.trim()}
              >
                {t("common.create", "Create")}
              </button>
            </div>
          </form>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button onClick={() => setShowNewCollectionModal(false)}>
            close
          </button>
        </form>
      </dialog>

      {/* Save current search as a smart collection */}
      <dialog className={`modal ${showSaveSearchModal ? "modal-open" : ""}`}>
        <div className="modal-box">
          <h3 className="font-bold text-lg border-b border-base-200 pb-4 mb-4">
            {t("library.save_search", "Save current search")}
          </h3>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              const name = smartCollectionName.trim();
              if (!name) return;
              createSmartCollection.mutate(
                { name, rule: searchParams as SmartCollectionRule },
                { onSuccess: () => setShowSaveSearchModal(false) },
              );
            }}
            className="flex flex-col gap-4"
          >
            <div className="flex flex-col gap-1.5 w-full">
              <label className="text-sm font-medium pl-1">
                {t("library.smart_collection_name", "Smart collection name")}
              </label>
              <input
                type="text"
                className="input input-bordered w-full"
                value={smartCollectionName}
                onChange={(e) => setSmartCollectionName(e.target.value)}
                required
                autoFocus
              />
              <span className="text-xs text-base-content/50 pl-1">
                {t("library.save_search_desc", "Saves the current filters so you can reopen this exact view later.")}
              </span>
            </div>
            <div className="modal-action">
              <button type="button" onClick={() => setShowSaveSearchModal(false)} className="btn btn-ghost">
                {t("common.cancel", "Cancel")}
              </button>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={!smartCollectionName.trim() || createSmartCollection.isPending}
              >
                {t("common.save", "Save")}
              </button>
            </div>
          </form>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button onClick={() => setShowSaveSearchModal(false)}>close</button>
        </form>
      </dialog>

      <BulkActionToolbar
        selectedCount={selectedBookIds.length}
        onClearSelection={() => setSelectedBookIds([])}
        onBulkMove={() => setShowBulkMoveModal(true)}
        onBulkAddTags={() => setShowBulkTagModal(true)}
        onBulkDelete={() => setShowBulkDeleteModal(true)}
      />

      <BulkDeleteModal
        isOpen={showBulkDeleteModal}
        bookIds={selectedBookIds}
        onClose={() => setShowBulkDeleteModal(false)}
        onSuccess={() => setSelectedBookIds([])}
      />
      <BulkMoveModal
        isOpen={showBulkMoveModal}
        bookIds={selectedBookIds}
        onClose={() => setShowBulkMoveModal(false)}
        onSuccess={() => setSelectedBookIds([])}
      />
      <BulkTagModal
        isOpen={showBulkTagModal}
        bookIds={selectedBookIds}
        onClose={() => setShowBulkTagModal(false)}
        onSuccess={() => setSelectedBookIds([])}
      />
    </div>
  );
};
