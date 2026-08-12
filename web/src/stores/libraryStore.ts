import type {
  Book as BookType,
  Collection,
  DuplicateFileResult,
  LibraryStats,
  MetadataCount,
  ReadingHistory,
} from "@/types";
import { create } from "zustand";

interface LibraryState {
  books: BookType[];
  loading: boolean;
  search: string;
  selectedBook: BookType | null;
  activeNav: string;
  activeCollection: string;
  activeChip: string;
  activeFacet: { type: string; id: string; name: string } | null;
  stats: LibraryStats;
  metadataFacets: {
    authors: MetadataCount[];
    series: MetadataCount[];
    tags: MetadataCount[];
    publishers: MetadataCount[];
    languages: MetadataCount[];
    formats: MetadataCount[];
  };
  collections: Collection[];
  hasMoreCollections: boolean;
  collectionsCursor: string | null;
  recentHistory: ReadingHistory[];
  showNewCollectionModal: boolean;
  newCollectionName: string;
  collectionError: string;
  randomSeed: number;
  duplicates: DuplicateFileResult[];
  duplicatesLoading: boolean;
  metadataQuery: string;
  metadataAlpha: string;
  metadataSort: "name-asc" | "name-desc" | "count-desc";
  topBooks: BookType[];
  activeSmartFilterId: string | null;

  setBooks: (books: BookType[]) => void;
  setLoading: (loading: boolean) => void;
  setSearch: (search: string) => void;
  setSelectedBook: (book: BookType | null) => void;
  setActiveNav: (nav: string) => void;
  setActiveCollection: (collection: string) => void;
  setActiveChip: (chip: string) => void;
  setActiveFacet: (
    facet: { type: string; id: string; name: string } | null,
  ) => void;
  setStats: (stats: LibraryStats) => void;
  setMetadataFacets: (facets: Partial<LibraryState["metadataFacets"]>) => void;
  setCollections: (collections: Collection[]) => void;
  appendCollections: (collections: Collection[], cursor: string | null, hasMore: boolean) => void;
  addCollection: (collection: Collection) => void;
  updateCollection: (id: string, name: string) => void;
  deleteCollection: (id: string) => void;
  setRecentHistory: (history: ReadingHistory[]) => void;
  setShowNewCollectionModal: (show: boolean) => void;
  setNewCollectionName: (name: string) => void;
  setCollectionError: (error: string) => void;
  setRandomSeed: (seed: number | ((prev: number) => number)) => void;
  setDuplicates: (duplicates: DuplicateFileResult[]) => void;
  setDuplicatesLoading: (loading: boolean) => void;
  setMetadataQuery: (query: string) => void;
  setMetadataAlpha: (alpha: string) => void;
  setMetadataSort: (sort: "name-asc" | "name-desc" | "count-desc") => void;
  setTopBooks: (books: BookType[]) => void;
  setActiveSmartFilterId: (id: string | null) => void;
}

export const useLibraryStore = create<LibraryState>((set) => ({
  books: [],
  loading: true,
  search: "",
  selectedBook: null,
  activeNav: "books",
  activeCollection: "",
  activeChip: "All",
  activeFacet: null,
  stats: { total_books: 0, series_tracked: 0, need_review: 0 },
  metadataFacets: {
    authors: [],
    series: [],
    tags: [],
    publishers: [],
    languages: [],
    formats: [],
  },
  collections: [],
  hasMoreCollections: true,
  collectionsCursor: null,
  recentHistory: [],
  showNewCollectionModal: false,
  newCollectionName: "",
  collectionError: "",
  randomSeed: Date.now(),
  duplicates: [],
  duplicatesLoading: true,
  metadataQuery: "",
  metadataAlpha: "All",
  metadataSort: "name-asc",
  topBooks: [],
  activeSmartFilterId: null,

  setBooks: (books) => set({ books }),
  setCollections: (collections) => set({ collections }),
  appendCollections: (collections, cursor, hasMore) =>
    set((state) => ({
      collections: [...state.collections, ...collections],
      collectionsCursor: cursor,
      hasMoreCollections: hasMore,
    })),
  addCollection: (collection) =>
    set((state) => ({
      collections: [collection, ...state.collections],
    })),
  updateCollection: (id, name) =>
    set((state) => ({
      collections: state.collections.map((c) =>
        c.id === id ? { ...c, name } : c,
      ),
    })),
  deleteCollection: (id) =>
    set((state) => ({
      collections: state.collections.filter((c) => c.id !== id),
    })),
  setLoading: (loading) => set({ loading }),
  setSearch: (search) => set({ search }),
  setSelectedBook: (selectedBook) => set({ selectedBook }),
  setActiveNav: (activeNav) => set({ activeNav }),
  setActiveCollection: (activeCollection) => set({ activeCollection }),
  setActiveChip: (activeChip) => set({ activeChip }),
  setActiveFacet: (activeFacet) => set({ activeFacet }),
  setStats: (stats) => set({ stats }),
  setMetadataFacets: (facets) =>
    set((state) => ({
      metadataFacets: { ...state.metadataFacets, ...facets },
    })),
  setRecentHistory: (recentHistory) => set({ recentHistory }),
  setShowNewCollectionModal: (showNewCollectionModal) =>
    set({ showNewCollectionModal }),
  setNewCollectionName: (newCollectionName) => set({ newCollectionName }),
  setCollectionError: (collectionError) => set({ collectionError }),
  setRandomSeed: (seed) =>
    set((state) => ({
      randomSeed: typeof seed === "function" ? seed(state.randomSeed) : seed,
    })),
  setDuplicates: (duplicates) => set({ duplicates }),
  setDuplicatesLoading: (duplicatesLoading) => set({ duplicatesLoading }),
  setMetadataQuery: (metadataQuery) => set({ metadataQuery }),
  setMetadataAlpha: (metadataAlpha) => set({ metadataAlpha }),
  setMetadataSort: (metadataSort) => set({ metadataSort }),
  setTopBooks: (topBooks) => set({ topBooks }),
  setActiveSmartFilterId: (activeSmartFilterId) => set({ activeSmartFilterId }),
}));
