import type { RoleSimple } from "./admin";

export type AuthResponse = {
  access_token: string;
  refresh_token: string;
  totp_required?: boolean;
};

export type OTPPurpose = "email_verify" | "password_reset";

export type TOTPStatus = {
  enabled: boolean;
  pending_enrollment?: boolean;
  confirmed_at?: string;
  recovery_codes_remaining: number;
};

export type TOTPEnrollment = {
  secret: string;
  provisioning_uri: string;
};

export type TOTPRecoveryCodes = {
  codes: string[];
};

export type OTPRequestResponse = {
  expires_in_seconds: number;
  cooldown_seconds: number;
};

export type OTPVerifyResponse = {
  otp_ticket: string;
  expires_in_seconds: number;
};

export type RegisterRequest = {
  email: string;
  password: string;
  full_name?: string;
  otp_ticket?: string;
};

export type ResetPasswordWithOTPRequest = {
  email: string;
  otp_ticket: string;
  new_password: string;
};

export type User = {
  id: string;
  email: string;
  full_name: string;
  avatar_url: string;
  auth_provider: string;
  token_version: number;
  is_deleted: boolean;
  is_owner?: boolean;
  created_at?: string;
  updated_at?: string;
  roles: RoleSimple[];
};

export type SearchUserParams = {
  page?: number;
  limit?: number;
  search?: string;
  is_deleted?: boolean;
  role_ids?: string[];
  sort?: "id" | "created_at" | "updated_at" | "email" | "is_deleted" | "auth_provider";
  order?: "asc" | "desc";
};

export type CreateUserRequest = {
  email: string;
  password: string;
  full_name: string;
  avatar_url?: string;
  role_ids?: string[];
};

export type UpdateProfileRequest = {
  full_name?: string;
  avatar_url?: string;
};

export type ChangePasswordRequest = {
  old_password: string;
  new_password: string;
};
