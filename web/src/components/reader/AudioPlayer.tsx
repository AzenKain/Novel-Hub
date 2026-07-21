import { useEffect, useRef, useState } from "react";
import { Play, Pause, SkipBack, SkipForward, Volume2, VolumeX } from "lucide-react";

interface AudioPlayerProps {
  rawUrl: string;
  initialTime?: number;
  onTimeUpdate?: (time: number) => void;
  title?: string;
  author?: string;
  coverUrl?: string;
}

export function AudioPlayer({
  rawUrl,
  initialTime = 0,
  onTimeUpdate,
  title = "Audiobook",
  author = "Unknown Author",
  coverUrl,
}: AudioPlayerProps) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(initialTime);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);
  const [isMuted, setIsMuted] = useState(false);

  useEffect(() => {
    if (audioRef.current && initialTime > 0) {
      audioRef.current.currentTime = initialTime;
    }
  }, []);

  const togglePlay = () => {
    if (!audioRef.current) return;
    if (isPlaying) {
      audioRef.current.pause();
    } else {
      audioRef.current.play().catch(console.error);
    }
    setIsPlaying(!isPlaying);
  };

  const skip = (seconds: number) => {
    if (!audioRef.current) return;
    audioRef.current.currentTime += seconds;
  };

  const handleTimeUpdate = () => {
    if (!audioRef.current) return;
    const time = audioRef.current.currentTime;
    setCurrentTime(time);
    if (onTimeUpdate) {
      onTimeUpdate(time);
    }
  };

  const handleLoadedMetadata = () => {
    if (!audioRef.current) return;
    setDuration(audioRef.current.duration);
  };

  const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const time = parseFloat(e.target.value);
    setCurrentTime(time);
    if (audioRef.current) {
      audioRef.current.currentTime = time;
    }
  };

  const handleVolumeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = parseFloat(e.target.value);
    setVolume(val);
    if (audioRef.current) {
      audioRef.current.volume = val;
    }
    setIsMuted(val === 0);
  };

  const toggleMute = () => {
    if (!audioRef.current) return;
    const newMuted = !isMuted;
    setIsMuted(newMuted);
    audioRef.current.muted = newMuted;
    if (newMuted) {
      setVolume(0);
    } else {
      setVolume(1);
      audioRef.current.volume = 1;
    }
  };

  const formatTime = (time: number) => {
    if (isNaN(time)) return "00:00";
    const minutes = Math.floor(time / 60);
    const seconds = Math.floor(time % 60);
    return `${minutes.toString().padStart(2, "0")}:${seconds.toString().padStart(2, "0")}`;
  };

  return (
    <div className="flex flex-col items-center justify-center h-full w-full bg-base-100 p-6">
      <audio
        ref={audioRef}
        src={rawUrl}
        onTimeUpdate={handleTimeUpdate}
        onLoadedMetadata={handleLoadedMetadata}
        onEnded={() => setIsPlaying(false)}
        preload="metadata"
      />

      <div className="card w-full max-w-lg bg-base-200 shadow-xl p-6">
        {coverUrl && (
          <figure className="mb-6">
            <img src={coverUrl} alt="Cover" className="w-64 h-64 object-cover rounded-xl shadow-lg" />
          </figure>
        )}
        <div className="text-center mb-8">
          <h2 className="text-2xl font-bold line-clamp-2">{title}</h2>
          <p className="text-base-content/70 mt-2">{author}</p>
        </div>

        <div className="w-full flex items-center gap-4 text-sm font-medium mb-6">
          <span>{formatTime(currentTime)}</span>
          <input
            type="range"
            min={0}
            max={duration || 100}
            value={currentTime}
            onChange={handleSeek}
            className="range range-primary range-sm flex-1"
          />
          <span>{formatTime(duration)}</span>
        </div>

        <div className="flex items-center justify-center gap-6 mb-6">
          <button className="btn btn-circle btn-ghost" onClick={() => skip(-15)} aria-label="Skip Back 15s">
            <SkipBack size={24} />
          </button>
          <button className="btn btn-circle btn-primary btn-lg" onClick={togglePlay} aria-label="Play/Pause">
            {isPlaying ? <Pause size={32} /> : <Play size={32} className="ml-1" />}
          </button>
          <button className="btn btn-circle btn-ghost" onClick={() => skip(15)} aria-label="Skip Forward 15s">
            <SkipForward size={24} />
          </button>
        </div>

        <div className="flex items-center gap-4 px-4">
          <button className="btn btn-circle btn-sm btn-ghost" onClick={toggleMute}>
            {isMuted || volume === 0 ? <VolumeX size={18} /> : <Volume2 size={18} />}
          </button>
          <input
            type="range"
            min={0}
            max={1}
            step={0.01}
            value={volume}
            onChange={handleVolumeChange}
            className="range range-xs flex-1"
          />
        </div>
      </div>
    </div>
  );
}
