import { koboService } from "@/services";
import type { KoboSetup } from "@/types";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

const KOBO_SETUP_KEY = ["kobo", "setup"];

export function useKoboSetupQuery(enabled = true) {
  return useQuery<KoboSetup | null>({
    queryKey: KOBO_SETUP_KEY,
    queryFn: async () => {
      const res = await koboService.getSetup();
      if (!res.status) throw new Error(res.message || "Failed to load Kobo endpoint");
      return res.data ?? null;
    },
    enabled,
    // The token is stable until regenerated, so there is nothing to poll for.
    staleTime: Infinity,
    retry: false,
  });
}

export function useRegenerateKoboSetupMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await koboService.regenerate();
      if (!res.status || !res.data) throw new Error(res.message || "Failed to regenerate Kobo endpoint");
      return res.data;
    },
    onSuccess: (data) => {
      // Write through rather than invalidate: the response already carries the new URL, and a
      // refetch would show the old one until it lands.
      queryClient.setQueryData(KOBO_SETUP_KEY, data);
    },
  });
}

export function useRevokeKoboSetupMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await koboService.revoke();
      if (!res.status) throw new Error(res.message || "Failed to revoke Kobo endpoint");
      return res;
    },
    onSuccess: () => {
      queryClient.setQueryData(KOBO_SETUP_KEY, null);
    },
  });
}
