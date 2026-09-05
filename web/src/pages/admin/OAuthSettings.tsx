import { useAdminSettingsQuery, useUpdateAdminSettingsMutation } from "@/hooks";
import { invalidatePublicSettings } from "@/hooks/useSettings";
import {
  ArrowLeft,
  Check,
  Key,
  Loader2,
  Save,
  Shield,
  HelpCircle,
} from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { toast } from "react-toastify";

export function OAuthSettings() {
  const { t } = useTranslation();
  const {
    data: adminSettings,
    isLoading: settingsLoading,
    refetch,
  } = useAdminSettingsQuery();
  const updateSettingsMutation = useUpdateAdminSettingsMutation();

  const [googleEnabled, setGoogleEnabled] = useState(false);
  const [googleClientId, setGoogleClientId] = useState("");
  const [googleClientSecret, setGoogleClientSecret] = useState("");
  const [googleRedirectUri, setGoogleRedirectUri] = useState("");

  const [githubEnabled, setGithubEnabled] = useState(false);
  const [githubClientId, setGithubClientId] = useState("");
  const [githubClientSecret, setGithubClientSecret] = useState("");
  const [githubRedirectUri, setGithubRedirectUri] = useState("");

  const [discordEnabled, setDiscordEnabled] = useState(false);
  const [discordClientId, setDiscordClientId] = useState("");
  const [discordClientSecret, setDiscordClientSecret] = useState("");
  const [discordRedirectUri, setDiscordRedirectUri] = useState("");

  const [oidcEnabled, setOidcEnabled] = useState(false);
  const [oidcName, setOidcName] = useState("");
  const [oidcIssuerUrl, setOidcIssuerUrl] = useState("");
  const [oidcClientId, setOidcClientId] = useState("");
  const [oidcClientSecret, setOidcClientSecret] = useState("");
  const [oidcRedirectUri, setOidcRedirectUri] = useState("");
  const [oidcScopes, setOidcScopes] = useState("");

  const [savingProvider, setSavingProvider] = useState<string | null>(null);

  useEffect(() => {
    if (adminSettings?.oauth) {
      const { google, github, discord, oidc } = adminSettings.oauth;

      setGoogleEnabled(google.enabled);
      setGoogleClientId(google.client_id || "");
      setGoogleClientSecret(google.client_secret_set ? "********" : "");
      setGoogleRedirectUri(google.redirect_uri || "");

      setGithubEnabled(github.enabled);
      setGithubClientId(github.client_id || "");
      setGithubClientSecret(github.client_secret_set ? "********" : "");
      setGithubRedirectUri(github.redirect_uri || "");

      setDiscordEnabled(discord.enabled);
      setDiscordClientId(discord.client_id || "");
      setDiscordClientSecret(discord.client_secret_set ? "********" : "");
      setDiscordRedirectUri(discord.redirect_uri || "");

      setOidcEnabled(oidc.enabled);
      setOidcName(oidc.name || "OpenID Connect");
      setOidcIssuerUrl(oidc.issuer_url || "");
      setOidcClientId(oidc.client_id || "");
      setOidcClientSecret(oidc.client_secret_set ? "********" : "");
      setOidcRedirectUri(oidc.redirect_uri || "");
      setOidcScopes(
        oidc.scopes ? oidc.scopes.join(", ") : "openid, profile, email",
      );
    }
  }, [adminSettings]);

  const handleSave = (provider: "google" | "github" | "discord" | "oidc") => {
    setSavingProvider(provider);

    let payload: Record<string, unknown> = {};

    if (provider === "google") {
      payload = {
        "oauth.google.enabled": googleEnabled,
        "oauth.google.client_id": googleClientId.trim(),
        "oauth.google.redirect_uri": googleRedirectUri.trim(),
      };
      if (googleClientSecret && googleClientSecret !== "********") {
        payload["oauth.google.client_secret"] = googleClientSecret;
      }
    } else if (provider === "github") {
      payload = {
        "oauth.github.enabled": githubEnabled,
        "oauth.github.client_id": githubClientId.trim(),
        "oauth.github.redirect_uri": githubRedirectUri.trim(),
      };
      if (githubClientSecret && githubClientSecret !== "********") {
        payload["oauth.github.client_secret"] = githubClientSecret;
      }
    } else if (provider === "discord") {
      payload = {
        "oauth.discord.enabled": discordEnabled,
        "oauth.discord.client_id": discordClientId.trim(),
        "oauth.discord.redirect_uri": discordRedirectUri.trim(),
      };
      if (discordClientSecret && discordClientSecret !== "********") {
        payload["oauth.discord.client_secret"] = discordClientSecret;
      }
    } else if (provider === "oidc") {
      const parsedScopes = oidcScopes
        .split(",")
        .map((s) => s.trim())
        .filter((s) => s !== "");

      payload = {
        "oauth.oidc.enabled": oidcEnabled,
        "oauth.oidc.name": oidcName.trim() || "OpenID Connect",
        "oauth.oidc.issuer_url": oidcIssuerUrl.trim(),
        "oauth.oidc.client_id": oidcClientId.trim(),
        "oauth.oidc.redirect_uri": oidcRedirectUri.trim(),
        "oauth.oidc.scopes": parsedScopes,
      };
      if (oidcClientSecret && oidcClientSecret !== "********") {
        payload["oauth.oidc.client_secret"] = oidcClientSecret;
      }
    }

    updateSettingsMutation.mutate(payload, {
      onSuccess: async () => {
        toast.success(t("settings.saved_success", "Saved successfully"));
        await invalidatePublicSettings();
        await refetch();
        setSavingProvider(null);
      },
      onError: (err) => {
        toast.error(err instanceof Error ? err.message : String(err));
        setSavingProvider(null);
      },
    });
  };

  if (settingsLoading) {
    return (
      <div className="flex h-full items-center justify-center bg-base-100">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  const isSaving = (p: string) => savingProvider === p;

  return (
    <div className="flex flex-col h-full bg-base-100 font-sans">
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex items-center justify-between bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div className="flex items-center gap-3">
          <Link
            to="/admin/settings"
            className="btn btn-ghost btn-circle btn-sm"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {t("settings.oauth.title", "OAuth & OIDC Settings")}
            </h1>
            <p className="text-sm text-base-content/60 mt-1">
              {t(
                "settings.oauth.subtitle",
                "Configure external identity providers for social and enterprise single sign-on",
              )}
            </p>
          </div>
        </div>
      </header>

      <div className="flex-1 overflow-auto p-4 sm:p-6 lg:p-8">
        <div className="max-w-4xl mx-auto space-y-6">
          <div className="rounded-2xl border border-info/30 bg-info/5 p-4 flex items-start sm:items-center gap-3 text-sm">
            <HelpCircle className="h-5 w-5 shrink-0 text-info mt-0.5 sm:mt-0" />
            <div className="flex-1">
              <p className="text-sm text-base-content/85 font-medium leading-relaxed">
                {t("settings.oauth.setup_info")}
              </p>
            </div>
          </div>

          <div className="card bg-base-100 border border-base-200 shadow-sm overflow-hidden transition-all duration-300 hover:shadow-md">
            <div className="card-body p-5">
              <div className="flex items-center justify-between border-b border-base-200/60 pb-3 mb-4">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-red-500/10 flex items-center justify-center text-red-500 shrink-0">
                    <svg
                      className="h-5 w-5"
                      viewBox="0 0 24 24"
                      fill="currentColor"
                    >
                      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
                      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
                      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z" />
                      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z" />
                    </svg>
                  </div>
                  <div>
                    <h2 className="card-title text-base font-bold">
                      {t("settings.oauth.google_title")}
                    </h2>
                    <p className="text-xs text-base-content/50">
                      {t("settings.oauth.google_desc")}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary toggle-sm"
                    checked={googleEnabled}
                    onChange={(e) => setGoogleEnabled(e.target.checked)}
                  />
                  <button
                    onClick={() => handleSave("google")}
                    disabled={isSaving("google")}
                    className="btn btn-primary btn-sm gap-1"
                  >
                    {isSaving("google") ? (
                      <Loader2 className="h-3 w-3 animate-spin" />
                    ) : (
                      <Save className="h-3.5 w-3.5" />
                    )}
                    {t("admin.save", "Save")}
                  </button>
                </div>
              </div>

              {googleEnabled && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 animate-fadeIn">
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.client_id")}
                    </label>
                    <input
                      type="text"
                      className="input input-bordered w-full input-sm h-10"
                      value={googleClientId}
                      onChange={(e) => setGoogleClientId(e.target.value)}
                      placeholder="12345678-abc.apps.googleusercontent.com"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.client_secret")}
                    </label>
                    <input
                      type="password"
                      className="input input-bordered w-full input-sm h-10"
                      value={googleClientSecret}
                      onChange={(e) => setGoogleClientSecret(e.target.value)}
                      placeholder="••••••••••••••••••••••••"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5 col-span-1 sm:col-span-2">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.redirect_uri")}
                    </label>
                    <input
                      type="text"
                      className="input input-bordered w-full input-sm h-10"
                      value={googleRedirectUri}
                      onChange={(e) => setGoogleRedirectUri(e.target.value)}
                      placeholder="https://books.example.com/api/v1/auth/oauth2/google/callback"
                    />
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className="card bg-base-100 border border-base-200 shadow-sm overflow-hidden transition-all duration-300 hover:shadow-md">
            <div className="card-body p-5">
              <div className="flex items-center justify-between border-b border-base-200/60 pb-3 mb-4">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-base-content/10 flex items-center justify-center text-base-content shrink-0">
                    <svg
                      className="h-5 w-5"
                      viewBox="0 0 24 24"
                      fill="currentColor"
                    >
                      <path
                        fillRule="evenodd"
                        clipRule="evenodd"
                        d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482C19.138 20.193 22 16.44 22 12.017 22 6.484 17.522 2 12 2z"
                      />
                    </svg>
                  </div>
                  <div>
                    <h2 className="card-title text-base font-bold">
                      {t("settings.oauth.github_title")}
                    </h2>
                    <p className="text-xs text-base-content/50">
                      {t("settings.oauth.github_desc")}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary toggle-sm"
                    checked={githubEnabled}
                    onChange={(e) => setGithubEnabled(e.target.checked)}
                  />
                  <button
                    onClick={() => handleSave("github")}
                    disabled={isSaving("github")}
                    className="btn btn-primary btn-sm gap-1"
                  >
                    {isSaving("github") ? (
                      <Loader2 className="h-3 w-3 animate-spin" />
                    ) : (
                      <Save className="h-3.5 w-3.5" />
                    )}
                    {t("admin.save", "Save")}
                  </button>
                </div>
              </div>

              {githubEnabled && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 animate-fadeIn">
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.client_id")}
                    </label>
                    <input
                      type="text"
                      className="input input-bordered w-full input-sm h-10"
                      value={githubClientId}
                      onChange={(e) => setGithubClientId(e.target.value)}
                      placeholder="Ov23cixxxxxx"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.client_secret")}
                    </label>
                    <input
                      type="password"
                      className="input input-bordered w-full input-sm h-10"
                      value={githubClientSecret}
                      onChange={(e) => setGithubClientSecret(e.target.value)}
                      placeholder="••••••••••••••••••••••••"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5 col-span-1 sm:col-span-2">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.redirect_uri")}
                    </label>
                    <input
                      type="text"
                      className="input input-bordered w-full input-sm h-10"
                      value={githubRedirectUri}
                      onChange={(e) => setGithubRedirectUri(e.target.value)}
                      placeholder="https://books.example.com/api/v1/auth/oauth2/github/callback"
                    />
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className="card bg-base-100 border border-base-200 shadow-sm overflow-hidden transition-all duration-300 hover:shadow-md">
            <div className="card-body p-5">
              <div className="flex items-center justify-between border-b border-base-200/60 pb-3 mb-4">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-indigo-500/10 flex items-center justify-center text-indigo-500 shrink-0">
                    <svg
                      className="h-5 w-5"
                      viewBox="0 0 24 24"
                      fill="currentColor"
                    >
                      <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994.021-.041.001-.09-.041-.106a13.094 13.094 0 0 1-1.873-.894.077.077 0 0 1-.008-.128c.126-.093.252-.19.372-.287a.075.075 0 0 1 .077-.011c3.92 1.793 8.18 1.793 12.061 0a.073.073 0 0 1 .078.009c.12.099.246.195.373.289a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.894.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.156-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.156 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.156-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.156 2.418z" />
                    </svg>
                  </div>
                  <div>
                    <h2 className="card-title text-base font-bold">
                      {t("settings.oauth.discord_title")}
                    </h2>
                    <p className="text-xs text-base-content/50">
                      {t("settings.oauth.discord_desc")}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary toggle-sm"
                    checked={discordEnabled}
                    onChange={(e) => setDiscordEnabled(e.target.checked)}
                  />
                  <button
                    onClick={() => handleSave("discord")}
                    disabled={isSaving("discord")}
                    className="btn btn-primary btn-sm gap-1"
                  >
                    {isSaving("discord") ? (
                      <Loader2 className="h-3 w-3 animate-spin" />
                    ) : (
                      <Save className="h-3.5 w-3.5" />
                    )}
                    {t("admin.save", "Save")}
                  </button>
                </div>
              </div>

              {discordEnabled && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 animate-fadeIn">
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.client_id")}
                    </label>
                    <input
                      type="text"
                      className="input input-bordered w-full input-sm h-10"
                      value={discordClientId}
                      onChange={(e) => setDiscordClientId(e.target.value)}
                      placeholder="10928374950293"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.client_secret")}
                    </label>
                    <input
                      type="password"
                      className="input input-bordered w-full input-sm h-10"
                      value={discordClientSecret}
                      onChange={(e) => setDiscordClientSecret(e.target.value)}
                      placeholder="••••••••••••••••••••••••"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5 col-span-1 sm:col-span-2">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.redirect_uri")}
                    </label>
                    <input
                      type="text"
                      className="input input-bordered w-full input-sm h-10"
                      value={discordRedirectUri}
                      onChange={(e) => setDiscordRedirectUri(e.target.value)}
                      placeholder="https://books.example.com/api/v1/auth/oauth2/discord/callback"
                    />
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className="card bg-base-100 border border-base-200 shadow-sm overflow-hidden transition-all duration-300 hover:shadow-md">
            <div className="card-body p-5">
              <div className="flex items-center justify-between border-b border-base-200/60 pb-3 mb-4">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-purple-500/10 flex items-center justify-center text-purple-500 shrink-0">
                    <Shield className="h-5 w-5" />
                  </div>
                  <div>
                    <h2 className="card-title text-base font-bold">
                      {t("settings.oauth.oidc_title")}
                    </h2>
                    <p className="text-xs text-base-content/50">
                      {t("settings.oauth.oidc_desc")}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary toggle-sm"
                    checked={oidcEnabled}
                    onChange={(e) => setOidcEnabled(e.target.checked)}
                  />
                  <button
                    onClick={() => handleSave("oidc")}
                    disabled={isSaving("oidc")}
                    className="btn btn-primary btn-sm gap-1"
                  >
                    {isSaving("oidc") ? (
                      <Loader2 className="h-3 w-3 animate-spin" />
                    ) : (
                      <Save className="h-3.5 w-3.5" />
                    )}
                    {t("admin.save", "Save")}
                  </button>
                </div>
              </div>

              {oidcEnabled && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 animate-fadeIn">
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.provider_name")}
                    </label>
                    <input
                      type="text"
                      className="input input-bordered w-full input-sm h-10"
                      value={oidcName}
                      onChange={(e) => setOidcName(e.target.value)}
                      placeholder="e.g. Keycloak"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.issuer_url")}
                    </label>
                    <input
                      type="text"
                      className="input input-bordered w-full input-sm h-10"
                      value={oidcIssuerUrl}
                      onChange={(e) => setOidcIssuerUrl(e.target.value)}
                      placeholder="https://keycloak.example.com/realms/master"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.client_id")}
                    </label>
                    <input
                      type="text"
                      className="input input-bordered w-full input-sm h-10"
                      value={oidcClientId}
                      onChange={(e) => setOidcClientId(e.target.value)}
                      placeholder="novelhub-client"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.client_secret")}
                    </label>
                    <input
                      type="password"
                      className="input input-bordered w-full input-sm h-10"
                      value={oidcClientSecret}
                      onChange={(e) => setOidcClientSecret(e.target.value)}
                      placeholder="••••••••••••••••••••••••"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.redirect_uri")}
                    </label>
                    <input
                      type="text"
                      className="input input-bordered w-full input-sm h-10"
                      value={oidcRedirectUri}
                      onChange={(e) => setOidcRedirectUri(e.target.value)}
                      placeholder="https://books.example.com/api/v1/auth/oauth2/oidc/callback"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">
                      {t("settings.oauth.scopes")}
                    </label>
                    <input
                      type="text"
                      className="input input-bordered w-full input-sm h-10"
                      value={oidcScopes}
                      onChange={(e) => setOidcScopes(e.target.value)}
                      placeholder="openid, profile, email"
                    />
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
