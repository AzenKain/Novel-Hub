import { api } from "@/config/api";
import type {
  CommonResponse,
  CustomFont,
  CustomTheme,
  Soundscape,
  UploadFontPayload,
  UploadSoundscapePayload,
  CreateCustomThemePayload,
  UpdateCustomThemePayload,
  CustomizationListParams,
} from "@/types";

export const customizationService = {
  async getSoundscapes(
    params?: CustomizationListParams,
  ): Promise<Soundscape[]> {
    const res = await api.get<{
      status: boolean;
      data: Soundscape[];
      next_cursor?: string | null;
    }>("/soundscapes", { params });
    return Array.isArray(res.data.data) ? res.data.data : [];
  },

  async uploadSoundscape(
    payload: UploadSoundscapePayload,
  ): Promise<Soundscape> {
    const formData = new FormData();
    formData.append("name", payload.name);
    if (payload.category) formData.append("category", payload.category);
    if (payload.icon) formData.append("icon", payload.icon);
    if (typeof payload.volume === "number")
      formData.append("volume", String(payload.volume));
    if (payload.audio_url) formData.append("audio_url", payload.audio_url);
    if (payload.is_system) formData.append("is_system", "true");
    if (payload.file) formData.append("audio", payload.file);

    const res = await api.post<CommonResponse<Soundscape>>(
      "/soundscapes/upload",
      formData,
      {
        headers: { "Content-Type": "multipart/form-data" },
      },
    );
    return res.data.data as Soundscape;
  },

  async deleteSoundscape(id: string): Promise<void> {
    await api.delete(`/soundscapes/${id}`);
  },

  // Custom Fonts
  async getFonts(params?: CustomizationListParams): Promise<CustomFont[]> {
    const res = await api.get<{
      status: boolean;
      data: CustomFont[];
      next_cursor?: string | null;
    }>("/fonts", { params });
    return Array.isArray(res.data.data) ? res.data.data : [];
  },

  async uploadFont(payload: UploadFontPayload): Promise<CustomFont> {
    const formData = new FormData();
    formData.append("name", payload.name);
    formData.append("font_family", payload.font_family);
    formData.append("source_type", payload.source_type);
    if (payload.font_url) formData.append("font_url", payload.font_url);
    if (payload.is_system) formData.append("is_system", "true");
    if (payload.file) formData.append("font", payload.file);

    const res = await api.post<CommonResponse<CustomFont>>(
      "/fonts/upload",
      formData,
      {
        headers: { "Content-Type": "multipart/form-data" },
      },
    );
    return res.data.data as CustomFont;
  },

  async deleteFont(id: string): Promise<void> {
    await api.delete(`/fonts/${id}`);
  },

  // Custom Themes
  async getThemes(params?: CustomizationListParams): Promise<CustomTheme[]> {
    const res = await api.get<{
      status: boolean;
      data: CustomTheme[];
      next_cursor?: string | null;
    }>("/themes", { params });
    return Array.isArray(res.data.data) ? res.data.data : [];
  },

  async createTheme(payload: CreateCustomThemePayload): Promise<CustomTheme> {
    const res = await api.post<CommonResponse<CustomTheme>>("/themes", payload);
    return res.data.data as CustomTheme;
  },

  async updateTheme(
    id: string,
    payload: UpdateCustomThemePayload,
  ): Promise<CustomTheme> {
    const res = await api.put<CommonResponse<CustomTheme>>(
      `/themes/${id}`,
      payload,
    );
    return res.data.data as CustomTheme;
  },

  async deleteTheme(id: string): Promise<void> {
    await api.delete(`/themes/${id}`);
  },
};
