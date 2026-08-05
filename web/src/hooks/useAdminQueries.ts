import { adminService, webhookService } from "@/services";
import type {
  AdminReview,
  AdminSettings,
  CalibreImportResult,
  CreateRoleRequest,
  CreateUserRequest,
  CreateWebhookInput,
  Permission,
  Role,
  SearchUserParams,
  SendUserEmailRequest,
  SmtpTestRequest,
  UpdateRoleRequest,
  User,
  Webhook,
} from "@/types";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

// Settings
export function useAdminSettingsQuery() {
  return useQuery<AdminSettings>({
    queryKey: ["admin", "settings"],
    queryFn: async () => {
      const res = await adminService.getAdminSettings();
      if (!res.status) throw new Error(res.message || "Failed to fetch admin settings");
      return res.data!;
    },
  });
}

export function useUpdateAdminSettingsMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: Record<string, unknown>) => {
      const res = await adminService.updateSettings(data);
      if (!res.status) throw new Error(res.message || "Failed to update settings");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "settings"] });
      void queryClient.invalidateQueries({ queryKey: ["public", "settings"] });
    },
  });
}

export function useTestSmtpMutation() {
  return useMutation({
    mutationFn: async (data: SmtpTestRequest) => {
      const res = await adminService.testSmtp(data);
      if (!res.status) throw new Error(res.message || "SMTP connection failed");
      return res;
    },
  });
}

export function useUploadAdminLogoMutation() {
  return useMutation({
    mutationFn: async (fd: FormData) => {
      const res = await adminService.uploadAdminLogo(fd);
      if (!res.status || !res.data) throw new Error(res.message || "Failed to upload logo/favicon");
      return res.data.url;
    },
  });
}

// Calibre
export function useCalibreImportMutation() {
  const queryClient = useQueryClient();
  return useMutation<CalibreImportResult, Error, { path: string; library_id?: string }>({
    mutationFn: async ({ path, library_id }) => {
      const res = await adminService.importCalibre(path, library_id);
      if (!res.status) throw new Error(res.message || "Failed to import Calibre library");
      return res.data ?? { imported_count: 0 };
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
    },
  });
}

// Users
export function useUsersQuery(params: SearchUserParams) {
  return useQuery<{ users: User[]; total: number }>({
    queryKey: ["admin", "users", params],
    queryFn: async () => {
      const res = await adminService.searchUsers(params);
      if (!res.status) throw new Error(res.message || "Failed to fetch users");
      return { users: res.data || [], total: res.pagination?.total_records || 0 };
    },
  });
}

export function useCreateUserMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateUserRequest) => {
      const res = await adminService.createUser(input);
      if (!res.status) throw new Error(res.message || "Failed to create user");
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
    },
  });
}

export function useUpdateUserMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: { full_name: string; avatar_url?: string } }) => {
      const res = await adminService.updateUser(id, data);
      if (!res.status) throw new Error(res.message || "Failed to update user");
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
    },
  });
}

export function useResetUserPasswordMutation() {
  return useMutation({
    mutationFn: async ({ id, password }: { id: string; password: string }) => {
      const res = await adminService.resetPassword(id, password);
      if (!res.status) throw new Error(res.message || "Failed to reset password");
      return res;
    },
  });
}

export function useSendUserEmailMutation() {
  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: SendUserEmailRequest }) => {
      const res = await adminService.sendUserEmail(id, data);
      if (!res.status) throw new Error(res.message || "Failed to send the email");
      return res;
    },
  });
}

export function useChangeUserRolesMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, roleIDs }: { id: string; roleIDs: string[] }) => {
      const res = await adminService.changeRoles(id, roleIDs);
      if (!res.status) throw new Error(res.message || "Failed to change roles");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
    },
  });
}

export function useDeleteUserMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await adminService.deleteUser(id);
      if (!res.status) throw new Error(res.message || "Failed to delete user");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
    },
  });
}

// Roles
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

export function useCreateRoleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateRoleRequest) => {
      const res = await adminService.createRole(input);
      if (!res.status) throw new Error(res.message || "Failed to create role");
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
    },
  });
}

export function useUpdateRoleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: UpdateRoleRequest }) => {
      const res = await adminService.updateRole(id, data);
      if (!res.status) throw new Error(res.message || "Failed to update role");
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
    },
  });
}

export function useDeleteRoleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await adminService.deleteRole(id);
      if (!res.status) throw new Error(res.message || "Failed to delete role");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
    },
  });
}

export function useReorderRolesMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (roleIDs: string[]) => {
      const res = await adminService.reorderRoles(roleIDs);
      if (!res.status) throw new Error(res.message || "Failed to reorder roles");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
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

export function useAssignRolePermissionsMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ roleID, assignments }: { roleID: string; assignments: any[] }) => {
      const res = await adminService.updateRolePermissions(roleID, { permissions: assignments });
      if (!res.status) throw new Error(res.message || "Failed to update role permissions");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
    },
  });
}

// Reviews
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

export function useDeleteReviewMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ book_id, user_id }: { book_id: string; user_id: string }) => {
      const res = await adminService.deleteReview(book_id, user_id);
      if (!res.status) throw new Error(res.message || "Failed to delete review");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "reviews"] });
    },
  });
}

// Webhooks
export function useWebhooksQuery() {
  return useQuery<Webhook[]>({
    queryKey: ["admin", "webhooks"],
    queryFn: async () => {
      const res = await webhookService.listWebhooks();
      if (!res.status) throw new Error(res.message || "Failed to fetch webhooks");
      return res.data || [];
    },
    retry: 1,
  });
}

export function useCreateWebhookMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateWebhookInput) => {
      const res = await webhookService.createWebhook(input);
      if (!res.status) throw new Error(res.message || "Failed to create webhook");
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "webhooks"] });
    },
  });
}

export function useUpdateWebhookMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id: string; input: CreateWebhookInput }) => {
      const res = await webhookService.updateWebhook(id, input);
      if (!res.status) throw new Error(res.message || "Failed to update webhook");
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "webhooks"] });
    },
  });
}

export function useDeleteWebhookMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await webhookService.deleteWebhook(id);
      if (!res.status) throw new Error(res.message || "Failed to delete webhook");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "webhooks"] });
    },
  });
}

export function useTestWebhookMutation() {
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await webhookService.testPingWebhook(id);
      if (!res.status) throw new Error(res.message || "Webhook test failed");
      return res;
    },
  });
}
