import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { customizationService } from "@/services";
import type {
  CreateCustomThemePayload,
  CustomFont,
  CustomTheme,
  Soundscape,
  UpdateCustomThemePayload,
  UploadFontPayload,
  UploadSoundscapePayload,
} from "@/types";

export const useCustomization = () => {
  const queryClient = useQueryClient();

  const soundscapesQuery = useQuery<Soundscape[]>({
    queryKey: ["soundscapes"],
    queryFn: () => customizationService.getSoundscapes(),
    staleTime: 60000,
  });

  const uploadSoundscapeMutation = useMutation({
    mutationFn: (payload: UploadSoundscapePayload) =>
      customizationService.uploadSoundscape(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["soundscapes"] });
    },
  });

  const deleteSoundscapeMutation = useMutation({
    mutationFn: (id: string) => customizationService.deleteSoundscape(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["soundscapes"] });
    },
  });

  const customFontsQuery = useQuery<CustomFont[]>({
    queryKey: ["custom_fonts"],
    queryFn: () => customizationService.getFonts(),
    staleTime: 60000,
  });

  const uploadCustomFontMutation = useMutation({
    mutationFn: (payload: UploadFontPayload) =>
      customizationService.uploadFont(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["custom_fonts"] });
    },
  });

  const deleteCustomFontMutation = useMutation({
    mutationFn: (id: string) => customizationService.deleteFont(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["custom_fonts"] });
    },
  });

  const customThemesQuery = useQuery<CustomTheme[]>({
    queryKey: ["custom_themes"],
    queryFn: () => customizationService.getThemes(),
    staleTime: 60000,
  });

  const createCustomThemeMutation = useMutation({
    mutationFn: (payload: CreateCustomThemePayload) =>
      customizationService.createTheme(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["custom_themes"] });
    },
  });

  const updateCustomThemeMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateCustomThemePayload }) =>
      customizationService.updateTheme(id, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["custom_themes"] });
    },
  });

  const deleteCustomThemeMutation = useMutation({
    mutationFn: (id: string) => customizationService.deleteTheme(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["custom_themes"] });
    },
  });

  return {
    soundscapes: soundscapesQuery.data || [],
    isSoundscapesLoading: soundscapesQuery.isLoading,
    refetchSoundscapes: soundscapesQuery.refetch,
    uploadSoundscape: uploadSoundscapeMutation.mutateAsync,
    isUploadingSoundscape: uploadSoundscapeMutation.isPending,
    deleteSoundscape: deleteSoundscapeMutation.mutateAsync,

    customFonts: customFontsQuery.data || [],
    isFontsLoading: customFontsQuery.isLoading,
    refetchFonts: customFontsQuery.refetch,
    uploadCustomFont: uploadCustomFontMutation.mutateAsync,
    isUploadingFont: uploadCustomFontMutation.isPending,
    deleteCustomFont: deleteCustomFontMutation.mutateAsync,

    customThemes: customThemesQuery.data || [],
    isThemesLoading: customThemesQuery.isLoading,
    refetchThemes: customThemesQuery.refetch,
    createCustomTheme: createCustomThemeMutation.mutateAsync,
    updateCustomTheme: updateCustomThemeMutation.mutateAsync,
    deleteCustomTheme: deleteCustomThemeMutation.mutateAsync,
  };
};

export const useSoundscapesQuery = () =>
  useQuery<Soundscape[]>({
    queryKey: ["soundscapes"],
    queryFn: () => customizationService.getSoundscapes(),
    staleTime: 60000,
  });

export const useCustomFontsQuery = () =>
  useQuery<CustomFont[]>({
    queryKey: ["custom_fonts"],
    queryFn: () => customizationService.getFonts(),
    staleTime: 60000,
  });

export const useCustomThemesQuery = () =>
  useQuery<CustomTheme[]>({
    queryKey: ["custom_themes"],
    queryFn: () => customizationService.getThemes(),
    staleTime: 60000,
  });
