import { useMutation } from "@tanstack/react-query";
import { magicCodeService } from "@/services/magicCodeService";

export const useRequestMagicCodeMutation = () => {
  return useMutation({
    mutationFn: (deviceInfo?: string) =>
      magicCodeService.requestCode(deviceInfo),
  });
};

export const usePollMagicCodeMutation = () => {
  return useMutation({
    mutationFn: (pollToken: string) => magicCodeService.pollCode(pollToken),
  });
};

export const useActivateMagicCodeMutation = () => {
  return useMutation({
    mutationFn: (code: string) => magicCodeService.activateCode(code),
  });
};
