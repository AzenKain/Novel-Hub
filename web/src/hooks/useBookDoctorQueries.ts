import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { bookDoctorService } from "@/services";
import type { RepairOptions } from "@/types";

export const useBookValidationQuery = (
  bookId: string,
  fileId?: string,
  enabled = true,
) => {
  return useQuery({
    queryKey: ["book-doctor-validate", bookId, fileId],
    queryFn: () => bookDoctorService.validateBook(bookId, fileId),
    enabled: !!bookId && enabled,
    staleTime: 0,
  });
};

export const useRepairBookMutation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      bookId,
      options,
      fileId,
    }: {
      bookId: string;
      options?: RepairOptions;
      fileId?: string;
    }) => bookDoctorService.repairBook(bookId, options, fileId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["book-doctor-validate", variables.bookId],
      });
      queryClient.invalidateQueries({ queryKey: ["book", variables.bookId] });
      queryClient.invalidateQueries({
        queryKey: ["reader-bootstrap", variables.bookId],
      });
      queryClient.invalidateQueries({
        queryKey: ["book-files", variables.bookId],
      });
    },
  });
};
