import { Check, Circle } from "lucide-react";
import React from "react";
import { useTranslation } from "react-i18next";

interface PasswordStrengthProps {
  password?: string;
}

export function PasswordStrength({ password = "" }: PasswordStrengthProps) {
  const { t } = useTranslation();
  if (password.length === 0) return null;

  const passwordReqs = [
    { label: t("auth.req_length"), valid: password.length >= 8 },
    { label: t("auth.req_upper"), valid: /[A-Z]/.test(password) },
    { label: t("auth.req_lower"), valid: /[a-z]/.test(password) },
    { label: t("auth.req_number"), valid: /\d/.test(password) },
    { label: t("auth.req_special"), valid: /[^A-Za-z0-9]/.test(password) },
  ];

  const validReqCount = passwordReqs.filter((r) => r.valid).length;

  const getStrengthBadgeClass = () => {
    if (validReqCount <= 2) return "bg-error/20 text-error";
    if (validReqCount <= 4) return "bg-warning/20 text-warning";
    return "bg-success/20 text-success";
  };

  const getStrengthLabel = () => {
    if (validReqCount <= 2) return t("auth.strength_weak");
    if (validReqCount <= 4) return t("auth.strength_fair");
    return t("auth.strength_strong");
  };

  return (
    <div className="flex flex-col gap-2.5 p-3 rounded-xl bg-base-200/50 border border-base-300/70 mt-2 transition-all">
      <div className="flex justify-between items-center text-xs">
        <span className="font-semibold text-base-content/90">
          {t("auth.password_strength")}
        </span>
        <span
          className={`badge badge-sm font-bold border-0 ${getStrengthBadgeClass()}`}
        >
          {getStrengthLabel()}
        </span>
      </div>

      {/* 5-segment strength indicator */}
      <div className="grid grid-cols-5 gap-1.5 h-1.5 w-full">
        {[1, 2, 3, 4, 5].map((step) => {
          const isFilled = validReqCount >= step;
          let barColor = "bg-base-300 dark:bg-base-100/60";
          if (isFilled) {
            if (validReqCount <= 2) barColor = "bg-error";
            else if (validReqCount <= 4) barColor = "bg-warning";
            else barColor = "bg-success";
          }
          return (
            <div
              key={step}
              className={`h-full rounded-full transition-all duration-300 ${barColor}`}
            />
          );
        })}
      </div>

      {/* Requirements checklist */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-3 gap-y-1.5 mt-0.5">
        {passwordReqs.map((req, i) => (
          <div
            key={i}
            className={`text-xs flex items-center gap-1.5 transition-colors ${
              req.valid
                ? "text-success font-medium"
                : "text-base-content/85 font-normal"
            }`}
          >
            {req.valid ? (
              <Check className="w-3.5 h-3.5 text-success shrink-0 stroke-[2.5]" />
            ) : (
              <Circle className="w-3 h-3 text-base-content/40 shrink-0 stroke-2" />
            )}
            <span>{req.label}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
