import {
  AlertCircle,
  HelpCircle,
  AlertTriangle,
  CheckCircle,
  Loader2,
  X,
} from "lucide-react";
import React from "react";
import { useTranslation } from "react-i18next";

type ConfirmModalProps = {
  open: boolean;
  title: string;
  message: React.ReactNode;
  onClose: () => void;
  onConfirm: () => void;
  confirmText?: string;
  cancelText?: string;
  loading?: boolean;
  variant?: "danger" | "warning" | "info" | "success";
};

export const ConfirmModal: React.FC<ConfirmModalProps> = ({
  open,
  title,
  message,
  onClose,
  onConfirm,
  confirmText,
  cancelText,
  loading = false,
  variant = "info",
}) => {
  const { t } = useTranslation();

  const getIcon = () => {
    switch (variant) {
      case "danger":
        return <AlertCircle className="h-6 w-6 text-error" />;
      case "warning":
        return <AlertTriangle className="h-6 w-6 text-warning" />;
      case "success":
        return <CheckCircle className="h-6 w-6 text-success" />;
      default:
        return <HelpCircle className="h-6 w-6 text-info" />;
    }
  };

  const getConfirmBtnClass = () => {
    switch (variant) {
      case "danger":
        return "btn-error text-white";
      case "warning":
        return "btn-warning text-warning-content";
      case "success":
        return "btn-success text-success-content";
      default:
        return "btn-primary text-primary-content";
    }
  };

  return (
    <dialog className={`modal ${open ? "modal-open" : ""}`}>
      <div className="modal-box max-w-md w-11/12 sm:w-full max-h-[85vh] p-0 overflow-hidden flex flex-col">
        <div className="px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-200 flex items-center justify-between gap-3 shrink-0 bg-base-100">
          <h3 className="flex items-center gap-2 text-lg font-bold leading-tight">
            {getIcon()}
            <span>{title}</span>
          </h3>
          <button
            type="button"
            onClick={onClose}
            disabled={loading}
            className="btn btn-ghost btn-circle btn-sm -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
            aria-label={t("common.close", "Close")}
          >
            <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
          </button>
        </div>
        <div className="p-4 sm:p-6 overflow-y-auto flex-1 min-h-0">
          <div className="text-sm opacity-80 wrap-break-word min-w-0">{message}</div>
          <div className="modal-action border-t border-base-200 pt-4 mt-6">
            <button
              onClick={onClose}
              disabled={loading}
              className="btn btn-ghost"
            >
              {cancelText || t("common.cancel", "Cancel")}
            </button>
            <button
              onClick={() => {
                onConfirm();
              }}
              disabled={loading}
              className={`btn ${getConfirmBtnClass()} flex items-center gap-2`}
            >
              {loading && <Loader2 className="w-4 h-4 animate-spin" />}
              {confirmText || t("common.confirm", "Confirm")}
            </button>
          </div>
        </div>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose} disabled={loading}>
          close
        </button>
      </form>
    </dialog>
  );
};
