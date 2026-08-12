import { api } from "@/config/api";
import { CommonResponse } from "@/types";
import { ContentWarning, KidsModeInfo } from "@/types/ageRating";

export const ageRatingService = {
  getContentWarnings: async () => {
    const res = await api.get<CommonResponse<ContentWarning[]>>("/content-warnings");
    return res.data;
  },

  getBookContentWarnings: async (bookId: string) => {
    const res = await api.get<CommonResponse<ContentWarning[]>>(`/books/${bookId}/content-warnings`);
    return res.data;
  },

  updateBookAgeRating: async (
    bookId: string,
    payload: { age_rating: string; content_warning_ids: string[] }
  ) => {
    const res = await api.put<CommonResponse<null>>(
      `/books/${bookId}/age-rating`,
      payload
    );
    return res.data;
  },

  getKidsModeInfo: async () => {
    const res = await api.get<CommonResponse<KidsModeInfo>>("/user/kids-mode/info");
    return res.data;
  },

  setKidsModePin: async (pin: string) => {
    const res = await api.post<CommonResponse<null>>("/user/kids-mode/pin", { pin });
    return res.data;
  },

  toggleKidsMode: async (enable: boolean, pin?: string) => {
    const res = await api.post<CommonResponse<null>>("/user/kids-mode/toggle", { enable, pin });
    return res.data;
  },
};
