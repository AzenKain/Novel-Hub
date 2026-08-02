import React, { useState } from "react";
import { Mail, Send, X } from "lucide-react";
import { useSendBookToEmailMutation } from "@/hooks";
import type { Book } from "@/types";

type SendToKindleModalProps = {
  open: boolean;
  book: Book;
  t: (key: string, fallback: string) => string;
  onClose: () => void;
  onSuccess?: () => void;
};

export const SendToKindleModal: React.FC<SendToKindleModalProps> = ({
  open,
  book,
  t,
  onClose,
  onSuccess,
}) => {
  const [email, setEmail] = useState("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const sendEmailMutation = useSendBookToEmailMutation();

  if (!open) return null;

  const handleSend = (e: React.FormEvent) => {
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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        aria-label={t("common.close", "Close")}
        onClick={onClose}
      />
      <section className="relative z-10 w-full max-w-md rounded-2xl border border-base-300 bg-base-100 p-6 shadow-2xl">
        <div className="mb-4 flex items-start justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="grid h-10 w-10 place-items-center rounded-xl bg-primary/10 text-primary">
              <Mail className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-xl font-bold">
                {t("library.send_to_kindle", "Send to Kindle")}
              </h2>
              <p className="text-xs text-base-content/60">{book.title}</p>
            </div>
          </div>
          <button
            className="btn btn-ghost btn-circle btn-sm"
            onClick={onClose}
            aria-label={t("common.close", "Close")}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <form onSubmit={handleSend} className="space-y-4">
          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-base-content/70 mb-1">
              {t("email.recipient_label", "Recipient Email Address")}
            </label>
            <input
              type="email"
              placeholder="user@kindle.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="input input-bordered w-full"
              required
            />
            <p className="mt-1.5 text-xs text-base-content/50">
              {t("email.kindle_tip", "Enter your Kindle device email or personal email address.")}
            </p>
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

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              className="btn btn-ghost text-sm"
              onClick={onClose}
              disabled={sendEmailMutation.isPending}
            >
              {t("admin.cancel", "Cancel")}
            </button>
            <button
              type="submit"
              className="btn btn-primary text-sm gap-2"
              disabled={sendEmailMutation.isPending}
            >
              {sendEmailMutation.isPending ? (
                <span className="loading loading-spinner loading-xs" />
              ) : (
                <Send className="h-4 w-4" />
              )}
              {t("email.send_btn", "Send Book")}
            </button>
          </div>
        </form>
      </section>
    </div>
  );
};
