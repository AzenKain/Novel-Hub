import { RecentlyReadPanel, BookDetailModal } from "@/components/book-detail";
import { LoginView } from "@/components/common";
import { LibrarySidebar, MetadataIndexView, type LibraryNavItem, type MetadataFacetSection } from "@/components/library";
import { BookCard, BookGrid, LanguageSwitcher, ThemeController } from "@/components/ui";
import { UserProfile } from "@/pages/user";
import { featureService } from "@/services";
import type { Book, MetadataCount } from "@/types";
import { useQueryClient } from "@tanstack/react-query";
import React, { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";

import { settingsKeyToNavId } from "@/constants";
import {
  useBookmarkedBooksQuery,
  useBooksQuery,
  useCollectionsQuery,
  useDebounce,
  useHotBooksQuery,
  useLibraryStatsQuery,
  useMetadataFacetQuery,
  usePublicSettings,
  useRandomBooksQuery,
  useReadingHistoryQuery,
} from "@/hooks";
import {
  alphabetFilters,
  filterMetadataItems,
  metadataNavIds,
} from "@/lib/libraryMetadata";
import { useAuthStore, useLibraryStore } from "@/stores";
import {
  AlertCircle,
  Archive,
  ArrowDownAZ,
  ArrowUpAZ,
  Bookmark,
  BookOpen,
  Building2,
  Download,
  Eye,
  EyeOff,
  FileType,
  Flame,
  Languages,
  Layers,
  LayoutDashboard,
  LayoutGrid,
  Menu,
  Search,
  Shuffle,
  Star,
  Users
} from "lucide-react";

function isItemVisible(visibleKeys: string[] | undefined, id: string): boolean {
  if (!visibleKeys || visibleKeys.length === 0) return true;
  return visibleKeys.some((key) => (settingsKeyToNavId[key] || key) === id);
}

export const LibraryWorkspace = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { bookId } = useParams<{ bookId: string }>();
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
  })));
  const publicSettings = usePublicSettings();
  const debouncedSearch = useDebounce(search, 500);

  const { data: hotBooksData } = useHotBooksQuery(6);
  const topBooks = hotBooksData || [];

  const { data: randomBooksData, refetch: refetchRandomBooks } = useRandomBooksQuery(6);
  const randomBooks = randomBooksData || [];

  const recentReading = recentHistory.slice(0, 4);

  const openBookDetail = (book: Book) => {
    setSelectedBook(book);
    navigate(`/books/${encodeURIComponent(book.id)}`);
  };

  const handleNavClick = (nav: string) => {
    setActiveNav(nav);
    setActiveCollection("");
    setActiveFacet(null);
    setActiveChip("All");
    setMetadataQuery("");
    setMetadataAlpha("All");
  };

  const handleCollectionClick = (collection: string) => {
    setActiveCollection(collection);
    setActiveNav("");
    setActiveFacet(null);
    setActiveChip("All");
    setMetadataQuery("");
    setMetadataAlpha("All");
  };

  const handleFacetClick = (type: string, item: MetadataCount, nav: string) => {
    setActiveNav(nav);
    setActiveCollection("");
    setActiveFacet({ type, id: item.id, name: item.name });
    setActiveChip("All");
  };

  const queryClient = useQueryClient();
  const isMetadataNav = metadataNavIds.includes(activeNav) && !activeFacet;

  const searchParams = useMemo(() => ({
    search: debouncedSearch,
    nav: activeNav,
    collection: activeCollection,
    chip: activeChip,
    facet: activeFacet?.type,
    facet_id: activeFacet?.id,
  }), [debouncedSearch, activeNav, activeCollection, activeChip, activeFacet]);

  const { data: booksData, isLoading: normalLoading } = useBooksQuery(
    searchParams,
    !isMetadataNav && activeNav !== "bookmarks"
  );

  const { data: bookmarkedBooksData, isLoading: bookmarksLoading } = useBookmarkedBooksQuery(
    activeNav === "bookmarks" && !!user
  );

  const { data: statsData } = useLibraryStatsQuery();
  const { data: collectionsData } = useCollectionsQuery(!!user);
  const { data: historyData } = useReadingHistoryQuery(!!user);

  const { data: authorsFacet } = useMetadataFacetQuery("authors");
  const { data: seriesFacet } = useMetadataFacetQuery("series");
  const { data: tagsFacet } = useMetadataFacetQuery("tags");
  const { data: publishersFacet } = useMetadataFacetQuery("publishers");
  const { data: languagesFacet } = useMetadataFacetQuery("languages");
  const { data: formatsFacet } = useMetadataFacetQuery("formats");

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
    } else {
      if (booksData) {
        setBooks(booksData);
        if (booksData.length > 0 && !selectedBook) {
          setSelectedBook(booksData[0]);
        }
      }
    }
  }, [isMetadataNav, activeNav, booksData, bookmarkedBooksData, setBooks, setSelectedBook]);

  useEffect(() => {
    if (activeNav === "bookmarks") {
      setLoading(bookmarksLoading);
    } else {
      setLoading(normalLoading);
    }
  }, [activeNav, bookmarksLoading, normalLoading, setLoading]);

  useEffect(() => {
    if (statsData) setStats(statsData);
  }, [statsData, setStats]);

  useEffect(() => {
    if (collectionsData) setCollections(collectionsData);
  }, [collectionsData, setCollections]);

  useEffect(() => {
    if (historyData) setRecentHistory(historyData);
  }, [historyData, setRecentHistory]);

  useEffect(() => {
    setMetadataFacets({
      authors: authorsFacet || [],
      series: seriesFacet || [],
      tags: tagsFacet || [],
      publishers: publishersFacet || [],
      languages: languagesFacet || [],
      formats: formatsFacet || [],
    });
  }, [authorsFacet, seriesFacet, tagsFacet, publishersFacet, languagesFacet, formatsFacet, setMetadataFacets]);

  const handleCreateCollection = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!newCollectionName.trim()) return;
    setCollectionError("");
    try {
      const res = await featureService.createCollection(
        newCollectionName.trim(),
      );
      if (res.status && res.data) {
        setNewCollectionName("");
        setShowNewCollectionModal(false);
        await queryClient.invalidateQueries({ queryKey: ["collections"] });
      }
    } catch (err) {
      setCollectionError(
        t("library.fail_create_collection", "Failed to create collection"),
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
}, [visibleItems]);

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
}, [visibleItems, metadataFacets]);

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
}, [visibleItems]);

  const currentFacetSection = facetSections.find(
    (section) => section.nav === activeNav,
  );
  const isMetadataIndex = !!currentFacetSection && !activeFacet;
  const isCatalogPage = !!currentFacetSection;
  const activeNavLabel =
    primaryNavItems.find((item) => item.id === activeNav)?.label ||
    currentFacetSection?.label ||
    secondaryNavItems.find((item) => item.id === activeNav)?.label;
  const bookListTitle = activeFacet
    ? `${currentFacetSection?.label || activeNavLabel}: ${activeFacet.name}`
    : activeNavLabel || t("library.all_books", "All books");
  const filteredMetadataItems = useMemo(() => {
    const items = currentFacetSection?.items || [];
    return filterMetadataItems(
      items,
      metadataQuery,
      metadataAlpha,
      metadataSort,
    );
  }, [currentFacetSection, metadataAlpha, metadataQuery, metadataSort]);

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const nav = params.get("nav") || "";
    const facet = params.get("facet") || "";
    const facetId = params.get("facet_id") || "";
    const name = params.get("name") || "";
    if (!nav || !facet || !metadataNavIds.includes(nav)) return;

    const section = facetSections.find(
      (item) => item.nav === nav && item.type === facet,
    );
    if (!section) return;

    const normalizedName = name.trim().toLowerCase();
    const matched =
      section.items.find((item) => item.id === facetId) ||
      section.items.find(
        (item) => item.name.trim().toLowerCase() === normalizedName,
      );

    if (!matched && !facetId) return;

    setActiveNav(nav);
    setActiveCollection("");
    setActiveChip("All");
    setMetadataQuery("");
    setMetadataAlpha("All");
    setActiveFacet({
      type: facet,
      id: matched?.id || facetId,
      name: matched?.name || name || facetId,
    });
  }, [location.search, metadataFacets]);

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

              <div className="mt-5 grid grid-cols-1 overflow-hidden rounded-xl border border-base-200 bg-base-200/30 sm:grid-cols-3">
                <div className="p-3.5 sm:p-4">
                  <div className="text-[11px] font-medium uppercase tracking-wider text-base-content/50">
                    {t("library.books_indexed", "books indexed")}
                  </div>
                  <div className="mt-1.5 text-2xl font-black">
                    {stats.totalBooks || 0}
                  </div>
                </div>
                <div className="border-t border-base-200 p-3.5 sm:border-l sm:border-t-0 sm:p-4">
                  <div className="text-[11px] font-medium uppercase tracking-wider text-base-content/50">
                    {t("library.series_tracked", "series tracked")}
                  </div>
                  <div className="mt-1.5 text-2xl font-black">
                    {stats.seriesTracked || 0}
                  </div>
                </div>
                <div className="border-t border-base-200 p-3.5 sm:border-l sm:border-t-0 sm:p-4">
                  <div className="text-[11px] font-medium uppercase tracking-wider text-base-content/50">
                    {t("library.need_review", "need review")}
                  </div>
                  <div className="mt-1.5 text-2xl font-black text-secondary">
                    {stats.needReview || 0}
                  </div>
                </div>
              </div>
            </div>
          </section>

          {(!publicSettings || publicSettings.home_sections.random_books !== false) && (
          <section className="rounded-2xl bg-base-100 shadow-sm border border-base-200 p-4 sm:p-5">
            <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex items-center gap-3">
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
              <button
                className="btn btn-outline btn-sm w-full gap-2 sm:w-auto"
                onClick={() => refetchRandomBooks()}
              >
                <Shuffle className="h-4 w-4" />
                {t("library.shuffle_books", "Shuffle")}
              </button>
            </div>

            {randomBooks.length > 0 ? (
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 2xl:grid-cols-6">
                {randomBooks.map((book) => (
                  <BookCard
                    key={book.id}
                    book={book}
                    onClick={openBookDetail}
                  />
                ))}
              </div>
            ) : (
              <div className="rounded-xl border border-dashed border-base-300 bg-base-200/30 p-8 text-center text-sm text-base-content/45">
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
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 2xl:grid-cols-6">
                {topBooks.map((book) => (
                  <BookCard key={book.id} book={book} onClick={openBookDetail} />
                ))}
              </div>
            ) : (
              <div className="rounded-xl border border-dashed border-base-300 bg-base-200/30 p-8 text-center text-sm text-base-content/45">
                {t(
                  "library.no_top_books",
                  "Books will appear here once they start getting reads.",
                )}
              </div>
            )}
          </section>
          )}
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
          <button
            className="btn btn-sm btn-ghost bg-base-100 rounded-full border-base-300 hover:bg-base-200"
            onClick={() => navigate("/duplicates")}
          >
            {t("library.duplicates", "Duplicates")}
          </button>
        </div>

        <select className="select select-bordered select-sm w-full sm:w-auto bg-base-100">
          <option>{t("library.recently_added", "Recently added")}</option>
          <option>{t("library.title_az", "Title A-Z")}</option>
          <option>{t("library.series_order", "Series order")}</option>
        </select>
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

        {loading ? (
          <div className="flex justify-center items-center py-20">
            <span className="loading loading-spinner loading-lg text-primary"></span>
          </div>
        ) : books.length > 0 ? (
          <BookGrid books={books} onBookClick={openBookDetail} />
        ) : (
          <div className="rounded-xl border border-dashed border-base-300 bg-base-200/30 p-12 text-center">
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
    <div className="drawer lg:drawer-open bg-base-200 min-h-screen">
      <input id="main-drawer" type="checkbox" className="drawer-toggle" />

      {/* Main Content */}
      <div className="drawer-content flex flex-col h-screen overflow-hidden">
        {/* Navbar */}
        <div className="navbar flex-wrap gap-2 bg-base-100 shadow-sm border-b border-base-200 z-10 px-3 sm:px-4">
          <div className="flex-none lg:hidden">
            <label
              htmlFor="main-drawer"
              aria-label="open sidebar"
              className="btn btn-square btn-ghost"
            >
              <Menu className="w-5 h-5" />
            </label>
          </div>

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
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
          </div>

          <div className="flex shrink-0 items-center gap-1 sm:gap-2">
            <ThemeController />
            <LanguageSwitcher />

            {user ? (
              <>
                {user.roles?.some(
                  (r) => r.name === "ADMIN" || r.name === "MOD",
                ) && (
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
                    className="mt-3 z-1 p-2 shadow menu menu-sm dropdown-content bg-base-100 rounded-box w-52 border border-base-200"
                  >
                    <li>
                      <span className="font-semibold opacity-60 px-4 py-2 truncate block">
                        {user.email}
                      </span>
                    </li>
                    <li>
                      <button onClick={() => setProfileModalOpen(true)}>
                        {t("user.profile", "Profile")}
                      </button>
                    </li>
                    <li className="sm:hidden">
                      {user.roles?.some(
                        (r) => r.name === "ADMIN" || r.name === "MOD",
                      ) && (
                        <Link to="/admin">{t("admin.dashboard", "Admin")}</Link>
                      )}
                    </li>
                    <li>
                      <button className="text-error" onClick={() => logout()}>
                        {t("auth.logout", "Logout")}
                      </button>
                    </li>
                  </ul>
                </div>
              </>
            ) : (
              <>
                <Link to="/register" className="btn btn-ghost btn-sm sm:btn-md">
                  {t("auth.register", "Register")}
                </Link>
                <button
                  onClick={() => setLoginModalOpen(true)}
                  className="btn btn-primary btn-sm sm:btn-md"
                >
                  {t("auth.login", "Login")}
                </button>
              </>
            )}
          </div>
        </div>

        {/* Scrollable Main Area */}
        <div className="flex-1 overflow-y-auto p-4 sm:p-5 lg:p-6">
          <div
            className={`mx-auto grid w-full max-w-375 grid-cols-1 gap-5 ${isCatalogPage ? "" : "xl:grid-cols-[minmax(0,1fr)_300px] 2xl:grid-cols-[minmax(0,1fr)_320px]"}`}
          >
            <main className="min-w-0 flex flex-col gap-5">
              {isMetadataIndex ? renderMetadataIndex() : renderBookList()}
            </main>

            {!isCatalogPage && (
              <aside className="min-w-0 xl:sticky xl:top-0 xl:self-start">
                <RecentlyReadPanel
                  className="mt-0"
                  items={recentReading}
                  onOpen={(item) =>
                    navigate(
                      `/reader/${item.bookId}${item.fileId ? `?file_id=${encodeURIComponent(item.fileId)}` : ""}`,
                    )
                  }
                  t={t}
                />
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
      {bookId && <BookDetailModal bookId={bookId} onClose={() => navigate("/")} />}
    </div>
  );
};
