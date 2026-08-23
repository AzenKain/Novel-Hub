import { AudioLines, X, Play, Pause, ChevronUp, ChevronDown, Volume2, Scissors, Trash2, ZoomIn, ZoomOut, GripVertical } from "lucide-react";
import React, { useState, useEffect, useMemo, useRef, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

import { useMergeAudiobookMutation } from "@/hooks";
import { bookService } from "@/services";
import type { BookFile } from "@/types";
import { formatTime, getPeaks } from "@/utils/audioEditUtils";

const MERGEABLE_FORMATS = new Set(["m4a", "m4b", "mp3", "flac", "ogg", "wav", "aac"]);

const TRACK_COLORS = [
  { bg: "bg-blue-500/5", border: "border-blue-500/40" },
  { bg: "bg-emerald-500/5", border: "border-emerald-500/40" },
  { bg: "bg-purple-500/5", border: "border-purple-500/40" },
  { bg: "bg-orange-500/5", border: "border-orange-500/40" },
  { bg: "bg-pink-500/5", border: "border-pink-500/40" },
  { bg: "bg-indigo-500/5", border: "border-indigo-500/40" },
  { bg: "bg-teal-500/5", border: "border-teal-500/40" },
];

type MergeAudiobookModalProps = {
  open: boolean;
  book_id: string;
  title: string;
  files: BookFile[];
  onClose: () => void;
};

const formatBytes = (n: number) => {
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
};

interface TimelineSegment {
  id: string;
  fileId: string;
  fileName: string;
  sizeBytes: number;
  startOffset: number;
  endOffset: number;
  duration: number;
  gain: number; // 1.0 = original volume
}

/* ── Waveform sub-component ────────────────────────────────────────── */
const TimelineTrackWaveform: React.FC<{
  peaks: number[] | undefined;
  isActive: boolean;
  localPlayheadPct: number;
}> = ({ peaks, isActive, localPlayheadPct }) => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  const draw = useCallback(() => {
    if (!canvasRef.current) return;
    const canvas = canvasRef.current;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    // Use a minimum of 1 to prevent layout collapse
    const width = canvas.clientWidth || 1;
    const height = canvas.clientHeight || 1;
    
    // Only update internal canvas size when actual client width/height changes
    if (canvas.width !== width || canvas.height !== height) {
      canvas.width = width;
      canvas.height = height;
    }

    ctx.clearRect(0, 0, width, height);

    if (!peaks) {
      ctx.fillStyle = "rgba(0, 0, 0, 0.03)";
      ctx.fillRect(0, 0, width, height);
      ctx.fillStyle = "#94a3b8";
      ctx.font = "9px sans-serif";
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText("Decoding...", width / 2, height / 2);
      return;
    }

    ctx.lineWidth = 1.5;
    const step = 3;

    // 1. Played portion (blue)
    const playedWidth = localPlayheadPct * width;
    if (playedWidth > 0) {
      ctx.beginPath();
      ctx.strokeStyle = "#3b82f6";
      for (let x = 0; x < playedWidth; x += step) {
        const peakIndex = Math.floor((x / width) * peaks.length);
        const amp = peaks[peakIndex] || 0;
        const h = Math.max(3, amp * height * 0.85);
        const y = (height - h) / 2;
        ctx.moveTo(x, y);
        ctx.lineTo(x, y + h);
      }
      ctx.stroke();
    }

    // 2. Unplayed portion (gray)
    if (playedWidth < width) {
      ctx.beginPath();
      ctx.strokeStyle = "#cbd5e1";
      const startX = Math.ceil(playedWidth / step) * step;
      for (let x = startX; x < width; x += step) {
        const peakIndex = Math.floor((x / width) * peaks.length);
        const amp = peaks[peakIndex] || 0;
        const h = Math.max(3, amp * height * 0.85);
        const y = (height - h) / 2;
        ctx.moveTo(x, y);
        ctx.lineTo(x, y + h);
      }
      ctx.stroke();
    }
  }, [peaks, isActive, localPlayheadPct]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    draw();

    const resizeObserver = new ResizeObserver(() => {
      draw();
    });
    resizeObserver.observe(canvas);

    return () => {
      resizeObserver.disconnect();
    };
  }, [draw]);

  return <canvas ref={canvasRef} className="absolute inset-0 w-full h-full rounded-lg" />;
};

/* ── Volume envelope overlay (CapCut-style drag up/down) ───────────── */
const VolumeEnvelope: React.FC<{
  gain: number;
  onGainChange: (g: number) => void;
}> = ({ gain, onGainChange }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [dragging, setDragging] = useState(false);

  const pct = Math.min(gain / 2, 1); // 0..2 → 0..1 (200% max)
  const lineBottom = `${pct * 100}%`;

  const calcGainFromY = useCallback((clientY: number) => {
    if (!containerRef.current) return gain;
    const rect = containerRef.current.getBoundingClientRect();
    const relY = rect.bottom - clientY; // distance from bottom
    const pctNew = Math.max(0, Math.min(relY / rect.height, 1));
    return Math.round(pctNew * 200) / 100; // 0..2
  }, [gain]);

  useEffect(() => {
    if (!dragging) return;
    const onMove = (e: PointerEvent) => {
      onGainChange(calcGainFromY(e.clientY));
    };
    const onUp = () => setDragging(false);
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    return () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
  }, [dragging, calcGainFromY, onGainChange]);

  return (
    <div
      ref={containerRef}
      className="absolute inset-0 z-10 pointer-events-none"
    >
      {/* Volume line – always visible */}
      <div
        className="absolute left-0 right-0 pointer-events-auto cursor-ns-resize group/vol"
        style={{ bottom: lineBottom, transform: "translateY(50%)" }}
        onPointerDown={(e) => {
          e.stopPropagation();
          e.preventDefault();
          setDragging(true);
        }}
      >
        {/* The draggable horizontal line */}
        <div className="h-0.5 bg-amber-400/70 group-hover/vol:bg-amber-400 group-hover/vol:h-0.75 transition-all relative">
          {/* Volume badge in the center */}
          <div className="absolute left-1/2 -translate-x-1/2 -top-3.5 bg-amber-500/90 text-white text-[8px] font-bold px-1.5 py-0.5 rounded-sm shadow-sm whitespace-nowrap opacity-0 group-hover/vol:opacity-100 transition-opacity select-none">
            {Math.round(gain * 100)}%
          </div>
          {/* Small grab handles at edges */}
          <div className="absolute left-1 top-1/2 -translate-y-1/2 w-2 h-2 rounded-full bg-amber-400 border border-white shadow-sm opacity-0 group-hover/vol:opacity-100 transition-opacity" />
          <div className="absolute right-1 top-1/2 -translate-y-1/2 w-2 h-2 rounded-full bg-amber-400 border border-white shadow-sm opacity-0 group-hover/vol:opacity-100 transition-opacity" />
        </div>
      </div>
      {/* Filled region below the line */}
      <div
        className="absolute left-0 right-0 bottom-0 bg-amber-400/8 pointer-events-none"
        style={{ height: lineBottom }}
      />
    </div>
  );
};

/* ── Main Modal ────────────────────────────────────────────────────── */
export const MergeAudiobookModal: React.FC<MergeAudiobookModalProps> = ({ open, book_id, title, files, onClose }) => {
  const { t } = useTranslation();
  const merge = useMergeAudiobookMutation(book_id);
  const [mergedTitle, setMergedTitle] = useState(() =>
    t("audiobook.merged_title_default", "Merged - {{title}}", { title })
  );
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const [orderedFiles, setOrderedFiles] = useState<BookFile[]>(() =>
    files.filter((f) => MERGEABLE_FORMATS.has(f.format.toLowerCase()))
  );

  useEffect(() => {
    setSelected(new Set(orderedFiles.map(f => f.id)));
  }, [files]);

  const [decodedBuffers, setDecodedBuffers] = useState<Record<string, AudioBuffer>>({});
  const [cachedPeaks, setCachedPeaks] = useState<Record<string, number[]>>({});
  const [decodingStatus, setDecodingStatus] = useState<Record<string, string>>({});
  const [currentTime, setCurrentTime] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [segments, setSegments] = useState<TimelineSegment[]>([]);
  const [zoomLevel, setZoomLevel] = useState(1);

  const audioRef = useRef<HTMLAudioElement | null>(null);
  const pendingSeekTime = useRef<number | null>(null);
  const timelineRef = useRef<HTMLDivElement | null>(null);
  const [isDraggingPlayhead, setIsDraggingPlayhead] = useState(false);

  // --- Pointer-based drag reorder state ---
  const dragState = useRef<{
    sourceIndex: number;
    active: boolean;
    startX: number;
    startY: number;
    didMove: boolean;
  } | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  const handleTrackPointerDown = useCallback((e: React.PointerEvent, index: number) => {
    if (e.button !== 0) return;
    dragState.current = { sourceIndex: index, active: false, startX: e.clientX, startY: e.clientY, didMove: false };
  }, []);

  useEffect(() => {
    const handlePointerMove = (e: PointerEvent) => {
      const ds = dragState.current;
      if (!ds) return;
      
      if (!ds.active) {
        const dx = Math.abs(e.clientX - ds.startX);
        const dy = Math.abs(e.clientY - ds.startY);
        if (dx < 8 && dy < 8) return;
        // Only activate horizontal drag (not vertical volume drag)
        if (dy > dx) { dragState.current = null; return; }
        ds.active = true;
        ds.didMove = true;
      }

      if (!timelineRef.current) return;
      const children = Array.from(timelineRef.current.children).filter(
        (c) => (c as HTMLElement).dataset.trackIndex !== undefined
      );
      if (children.length === 0) return;

      const firstRect = (children[0] as HTMLElement).getBoundingClientRect();
      const lastRect = (children[children.length - 1] as HTMLElement).getBoundingClientRect();

      if (e.clientX < firstRect.left) {
        setDragOverIndex(0);
        return;
      }
      if (e.clientX >= lastRect.right) {
        setDragOverIndex(children.length - 1);
        return;
      }
      
      for (let i = 0; i < children.length; i++) {
        const rect = (children[i] as HTMLElement).getBoundingClientRect();
        if (e.clientX >= rect.left && e.clientX < rect.right) {
          setDragOverIndex(i);
          return;
        }
      }
    };

    const handlePointerUp = () => {
      const ds = dragState.current;
      if (ds && ds.active && dragOverIndex !== null && ds.sourceIndex !== dragOverIndex) {
        setSegments((prev) => {
          const next = [...prev];
          const [dragged] = next.splice(ds.sourceIndex, 1);
          next.splice(dragOverIndex, 0, dragged);
          return next;
        });
      }
      dragState.current = null;
      setDragOverIndex(null);
    };

    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerup", handlePointerUp);
    return () => {
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", handlePointerUp);
    };
  }, [dragOverIndex]);



  // Background audio decoding
  useEffect(() => {
    let active = true;
    orderedFiles.forEach(async (f) => {
      if (decodedBuffers[f.id] || decodingStatus[f.id] === "loading") return;

      setDecodingStatus((prev) => ({ ...prev, [f.id]: "loading" }));
      try {
        const url = bookService.getDownloadUrl(f.book_id, f.id);
        const response = await fetch(url);
        if (!response.ok) throw new Error("Fetch failed");
        const arrayBuffer = await response.arrayBuffer();

        if (!active) return;
        const ctx = new (window.AudioContext || (window as any).webkitAudioContext)();
        const decoded = await ctx.decodeAudioData(arrayBuffer);

        if (!active) return;
        setDecodedBuffers((prev) => ({ ...prev, [f.id]: decoded }));
        
        const peaks = getPeaks(decoded, 300);
        setCachedPeaks((prev) => ({ ...prev, [f.id]: peaks }));
        setDecodingStatus((prev) => ({ ...prev, [f.id]: "ready" }));
      } catch (err) {
        console.error(err);
        if (active) {
          setDecodingStatus((prev) => ({ ...prev, [f.id]: "error" }));
        }
      }
    });
    return () => { active = false; };
  }, [orderedFiles]);

  const activeFiles = useMemo(() => {
    return orderedFiles.filter((f) => selected.has(f.id));
  }, [orderedFiles, selected]);

  useEffect(() => {
    const nextSegments = activeFiles.map((f) => {
      const buffer = decodedBuffers[f.id];
      const dur = buffer ? buffer.duration : 0;
      return {
        id: `${f.id}-segment-${Math.random()}`,
        fileId: f.id,
        fileName: f.path.split("/").pop() || "track",
        sizeBytes: f.size_bytes,
        startOffset: 0,
        endOffset: dur,
        duration: dur,
        gain: 1.0,
      };
    });
    setSegments(nextSegments);
  }, [orderedFiles, selected]);

  useEffect(() => {
    setSegments((prev) =>
      prev.map((seg) => {
        if (seg.duration === 0) {
          const buffer = decodedBuffers[seg.fileId];
          if (buffer) return { ...seg, endOffset: buffer.duration, duration: buffer.duration };
        }
        return seg;
      })
    );
  }, [decodedBuffers]);

  const segmentRanges = useMemo(() => {
    let start = 0;
    return segments.map((seg) => {
      const range = { segment: seg, start, end: start + seg.duration };
      start += seg.duration;
      return range;
    });
  }, [segments]);

  const totalDuration = useMemo(() => {
    return segmentRanges.reduce((acc, r) => acc + r.segment.duration, 0);
  }, [segmentRanges]);

  const handleTimeUpdate = () => {
    if (!audioRef.current || !playing) return;
    const activeRange = segmentRanges.find((r) => currentTime >= r.start && currentTime <= r.end) || segmentRanges[0];
    if (!activeRange) return;

    const localPlayhead = audioRef.current.currentTime - activeRange.segment.startOffset;
    const newGlobalTime = activeRange.start + localPlayhead;
    setCurrentTime(newGlobalTime);

    const fileOffset = audioRef.current.currentTime;
    if (fileOffset >= activeRange.segment.endOffset - 0.08) {
      const nextIndex = segmentRanges.indexOf(activeRange) + 1;
      if (nextIndex < segmentRanges.length) {
        const nextRange = segmentRanges[nextIndex];
        const nextSrc = bookService.getDownloadUrl(book_id, nextRange.segment.fileId);
        
        const isSameSrc = audioRef.current.src.endsWith(nextSrc);
        if (!isSameSrc) {
          pendingSeekTime.current = nextRange.segment.startOffset;
          audioRef.current.src = nextSrc;
        } else {
          audioRef.current.currentTime = nextRange.segment.startOffset;
        }
        setCurrentTime(nextRange.start);
        audioRef.current.play().catch(console.error);
      } else {
        setPlaying(false);
        setCurrentTime(0);
      }
    }
  };

  const handleScrub = (newGlobalTime: number) => {
    const clampedTime = Math.max(0, Math.min(newGlobalTime, totalDuration));
    setCurrentTime(clampedTime);
    if (audioRef.current) {
      const activeRange = segmentRanges.find((r) => clampedTime >= r.start && clampedTime <= r.end) || segmentRanges[0];
      if (activeRange && activeRange.segment.duration > 0) {
        const newSrc = bookService.getDownloadUrl(book_id, activeRange.segment.fileId);
        const localOffset = clampedTime - activeRange.start;
        const fileOffset = activeRange.segment.startOffset + localOffset;

        const isSameSrc = audioRef.current.src.endsWith(newSrc);
        if (!isSameSrc) {
          pendingSeekTime.current = fileOffset;
          audioRef.current.src = newSrc;
        } else {
          audioRef.current.currentTime = fileOffset;
          if (playing) {
            audioRef.current.play().catch(console.error);
          }
        }
      }
    }
  };

  const handlePlayheadMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDraggingPlayhead(true);
  };

  useEffect(() => {
    if (!isDraggingPlayhead) return;

    const handleMouseMove = (e: MouseEvent) => {
      if (!timelineRef.current || totalDuration <= 0) return;
      const rect = timelineRef.current.getBoundingClientRect();
      const scrollLeft = timelineRef.current.parentElement?.scrollLeft || 0;
      const x = e.clientX - rect.left + scrollLeft;
      const pct = Math.max(0, Math.min(x / timelineRef.current.scrollWidth, 1));
      handleScrub(pct * totalDuration);
    };

    const handleMouseUp = () => setIsDraggingPlayhead(false);

    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mouseup", handleMouseUp);
    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isDraggingPlayhead, totalDuration]);

  const togglePlay = () => {
    if (!audioRef.current) return;
    if (playing) {
      audioRef.current.pause();
      setPlaying(false);
    } else {
      const activeRange = segmentRanges.find((r) => currentTime >= r.start && currentTime <= r.end) || segmentRanges[0];
      if (activeRange) {
        const newSrc = bookService.getDownloadUrl(book_id, activeRange.segment.fileId);
        const targetTime = (currentTime - activeRange.start) + activeRange.segment.startOffset;
        const isSameSrc = audioRef.current.src.endsWith(newSrc);
        
        setPlaying(true);
        
        if (!isSameSrc) {
          pendingSeekTime.current = targetTime;
          audioRef.current.src = newSrc;
        } else {
          audioRef.current.currentTime = targetTime;
          audioRef.current.play().catch((err) => {
            console.error(err);
            setPlaying(false);
          });
        }
      }
    }
  };

  const handleSplitAtPlayhead = () => {
    const activeRange = segmentRanges.find((r) => currentTime >= r.start && currentTime < r.end) || segmentRanges[0];
    if (!activeRange) return;

    const localSplitTime = currentTime - activeRange.start;
    if (localSplitTime <= 0.5 || localSplitTime >= activeRange.segment.duration - 0.5) {
      toast.error(t("audiobook.cannot_split_close", "Cannot split too close to track boundaries (minimum 0.5s from edge)"));
      return;
    }

    const splitOffsetInFile = activeRange.segment.startOffset + localSplitTime;
    const baseName = activeRange.segment.fileName.substring(0, activeRange.segment.fileName.lastIndexOf(".")) || activeRange.segment.fileName;

    const seg1: TimelineSegment = {
      id: `${activeRange.segment.fileId}-part1-${Date.now()}`,
      fileId: activeRange.segment.fileId,
      fileName: `${baseName}_part1`,
      sizeBytes: Math.floor(activeRange.segment.sizeBytes * (localSplitTime / activeRange.segment.duration)),
      startOffset: activeRange.segment.startOffset,
      endOffset: splitOffsetInFile,
      duration: localSplitTime,
      gain: activeRange.segment.gain,
    };

    const seg2: TimelineSegment = {
      id: `${activeRange.segment.fileId}-part2-${Date.now()}`,
      fileId: activeRange.segment.fileId,
      fileName: `${baseName}_part2`,
      sizeBytes: Math.floor(activeRange.segment.sizeBytes * ((activeRange.segment.duration - localSplitTime) / activeRange.segment.duration)),
      startOffset: splitOffsetInFile,
      endOffset: activeRange.segment.endOffset,
      duration: activeRange.segment.duration - localSplitTime,
      gain: activeRange.segment.gain,
    };

    setSegments((prev) => {
      const next: TimelineSegment[] = [];
      prev.forEach((seg) => {
        if (seg.id === activeRange.segment.id) {
          next.push(seg1, seg2);
        } else {
          next.push(seg);
        }
      });
      return next;
    });

    toast.success(t("audiobook.timeline_split_success", "Timeline split locally!"));
  };

  const updateSegmentGain = useCallback((segId: string, newGain: number) => {
    setSegments((prev) =>
      prev.map((seg) => (seg.id === segId ? { ...seg, gain: newGain } : seg))
    );
  }, []);

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const moveUp = (index: number) => {
    if (index === 0) return;
    setOrderedFiles((prev) => {
      const next = [...prev];
      [next[index - 1], next[index]] = [next[index], next[index - 1]];
      return next;
    });
  };

  const moveDown = (index: number) => {
    if (index === orderedFiles.length - 1) return;
    setOrderedFiles((prev) => {
      const next = [...prev];
      [next[index], next[index + 1]] = [next[index + 1], next[index]];
      return next;
    });
  };

  const removeTrack = (id: string) => {
    setOrderedFiles((prev) => prev.filter(f => f.id !== id));
    setSelected((prev) => { const next = new Set(prev); next.delete(id); return next; });
  };

  const removeSegment = (segId: string) => {
    setSegments((prev) => prev.filter((s) => s.id !== segId));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (segments.length < 2) {
      toast.error(t("audiobook.merge_need_two", "Timeline must contain at least 2 segments to merge"));
      return;
    }
    if (segments.some((s) => s.duration <= 0)) {
      toast.error(t("audiobook.merge_not_ready", "Still decoding audio — wait until all segments have a duration"));
      return;
    }

    toast.info(t("audiobook.submitting_merge", "Submitting merge request to server..."));
    merge.mutate(
      {
        title: mergedTitle.trim() || title,
        segments: segments.map((s) => ({
          file_id: s.fileId,
          start_sec: s.startOffset,
          end_sec: s.endOffset,
          gain: s.gain,
        })),
      },
      {
        onSuccess: () => {
          toast.success(t("audiobook.merge_queued", "Merge job started"));
          onClose();
        },
        onError: (err) => {
          toast.error(err.message || t("audiobook.merge_failed", "Failed to start merge"));
        },
      }
    );
  };

  if (!open) return null;

  return (
    <dialog className="modal modal-open">
      {/* Near full-screen modal */}
      <div className="modal-box max-w-[95vw] w-full max-h-[92vh] p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <AudioLines className="w-5 h-5 text-primary" />
            {t("audiobook.merge_into", "Merge & Timeline Editor")}
          </h3>
          <button className="btn btn-square btn-sm btn-ghost" onClick={onClose} aria-label={t("common.close")}>
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
            {/* ═══ Audio Timeline ═══ */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <label className="text-xs font-bold uppercase opacity-60 block">{t("audiobook.timeline", "Audio Timeline")}</label>
                {/* Zoom controls */}
                <div className="flex items-center gap-2 text-xs">
                  <button type="button" onClick={() => setZoomLevel((z) => Math.max(1, z - 0.5))} className="btn btn-ghost btn-xs btn-circle" title={t("audiobook.zoom_out", "Zoom Out")}>
                    <ZoomOut className="w-3.5 h-3.5" />
                  </button>
                  <input
                    type="range"
                    min={1}
                    max={10}
                    step={0.1}
                    value={zoomLevel}
                    onChange={(e) => setZoomLevel(parseFloat(e.target.value))}
                    className="range range-primary range-xs w-24"
                  />
                  <button type="button" onClick={() => setZoomLevel((z) => Math.min(10, z + 0.5))} className="btn btn-ghost btn-xs btn-circle" title={t("audiobook.zoom_in", "Zoom In")}>
                    <ZoomIn className="w-3.5 h-3.5" />
                  </button>
                  <span className="font-mono opacity-60 w-10 text-right">{zoomLevel.toFixed(1)}x</span>
                </div>
              </div>
              
              <div
                className="relative border border-base-300 rounded-2xl bg-base-200/20 overflow-x-auto overflow-y-hidden shadow-xs"
              >
                <div 
                  ref={timelineRef}
                  className="flex items-stretch relative select-none cursor-pointer"
                  style={{
                    width: `${zoomLevel * 100}%`,
                    minWidth: "100%",
                    height: "180px",
                  }}
                  onClick={(e) => {
                    if (totalDuration <= 0 || isDraggingPlayhead) return;
                    if (dragState.current?.didMove) return;
                    const rect = e.currentTarget.getBoundingClientRect();
                    const x = e.clientX - rect.left;
                    const clickPct = x / e.currentTarget.offsetWidth;
                    handleScrub(clickPct * totalDuration);
                  }}
                >
                  {segmentRanges.map((range, index) => {
                    const peaks = cachedPeaks[range.segment.fileId];
                    const isActive = currentTime >= range.start && currentTime <= range.end;
                    const localPlayheadPct = isActive && range.segment.duration > 0 ? (currentTime - range.start) / range.segment.duration : 0;
                    const isDragSource = dragState.current?.active && dragState.current.sourceIndex === index;
                    
                    const color = TRACK_COLORS[index % TRACK_COLORS.length];
                    const isActiveClass = isActive ? "ring-2 ring-primary/50 shadow-md scale-[0.99] z-10" : "";
                    const isDragOver = dragOverIndex === index && dragState.current?.active;
                    const dragOverBorderClass = isDragOver && dragState.current
                      ? (dragState.current.sourceIndex < index 
                          ? "border-r-4 border-r-blue-500 bg-blue-500/20" 
                          : "border-l-4 border-l-blue-500 bg-blue-500/20")
                      : "";

                    return (
                      <div
                        key={range.segment.id}
                        data-track-index={index}
                        onPointerDown={(e) => handleTrackPointerDown(e, index)}
                        className={`h-full relative select-none p-1 shrink-0 ${
                          isDragSource ? "opacity-30" : ""
                        }`}
                        style={{
                          width: totalDuration > 0 ? `${(range.segment.duration / totalDuration) * 100}%` : "0%",
                          minWidth: "80px",
                          cursor: dragState.current?.active ? "grabbing" : "grab",
                        }}
                      >
                        {/* Inner card containing the visual track */}
                        <div
                          className={`flex flex-col h-full w-full rounded-xl border-2 transition-all ${color.bg} ${color.border} ${isActiveClass} ${dragOverBorderClass}`}
                        >
                          {/* Track header – compact */}
                          <div className="flex items-center justify-between text-[9px] font-bold text-base-content/60 px-2 pt-1.5 pb-0.5 gap-1">
                            <div className="flex items-center gap-0.5 truncate min-w-0">
                              <GripVertical className="w-2.5 h-2.5 shrink-0 opacity-40" />
                              <span className="truncate">{range.segment.fileName}</span>
                            </div>
                            <div className="flex items-center gap-1.5 shrink-0">
                              <Volume2 className="w-2.5 h-2.5 opacity-50" />
                              <span className="font-mono opacity-70">{Math.round(range.segment.gain * 100)}%</span>
                              <button
                                type="button"
                                onClick={(e) => { e.stopPropagation(); removeSegment(range.segment.id); }}
                                onPointerDown={(e) => e.stopPropagation()}
                                className="btn btn-ghost btn-circle btn-xs h-3.5 w-3.5 min-h-0 p-0 text-error/60 hover:text-error hover:bg-error/10"
                                title={t("audiobook.remove_segment", "Remove segment")}
                              >
                                <X className="w-2.5 h-2.5" />
                              </button>
                              <span className="font-mono opacity-80">{formatTime(range.segment.duration)}</span>
                            </div>
                          </div>

                          {/* Waveform + Volume envelope overlay */}
                          <div className="flex-1 relative px-1 pb-1">
                            <TimelineTrackWaveform
                              peaks={peaks}
                              isActive={isActive}
                              localPlayheadPct={localPlayheadPct}
                            />
                            {/* CapCut-style volume envelope: drag line up/down */}
                            <VolumeEnvelope
                              gain={range.segment.gain}
                              onGainChange={(g) => updateSegmentGain(range.segment.id, g)}
                            />
                          </div>
                        </div>
                      </div>
                    );
                  })}

                  {/* Absolute playhead overlay */}
                  {totalDuration > 0 && (
                    <div
                      onMouseDown={handlePlayheadMouseDown}
                      className="absolute top-0 bottom-0 w-5 -ml-2.5 z-30 cursor-ew-resize group/playhead pointer-events-auto"
                      style={{
                        left: `${(currentTime / totalDuration) * 100}%`,
                      }}
                    >
                      <div className="absolute -top-0.5 left-1/2 -translate-x-1/2 w-4 h-4 rounded-full bg-blue-500 border-2 border-white shadow-md group-hover/playhead:bg-blue-600 group-hover/playhead:scale-110 group-active/playhead:scale-95 transition-all" />
                      <div className="w-0.5 h-full bg-blue-500 mx-auto group-hover/playhead:w-1 group-hover/playhead:bg-blue-600 transition-all" />
                    </div>
                  )}
                </div>
              </div>

              {/* Time display & play controls */}
              <div className="flex items-center justify-between text-xs font-mono text-base-content/75 bg-base-200/50 p-2 rounded-xl">
                <div className="flex items-center gap-1.5">
                  <button
                    type="button"
                    onClick={togglePlay}
                    className="btn btn-primary btn-xs btn-circle text-white shadow-xs"
                    title={playing ? t("audiobook.pause", "Pause") : t("audiobook.play", "Play")}
                  >
                    {playing ? <Pause className="w-3.5 h-3.5" /> : <Play className="w-3.5 h-3.5" />}
                  </button>
                  <span className="ml-1">{t("audiobook.playhead", "Playhead")}: {formatTime(currentTime)}</span>
                </div>
                <div className="flex items-center gap-3">
                  {totalDuration > 0 && (
                    <button
                      type="button"
                      onClick={handleSplitAtPlayhead}
                      className="btn btn-error btn-xs gap-1 text-white shadow-xs"
                      title={t("audiobook.split_track_desc", "Split track at current playhead position")}
                    >
                      <Scissors className="w-3 h-3" />
                      {t("audiobook.split_track", "Split Track")}
                    </button>
                  )}
                  <span>{t("audiobook.total_duration", "Total Duration")}: {formatTime(totalDuration)}</span>
                </div>
              </div>

              {/* Range slider for scrubber */}
              {totalDuration > 0 && (
                <input
                  type="range"
                  min={0}
                  max={totalDuration}
                  step={0.01}
                  value={currentTime}
                  onChange={(e) => handleScrub(parseFloat(e.target.value))}
                  className="range range-primary range-xs mt-1"
                />
              )}
            </div>

            {/* Invisible Audio Element */}
            <audio
              ref={audioRef}
              onTimeUpdate={handleTimeUpdate}
              onEnded={() => setPlaying(false)}
              onLoadedMetadata={() => {
                if (audioRef.current && pendingSeekTime.current !== null) {
                  audioRef.current.currentTime = pendingSeekTime.current;
                  pendingSeekTime.current = null;
                }
              }}
              className="hidden"
            />

            {/* Merged settings */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="label label-text font-semibold">{t("audiobook.merged_title", "Merged title")}</label>
                <input
                  type="text"
                  value={mergedTitle}
                  onChange={(e) => setMergedTitle(e.target.value)}
                  className="input input-bordered w-full"
                />
              </div>
            </div>

            {/* Source files reorder & checklist */}
            <div>
              <label className="label label-text font-semibold">
                {t("audiobook.select_tracks", "Source files")}
                {orderedFiles.length > 0 && (
                  <span className="badge badge-ghost badge-sm">{selected.size}/{orderedFiles.length}</span>
                )}
              </label>
              {orderedFiles.length === 0 ? (
                <p className="text-sm opacity-60">{t("audiobook.no_mergeable_files", "No audio files on this book")}</p>
              ) : (
                <div className="max-h-52 overflow-y-auto space-y-1.5 rounded-box border border-base-300 p-2">
                  {orderedFiles.map((f, index) => {
                    return (
                      <div key={f.id} className="flex items-center justify-between gap-2 p-2 rounded-lg hover:bg-base-200 border border-base-200 bg-base-100/50">
                        <div className="flex items-center gap-1.5 flex-1 min-w-0">
                          <input
                            type="checkbox"
                            className="checkbox checkbox-sm checkbox-primary shrink-0 mr-1"
                            checked={selected.has(f.id)}
                            onChange={() => toggle(f.id)}
                          />
                          <span className="text-sm font-semibold truncate text-base-content" title={f.path.split("/").pop()}>
                            {f.path.split("/").pop()}
                          </span>
                        </div>
                        <div className="flex items-center gap-2 shrink-0">
                          <span className="text-[10px] opacity-50 uppercase font-mono">{formatBytes(f.size_bytes)}</span>
                          <div className="join border border-base-300">
                            <button
                              type="button"
                              onClick={() => moveUp(index)}
                              disabled={index === 0}
                              className="btn btn-ghost btn-xs join-item px-1 disabled:opacity-30"
                              title={t("common.move_up", "Move Up")}
                            >
                              <ChevronUp className="w-3.5 h-3.5" />
                            </button>
                            <button
                              type="button"
                              onClick={() => moveDown(index)}
                              disabled={index === orderedFiles.length - 1}
                              className="btn btn-ghost btn-xs join-item px-1 disabled:opacity-30"
                              title={t("common.move_down", "Move Down")}
                            >
                              <ChevronDown className="w-3.5 h-3.5" />
                            </button>
                          </div>
                          <button
                            type="button"
                            onClick={() => removeTrack(f.id)}
                            className="btn btn-ghost btn-xs btn-square text-error"
                            title={t("audiobook.remove_from_timeline", "Remove from timeline")}
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>

            <div className="modal-action">
              <button type="button" className="btn btn-ghost" onClick={onClose}>
                {t("common.cancel")}
              </button>
              <button type="submit" className="btn btn-primary min-w-30" disabled={merge.isPending || selected.size < 2}>
                {merge.isPending ? t("common.loading") : t("audiobook.start_merge", "Start merge")}
              </button>
            </div>
          </form>
      </div>
    </dialog>
  );
};
