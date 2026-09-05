import { deviceService } from "@/services";
import type { CreateUserDeviceInput } from "@/types";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export function useDevicesQuery(
  enabled = true,
  params?: { cursor?: string; limit?: number },
) {
  return useQuery({
    queryKey: ["user", "devices", params?.cursor, params?.limit],
    enabled,
    queryFn: async () => {
      const res = await deviceService.listDevices(params);
      if (!res.status) throw new Error(res.message || "Failed to load devices");
      return res.data || [];
    },
  });
}

export function useCreateDeviceMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateUserDeviceInput) =>
      deviceService.createDevice(input),
    onSuccess: () =>
      void client.invalidateQueries({ queryKey: ["user", "devices"] }),
  });
}

export function useDeleteDeviceMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deviceService.deleteDevice(id),
    onSuccess: () =>
      void client.invalidateQueries({ queryKey: ["user", "devices"] }),
  });
}

export function usePushBookMutation() {
  return useMutation({
    mutationFn: ({ bookId, deviceId }: { bookId: string; deviceId: string }) =>
      deviceService.pushBook(bookId, deviceId),
  });
}
