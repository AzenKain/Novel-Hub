import { api, toQuery } from "@/config/api";
import type {
  AdminReview,
  CommonResponse,
  CreateRoleRequest,
  CreateUserRequest,
  PaginatedResponse,
  Permission,
  PublicSettings,
  Role,
  SearchUserParams,
  UpdateProfileRequest,
  UpdateRolePermissionsRequest,
  UpdateRoleRequest,
  UpdateSettingsRequest,
  User
} from "@/types";
import axios from "axios";

export const adminService = {
  async searchUsers(params: SearchUserParams): Promise<PaginatedResponse<User>> {
    try {
      const res = await api.get(`/users${toQuery(params)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as PaginatedResponse<User>;
      }
      throw error;
    }
  },

  async createUser(data: CreateUserRequest): Promise<CommonResponse<User>> {
    try {
      const res = await api.post("/users", data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<User>;
      }
      throw error;
    }
  },

  async updateUser(id: number, data: UpdateProfileRequest): Promise<CommonResponse<User>> {
    try {
      const res = await api.put(`/users/${id}`, data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<User>;
      }
      throw error;
    }
  },

  async resetPassword(id: number, newPassword: string): Promise<CommonResponse<unknown>> {
    try {
      const res = await api.patch(`/users/${id}/password`, { new_password: newPassword });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<unknown>;
      }
      throw error;
    }
  },

  async changeRoles(id: number, roleIDs: number[]): Promise<CommonResponse<User>> {
    try {
      const res = await api.patch(`/users/${id}/role`, { role_ids: roleIDs });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<User>;
      }
      throw error;
    }
  },

  async deleteUser(id: number): Promise<CommonResponse<unknown>> {
    try {
      const res = await api.delete(`/users/${id}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<unknown>;
      }
      throw error;
    }
  },

  async restoreUser(id: number): Promise<CommonResponse<User>> {
    try {
      const res = await api.patch(`/users/${id}/restore`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<User>;
      }
      throw error;
    }
  },

  // ─── Role CRUD ──────────────────────────────────────────────

  async getRoles(): Promise<CommonResponse<Role[]>> {
    try {
      const res = await api.get("/roles");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Role[]>;
      }
      throw error;
    }
  },

  async getRoleById(id: number): Promise<CommonResponse<Role>> {
    try {
      const res = await api.get(`/roles/${id}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Role>;
      }
      throw error;
    }
  },

  async createRole(data: CreateRoleRequest): Promise<CommonResponse<Role>> {
    try {
      const res = await api.post("/roles", data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Role>;
      }
      throw error;
    }
  },

  async updateRole(id: number, data: UpdateRoleRequest): Promise<CommonResponse<Role>> {
    try {
      const res = await api.put(`/roles/${id}`, data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Role>;
      }
      throw error;
    }
  },

  async updateRolePermissions(id: number, data: UpdateRolePermissionsRequest): Promise<CommonResponse<Role>> {
    try {
      const res = await api.put(`/roles/${id}/permissions`, data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Role>;
      }
      throw error;
    }
  },

  async deleteRole(id: number): Promise<CommonResponse<unknown>> {
    try {
      const res = await api.delete(`/roles/${id}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<unknown>;
      }
      throw error;
    }
  },

  async getPermissions(): Promise<CommonResponse<Permission[]>> {
    try {
      const res = await api.get("/roles/permissions");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Permission[]>;
      }
      throw error;
    }
  },

  // ─── Settings ───────────────────────────────────────────────

  async getAdminSettings(): Promise<CommonResponse<PublicSettings>> {
    try {
      const res = await api.get("/settings");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<PublicSettings>;
      }
      throw error;
    }
  },

  async updateSettings(data: UpdateSettingsRequest): Promise<CommonResponse<PublicSettings>> {
    try {
      const res = await api.put("/settings", data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<PublicSettings>;
      }
      throw error;
    }
  },

  // ─── Reviews ────────────────────────────────────────────────

  async listAllReviews(limit = 50, offset = 0): Promise<CommonResponse<AdminReview[]>> {
    try {
      const res = await api.get(`/admin/reviews?limit=${limit}&offset=${offset}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<AdminReview[]>;
      }
      throw error;
    }
  },

  async deleteReview(bookId: string, userId: number): Promise<CommonResponse<unknown>> {
    try {
      const res = await api.delete(`/admin/reviews/${encodeURIComponent(bookId)}/${userId}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<unknown>;
      }
      throw error;
    }
  },

  async uploadAdminLogo(data: FormData): Promise<CommonResponse<{ url: string }>> {
    try {
      const res = await api.post("/settings/logo", data, {
        headers: {
          "Content-Type": "multipart/form-data"
        }
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<{ url: string }>;
      throw error;
    }
  },
};
