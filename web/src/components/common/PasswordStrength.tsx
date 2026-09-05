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

  const getStrengthColor = () => {
    if (validReqCount <= 2) return "progress-error";
    if (validReqCount <= 4) return "progress-warning";
    return "progress-success";
  };

  const getStrengthLabel = () => {
    if (validReqCount <= 2) return t("auth.strength_weak");
    if (validReqCount <= 4) return t("auth.strength_fair");
    return t("auth.strength_strong");
  };

  return (
    <div className="flex flex-col gap-2 mt-2">
      <div className="flex justify-between items-center text-xs font-semibold">
        <span>{t("auth.password_strength")}</span>
        <span
          className={
            validReqCount <= 2
              ? "text-error"
              : validReqCount <= 4
                ? "text-warning"
                : "text-success"
          }
        >
          {getStrengthLabel()}
        </span>
      </div>
      <progress
        className={`progress w-full ${getStrengthColor()}`}
        value={validReqCount}
        max="5"
      ></progress>
      <div className="flex flex-col gap-1 mt-1">
        {passwordReqs.map((req, i) => (
          <div
            key={i}
            className={`text-xs flex items-center gap-1.5 ${
              req.valid ? "text-success font-medium" : "text-base-content/50"
            }`}
          >
            <span className="w-3 inline-block">{req.valid ? "✓" : "○"}</span>
            {req.label}
          </div>
        ))}
      </div>
    </div>
  );
}
