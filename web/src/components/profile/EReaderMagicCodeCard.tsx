import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { QrCode, CheckCircle2, AlertCircle, Smartphone } from "lucide-react";
import { useActivateMagicCodeMutation } from "@/hooks";
import { toast } from "react-toastify";

export const EReaderMagicCodeCard: React.FC = () => {
  const { t } = useTranslation();
  const [code, setCode] = useState("");
  const [loading, setLoading] = useState(false);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const activateMutation = useActivateMagicCodeMutation();

  const handleActivate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!code.trim()) return;

    setSuccessMessage(null);
    setErrorMessage(null);

    try {
      const data = await activateMutation.mutateAsync(code.trim());
      if (data?.status) {
        setSuccessMessage(
          t(
            "profile.ereader_activated",
            "Device activated successfully! Your eReader is now logged in.",
          ),
        );
        setCode("");
        toast.success(
          t("profile.ereader_activated", "Device activated successfully!"),
        );
      } else {
        setErrorMessage(
          data?.message ||
            t(
              "profile.ereader_activate_failed",
              "Failed to activate eReader code",
            ),
        );
      }
    } catch (err: any) {
      const msg =
        err.response?.data?.message ||
        err.message ||
        t("profile.ereader_activate_failed", "Failed to activate eReader code");
      setErrorMessage(msg);
    }
  };

  return (
    <div className="card bg-base-100 shadow-sm border border-base-200">
      <div className="card-body p-5">
        <div className="flex items-center gap-3 mb-2">
          <div className="p-2 rounded-lg bg-primary/10 text-primary">
            <QrCode className="w-5 h-5" />
          </div>
          <div>
            <h3 className="card-title text-base">
              {t("profile.ereader_login_title", "eReader Quick Login")}
            </h3>
            <p className="text-xs text-base-content/70">
              {t(
                "profile.ereader_login_desc",
                "Log in to your Kindle, Kobo, Boox or mobile eReader without typing your password.",
              )}
            </p>
          </div>
        </div>

        <form onSubmit={handleActivate} className="mt-3 flex flex-col gap-3">
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="e.g. 849-102"
              maxLength={10}
              className="input input-bordered input-sm flex-1 font-mono tracking-widest text-center uppercase"
            />
            <button
              type="submit"
              disabled={activateMutation.isPending || !code.trim()}
              className="btn btn-primary btn-sm"
            >
              {activateMutation.isPending ? (
                <span className="loading loading-spinner loading-xs"></span>
              ) : (
                <>
                  <Smartphone className="w-4 h-4" />
                  {t("profile.activate_device", "Activate Device")}
                </>
              )}
            </button>
          </div>

          {successMessage && (
            <div className="alert alert-success py-2 px-3 text-xs flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 shrink-0" />
              <span>{successMessage}</span>
            </div>
          )}

          {errorMessage && (
            <div className="alert alert-error py-2 px-3 text-xs flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{errorMessage}</span>
            </div>
          )}
        </form>
      </div>
    </div>
  );
};
