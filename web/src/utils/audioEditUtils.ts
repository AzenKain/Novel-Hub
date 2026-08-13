export function getPeaks(buffer: AudioBuffer, width: number): number[] {
  const channelData = buffer.getChannelData(0);
  const step = Math.floor(channelData.length / width);
  const peaks: number[] = [];
  for (let i = 0; i < width; i++) {
    let min = 1.0;
    let max = -1.0;
    for (let j = 0; j < step; j++) {
      const idx = i * step + j;
      if (idx >= channelData.length) break;
      const datum = channelData[idx];
      if (datum < min) min = datum;
      if (datum > max) max = datum;
    }
    peaks.push(max - min);
  }
  return peaks;
}

export function formatTime(secs: number): string {
  const m = Math.floor(secs / 60);
  const s = Math.floor(secs % 60);
  const ms = Math.floor((secs % 1) * 10);
  return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}.${ms}`;
}

export interface FileRange<T> {
  file: T;
  start: number;
  end: number;
  duration: number;
}

export function getActiveRange<T>(time: number, ranges: FileRange<T>[]): FileRange<T> | undefined {
  const match = ranges.find((r, idx) => {
    if (idx === ranges.length - 1) {
      return time >= r.start && time <= r.end;
    }
    return time >= r.start && time < r.end;
  });
  return match || ranges[0];
}