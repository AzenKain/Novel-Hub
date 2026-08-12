import { AuthResponse } from "./auth";

export interface MagicCodeResponse {
  id: string;
  code: string;
  poll_token: string;
  device_info: string;
  user_id?: string;
  jwt_token?: string;
  status: string;
  expires_at: string;
  created_at: string;
}

export interface RequestMagicCodeResponse {
  code: string;
  poll_token: string;
  activate_url: string;
  expires_in_seconds: number;
}

export interface PollMagicCodeResponse {
  status: string;
  auth?: AuthResponse;
}
