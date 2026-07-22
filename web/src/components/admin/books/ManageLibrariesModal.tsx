import React from "react";

import type { Library } from "@/types";

type ManageLibrariesModalProps = {
  open: boolean;
  libraries: Library[];
  newLibraryName: string;
  onClose: () => void;
  onNameChange: (name: string) => void;
  onCreate: (event: React.SyntheticEvent) => void;
  onDelete: (library: Library) => void;
};

export const ManageLibrariesModal: React.FC<ManageLibrariesModalProps> = ({
  open,
  libraries,
  newLibraryName,
  onClose,
  onNameChange,
  onCreate,
  onDelete,
}) => (
  <dialog className={`modal ${open ? "modal-open" : ""}`}>
    <div className="modal-box">
      <button
        onClick={onClose}
        className="btn btn-ghost btn-circle btn-sm absolute right-2 top-2"
      >
        ✕
      </button>
      <h3 className="mb-4 border-b border-base-200 pb-4 text-lg font-bold">
        Manage Libraries
      </h3>

      <form onSubmit={onCreate} className="mb-6 flex gap-2">
        <input
          type="text"
          placeholder="New library name..."
          className="input input-bordered flex-1"
          value={newLibraryName}
          onChange={(event) => onNameChange(event.target.value)}
        />
        <button
          type="submit"
          className="btn btn-primary"
          disabled={!newLibraryName.trim()}
        >
          Create
        </button>
      </form>

      <div className="flex max-h-64 flex-col gap-2 overflow-y-auto">
        {libraries.length === 0 ? (
          <p className="py-4 text-center text-base-content/50">
            No libraries found.
          </p>
        ) : (
          libraries.map((library) => (
            <div
              key={library.id}
              className="flex items-center justify-between rounded-lg bg-base-200 p-3"
            >
              <span className="font-medium">{library.name}</span>
              <button
                onClick={() => onDelete(library)}
                className="btn btn-error btn-outline btn-xs"
              >
                Delete
              </button>
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
