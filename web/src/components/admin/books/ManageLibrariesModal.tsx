import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Check, Copy } from "lucide-react";

import type { Library } from "@/types";
import { copyText } from "@/utils/clipboard";

type ManageLibrariesModalProps = {
  open: boolean;
  libraries: Library[];
  newLibraryName: string;
  onClose: () => void;
  onNameChange: (name: string) => void;
  onCreate: (event: React.SyntheticEvent) => void;
  onDelete: (library: Library) => void;
  onRename: (id: string, name: string) => void;
};

export const ManageLibrariesModal: React.FC<ManageLibrariesModalProps> = ({
  open,
  libraries,
  newLibraryName,
  onClose,
  onNameChange,
  onCreate,
  onDelete,
  onRename,
}) => {
  const { t } = useTranslation();
  const [renaming, setRenaming] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const startRename = (lib: Library) => {
    setRenaming(lib.id);
    setRenameValue(lib.name);
  };

  const handleCopyId = (id: string) => {
    copyText(id).then((success) => {
      if (success) {
        setCopiedId(id);
        setTimeout(() => setCopiedId(null), 2000);
      }
    });
  };

  const submitRename = (id: string) => {
    if (
      renameValue.trim() &&
      renameValue.trim() !== libraries.find((l) => l.id === id)?.name
    ) {
      onRename(id, renameValue.trim());
    }
    setRenaming(null);
  };

  return (
    <dialog className={`modal ${open ? "modal-open" : ""}`}>
      <div className="modal-box">
        <button
          onClick={onClose}
          className="btn btn-ghost btn-circle btn-sm absolute right-2 top-2"
        >
          ✕
        </button>
        <h3 className="mb-4 border-b border-base-200 pb-4 text-lg font-bold">
          {t("admin.manage_libraries", "Manage Libraries")}
        </h3>

        <form onSubmit={onCreate} className="mb-6 flex gap-2">
          <input
            type="text"
            placeholder={t(
              "admin.library_name_placeholder",
              "New library name...",
            )}
            className="input input-bordered flex-1"
            value={newLibraryName}
            onChange={(event) => onNameChange(event.target.value)}
          />
          <button
            type="submit"
            className="btn btn-primary"
            disabled={!newLibraryName.trim()}
          >
            {t("admin.add_library", "Add Library")}
          </button>
        </form>

        <div className="flex max-h-64 flex-col gap-2 overflow-y-auto">
          {libraries.length === 0 ? (
            <p className="py-4 text-center text-base-content/50">
              {t("admin.no_libraries", "No libraries found.")}
            </p>
          ) : (
            libraries.map((library) => (
              <div
                key={library.id}
                className="flex items-center justify-between gap-2 rounded-lg bg-base-200 p-3"
              >
                {renaming === library.id ? (
                  <input
                    type="text"
                    className="input input-bordered input-sm flex-1"
                    value={renameValue}
                    onChange={(e) => setRenameValue(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") submitRename(library.id);
                      if (e.key === "Escape") setRenaming(null);
                    }}
                    autoFocus
                  />
                ) : (
                  <div className="flex flex-col min-w-0">
                    <span className="font-medium truncate">{library.name}</span>
                    <div className="flex items-center gap-1.5 mt-0.5 text-xs text-base-content/50">
                      <span
                        className="font-mono text-[10px] select-all truncate max-w-[150px] sm:max-w-[200px]"
                        title={library.id}
                      >
                        {library.id}
                      </span>
                      <button
                        type="button"
                        onClick={() => handleCopyId(library.id)}
                        className="btn btn-ghost btn-circle btn-xs hover:bg-base-300 h-5 w-5 min-h-0"
                        title={t("admin.copy_library_id")}
                      >
                        {copiedId === library.id ? (
                          <Check size={11} className="text-success" />
                        ) : (
                          <Copy size={11} />
                        )}
                      </button>
                    </div>
                  </div>
                )}
                <div className="flex gap-1 shrink-0">
                  <button
                    onClick={() =>
                      renaming === library.id
                        ? submitRename(library.id)
                        : startRename(library)
                    }
                    className="btn btn-ghost btn-xs"
                  >
                    {renaming === library.id
                      ? t("common.save", "Save")
                      : t("admin.rename_library", "Rename")}
                  </button>
                  <button
                    onClick={() => onDelete(library)}
                    className="btn btn-error btn-outline btn-xs"
                  >
                    {t("admin.delete", "Delete")}
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose}>close</button>
      </form>
    </dialog>
  );
};
