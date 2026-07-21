import React, { useEffect, useState } from 'react';
import { ActivityCalendar } from 'react-activity-calendar';
import { getReadingHeatmap } from '@/hooks/useReadingStats';

export const ReadingHeatmap: React.FC = () => {
  const [data, setData] = useState<{ date: string; count: number; level: number }[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const heatmapData = await getReadingHeatmap();
        if (!heatmapData) return;
        
        const formattedData = Object.keys(heatmapData).map(dateStr => {
          const stats = heatmapData[dateStr];
          const words = stats.words || 0;
          let level = 0;
          if (words > 0) level = 1;
          if (words > 1000) level = 2;
          if (words > 5000) level = 3;
          if (words > 10000) level = 4;
          return {
            date: dateStr,
            count: words,
            level
          };
        });
        
        if (formattedData.length === 0) {
           const today = new Date().toISOString().split('T')[0];
           formattedData.push({ date: today, count: 0, level: 0 });
        }
        setData(formattedData);
      } catch (err) {
        console.error("Failed to load heatmap", err);
      }
    };
    fetchData();
  }, []);

  if (data.length === 0) return null;

  return (
    <div className="bg-base-200 p-4 rounded-xl">
      <h2 className="text-lg font-bold mb-4">Reading Activity (Words)</h2>
      <ActivityCalendar 
        data={data}
        theme={{
          light: ['#ebedf0', '#9be9a8', '#40c463', '#30a14e', '#216e39'],
          dark: ['#161b22', '#0e4429', '#006d32', '#26a641', '#39d353'],
        }}
      />
    </div>
  );
};
