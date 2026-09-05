import {
  AlertCircle,
  HelpCircle,
  AlertTriangle,
  CheckCircle,
  Loader2,
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
      <div className="modal-box">
        <h3 className="flex items-center gap-2 text-lg font-bold">
          {getIcon()}
          {title}
        </h3>
        <div className="py-4 text-sm opacity-80">{message}</div>
        <div className="modal-action">
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
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose} disabled={loading}>
          close
        </button>
      </form>
    </dialog>
  );
};
