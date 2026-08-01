import React from "react";

type InfoLineProps = {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
};

export const InfoLine: React.FC<InfoLineProps> = ({ icon, label, value }) => (
  <div className="flex min-w-0 items-center gap-2 text-sm text-base-content/70">
    <span className="[&_svg]:h-4 [&_svg]:w-4 [&_svg]:shrink-0 opacity-70">{icon}</span>
    <span className="shrink-0 font-medium text-base-content/60">{label}:</span>
    <span className="min-w-0 truncate text-sm">{value}</span>
  </div>
);
