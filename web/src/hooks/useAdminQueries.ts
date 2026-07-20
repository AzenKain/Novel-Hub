import { adminService } from "@/services";
import type { AdminReview, Permission, PublicSettings, Role } from "@/types";
import { useQuery } from "@tanstack/react-query";

export function useAdminSettingsQuery() {
  return useQuery<PublicSettings>({
    queryKey: ["admin", "settings"],
    queryFn: async () => {
      const res = await adminService.getAdminSettings();
      if (!res.status) throw new Error(res.message || "Failed to fetch admin settings");
      return res.data!;
    },
  });
}

export function useRolesQuery() {
  return useQuery<Role[]>({
    queryKey: ["admin", "roles"],
    queryFn: async () => {
      const res = await adminService.getRoles();
      if (!res.status) throw new Error(res.message || "Failed to fetch roles");
      return res.data || [];
    },
  });
}

export function usePermissionsQuery() {
  return useQuery<Permission[]>({
    queryKey: ["admin", "permissions"],
    queryFn: async () => {
      const res = await adminService.getPermissions();
      if (!res.status) throw new Error(res.message || "Failed to fetch permissions");
      return res.data || [];
    },
  });
}

export function useReviewsQuery(page: number, limit = 20) {
  return useQuery<AdminReview[]>({
    queryKey: ["admin", "reviews", page, limit],
    queryFn: async () => {
      const res = await adminService.listAllReviews(limit, page * limit);
      if (!res.status) throw new Error(res.message || "Failed to fetch reviews");
      return res.data || [];
    },
  });
}
