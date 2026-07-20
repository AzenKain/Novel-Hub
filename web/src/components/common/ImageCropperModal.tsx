import React, { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

type ImageCropperModalProps = {
  imageSrc: string;
  onCrop: (base64: string) => void;
  onCancel: () => void;
  cropSize?: number; // Size of the final cropped square image
};

export const ImageCropperModal: React.FC<ImageCropperModalProps> = ({
  imageSrc,
  onCrop,
  onCancel,
  cropSize = 200,
}) => {
  const { t } = useTranslation();
  const [zoom, setZoom] = useState(1);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [isLandscape, setIsLandscape] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const imgRef = useRef<HTMLImageElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    
    const handleWheel = (e: WheelEvent) => {
      e.preventDefault();
      setZoom((prevZoom) => {
        return Math.min(Math.max(1, prevZoom - e.deltaY * 0.005), 3);
      });
    };

    container.addEventListener("wheel", handleWheel, { passive: false });
    return () => {
      container.removeEventListener("wheel", handleWheel);
    };
  }, []);

  useEffect(() => {
    if (imageSrc) {
      const img = new Image();
      img.crossOrigin = "anonymous";
      img.src = imageSrc;
      img.onload = () => {
        setIsLandscape(img.naturalWidth > img.naturalHeight);
      };
    }
  }, [imageSrc]);

  const handlePointerDown = (e: React.PointerEvent<HTMLImageElement>) => {
    e.preventDefault();
    setIsDragging(true);
    setDragStart({ x: e.clientX - position.x, y: e.clientY - position.y });
    e.currentTarget.setPointerCapture(e.pointerId);
  };

  const handlePointerMove = (e: React.PointerEvent<HTMLImageElement>) => {
    if (!isDragging) return;
    setPosition({
      x: e.clientX - dragStart.x,
      y: e.clientY - dragStart.y
    });
  };

  const handlePointerUp = (e: React.PointerEvent<HTMLImageElement>) => {
    setIsDragging(false);
    e.currentTarget.releasePointerCapture(e.pointerId);
  };

  const handleCropApply = () => {
    const img = imgRef.current;
    if (!img) return;

    const naturalWidth = img.naturalWidth;
    const naturalHeight = img.naturalHeight;

    let drawWidth = cropSize;
    let drawHeight = cropSize;

    if (naturalWidth > naturalHeight) {
      drawHeight = cropSize;
      drawWidth = (naturalWidth / naturalHeight) * cropSize;
    } else {
      drawWidth = cropSize;
      drawHeight = (naturalHeight / naturalWidth) * cropSize;
    }

    const canvas = document.createElement("canvas");
    canvas.width = cropSize;
    canvas.height = cropSize;
    const ctx = canvas.getContext("2d");

    if (ctx) {
      ctx.clearRect(0, 0, cropSize, cropSize);
      ctx.save();
      ctx.translate(cropSize / 2, cropSize / 2);
      ctx.translate(position.x * (cropSize/200), position.y * (cropSize/200));
      ctx.scale(zoom, zoom);
      ctx.drawImage(img, -drawWidth / 2, -drawHeight / 2, drawWidth, drawHeight);
      ctx.restore();

      const base64 = canvas.toDataURL("image/png", 1.0);
      onCrop(base64);
    }
  };

  return (
    <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-base-100 p-6 rounded-2xl shadow-2xl max-w-md w-full flex flex-col items-center gap-4">
        <h4 className="font-bold text-lg">{t('common.crop_image', 'Crop & Adjust Photo')}</h4>
      
      <div 
        ref={containerRef}
        className="w-[200px] h-[200px] overflow-hidden relative border-2 border-primary shadow-lg bg-base-300 mx-auto select-none"
        style={{ borderRadius: cropSize > 200 ? '0.5rem' : '9999px' }} // Square for large logos, circle for avatars
      >
        <img
          ref={imgRef}
          src={imageSrc}
          crossOrigin="anonymous"
          alt="Crop preview"
          draggable={false}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onPointerCancel={handlePointerUp}
          className="absolute cursor-move select-none max-w-none origin-center"
          style={{
            width: isLandscape ? 'auto' : '200px',
            height: isLandscape ? '200px' : 'auto',
            top: '50%',
            left: '50%',
            transform: `translate(calc(-50% + ${position.x}px), calc(-50% + ${position.y}px)) scale(${zoom})`,
          }}
        />
      </div>
      
      <div className="w-full max-w-xs flex flex-col gap-2 mt-2">
        <div className="flex justify-between text-xs text-base-content/60 px-1">
          <span>{t('user.zoom', 'Zoom')}</span>
          <span>{Math.round(zoom * 100)}%</span>
        </div>
        <input
          type="range"
          min="1"
          max="3"
          step="0.05"
          value={zoom}
          onChange={(e) => setZoom(parseFloat(e.target.value))}
          className="range range-primary range-sm"
        />
        <p className="text-center text-xs text-base-content/50 mt-1">
          {t('user.crop_instructions', 'Drag image to reposition')}
        </p>
      </div>
      
        <div className="flex gap-2 w-full justify-end mt-4">
          <button
            type="button"
            className="btn btn-ghost"
            onClick={onCancel}
          >
            {t('common.cancel', 'Cancel')}
          </button>
          <button
            type="button"
            className="btn btn-primary"
            onClick={handleCropApply}
          >
            {t('common.apply', 'Apply')}
          </button>
        </div>
      </div>
    </div>
  );
};
