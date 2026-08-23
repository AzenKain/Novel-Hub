import React, { useState, useEffect, useMemo } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useShallow } from "zustand/react/shallow";
import {
  Search,
  Filter,
  X,
  RotateCcw,
  BookOpen,
  SlidersHorizontal,
  Layers,
  User,
  Building2,
  Globe,
  FileType,
  Sparkles,
  ArrowLeft,
} from "lucide-react";

import { TopNav } from "@/components/common/TopNav";
import { LibrarySidebar } from "@/components/library/LibrarySidebar";
import { BookGrid } from "@/components/ui/BookGrid";
import { useBooksQuery } from "@/hooks/useBooksQuery";
import { useAdvancedSearchFacets } from "@/hooks/useAdvancedSearchQueries";
import { useDebounce } from "@/hooks/useDebounce";
import { useAuthStore, useLibraryStore } from "@/stores";
import type { MetadataCount } from "@/types";

export const AdvancedSearchPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const user = useAuthStore((state) => state.user);
  const {
    activeNav,
    setActiveNav,
    activeFacet,
    setActiveFacet,
    activeCollection,
    setActiveCollection,
  } = useLibraryStore(
    useShallow((state) => ({
      activeNav: state.activeNav,
      setActiveNav: state.setActiveNav,
      activeFacet: state.activeFacet,
      setActiveFacet: state.setActiveFacet,
      activeCollection: state.activeCollection,
      setActiveCollection: state.setActiveCollection,
    }))
  );

  // Form State initialized from URL search params
  const [queryInput, setQueryInput] = useState(searchParams.get("q") || searchParams.get("search") || "");
  const debouncedQuery = useDebounce(queryInput, 400);
  const [selectedFormat, setSelectedFormat] = useState(searchParams.get("format") || "");
  const [selectedSeries, setSelectedSeries] = useState(searchParams.get("series") || "");
  const [selectedAuthor, setSelectedAuthor] = useState(searchParams.get("author") || "");
  const [selectedPublisher, setSelectedPublisher] = useState(searchParams.get("publisher") || "");
  const [selectedLanguage, setSelectedLanguage] = useState(searchParams.get("language") || "");
  const [selectedTag, setSelectedTag] = useState(searchParams.get("tag") || "");
  const [selectedSort, setSelectedSort] = useState(searchParams.get("sort") || "title_asc");

  const [isFilterPanelOpen, setIsFilterPanelOpen] = useState(true);

  // Sync state when URL searchParams change
  useEffect(() => {
    setQueryInput(searchParams.get("q") || searchParams.get("search") || "");
    setSelectedFormat(searchParams.get("format") || "");
    setSelectedSeries(searchParams.get("series") || "");
    setSelectedAuthor(searchParams.get("author") || "");
    setSelectedPublisher(searchParams.get("publisher") || "");
    setSelectedLanguage(searchParams.get("language") || "");
    setSelectedTag(searchParams.get("tag") || "");
    setSelectedSort(searchParams.get("sort") || "title_asc");
  }, [searchParams]);

  // Update URL helper
  const updateUrlParams = (updates: Record<string, string>) => {
    const params = new URLSearchParams(searchParams);
    Object.entries(updates).forEach(([key, val]) => {
      if (val) {
        params.set(key, val);
      } else {
        params.delete(key);
      }
    });
    setSearchParams(params, { replace: true });
  };

  // Fetch facet metadata via extracted hook
  const { formats, series, authors, publishers, languages } = useAdvancedSearchFacets();

  // Construct Search Query Params for API
  const apiQueryParams = useMemo(() => {
    const params: Record<string, unknown> = {
      limit: 24,
    };
    if (debouncedQuery.trim()) params.search = debouncedQuery.trim();
    if (selectedFormat) params.chip = selectedFormat;
    if (selectedSeries) {
      params.facet = "series";
      params.facet_id = selectedSeries;
    } else if (selectedAuthor) {
      params.facet = "authors";
      params.facet_id = selectedAuthor;
    } else if (selectedPublisher) {
      params.facet = "publishers";
      params.facet_id = selectedPublisher;
    } else if (selectedLanguage) {
      params.facet = "languages";
      params.facet_id = selectedLanguage;
    } else if (selectedTag) {
      params.facet = "tags";
      params.facet_id = selectedTag;
    }
    return params;
  }, [
    debouncedQuery,
    selectedFormat,
    selectedSeries,
    selectedAuthor,
    selectedPublisher,
    selectedLanguage,
    selectedTag,
  ]);

  const {
    data: searchResultsData,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    fetchNextPage,
  } = useBooksQuery(apiQueryParams);

  const books = useMemo(() => {
    let allBooks = searchResultsData?.pages.flatMap((page) => page.data || []) || [];
    // Filter by format if selected
    if (selectedFormat) {
      allBooks = allBooks.filter((b) =>
        b.file_path?.toLowerCase().endsWith(`.${selectedFormat.toLowerCase()}`)
      );
    }
    // Client-side sort options if needed
    if (selectedSort === "title_desc") {
      allBooks = [...allBooks].sort((a, b) => b.title.localeCompare(a.title));
    } else if (selectedSort === "title_asc") {
      allBooks = [...allBooks].sort((a, b) => a.title.localeCompare(b.title));
    }
    return allBooks;
  }, [searchResultsData, selectedFormat, selectedSort]);

  const availableFormats = useMemo(() => {
    const fetchedFormats = formats.map((f) => f.name.toLowerCase());
    // Fallback list ensures chips always appear even before API responds
    const defaultFormats = ["epub", "pdf", "mobi", "azw3", "fb2", "cbz", "cbr", "docx", "odt", "html", "txt", "rtf", "djvu", "chm", "mp3", "m4b", "flac", "zip", "rar", "csv", "tex", "pptx", "ppt", "odp", "xlsx", "xls", "ods"];
    const set = new Set([...fetchedFormats, ...defaultFormats]);
    return Array.from(set);
  }, [formats]);

  const hasActiveFilters = Boolean(
    debouncedQuery.trim() ||
      selectedFormat ||
      selectedSeries ||
      selectedAuthor ||
      selectedPublisher ||
      selectedLanguage ||
      selectedTag
  );

  const handleResetFilters = () => {
    setQueryInput("");
    setSelectedFormat("");
    setSelectedSeries("");
    setSelectedAuthor("");
    setSelectedPublisher("");
    setSelectedLanguage("");
    setSelectedTag("");
    setSelectedSort("title_asc");
    setSearchParams({}, { replace: true });
  };

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    updateUrlParams({ q: queryInput.trim() });
  };

  const primaryNavItems = [
    { id: "", label: t("library.all_books", "All Books"), icon: <BookOpen className="w-4 h-4" /> },
  ];

  return (
    <div className="drawer min-h-screen bg-base-200/40 text-base-content">
      <input id="main-drawer" type="checkbox" className="drawer-toggle" />
      <div className="drawer-content flex flex-col min-h-screen">
        <TopNav showSidebarToggle={true} />

        <div className="flex-1 p-3 sm:p-6 max-w-7xl w-full mx-auto space-y-4">
          {/* Back Button */}
          <button
            type="button"
            onClick={() => navigate(-1)}
            className="btn btn-ghost btn-sm rounded-xl gap-1.5 text-base-content/70 hover:text-base-content -mb-2"
          >
            <ArrowLeft className="w-4 h-4" />
            {t("common.back", "Back")}
          </button>

          {/* Header Section */}
          <div className="bg-base-100 border border-base-200 shadow-sm rounded-3xl p-5 sm:p-7 transition-all">
            <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="badge badge-primary badge-sm font-bold uppercase tracking-wider">
                    <Sparkles className="w-3 h-3 mr-1" />
                    {t("search.advanced_title", "Advanced Search")}
                  </span>
                  {books.length > 0 && (
                    <span className="badge badge-ghost badge-sm font-semibold">
                      {books.length} {t("search.results_suffix", "results")}
                    </span>
                  )}
                </div>
                <h1 className="text-2xl sm:text-3xl font-black tracking-tight text-base-content">
                  {t("search.header_title", "Search Library")}
                </h1>
                <p className="text-xs sm:text-sm text-base-content/60">
                  {t("search.header_subtitle", "Find any book by title, author, format, language, or series.")}
                </p>
              </div>

              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setIsFilterPanelOpen(!isFilterPanelOpen)}
                  className={`btn btn-sm sm:btn-md gap-2 rounded-xl transition-all ${
                    isFilterPanelOpen ? "btn-primary" : "btn-outline btn-neutral"
                  }`}
                >
                  <SlidersHorizontal className="w-4 h-4" />
                  <span>{t("search.filters", "Filters")}</span>
                  {hasActiveFilters && (
                    <span className="badge badge-xs badge-secondary font-bold">!</span>
                  )}
                </button>
                {hasActiveFilters && (
                  <button
                    type="button"
                    onClick={handleResetFilters}
                    className="btn btn-ghost btn-sm sm:btn-md text-error hover:bg-error/10 rounded-xl gap-1.5"
                    title={t("search.reset_filters", "Reset Filters")}
                  >
                    <RotateCcw className="w-4 h-4" />
                    <span className="hidden sm:inline">{t("search.reset_filters", "Reset")}</span>
                  </button>
                )}
              </div>
            </div>

            {/* Main Search Input Form */}
            <form onSubmit={handleSearchSubmit} className="mt-5 relative">
              <div className="relative flex items-center">
                <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-base-content/40">
                  <Search className="w-5 h-5" />
                </div>
                <input
                  type="text"
                  placeholder={t("search.input_placeholder", "Enter title, author, description, or keyword...")}
                  className="input input-bordered input-md sm:input-lg w-full pl-12 pr-28 rounded-2xl bg-base-200/40 focus:bg-base-100 font-medium text-sm sm:text-base transition-all"
                  value={queryInput}
                  onChange={(e) => setQueryInput(e.target.value)}
                />
                {queryInput && (
                  <button
                    type="button"
                    onClick={() => {
                      setQueryInput("");
                      updateUrlParams({ q: "" });
                    }}
                    className="absolute right-24 text-base-content/40 hover:text-base-content p-1"
                  >
                    <X className="w-4 h-4" />
                  </button>
                )}
                <button
                  type="submit"
                  className="absolute right-2 btn btn-primary btn-sm sm:btn-md rounded-xl font-bold px-4"
                >
                  {t("common.search", "Search")}
                </button>
              </div>
            </form>
          </div>

          {/* Filter Panel (Collapsible) */}
          {isFilterPanelOpen && (
            <div className="bg-base-100 border border-base-200 shadow-sm rounded-3xl p-5 space-y-5 animate-in fade-in slide-in-from-top-3">
              <div className="flex items-center justify-between border-b border-base-200 pb-3">
                <h3 className="text-sm font-bold uppercase tracking-wider text-base-content/70 flex items-center gap-2">
                  <Filter className="w-4 h-4 text-primary" />
                  {t("search.filter_options", "Refine Results")}
                </h3>
                {hasActiveFilters && (
                  <span className="text-xs text-primary font-medium">
                    {t("search.active_filters_count", "Filters applied")}
                  </span>
                )}
              </div>

              {/* Format Filter Chips */}
              <div className="space-y-2">
                <label className="text-xs font-bold uppercase tracking-wider text-base-content/50 flex items-center gap-1.5">
                  <FileType className="w-3.5 h-3.5" />
                  {t("search.format", "File Format")}
                </label>
                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    onClick={() => {
                      setSelectedFormat("");
                      updateUrlParams({ format: "" });
                    }}
                    className={`btn btn-xs sm:btn-sm rounded-xl font-semibold transition-all ${
                      !selectedFormat ? "btn-primary" : "btn-ghost bg-base-200/60"
                    }`}
                  >
                    {t("common.all", "All Formats")}
                  </button>
                  {availableFormats.map((fmt) => {
                    const facetItem = formats.find((f) => f.name.toLowerCase() === fmt.toLowerCase());
                    return (
                      <button
                        key={fmt}
                        type="button"
                        onClick={() => {
                          const val = selectedFormat === fmt ? "" : fmt;
                          setSelectedFormat(val);
                          updateUrlParams({ format: val });
                        }}
                        className={`btn btn-xs sm:btn-sm rounded-xl font-semibold uppercase transition-all ${
                          selectedFormat === fmt
                            ? "btn-primary shadow-xs"
                            : "btn-ghost bg-base-200/60 hover:bg-base-200"
                        }`}
                      >
                        {fmt} {facetItem ? `(${facetItem.book_count})` : ""}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Facet Dropdowns Grid */}
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
                {/* Series Select */}
                <div className="space-y-1.5">
                  <label className="text-xs font-bold text-base-content/60 flex items-center gap-1.5">
                    <Layers className="w-3.5 h-3.5 text-primary" />
                    {t("library.facets.series", "Series")}
                  </label>
                  <select
                    className="select select-bordered select-sm w-full rounded-xl bg-base-200/40 text-xs font-medium"
                    value={selectedSeries}
                    onChange={(e) => {
                      setSelectedSeries(e.target.value);
                      updateUrlParams({ series: e.target.value });
                    }}
                  >
                    <option value="">{t("search.all_series", "All Series")}</option>
                    {series.map((item: MetadataCount) => (
                      <option key={item.name} value={item.name}>
                        {item.name} ({item.book_count})
                      </option>
                    ))}
                  </select>
                </div>

                {/* Author Select */}
                <div className="space-y-1.5">
                  <label className="text-xs font-bold text-base-content/60 flex items-center gap-1.5">
                    <User className="w-3.5 h-3.5 text-primary" />
                    {t("library.facets.authors", "Author")}
                  </label>
                  <select
                    className="select select-bordered select-sm w-full rounded-xl bg-base-200/40 text-xs font-medium"
                    value={selectedAuthor}
                    onChange={(e) => {
                      setSelectedAuthor(e.target.value);
                      updateUrlParams({ author: e.target.value });
                    }}
                  >
                    <option value="">{t("search.all_authors", "All Authors")}</option>
                    {authors.map((item: MetadataCount) => (
                      <option key={item.name} value={item.name}>
                        {item.name} ({item.book_count})
                      </option>
                    ))}
                  </select>
                </div>

                {/* Publisher Select */}
                <div className="space-y-1.5">
                  <label className="text-xs font-bold text-base-content/60 flex items-center gap-1.5">
                    <Building2 className="w-3.5 h-3.5 text-primary" />
                    {t("library.facets.publishers", "Publisher")}
                  </label>
                  <select
                    className="select select-bordered select-sm w-full rounded-xl bg-base-200/40 text-xs font-medium"
                    value={selectedPublisher}
                    onChange={(e) => {
                      setSelectedPublisher(e.target.value);
                      updateUrlParams({ publisher: e.target.value });
                    }}
                  >
                    <option value="">{t("search.all_publishers", "All Publishers")}</option>
                    {publishers.map((item: MetadataCount) => (
                      <option key={item.name} value={item.name}>
                        {item.name} ({item.book_count})
                      </option>
                    ))}
                  </select>
                </div>

                {/* Language Select */}
                <div className="space-y-1.5">
                  <label className="text-xs font-bold text-base-content/60 flex items-center gap-1.5">
                    <Globe className="w-3.5 h-3.5 text-primary" />
                    {t("library.facets.languages", "Language")}
                  </label>
                  <select
                    className="select select-bordered select-sm w-full rounded-xl bg-base-200/40 text-xs font-medium"
                    value={selectedLanguage}
                    onChange={(e) => {
                      setSelectedLanguage(e.target.value);
                      updateUrlParams({ language: e.target.value });
                    }}
                  >
                    <option value="">{t("search.all_languages", "All Languages")}</option>
                    {languages.map((item: MetadataCount) => (
                      <option key={item.name} value={item.name}>
                        {item.name} ({item.book_count})
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              {/* Active Filter Badges */}
              {hasActiveFilters && (
                <div className="flex flex-wrap items-center gap-2 pt-2 border-t border-base-200">
                  <span className="text-xs font-bold text-base-content/50 uppercase mr-1">
                    {t("search.applied", "Applied:")}
                  </span>
                  {queryInput && (
                    <span className="badge badge-primary badge-sm gap-1.5 py-2 px-3 rounded-lg font-medium">
                      {t("search.query_label", "Query")}: "{queryInput}"
                      <X
                        className="w-3 h-3 cursor-pointer hover:opacity-80"
                        onClick={() => {
                          setQueryInput("");
                          updateUrlParams({ q: "" });
                        }}
                      />
                    </span>
                  )}
                  {selectedFormat && (
                    <span className="badge badge-secondary badge-sm gap-1.5 py-2 px-3 rounded-lg uppercase font-bold">
                      {t("search.format", "Format")}: {selectedFormat}
                      <X
                        className="w-3 h-3 cursor-pointer hover:opacity-80"
                        onClick={() => {
                          setSelectedFormat("");
                          updateUrlParams({ format: "" });
                        }}
                      />
                    </span>
                  )}
                  {selectedSeries && (
                    <span className="badge badge-accent badge-sm gap-1.5 py-2 px-3 rounded-lg font-medium">
                      {t("library.facets.series", "Series")}: {selectedSeries}
                      <X
                        className="w-3 h-3 cursor-pointer hover:opacity-80"
                        onClick={() => {
                          setSelectedSeries("");
                          updateUrlParams({ series: "" });
                        }}
                      />
                    </span>
                  )}
                  {selectedAuthor && (
                    <span className="badge badge-info badge-sm gap-1.5 py-2 px-3 rounded-lg font-medium">
                      {t("library.facets.authors", "Author")}: {selectedAuthor}
                      <X
                        className="w-3 h-3 cursor-pointer hover:opacity-80"
                        onClick={() => {
                          setSelectedAuthor("");
                          updateUrlParams({ author: "" });
                        }}
                      />
                    </span>
                  )}
                  {selectedPublisher && (
                    <span className="badge badge-neutral badge-sm gap-1.5 py-2 px-3 rounded-lg font-medium">
                      {t("library.facets.publishers", "Publisher")}: {selectedPublisher}
                      <X
                        className="w-3 h-3 cursor-pointer hover:opacity-80"
                        onClick={() => {
                          setSelectedPublisher("");
                          updateUrlParams({ publisher: "" });
                        }}
                      />
                    </span>
                  )}
                  {selectedLanguage && (
                    <span className="badge badge-ghost badge-sm gap-1.5 py-2 px-3 rounded-lg font-medium">
                      {t("library.facets.languages", "Language")}: {selectedLanguage}
                      <X
                        className="w-3 h-3 cursor-pointer hover:opacity-80"
                        onClick={() => {
                          setSelectedLanguage("");
                          updateUrlParams({ language: "" });
                        }}
                      />
                    </span>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Results Grid Section */}
          <div className="space-y-4">
            <div className="flex items-center justify-between px-1">
              <div className="text-sm font-bold text-base-content/70">
                {isLoading
                  ? t("common.loading", "Searching books...")
                  : books.length > 0
                  ? t("search.found_books_count", "Found {{count}} books", { count: books.length })
                  : t("search.no_books_found", "No books found")}
              </div>
            </div>

            {isLoading ? (
              <div className="py-16 text-center space-y-3">
                <span className="loading loading-spinner loading-lg text-primary"></span>
                <p className="text-xs text-base-content/60 font-medium">
                  {t("common.loading", "Loading books...")}
                </p>
              </div>
            ) : books.length > 0 ? (
              <>
                <BookGrid books={books} onBookClick={(book) => navigate(`/books/${book.id}`)} compact />
                {hasNextPage && (
                  <div className="text-center pt-6">
                    <button
                      type="button"
                      onClick={() => fetchNextPage()}
                      disabled={isFetchingNextPage}
                      className="btn btn-outline btn-primary rounded-xl px-8 font-bold"
                    >
                      {isFetchingNextPage ? (
                        <>
                          <span className="loading loading-spinner loading-xs mr-2"></span>
                          {t("common.loading", "Loading...")}
                        </>
                      ) : (
                        t("common.load_more", "Load More Books")
                      )}
                    </button>
                  </div>
                )}
              </>
            ) : (
              <div className="bg-base-100 border border-base-200 rounded-3xl p-12 text-center space-y-4">
                <div className="w-16 h-16 rounded-full bg-primary/10 text-primary grid place-items-center mx-auto">
                  <BookOpen className="w-8 h-8" />
                </div>
                <div className="max-w-md mx-auto space-y-1">
                  <h3 className="text-lg font-bold text-base-content">
                    {t("search.no_results_title", "No matching books found")}
                  </h3>
                  <p className="text-xs text-base-content/60">
                    {t("search.no_results_desc", "Try adjusting your search query, clearing filter options, or searching by a different term.")}
                  </p>
                </div>
                {hasActiveFilters && (
                  <button
                    type="button"
                    onClick={handleResetFilters}
                    className="btn btn-primary btn-sm rounded-xl font-bold gap-2"
                  >
                    <RotateCcw className="w-4 h-4" />
                    {t("search.reset_filters", "Clear All Filters")}
                  </button>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Sidebar Drawer */}
      <LibrarySidebar
        t={t}
        user={user}
        primaryNavItems={primaryNavItems}
        facetSections={[]}
        secondaryNavItems={[]}
        collections={[]}
        activeNav={activeNav}
        activeFacet={activeFacet}
        activeCollection={activeCollection}
        onNavClick={(nav) => {
          setActiveNav(nav);
          navigate("/");
        }}
        onCollectionClick={(coll) => {
          setActiveCollection(coll);
          navigate("/");
        }}
        onNewCollection={() => {}}
      />
    </div>
  );
};
