import { useTestSmtpMutation, useUpdateAdminSettingsMutation } from "@/hooks";
import type { AdminSettings, SmtpTlsMode } from "@/types";
import { AlertTriangle, CheckCircle2, Mail, Save } from "lucide-react";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

const TLS_PORTS: Record<SmtpTlsMode, number> = {
  none: 25,
  starttls: 587,
  implicit_tls: 465,
};

const TLS_LABEL_KEYS: Record<SmtpTlsMode, string> = {
  none: "settings.smtp_tls_none",
  starttls: "settings.smtp_tls_starttls",
  implicit_tls: "settings.smtp_tls_implicit",
};

type Props = { settings?: AdminSettings };

export const SmtpSettingsTab: React.FC<Props> = ({ settings }) => {
  const { t } = useTranslation();
  const updateMutation = useUpdateAdminSettingsMutation();
  const testMutation = useTestSmtpMutation();

  const [form, setForm] = useState({
    enabled: false,
    host: "",
    port: 587,
    username: "",
    from_email: "",
    tls_mode: "starttls" as SmtpTlsMode,
    allow_private_networks: false,
    max_attachment_mb: 50,
  });
  const [password, setPassword] = useState("");
  const [clearPassword, setClearPassword] = useState(false);

  const smtp = settings?.smtp;

  useEffect(() => {
    if (!smtp) return;
    setForm({
      enabled: smtp.enabled,
      host: smtp.host,
      port: smtp.port,
      username: smtp.username,
      from_email: smtp.from_email,
      tls_mode: smtp.tls_mode,
      allow_private_networks: smtp.allow_private_networks,
      max_attachment_mb: smtp.max_attachment_mb || 50,
    });
    setPassword("");
    setClearPassword(false);
  }, [smtp]);

  const tlsModes = smtp?.available_tls_modes?.length
    ? smtp.available_tls_modes
    : (["none", "starttls", "implicit_tls"] as SmtpTlsMode[]);

  const changeTlsMode = (mode: SmtpTlsMode) => {
    setForm((prev) => {
      const isDefaultPort = Object.values(TLS_PORTS).includes(prev.port);
      return {
        ...prev,
        tls_mode: mode,
        port: isDefaultPort ? TLS_PORTS[mode] : prev.port,
      };
    });
  };

  const passwordPayload = () => {
    if (clearPassword) return "";
    return password.trim() !== "" ? password : undefined;
  };

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    const payload: Record<string, unknown> = {
      "smtp.enabled": form.enabled,
      "smtp.host": form.host.trim(),
      "smtp.port": form.port,
      "smtp.username": form.username.trim(),
      "smtp.from_email": form.from_email.trim(),
      "smtp.tls_mode": form.tls_mode,
      "smtp.allow_private_networks": form.allow_private_networks,
      "smtp.max_attachment_mb": form.max_attachment_mb,
    };
    const nextPassword = passwordPayload();
    if (nextPassword !== undefined) {
      payload["smtp.password"] = nextPassword;
    }

    updateMutation.mutate(payload, {
      onSuccess: () => {
        toast.success(t("settings.smtp_saved", "SMTP settings saved"));
        setPassword("");
        setClearPassword(false);
      },
      onError: (err) =>
        toast.error(err instanceof Error ? err.message : String(err)),
    });
  };

  const handleTest = () => {
    testMutation.mutate(
      {
        host: form.host.trim(),
        port: form.port,
        username: form.username.trim(),
        password: passwordPayload(),
        from_email: form.from_email.trim(),
        tls_mode: form.tls_mode,
        allow_private_networks: form.allow_private_networks,
      },
      {
        onSuccess: () =>
          toast.success(
            t("settings.smtp_test_success", "SMTP connection succeeded"),
          ),
        onError: (err) =>
          toast.error(err instanceof Error ? err.message : String(err)),
      },
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-3 border-b border-base-200 pb-3">
        <div>
          <h2 className="card-title text-lg flex items-center gap-2">
            <Mail className="h-5 w-5 text-primary" />
            {t("settings.smtp_title", "Email (SMTP)")}
          </h2>
          <p className="text-xs text-base-content/50 mt-1">
            {t(
              "settings.smtp_desc",
              "Outbound mail server used for Send-to-Kindle and system notifications.",
            )}
          </p>
        </div>
        <label className="flex items-center gap-2 shrink-0">
          <input
            type="checkbox"
            className="toggle toggle-primary toggle-sm"
            checked={form.enabled}
            onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
          />
          <span className="text-xs font-bold uppercase tracking-wider opacity-60">
            {t("settings.smtp_enabled", "Enabled")}
          </span>
        </label>
      </div>

      <form onSubmit={handleSave} className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div className="md:col-span-2 flex flex-col gap-1.5">
            <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
              {t("settings.smtp_host", "SMTP Host")}
            </label>
            <input
              type="text"
              value={form.host}
              onChange={(e) => setForm({ ...form, host: e.target.value })}
              placeholder="smtp.example.com"
              className="input input-bordered w-full"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
              {t("settings.smtp_port", "Port")}
            </label>
            <input
              type="number"
              min={1}
              max={65535}
              value={form.port}
              onChange={(e) =>
                setForm({ ...form, port: Number(e.target.value) })
              }
              className="input input-bordered w-full"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
              {t("settings.smtp_username", "Username")}
            </label>
            <input
              type="text"
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              placeholder="user@example.com"
              className="input input-bordered w-full"
              autoComplete="off"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
              {t("settings.smtp_password", "Password")}
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value);
                setClearPassword(false);
              }}
              placeholder={
                smtp?.password_configured
                  ? t(
                      "settings.smtp_password_keep",
                      "Leave blank to keep the saved password",
                    )
                  : t("settings.smtp_password_placeholder", "App password")
              }
              className="input input-bordered w-full"
              autoComplete="new-password"
              disabled={clearPassword}
            />
            {smtp?.password_configured && (
              <label className="flex items-center gap-2 pl-1">
                <input
                  type="checkbox"
                  className="checkbox checkbox-xs"
                  checked={clearPassword}
                  onChange={(e) => {
                    setClearPassword(e.target.checked);
                    if (e.target.checked) setPassword("");
                  }}
                />
                <span className="text-xs text-base-content/60">
                  {t(
                    "settings.smtp_password_clear",
                    "Remove the saved password",
                  )}
                </span>
              </label>
            )}
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
              {t("settings.smtp_from", "From Address")}
            </label>
            <input
              type="email"
              value={form.from_email}
              onChange={(e) => setForm({ ...form, from_email: e.target.value })}
              placeholder="library@example.com"
              className="input input-bordered w-full"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
              {t("settings.smtp_tls_mode", "Encryption")}
            </label>
            <select
              className="select select-bordered w-full"
              value={form.tls_mode}
              onChange={(e) => changeTlsMode(e.target.value as SmtpTlsMode)}
            >
              {tlsModes.map((mode) => (
                <option key={mode} value={mode}>
                  {t(TLS_LABEL_KEYS[mode], mode)}
                </option>
              ))}
            </select>
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
              {t("settings.smtp_max_attachment", "Max Attachment (MB)")}
            </label>
            <input
              type="number"
              min={1}
              max={500}
              value={form.max_attachment_mb}
              onChange={(e) =>
                setForm({
                  ...form,
                  max_attachment_mb: parseInt(e.target.value) || 50,
                })
              }
              placeholder="50"
              className="input input-bordered w-full"
            />
          </div>
        </div>

        <label className="flex items-start gap-3 rounded-lg bg-base-200/40 p-3">
          <input
            type="checkbox"
            className="toggle toggle-warning toggle-sm mt-0.5"
            checked={form.allow_private_networks}
            onChange={(e) =>
              setForm({ ...form, allow_private_networks: e.target.checked })
            }
          />
          <span>
            <span className="text-sm font-semibold flex items-center gap-1.5">
              <AlertTriangle className="h-3.5 w-3.5 text-warning" />
              {t(
                "settings.smtp_allow_private",
                "Allow private / local mail servers",
              )}
            </span>
            <span className="block text-xs text-base-content/50 mt-0.5">
              {t(
                "settings.smtp_allow_private_desc",
                "Required for a mail server on your LAN, localhost, or a sibling container. Leave off when using an internet provider.",
              )}
            </span>
          </span>
        </label>

        <div className="flex items-center gap-3 pt-3 border-t border-base-200">
          <button
            type="submit"
            className="btn btn-primary btn-sm gap-2"
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? (
              <span className="loading loading-spinner loading-xs" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            {t("settings.save_smtp", "Save SMTP")}
          </button>
          <button
            type="button"
            onClick={handleTest}
            className="btn btn-outline btn-sm gap-2"
            disabled={testMutation.isPending || !form.host.trim()}
          >
            {testMutation.isPending ? (
              <span className="loading loading-spinner loading-xs" />
            ) : (
              <CheckCircle2 className="h-4 w-4" />
            )}
            {t("settings.smtp_test", "Test Connection")}
          </button>
        </div>
      </form>
    </div>
  );
};
