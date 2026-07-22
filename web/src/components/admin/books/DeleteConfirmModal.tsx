import { AlertCircle, Loader2 } from "lucide-react";
import React from "react";
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
}) => {
  const { t } = useTranslation();

  return (
    <dialog className={`modal ${open ? "modal-open" : ""}`}>
      <div className="modal-box">
        <h3 className="flex items-center gap-2 text-lg font-bold text-error">
          <AlertCircle className="h-6 w-6" />
          {title}
        </h3>
        <div className="py-4 text-sm opacity-80">{message}</div>
        <div className="modal-action">
          <button onClick={onClose} disabled={loading} className="btn btn-ghost">
            {cancelText || t("common.cancel", "Cancel")}
          </button>
          <button onClick={onConfirm} disabled={loading} className="btn btn-error text-white flex items-center gap-2">
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
