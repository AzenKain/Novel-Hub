import { metadataService } from "@/services";
import type { MetadataCount, MetadataFacetParams } from "@/types";
import { useInfiniteQuery } from "@tanstack/react-query";

export type MetadataFacet = "authors" | "series" | "tags" | "publishers" | "languages" | "formats";

const fetchers: Record<MetadataFacet, (params: MetadataFacetParams) => ReturnType<typeof metadataService.listAuthors>> = {
  authors: (params) => metadataService.listAuthors(params),
  series: (params) => metadataService.listSeries(params),
  tags: (params) => metadataService.listTags(params),
  publishers: (params) => metadataService.listPublishers(params),
  languages: (params) => metadataService.listLanguages(params),
  formats: (params) => metadataService.listFormats(params),
};

export function useMetadataFacetQuery(
  facet: MetadataFacet,
  filters: { search?: string; alpha?: string } = {},
) {
  const search = filters.search?.trim() || undefined;
  const alpha = filters.alpha && filters.alpha !== "All" ? filters.alpha : undefined;

  const query = useInfiniteQuery({
    queryKey: ["metadata", facet, { search, alpha }],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const fetch = fetchers[facet];
      if (!fetch) throw new Error("Invalid facet type");
      const res = await fetch({ cursor: pageParam, limit: 50, search, alpha });
      if (!res.status) throw new Error(res.message || `Failed to fetch metadata ${facet}`);
      return res;
    },
    getNextPageParam: (lastPage) => lastPage.pagination?.next_cursor || undefined,
  });

  const items: MetadataCount[] = query.data?.pages.flatMap((page) => page.data || []) ?? [];
  return { ...query, items };
}
