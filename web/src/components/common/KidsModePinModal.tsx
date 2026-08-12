import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { ShieldCheck, Lock, AlertCircle, X } from "lucide-react";
import { useToggleKidsModeMutation } from "@/hooks";
import { toast } from "react-toastify";

interface KidsModePinModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  title?: string;
  description?: string;
}

export const KidsModePinModal: React.FC<KidsModePinModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
  title,
  description,
}) => {
  const { t } = useTranslation();
  const [pin, setPin] = useState("");
  const [error, setError] = useState<string | null>(null);
  
  const toggleMutation = useToggleKidsModeMutation();

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (pin.length !== 6 || !/^\d{6}$/.test(pin)) {
      setError(t("kids_mode.pin_must_be_6_digits", "PIN must be exactly 6 digits"));
      return;
    }

    setError(null);

    try {
      const data = await toggleMutation.mutateAsync({ enable: false, pin });
      if (data?.status) {
        toast.success(t("kids_mode.disabled_success", "Kids Mode disabled"));
        setPin("");
        onSuccess();
        onClose();
      } else {
        setError(data?.message || t("kids_mode.incorrect_pin", "Incorrect 6-digit PIN"));
      }
    } catch (err: any) {
      setError(err.response?.data?.message || err.message || t("kids_mode.incorrect_pin", "Incorrect 6-digit PIN"));
    }
  };

  return (
    <div className="modal modal-open">
      <div className="modal-box relative max-w-sm p-6 text-center">
        <button
          onClick={onClose}
          className="btn btn-sm btn-circle btn-ghost absolute right-3 top-3"
        >
          <X className="w-4 h-4" />
        </button>

        <div className="mx-auto grid h-14 w-14 place-items-center rounded-2xl bg-primary/10 text-primary mb-3">
          <ShieldCheck className="h-7 w-7" />
        </div>

        <h3 className="text-lg font-bold">
          {title || t("kids_mode.enter_pin_title", "Enter 6-Digit PIN")}
        </h3>

        <p className="text-xs text-base-content/70 mt-1">
          {description || t("kids_mode.enter_pin_desc", "Enter your 6-digit PIN to exit Kids Mode and access restricted content.")}
        </p>

        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          <div>
            <input
              type="password"
              value={pin}
              onChange={(e) => setPin(e.target.value.replace(/\D/g, "").slice(0, 6))}
              placeholder="••••••"
              maxLength={6}
              className="input input-bordered w-full text-center font-mono text-2xl tracking-[0.5em] font-bold"
              autoFocus
            />
          </div>

          {error && (
            <div className="alert alert-error py-2 px-3 text-xs flex items-center gap-2">
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <div className="flex items-center gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="btn btn-ghost btn-sm flex-1"
            >
              {t("common.cancel", "Cancel")}
            </button>
            <button
              type="submit"
              disabled={toggleMutation.isPending || pin.length !== 6}
              className="btn btn-primary btn-sm flex-1 gap-1"
            >
              {toggleMutation.isPending ? (
                <span className="loading loading-spinner loading-xs"></span>
              ) : (
                <>
                  <Lock className="w-4 h-4" />
                  {t("kids_mode.unlock", "Unlock")}
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
