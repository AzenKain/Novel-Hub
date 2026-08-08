import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { metadataService } from "@/services";
import type { MetadataCount } from "@/types";

const FACET_STALE_TIME = 60_000;
const FACET_LIMIT = 100;

/**
 * Fetches all metadata facets needed by the Advanced Search page in parallel.
 * Returns the raw query results so the page can check loading/error state per facet.
 */
export function useAdvancedSearchFacets() {
  const formatsQuery = useQuery({
    queryKey: ["metadata", "formats", { limit: FACET_LIMIT }],
    queryFn: () => metadataService.listFormats({ limit: FACET_LIMIT }),
    staleTime: FACET_STALE_TIME,
  });

  const seriesQuery = useQuery({
    queryKey: ["metadata", "series", { limit: FACET_LIMIT }],
    queryFn: () => metadataService.listSeries({ limit: FACET_LIMIT }),
    staleTime: FACET_STALE_TIME,
  });

  const authorsQuery = useQuery({
    queryKey: ["metadata", "authors", { limit: FACET_LIMIT }],
    queryFn: () => metadataService.listAuthors({ limit: FACET_LIMIT }),
    staleTime: FACET_STALE_TIME,
  });

  const publishersQuery = useQuery({
    queryKey: ["metadata", "publishers", { limit: FACET_LIMIT }],
    queryFn: () => metadataService.listPublishers({ limit: FACET_LIMIT }),
    staleTime: FACET_STALE_TIME,
  });

  const languagesQuery = useQuery({
    queryKey: ["metadata", "languages", { limit: FACET_LIMIT }],
    queryFn: () => metadataService.listLanguages({ limit: FACET_LIMIT }),
    staleTime: FACET_STALE_TIME,
  });

  const formats = useMemo<MetadataCount[]>(
    () => formatsQuery.data?.data ?? [],
    [formatsQuery.data],
  );

  const series = useMemo<MetadataCount[]>(
    () => seriesQuery.data?.data ?? [],
    [seriesQuery.data],
  );

  const authors = useMemo<MetadataCount[]>(
    () => authorsQuery.data?.data ?? [],
    [authorsQuery.data],
  );

  const publishers = useMemo<MetadataCount[]>(
    () => publishersQuery.data?.data ?? [],
    [publishersQuery.data],
  );

  const languages = useMemo<MetadataCount[]>(
    () => languagesQuery.data?.data ?? [],
    [languagesQuery.data],
  );

  const isLoading =
    formatsQuery.isLoading ||
    seriesQuery.isLoading ||
    authorsQuery.isLoading ||
    publishersQuery.isLoading ||
    languagesQuery.isLoading;

  return {
    formats,
    series,
    authors,
    publishers,
    languages,
    isLoading,
    formatsQuery,
    seriesQuery,
    authorsQuery,
    publishersQuery,
    languagesQuery,
  };
}
