import { AlertCircle, Loader2 } from "lucide-react";
import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";

type DeleteConfirmModalProps = {
  open: boolean;
  title: string;
  message: React.ReactNode;
  onClose: () => void;
  onConfirm: () => void;
  confirmText?: string;
  cancelText?: string;
  loading?: boolean;
  expectedConfirmationText?: string;
};

export const DeleteConfirmModal: React.FC<DeleteConfirmModalProps> = ({
  open,
  title,
  message,
  onClose,
  onConfirm,
  confirmText,
  cancelText,
  loading = false,
  expectedConfirmationText,
}) => {
  const { t } = useTranslation();
  const [confirmInput, setConfirmInput] = useState("");

  useEffect(() => {
    if (open) {
      setConfirmInput("");
    }
  }, [open]);

  const isConfirmed = expectedConfirmationText
    ? confirmInput.trim() === expectedConfirmationText.trim() ||
      confirmInput.trim().toUpperCase() === "DELETE"
    : true;

  return (
    <dialog className={`modal ${open ? "modal-open" : ""}`}>
      <div className="modal-box max-w-md p-6 rounded-3xl border border-error/20 bg-base-100 shadow-2xl">
        <h3 className="flex items-center gap-2.5 text-lg font-black text-error">
          <AlertCircle className="h-6 w-6 shrink-0" />
          <span>{title}</span>
        </h3>
        <div className="py-4 text-sm text-base-content/80 leading-relaxed">{message}</div>

        {expectedConfirmationText && (
          <div className="space-y-2 mb-2 p-3.5 bg-error/5 border border-error/15 rounded-2xl">
            <label className="text-xs font-bold text-base-content/80 block">
              {t("admin.type_to_confirm", "Vui lòng nhập")} <span className="font-mono text-error font-extrabold select-all">"{expectedConfirmationText}"</span> {t("admin.or_type_delete", 'hoặc "DELETE" để xác nhận:')}
            </label>
            <input
              type="text"
              value={confirmInput}
              onChange={(e) => setConfirmInput(e.target.value)}
              placeholder={expectedConfirmationText}
              className="input input-bordered input-sm w-full rounded-xl bg-base-100 font-mono text-sm focus:border-error focus:outline-hidden"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter" && isConfirmed && !loading) {
                  e.preventDefault();
                  onConfirm();
                }
              }}
            />
          </div>
        )}

        <div className="modal-action mt-5">
          <button onClick={onClose} disabled={loading} className="btn btn-md btn-ghost rounded-xl font-semibold">
            {cancelText || t("common.cancel", "Cancel")}
          </button>
          <button
            onClick={onConfirm}
            disabled={loading || !isConfirmed}
            className="btn btn-md btn-error text-white font-bold rounded-xl px-5 flex items-center gap-2 shadow-md shadow-error/20 disabled:bg-base-300 disabled:text-base-content/30 disabled:border-transparent"
          >
            {loading && <Loader2 className="w-4 h-4 animate-spin" />}
            {confirmText || t("common.delete", "Delete")}
          </button>
        </div>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose} disabled={loading}>close</button>
      </form>
    </dialog>
  );
};

