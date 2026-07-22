import type { TFunction } from "i18next";
import { Volume2, Play, Copy } from "lucide-react";
import React from "react";

type ReaderSelectionToolbarProps = {
  t: TFunction;
  toolbarPos: { top: number; left: number };
  isSupported: boolean;
  onReadSelection: () => void;
  onReadFromHere: () => void;
  onCopyText?: () => void;
  onHighlight: (color: string) => void;
};

export const ReaderSelectionToolbar: React.FC<ReaderSelectionToolbarProps> = ({
  t,
  toolbarPos,
  isSupported,
  onReadSelection,
  onReadFromHere,
  onCopyText,
  onHighlight,
}) => {
  return (
    <div
      data-reader-toolbar="true"
      onMouseDown={(e) => {
        e.preventDefault();
        e.stopPropagation();
      }}
      onMouseUp={(e) => {
        e.preventDefault();
        e.stopPropagation();
      }}
      className="fixed z-50 flex -translate-x-1/2 -translate-y-full items-center gap-1.5 rounded-full border border-slate-700 bg-slate-900/95 px-3.5 py-2 text-slate-100 shadow-2xl backdrop-blur-md transition-all duration-200"
      style={{ top: `${toolbarPos.top}px`, left: `${toolbarPos.left}px` }}
    >
      {onCopyText && (
        <button
          type="button"
          data-reader-toolbar="true"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onCopyText();
          }}
          className="btn btn-ghost btn-xs gap-1.5 text-xs font-medium text-slate-100 hover:bg-slate-800 hover:text-white"
          title={t("reader.copy_text", "Copy Text")}
        >
          <Copy className="h-3.5 w-3.5 text-amber-400 pointer-events-none" />
          <span className="pointer-events-none">{t("reader.copy_text", "Copy Text")}</span>
        </button>
      )}

      {isSupported && (
        <>
          <button
            type="button"
            data-reader-toolbar="true"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onReadSelection();
            }}
            className="btn btn-ghost btn-xs gap-1.5 text-xs font-medium text-slate-100 hover:bg-slate-800 hover:text-white"
            title={t("reader.read_selection", "Read Selection")}
          >
            <Volume2 className="h-3.5 w-3.5 text-sky-400 pointer-events-none" />
            <span className="pointer-events-none">{t("reader.read_selection", "Read Selection")}</span>
          </button>
          <button
            type="button"
            data-reader-toolbar="true"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onReadFromHere();
            }}
            className="btn btn-ghost btn-xs gap-1.5 text-xs font-medium text-slate-100 hover:bg-slate-800 hover:text-white"
            title={t("reader.read_from_here", "Read From Here")}
          >
            <Play className="h-3.5 w-3.5 text-emerald-400 pointer-events-none" />
            <span className="pointer-events-none">{t("reader.read_from_here", "Read From Here")}</span>
          </button>
        </>
      )}
      <div className="mx-1 h-4 w-px bg-slate-700 pointer-events-none" />

      <div data-reader-toolbar="true" className="flex items-center gap-1.5 px-1">
        <button
          type="button"
          data-reader-toolbar="true"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onHighlight("#fef08a");
          }}
          className="h-4 w-4 rounded-full bg-yellow-400 border border-yellow-200/50 shadow-sm transition-transform hover:scale-125"
          title="Highlight Yellow"
        />
        <button
          type="button"
          data-reader-toolbar="true"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onHighlight("#bbf7d0");
          }}
          className="h-4 w-4 rounded-full bg-emerald-400 border border-emerald-200/50 shadow-sm transition-transform hover:scale-125"
          title="Highlight Green"
        />
        <button
          type="button"
          data-reader-toolbar="true"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onHighlight("#bfdbfe");
          }}
          className="h-4 w-4 rounded-full bg-sky-400 border border-sky-200/50 shadow-sm transition-transform hover:scale-125"
          title="Highlight Blue"
        />
        <button
          type="button"
          data-reader-toolbar="true"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onHighlight("#e9d5ff");
          }}
          className="h-4 w-4 rounded-full bg-purple-400 border border-purple-200/50 shadow-sm transition-transform hover:scale-125"
          title="Highlight Purple"
        />
      </div>
    </div>
  );
};
