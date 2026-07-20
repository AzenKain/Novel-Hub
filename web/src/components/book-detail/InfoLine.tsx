import React from "react";

type InfoLineProps = {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
};

export const InfoLine: React.FC<InfoLineProps> = ({ icon, label, value }) => (
  <div className="flex min-w-0 items-center gap-2 text-base-content/65">
    <span className="[&_svg]:h-4 [&_svg]:w-4 [&_svg]:shrink-0">{icon}</span>
    <span className="shrink-0 font-bold text-base-content/45">{label}:</span>
    <span className="min-w-0 truncate">{value}</span>
  </div>
);
