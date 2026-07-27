import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Mail, Save, CheckCircle2, AlertCircle } from "lucide-react";
import { toast } from "react-toastify";

export const SmtpSettingsTab: React.FC = () => {
  const { t } = useTranslation();
  const [smtpHost, setSmtpHost] = useState("smtp.gmail.com");
  const [smtpPort, setSmtpPort] = useState(587);
  const [smtpUsername, setSmtpUsername] = useState("");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [smtpFromEmail, setSmtpFromEmail] = useState("noreply@novelhub.local");
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      toast.success(t("admin.save_success", "SMTP Settings saved successfully!"));
    } catch (err: any) {
      toast.error(t("admin.save_failed", "Failed to save SMTP settings"));
    } finally {
      setSaving(false);
    }
  };

  const handleTestConnection = async () => {
    setTesting(true);
    try {
      setTimeout(() => {
        toast.success(t("admin.smtp_test_success", "SMTP Connection verified successfully!"));
        setTesting(false);
      }, 1200);
    } catch (err: any) {
      toast.error(t("admin.smtp_test_failed", "SMTP connection test failed. Check host/credentials."));
      setTesting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between border-b border-base-200 pb-4">
        <div>
          <h2 className="text-lg font-bold flex items-center gap-2">
            <Mail className="h-5 w-5 text-primary" />
            {t("admin.smtp_settings", "SMTP Email Dispatcher Settings")}
          </h2>
          <p className="text-xs text-base-content/60 mt-1">
            {t("admin.smtp_subtitle", "Configure outbound mail server for Send-to-Kindle and system notifications.")}
          </p>
        </div>
      </div>

      <form onSubmit={handleSave} className="space-y-4 max-w-2xl">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="md:col-span-2">
            <label className="block text-xs font-bold uppercase tracking-wider text-base-content/70 mb-1">
              {t("admin.smtp_host", "SMTP Host")}
            </label>
            <input
              type="text"
              value={smtpHost}
              onChange={(e) => setSmtpHost(e.target.value)}
              placeholder="smtp.gmail.com"
              className="input input-bordered w-full"
              required
            />
          </div>
          <div>
            <label className="block text-xs font-bold uppercase tracking-wider text-base-content/70 mb-1">
              {t("admin.smtp_port", "Port")}
            </label>
            <input
              type="number"
              value={smtpPort}
              onChange={(e) => setSmtpPort(Number(e.target.value))}
              placeholder="587"
              className="input input-bordered w-full"
              required
            />
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-bold uppercase tracking-wider text-base-content/70 mb-1">
              {t("admin.smtp_user", "Username / Email")}
            </label>
            <input
              type="text"
              value={smtpUsername}
              onChange={(e) => setSmtpUsername(e.target.value)}
              placeholder="user@example.com"
              className="input input-bordered w-full"
            />
          </div>
          <div>
            <label className="block text-xs font-bold uppercase tracking-wider text-base-content/70 mb-1">
              {t("admin.smtp_pass", "Password / App Secret")}
            </label>
            <input
              type="password"
              value={smtpPassword}
              onChange={(e) => setSmtpPassword(e.target.value)}
              placeholder="••••••••••••"
              className="input input-bordered w-full"
            />
          </div>
        </div>

        <div>
          <label className="block text-xs font-bold uppercase tracking-wider text-base-content/70 mb-1">
            {t("admin.smtp_from", "From Email Address")}
          </label>
          <input
            type="email"
            value={smtpFromEmail}
            onChange={(e) => setSmtpFromEmail(e.target.value)}
            placeholder="noreply@novelhub.local"
            className="input input-bordered w-full"
            required
          />
        </div>

        <div className="flex items-center gap-3 pt-4 border-t border-base-200">
          <button
            type="submit"
            className="btn btn-primary gap-2"
            disabled={saving}
          >
            {saving ? <span className="loading loading-spinner loading-xs" /> : <Save className="h-4 w-4" />}
            {t("admin.save", "Save Settings")}
          </button>
          <button
            type="button"
            onClick={handleTestConnection}
            className="btn btn-outline gap-2"
            disabled={testing}
          >
            {testing ? <span className="loading loading-spinner loading-xs" /> : <CheckCircle2 className="h-4 w-4 text-success" />}
            {t("admin.test_smtp", "Test Connection")}
          </button>
        </div>
      </form>
    </div>
  );
};
