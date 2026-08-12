import { api } from "@/config/api";
import { CommonResponse } from "@/types";
import {
  RequestMagicCodeResponse,
  PollMagicCodeResponse,
} from "@/types/magicCode";
import { AuthResponse } from "@/types/auth";

export const magicCodeService = {
  requestCode: async (deviceInfo?: string) => {
    const res = await api.post<CommonResponse<RequestMagicCodeResponse>>(
      "/auth/magic-code/request",
      { device_info: deviceInfo }
    );
    return res.data;
  },

  pollCode: async (pollToken: string) => {
    const res = await api.post<CommonResponse<PollMagicCodeResponse>>(
      "/auth/magic-code/poll",
      { poll_token: pollToken }
    );
    return res.data;
  },

  activateCode: async (code: string) => {
    const res = await api.post<CommonResponse<AuthResponse>>(
      "/auth/magic-code/activate",
      { code }
    );
    return res.data;
  },
};
