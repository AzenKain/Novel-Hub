import { metadataService } from "@/services";
import type { MetadataCount } from "@/types";
import { useQuery } from "@tanstack/react-query";

export function useMetadataFacetQuery(facet: "authors" | "series" | "tags" | "publishers" | "languages" | "formats") {
  return useQuery<MetadataCount[]>({
    queryKey: ["metadata", facet],
    queryFn: async () => {
      let res;
      switch (facet) {
        case "authors":
          res = await metadataService.listAuthors();
          break;
        case "series":
          res = await metadataService.listSeries();
          break;
        case "tags":
          res = await metadataService.listTags();
          break;
        case "publishers":
          res = await metadataService.listPublishers();
          break;
        case "languages":
          res = await metadataService.listLanguages();
          break;
        case "formats":
          res = await metadataService.listFormats();
          break;
        default:
          throw new Error("Invalid facet type");
      }
      if (!res.status) throw new Error(res.message || `Failed to fetch metadata ${facet}`);
      return res.data || [];
    },
  });
}
