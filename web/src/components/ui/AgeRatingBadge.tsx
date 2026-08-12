import React from "react";

interface AgeRatingBadgeProps {
  rating?: string;
  className?: string;
}

export const AgeRatingBadge: React.FC<AgeRatingBadgeProps> = ({
  rating = "G",
  className = "",
}) => {
  const getBadgeStyle = (r: string) => {
    switch (r.toUpperCase()) {
      case "G":
        return "badge-success text-success-content";
      case "PG":
        return "badge-info text-info-content";
      case "PG-13":
        return "badge-warning text-warning-content";
      case "R17+":
      case "R18+":
        return "badge-error text-error-content font-bold";
      default:
        return "badge-neutral";
    }
  };

  return (
    <span className={`badge badge-sm uppercase text-[10px] tracking-wider font-semibold ${getBadgeStyle(rating)} ${className}`}>
      {rating}
    </span>
  );
};
