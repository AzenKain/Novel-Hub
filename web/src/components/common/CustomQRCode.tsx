import React from "react";
import { QRCodeSVG } from "qrcode.react";
import { usePublicSettings } from "@/hooks/useSettings";
import { getMediaUrl } from "@/config/api";

interface CustomQRCodeProps {
  value: string;
  size?: number;
  label?: string;
  className?: string;
}

export const CustomQRCode: React.FC<CustomQRCodeProps> = ({
  value,
  size = 180,
  label,
  className = "",
}) => {
  const publicSettings = usePublicSettings();
  const siteTitle = publicSettings?.site?.title || "NovelHub";
  const rawLogo = publicSettings?.site?.logo || publicSettings?.site?.favicon;
  const siteLogo = rawLogo ? getMediaUrl(rawLogo) : null;

  const getInitials = (name: string) => {
    const words = name.trim().split(/\s+/);
    if (words.length >= 2) {
      return (words[0][0] + words[1][0]).toUpperCase();
    }
    return name.slice(0, 2).toUpperCase();
  };

  const badgeSize = Math.max(36, Math.floor(size * 0.22));

  return (
    <div className={`flex flex-col items-center justify-center p-4 bg-base-200/50 rounded-2xl border border-base-300 backdrop-blur-sm space-y-3 transition-all ${className}`}>
      <div className="relative grid place-items-center rounded-2xl bg-white p-3 shadow-md border border-base-300/50">
        <QRCodeSVG
          value={value}
          size={size}
          level="H"
          bgColor="#ffffff"
          fgColor="#0f172a"
        />
        <div
          className="absolute grid place-items-center rounded-xl border-2 border-white bg-primary text-primary-content font-black shadow-md overflow-hidden"
          style={{ width: `${badgeSize}px`, height: `${badgeSize}px` }}
        >
          {siteLogo ? (
            <img
              src={siteLogo}
              alt={siteTitle}
              className="w-full h-full object-contain p-0.5 bg-white rounded-lg"
            />
          ) : (
            <span className="text-xs tracking-tighter uppercase">{getInitials(siteTitle)}</span>
          )}
        </div>
      </div>
      {label && (
        <p className="text-xs text-base-content/70 font-mono text-center break-all max-w-full px-2 selection:bg-primary/20">
          {label}
        </p>
      )}
    </div>
  );
};
