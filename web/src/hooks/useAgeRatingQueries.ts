import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ageRatingService } from "@/services/ageRatingService";

export const useKidsModeInfo = () => {
  return useQuery({
    queryKey: ["kidsModeInfo"],
    queryFn: ageRatingService.getKidsModeInfo,
    retry: false,
  });
};

export const useSetKidsModePinMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ageRatingService.setKidsModePin,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["kidsModeInfo"] });
    },
  });
};

export const useToggleKidsModeMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ enable, pin }: { enable: boolean; pin?: string }) =>
      ageRatingService.toggleKidsMode(enable, pin),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["kidsModeInfo"] });
    },
  });
};

export const useContentWarnings = () => {
  return useQuery({
    queryKey: ["contentWarnings"],
    queryFn: ageRatingService.getContentWarnings,
  });
};

export const useBookContentWarnings = (bookId: string) => {
  return useQuery({
    queryKey: ["bookContentWarnings", bookId],
    queryFn: () => ageRatingService.getBookContentWarnings(bookId),
    enabled: !!bookId,
  });
};

export const useUpdateBookAgeRatingMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      bookId,
      payload,
    }: {
      bookId: string;
      payload: { age_rating: string; content_warning_ids: string[] };
    }) => ageRatingService.updateBookAgeRating(bookId, payload),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["bookContentWarnings", variables.bookId],
      });
      queryClient.invalidateQueries({
        queryKey: ["book", variables.bookId],
      });
    },
  });
};
