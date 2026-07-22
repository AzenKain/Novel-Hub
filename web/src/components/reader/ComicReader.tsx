import React from "react";

interface ComicReaderProps {
  htmlContent: string;
  onContentClick?: (event: React.MouseEvent<HTMLDivElement>) => void;
}

export const ComicReader: React.FC<ComicReaderProps> = React.memo(({
  htmlContent,
  onContentClick,
}) => {
  if (!htmlContent) return null;

  return (
    <div
      className="webtoon-reader-container flex flex-col items-center w-full max-w-4xl mx-auto py-4 space-y-0"
      onClick={onContentClick}
      dangerouslySetInnerHTML={{ __html: htmlContent }}
    />
  );
});
