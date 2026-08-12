import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Shield, KeyRound, Check, AlertCircle } from "lucide-react";
import { useKidsModeInfo, useSetKidsModePinMutation } from "@/hooks";
import { toast } from "react-toastify";

export const KidsModePinCard: React.FC = () => {
  const { t } = useTranslation();
  const [pin, setPin] = useState("");
  const [confirmPin, setConfirmPin] = useState("");
  const [hasPin, setHasPin] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { data: kidsModeData } = useKidsModeInfo();
  const setPinMutation = useSetKidsModePinMutation();

  useEffect(() => {
    if (kidsModeData?.status && kidsModeData?.data) {
      setHasPin(kidsModeData.data.has_pin);
    }
  }, [kidsModeData]);

  const handleSetPin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (pin.length !== 6 || !/^\d{6}$/.test(pin)) {
      setError(t("kids_mode.pin_must_be_6_digits", "PIN must be exactly 6 digits"));
      return;
    }

    if (pin !== confirmPin) {
      setError(t("kids_mode.pins_do_not_match", "PINs do not match"));
      return;
    }

    setError(null);

    try {
      const data = await setPinMutation.mutateAsync(pin);
      if (data?.status) {
        toast.success(t("kids_mode.pin_saved", "6-digit Kids Mode PIN updated successfully!"));
        setPin("");
        setConfirmPin("");
        setHasPin(true);
      } else {
        setError(data?.message || t("kids_mode.pin_save_failed", "Failed to update PIN"));
      }
    } catch (err: any) {
      setError(err.response?.data?.message || err.message || t("kids_mode.pin_save_failed", "Failed to update PIN"));
    }
  };

  return (
    <div className="card bg-base-100 shadow-sm border border-base-200">
      <div className="card-body p-5">
        <div className="flex items-center gap-3 mb-2">
          <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-500">
            <Shield className="w-5 h-5" />
          </div>
          <div>
            <h3 className="card-title text-base">
              {t("kids_mode.parental_control_title", "Parental Control & Kids Mode PIN")}
            </h3>
            <p className="text-xs text-base-content/70">
              {t(
                "kids_mode.parental_control_desc",
                "Set a secure 6-digit PIN to prevent children from exiting Kids Mode or accessing R17+/R18+ content."
              )}
            </p>
          </div>
        </div>

        <form onSubmit={handleSetPin} className="mt-3 flex flex-col gap-3">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="label text-xs font-semibold text-base-content/70">
                {hasPin ? t("kids_mode.new_6_digit_pin", "New 6-Digit PIN") : t("kids_mode.enter_6_digit_pin_label", "Set 6-Digit PIN")}
              </label>
              <input
                type="password"
                value={pin}
                onChange={(e) => setPin(e.target.value.replace(/\D/g, "").slice(0, 6))}
                placeholder="••••••"
                maxLength={6}
                className="input input-bordered input-sm font-mono text-center text-lg tracking-widest w-full"
              />
            </div>

            <div>
              <label className="label text-xs font-semibold text-base-content/70">
                {t("kids_mode.confirm_6_digit_pin", "Confirm 6-Digit PIN")}
              </label>
              <input
                type="password"
                value={confirmPin}
                onChange={(e) => setConfirmPin(e.target.value.replace(/\D/g, "").slice(0, 6))}
                placeholder="••••••"
                maxLength={6}
                className="input input-bordered input-sm font-mono text-center text-lg tracking-widest w-full"
              />
            </div>
          </div>

          {error && (
            <div className="alert alert-error py-2 px-3 text-xs flex items-center gap-2">
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <div className="flex items-center justify-between pt-1">
            <span className="text-xs text-base-content/60">
              {hasPin
                ? t("kids_mode.pin_is_active", "Status: 6-digit PIN active")
                : t("kids_mode.no_pin_set", "Status: No PIN configured")}
            </span>

            <button
              type="submit"
              disabled={setPinMutation.isPending || pin.length !== 6 || confirmPin.length !== 6}
              className="btn btn-emerald btn-sm gap-1 text-white bg-emerald-600 hover:bg-emerald-700 border-none"
            >
              {setPinMutation.isPending ? (
                <span className="loading loading-spinner loading-xs"></span>
              ) : (
                <>
                  <Check className="w-4 h-4" />
                  {hasPin ? t("kids_mode.update_pin", "Update PIN") : t("kids_mode.save_pin", "Save 6-Digit PIN")}
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
