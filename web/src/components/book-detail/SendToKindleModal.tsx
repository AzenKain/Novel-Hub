import React, { useState } from "react";
import { Cpu, Send, Smartphone, Tablet, X } from "lucide-react";
import { useDevicesQuery, usePushBookMutation, useSendBookToEmailMutation } from "@/hooks";
import { usePublicSettings } from "@/hooks/useSettings";
import type { Book, UserDevice } from "@/types";

type SendToKindleModalProps = {
  open: boolean;
  book: Book;
  t: (key: string, fallback: string) => string;
  onClose: () => void;
  onSuccess?: () => void;
};

function getDeviceIcon(type: string) {
  switch (type?.toLowerCase()) {
    case "kindle":
      return <Tablet className="h-4 w-4 text-warning" />;
    case "pocketbook":
      return <Smartphone className="h-4 w-4 text-info" />;
    case "koreader":
      return <Cpu className="h-4 w-4 text-success" />;
    default:
      return <Tablet className="h-4 w-4 text-primary" />;
  }
}

export const SendToKindleModal: React.FC<SendToKindleModalProps> = ({
  open,
  book,
  t,
  onClose,
  onSuccess,
}) => {
  const [activeTab, setActiveTab] = useState<"saved" | "manual">("saved");
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>("");
  const [email, setEmail] = useState("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const publicSettings = usePublicSettings();
  const devicesQuery = useDevicesQuery(open);
  const sendEmailMutation = useSendBookToEmailMutation();
  const pushBookMutation = usePushBookMutation();

  if (!open) return null;

  const devices = devicesQuery.data || [];

  const handlePushToSavedDevice = (device: UserDevice) => {
    setErrorMsg(null);
    setSuccessMsg(null);
    setSelectedDeviceId(device.id);

    pushBookMutation.mutate(
      { bookId: book.id, deviceId: device.id },
      {
        onSuccess: () => {
          setSuccessMsg(t("email.sent_success", `Book successfully dispatched to ${device.name}!`));
          if (onSuccess) onSuccess();
          setTimeout(() => {
            onClose();
          }, 1800);
        },
        onError: (err: any) => {
          setErrorMsg(err?.message || t("email.send_failed", "Failed to deliver book to device."));
        },
      }
    );
  };

  const handleManualSend = (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !email.includes("@")) {
      setErrorMsg(t("email.invalid", "Please enter a valid email address"));
      return;
    }

    setErrorMsg(null);
    setSuccessMsg(null);

    sendEmailMutation.mutate(
      { book_id: book.id, recipientEmail: email },
      {
        onSuccess: () => {
          setSuccessMsg(t("email.sent_success", "Book successfully dispatched to your email!"));
          if (onSuccess) onSuccess();
          setTimeout(() => {
            onClose();
          }, 1800);
        },
        onError: (err: any) => {
          setErrorMsg(err?.message || t("email.send_failed", "Failed to send email. Check SMTP configuration."));
        },
      }
    );
  };

  const isLoading = sendEmailMutation.isPending || pushBookMutation.isPending;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        aria-label={t("common.close", "Close")}
        onClick={onClose}
      />
      <section className="relative z-10 w-full max-w-md rounded-2xl border border-base-300 bg-base-100 p-6 shadow-2xl space-y-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="grid h-10 w-10 place-items-center rounded-xl bg-primary/10 text-primary">
              <Send className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-xl font-bold">
                {t("library.send_to_device", "Send to Device")}
              </h2>
              <p className="text-xs text-base-content/60">{book.title}</p>
            </div>
          </div>
          <button
            className="btn btn-ghost btn-circle btn-sm text-base-content hover:bg-base-200 border border-base-300 shadow-xs"
            onClick={onClose}
            aria-label={t("common.close", "Close")}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Tab Selection */}
        <div className="flex border-b border-base-200 text-xs font-semibold">
          <button
            className={`pb-2 px-3 border-b-2 transition-colors ${activeTab === "saved" ? "border-primary text-primary" : "border-transparent text-base-content/60 hover:text-base-content"}`}
            onClick={() => setActiveTab("saved")}
          >
            {t("device.saved_devices", "Saved Devices")} ({devices.length})
          </button>
          <button
            className={`pb-2 px-3 border-b-2 transition-colors ${activeTab === "manual" ? "border-primary text-primary" : "border-transparent text-base-content/60 hover:text-base-content"}`}
            onClick={() => setActiveTab("manual")}
          >
            {t("device.manual_email", "Manual Email")}
          </button>
        </div>

        {errorMsg && (
          <div className="alert alert-error text-xs py-2 px-3 rounded-lg">
            <span>{errorMsg}</span>
          </div>
        )}

        {successMsg && (
          <div className="alert alert-success text-xs py-2 px-3 rounded-lg">
            <span>{successMsg}</span>
          </div>
        )}

        {activeTab === "saved" && (
          <div className="space-y-2 max-h-60 overflow-y-auto">
            {devices.length === 0 ? (
              <div className="text-center py-6 border border-dashed border-base-200 rounded-xl">
                <p className="text-xs text-base-content/60">
                  {t("device.no_devices", "No saved devices found.")}
                </p>
                <p className="text-[11px] text-base-content/40 mt-1">
                  {t("device.add_device_tip", "Use Manual Email or register devices in settings.")}
                </p>
              </div>
            ) : (
              devices.map((device) => (
                <div
                  key={device.id}
                  className="flex items-center justify-between p-3 border border-base-200 rounded-xl hover:border-primary/40 bg-base-100 transition-all"
                >
                  <div className="flex items-center gap-2.5">
                    <div className="p-2 rounded-lg bg-base-200/60">
                      {getDeviceIcon(device.device_type)}
                    </div>
                    <div>
                      <div className="font-semibold text-xs text-base-content">{device.name}</div>
                      <div className="text-[10px] text-base-content/50 font-mono truncate max-w-[180px]">
                        {device.target_address}
                      </div>
                    </div>
                  </div>
                  <button
                    className="btn btn-sm btn-primary rounded-lg text-xs gap-1.5"
                    disabled={isLoading}
                    onClick={() => handlePushToSavedDevice(device)}
                  >
                    {isLoading && selectedDeviceId === device.id ? (
                      <span className="loading loading-spinner loading-xs" />
                    ) : (
                      <Send className="h-3.5 w-3.5" />
                    )}
                    {t("device.push", "Push")}
                  </button>
                </div>
              ))
            )}
          </div>
        )}

        {activeTab === "manual" && (
          <form onSubmit={handleManualSend} className="space-y-4">
            {publicSettings?.smtp_enabled === false && (
              <div className="alert alert-warning text-xs py-2 px-3 rounded-lg">
                <span>{t("email.smtp_disabled", "SMTP server is not configured or disabled by administrator.")}</span>
              </div>
            )}
            <div>
              <label className="block text-xs font-semibold uppercase tracking-wider text-base-content/70 mb-1">
                {t("email.recipient_label", "Recipient Email Address")}
              </label>
              <input
                type="email"
                placeholder="user@kindle.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="input input-bordered w-full text-xs"
                disabled={publicSettings?.smtp_enabled === false}
                required
              />
              <p className="mt-1.5 text-xs text-base-content/50">
                {t("email.kindle_tip", "Enter your Kindle device email or personal email address.")}
              </p>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                className="btn btn-ghost text-xs"
                onClick={onClose}
                disabled={isLoading}
              >
                {t("admin.cancel", "Cancel")}
              </button>
              <button
                type="submit"
                className="btn btn-primary text-xs gap-2"
                disabled={isLoading || publicSettings?.smtp_enabled === false}
              >
                {isLoading ? (
                  <span className="loading loading-spinner loading-xs" />
                ) : (
                  <Send className="h-4 w-4" />
                )}
                {t("email.send_btn", "Send Book")}
              </button>
            </div>
          </form>
        )}
      </section>
    </div>
  );
};
