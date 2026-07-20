import { AlertCircle } from "lucide-react";
import React from "react";

type DeleteConfirmModalProps = {
  open: boolean;
  title: string;
  message: React.ReactNode;
  onClose: () => void;
  onConfirm: () => void;
};

export const DeleteConfirmModal: React.FC<DeleteConfirmModalProps> = ({
  open,
  title,
  message,
  onClose,
  onConfirm,
}) => (
  <dialog className={`modal ${open ? "modal-open" : ""}`}>
    <div className="modal-box">
      <h3 className="flex items-center gap-2 text-lg font-bold text-error">
        <AlertCircle className="h-6 w-6" />
        {title}
      </h3>
      <p className="py-4 text-sm opacity-80">{message}</p>
      <div className="modal-action">
        <button onClick={onClose} className="btn btn-ghost">
          Cancel
        </button>
        <button onClick={onConfirm} className="btn btn-error">
          Delete
        </button>
      </div>
    </div>
    <form method="dialog" className="modal-backdrop">
      <button onClick={onClose}>close</button>
    </form>
  </dialog>
);
