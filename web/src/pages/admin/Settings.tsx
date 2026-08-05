import { LibraryMultiSelect } from "@/components/admin/settings/LibraryMultiSelect";
import { RuntimeLimitsCard } from "@/components/admin/settings/RuntimeLimitsCard";
import { GUEST_MODES, SIDEBAR_LABELS } from "@/constants";
import { useAdminSettingsQuery, useLibrariesQuery, useUpdateAdminSettingsMutation } from "@/hooks";
import { invalidatePublicSettings } from "@/hooks/useSettings";
import { adminService } from "@/services";
import { useSettingsAdminStore } from "@/stores";
import {
  Eye,
  Globe,
  Home,
  Layout,
  Loader2,
  RefreshCw,
  Save,
  UserPlus,
} from "lucide-react";
import { SyntheticEvent, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";
import { useShallow } from "zustand/react/shallow";
import { ImageCropperModal } from "@/components/common/ImageCropperModal";
import { SmtpSettingsTab } from "@/components/admin/settings/SmtpSettingsTab";
import { WebhooksTab } from "@/components/admin/settings/WebhooksTab";

export function Settings() {
  const { t } = useTranslation();
  const { data: adminSettings, isLoading: settingsLoading, refetch: refetchSettings } = useAdminSettingsQuery();
  const { data: librariesList = [], isLoading: librariesLoading, refetch: refetchLibraries } = useLibrariesQuery();
  const updateSettingsMutation = useUpdateAdminSettingsMutation();

  const {
    site, setSite,
    serverUrl, setServerUrl,
    sidebarItems, setSidebarItems,
    homeSections, setHomeSections,
    registration, setRegistration,
    loginRequired, setLoginRequired,
    requireEmailVerify, setRequireEmailVerify,
    passwordResetEnabled, setPasswordResetEnabled,
    guestMode, setGuestMode,
    guestLibraryIds, setGuestLibraryIds,
    inBookSearch, setInBookSearch,
    customFontUpload, setCustomFontUpload,
    anilistTracking, setAnilistTracking,
    savingSection, setSavingSection,
    uploadingLogo, setUploadingLogo,
    uploadingFavicon, setUploadingFavicon,
    selectedCropImage, setSelectedCropImage,
    cropTarget, setCropTarget,
    initFromSettings,
  } = useSettingsAdminStore(useShallow((state) => ({
    site: state.site, setSite: state.setSite,
    serverUrl: state.serverUrl, setServerUrl: state.setServerUrl,
    sidebarItems: state.sidebarItems, setSidebarItems: state.setSidebarItems,
    homeSections: state.homeSections, setHomeSections: state.setHomeSections,
    registration: state.registration, setRegistration: state.setRegistration,
    loginRequired: state.loginRequired, setLoginRequired: state.setLoginRequired,
    requireEmailVerify: state.requireEmailVerify, setRequireEmailVerify: state.setRequireEmailVerify,
    passwordResetEnabled: state.passwordResetEnabled, setPasswordResetEnabled: state.setPasswordResetEnabled,
    guestMode: state.guestMode, setGuestMode: state.setGuestMode,
    guestLibraryIds: state.guestLibraryIds, setGuestLibraryIds: state.setGuestLibraryIds,
    inBookSearch: state.inBookSearch, setInBookSearch: state.setInBookSearch,
    customFontUpload: state.customFontUpload, setCustomFontUpload: state.setCustomFontUpload,
    anilistTracking: state.anilistTracking, setAnilistTracking: state.setAnilistTracking,
    savingSection: state.savingSection, setSavingSection: state.setSavingSection,
    uploadingLogo: state.uploadingLogo, setUploadingLogo: state.setUploadingLogo,
    uploadingFavicon: state.uploadingFavicon, setUploadingFavicon: state.setUploadingFavicon,
    selectedCropImage: state.selectedCropImage, setSelectedCropImage: state.setSelectedCropImage,
    cropTarget: state.cropTarget, setCropTarget: state.setCropTarget,
    initFromSettings: state.initFromSettings,
  })));

  const loading = settingsLoading || librariesLoading;
  const libraries = librariesList;

  useEffect(() => {
    if (adminSettings) {
      initFromSettings(adminSettings);
    }
  }, [adminSettings, initFromSettings]);

  function saveSection(section: string, data: Record<string, unknown>) {
    setSavingSection(section);
    updateSettingsMutation.mutate(data, {
      onSuccess: async () => {
        toast.success(t("settings.saved_success", "Saved successfully"));
        await invalidatePublicSettings();
        setSavingSection(null);
      },
      onError: (err) => {
        toast.error(err instanceof Error ? err.message : String(err));
        setSavingSection(null);
      },
    });
  }

  const handleUploadLink = async (type: "logo" | "favicon") => {
    const url = site[type];
    if (!url || !url.startsWith("http")) return;
    
    if (type === "logo") setUploadingLogo(true);
    else setUploadingFavicon(true);
    
    const fd = new FormData();
    fd.append("url", url);
    fd.append("target", type);
    try {
      const res = await adminService.uploadAdminLogo(fd);
      if (res.status && res.data?.url) {
        setSite({ ...site, [type]: res.data.url });
      } else {
        toast.error(res.message || `Failed to fetch ${type}`);
      }
    } catch {
      toast.error(`Failed to fetch ${type}`);
    } finally {
      if (type === "logo") setUploadingLogo(false);
      else setUploadingFavicon(false);
    }
  };

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>, type: "logo" | "favicon") => {
    const file = e.target.files?.[0];
    if (!file) return;
    
    const reader = new FileReader();
    reader.onload = (event) => {
      if (event.target?.result) {
        setSelectedCropImage(event.target.result as string);
        setCropTarget(type);
      }
    };
    reader.readAsDataURL(file);
    e.target.value = "";
  };

  const handleCropApply = async (base64: string) => {
    const target = cropTarget;
    setSelectedCropImage(null);
    setCropTarget(null);
    if (!target) return;

    if (target === "logo") setUploadingLogo(true);
    else setUploadingFavicon(true);
    
    try {
      const resBlob = await fetch(base64);
      const blob = await resBlob.blob();
      
      const fd = new FormData();
      fd.append("file", blob, `${target}.png`);
      fd.append("target", target);
      
      const res = await adminService.uploadAdminLogo(fd);
      if (res.status && res.data?.url) {
        setSite({ ...site, [target]: res.data.url });
      } else {
        toast.error(res.message || `Failed to upload ${target}`);
      }
    } catch {
      toast.error(`Failed to upload ${target}`);
    } finally {
      if (target === "logo") setUploadingLogo(false);
      else setUploadingFavicon(false);
    }
  };

  function handleSiteSave(e: SyntheticEvent) {
    e.preventDefault();
    void saveSection("Site settings", {
      "site.title": site.title,
      "site.description": site.description,
      "site.favicon": site.favicon,
      "site.logo": site.logo,
      "site.meta_description": site.meta_description,
      "server.url": serverUrl.trim(),
    });
  }

  function handleSidebarSave() {
    void saveSection("Sidebar", { "sidebar.visible_items": sidebarItems });
  }

  function handleHomeSave() {
    void saveSection("Home sections", { "home.sections": homeSections });
  }

  function handleReaderFeaturesSave() {
    void saveSection("Reader features", {
      "reader.enable_in_book_search": inBookSearch,
      "font.enable_custom_font_upload": customFontUpload,
      "tracker.anilist_enabled": anilistTracking,
    });
  }

  function handleRegistrationSave() {
    void saveSection("Registration & Guest", {
      "auth.registration_enabled": registration,
      "auth.login_required": loginRequired,
      "auth.require_email_verify": requireEmailVerify,
      "auth.password_reset_enabled": passwordResetEnabled,
      "guest_access.mode": guestMode,
      "guest_access.library_ids": guestMode === "selected_libraries" ? guestLibraryIds : [],
    });
  }

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center bg-base-100">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  const isSaving = (s: string) => savingSection === s;

  return (
    <div className="flex flex-col h-full bg-base-100">
      {/* Header */}
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex items-center justify-between bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("settings.title", "Settings")}</h1>
          <p className="text-sm text-base-content/60 mt-1">{t("settings.subtitle", "Website customization, policies, and access control")}</p>
        </div>
        <button
          onClick={() => {
            void refetchSettings();
            void refetchLibraries();
          }}
          className="btn btn-square btn-ghost btn-sm sm:btn-md"
          title={t("settings.refresh", "Refresh")}
        >
          <RefreshCw className={`h-5 w-5 ${loading ? "animate-spin" : ""}`} />
        </button>
      </header>

      <div className="flex-1 overflow-auto p-4 sm:p-6 lg:p-8">
        <div className="max-w-7xl mx-auto w-full space-y-3">
          {/* ────── Site Settings ────── */}
          <div className="card bg-base-100 border border-base-200 shadow-sm">
            <div className="card-body p-4 sm:p-5">
              <form onSubmit={handleSiteSave} className="flex flex-col gap-3">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2 border-b border-base-200/60">
                  <div>
                    <div className="flex items-center gap-2 mb-0.5">
                      <Globe className="h-5 w-5 text-primary" />
                      <h2 className="card-title text-lg">{t("settings.site_info", "Site Information")}</h2>
                    </div>
                    <p className="text-xs text-base-content/50">{t("settings.site_info_desc", "Customize how your library appears to visitors.")}</p>
                  </div>
                  <button
                    type="submit"
                    disabled={isSaving("Site settings")}
                    className="btn btn-primary btn-sm gap-1 shrink-0 self-start sm:self-center"
                  >
                    {isSaving("Site settings") ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                    {t("settings.save_site", "Save Site")}
                  </button>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">{t("settings.title_label", "Title")}</label>
                    <input
                      type="text"
                      className="input input-bordered w-full"
                      value={site.title}
                      onChange={(e) => setSite({ ...site, title: e.target.value })}
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">{t("settings.meta_desc_label", "Meta Description")}</label>
                    <input
                      type="text"
                      className="input input-bordered w-full"
                      value={site.meta_description}
                      onChange={(e) => setSite({ ...site, meta_description: e.target.value })}
                    />
                  </div>
                  <div className="flex flex-col gap-1.5 col-span-1 sm:col-span-2">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">{t("settings.server_url_label", "Server URL")}</label>
                    <input
                      type="text"
                      className="input input-bordered w-full"
                      value={serverUrl}
                      onChange={(e) => setServerUrl(e.target.value)}
                      placeholder="https://books.example.com"
                    />
                    <p className="text-xs text-base-content/50 pl-1">
                      {t("settings.server_url_desc", "Absolute base URL used in OPDS catalog and Kobo sync links. Leave empty to detect it from each request — set it only if the detected host is wrong, for example behind a path-rewriting proxy.")}
                    </p>
                  </div>
                  <div className="flex flex-col gap-1.5 col-span-1 sm:col-span-2">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">{t("settings.logo_url_label", "Logo URL")}</label>
                    <div className="flex gap-2 items-start">
                      <div className="flex flex-col gap-2 flex-1">
                        <div className="join w-full">
                          <input
                            type="text"
                            className="input input-bordered join-item w-full h-12"
                            value={site.logo}
                            onChange={(e) => setSite({ ...site, logo: e.target.value })}
                            placeholder="https://..."
                          />
                          <button 
                            type="button" 
                            className="btn btn-primary join-item h-12"
                            onClick={() => handleUploadLink("logo")}
                            disabled={!site.logo.startsWith('http') || uploadingLogo}
                          >
                            {uploadingLogo ? <Loader2 className="w-4 h-4 animate-spin" /> : t("settings.fetch", "Fetch")}
                          </button>
                        </div>
                        <div className="flex items-center gap-2">
                          <label className="btn btn-sm btn-outline cursor-pointer font-normal border-base-300">
                            {t("settings.upload_logo", "Upload Logo")}
                            <input type="file" accept="image/*" className="hidden" onChange={(e) => handleFileUpload(e, "logo")} />
                          </label>
                          <button 
                            type="button"
                            className="btn btn-sm btn-ghost font-normal"
                            onClick={() => setSite({ ...site, logo: '/logo.svg' })}
                          >
                            {t("settings.use_default", "Use Default")}
                          </button>
                        </div>
                      </div>
                      <div className="w-auto h-12 px-3 rounded-lg bg-base-200 border border-base-300 flex items-center justify-center overflow-hidden shrink-0">
                        {site.logo ? (
                          <div className="flex items-center gap-2">
                            <img src={site.logo} alt="Logo preview" className="w-8 h-8 rounded" onError={(e) => { e.currentTarget.style.display = 'none'; e.currentTarget.nextElementSibling?.classList.remove('hidden') }} onLoad={(e) => { e.currentTarget.style.display = 'block'; e.currentTarget.nextElementSibling?.classList.add('hidden') }} />
                            <div className={`w-5 h-5 opacity-40 hidden`} />
                            <span className="font-bold whitespace-nowrap">{site.title || "NovelHub"}</span>
                          </div>
                        ) : (
                          <div className="flex items-center gap-2">
                            <div className="w-5 h-5 bg-base-300 rounded opacity-40" />
                            <span className="font-bold whitespace-nowrap">{site.title || "NovelHub"}</span>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="flex flex-col gap-1.5 col-span-1 sm:col-span-2">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">{t("settings.favicon_url_label", "Favicon URL")}</label>
                    <div className="flex gap-2 items-start">
                      <div className="flex flex-col gap-2 flex-1">
                        <div className="join w-full">
                          <input
                            type="text"
                            className="input input-bordered join-item w-full h-12"
                            value={site.favicon}
                            onChange={(e) => setSite({ ...site, favicon: e.target.value })}
                            placeholder="https://..."
                          />
                          <button 
                            type="button" 
                            className="btn btn-primary join-item h-12"
                            onClick={() => handleUploadLink("favicon")}
                            disabled={!site.favicon.startsWith('http') || uploadingFavicon}
                          >
                            {uploadingFavicon ? <Loader2 className="w-4 h-4 animate-spin" /> : t("settings.fetch", "Fetch")}
                          </button>
                        </div>
                        <div className="flex items-center gap-2">
                          <label className="btn btn-sm btn-outline cursor-pointer font-normal border-base-300">
                            {t("settings.upload_favicon", "Upload Favicon")}
                            <input type="file" accept="image/*" className="hidden" onChange={(e) => handleFileUpload(e, "favicon")} />
                          </label>
                          <button 
                            type="button"
                            className="btn btn-sm btn-ghost font-normal"
                            onClick={() => setSite({ ...site, favicon: '/favicon.ico' })}
                          >
                            {t("settings.use_default", "Use Default")}
                          </button>
                        </div>
                      </div>
                      <div className="w-12 h-12 rounded-lg bg-base-200 border border-base-300 flex items-center justify-center overflow-hidden shrink-0">
                        {site.favicon ? (
                          <img src={site.favicon} alt="Favicon preview" className="w-8 h-8 rounded" onError={(e) => { e.currentTarget.style.display = 'none'; e.currentTarget.nextElementSibling?.classList.remove('hidden') }} onLoad={(e) => { e.currentTarget.style.display = 'block'; e.currentTarget.nextElementSibling?.classList.add('hidden') }} />
                        ) : null}
                        <div className={`w-5 h-5 bg-base-300 rounded opacity-40 ${site.favicon ? 'hidden' : ''}`} />
                      </div>
                    </div>
                  </div>
                </div>
                <div className="flex flex-col gap-1.5">
                  <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">{t("settings.description_label", "Description")}</label>
                  <textarea
                    className="textarea textarea-bordered w-full"
                    value={site.description}
                    onChange={(e) => setSite({ ...site, description: e.target.value })}
                    rows={2}
                  />
                </div>
              </form>
            </div>
          </div>

          {/* ────── Sidebar Visibility ────── */}
          <div className="card bg-base-100 border border-base-200 shadow-sm">
            <div className="card-body p-4 sm:p-5">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-3">
                <div>
                  <div className="flex items-center gap-2 mb-0.5">
                    <Layout className="h-5 w-5 text-primary" />
                    <h2 className="card-title text-lg">{t("settings.sidebar_nav", "Sidebar Navigation")}</h2>
                  </div>
                  <p className="text-xs text-base-content/50">{t("settings.sidebar_nav_desc", "Choose which navigation items appear in the library sidebar.")}</p>
                </div>
                <button
                  onClick={handleSidebarSave}
                  disabled={isSaving("Sidebar")}
                  className="btn btn-primary btn-sm gap-1 shrink-0 self-start sm:self-center"
                >
                  {isSaving("Sidebar") ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                  {t("settings.save_sidebar", "Save Sidebar")}
                </button>
              </div>

              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
                {(adminSettings?.available_sidebar_items || []).map((key: string) => (
                  <label
                    key={key}
                    className={`cursor-pointer flex items-center gap-2 p-2 rounded-lg border transition-colors ${
                      sidebarItems.includes(key)
                        ? "border-primary/30 bg-primary/5"
                        : "border-base-200 hover:bg-base-200/50"
                    }`}
                  >
                    <input
                      type="checkbox"
                      className="checkbox checkbox-xs checkbox-primary"
                      checked={sidebarItems.includes(key)}
                      onChange={() =>
                        setSidebarItems((prev) =>
                          prev.includes(key) ? prev.filter((i) => i !== key) : [...prev, key]
                        )
                      }
                    />
                    <span className="text-xs font-medium">{t(SIDEBAR_LABELS[key]) || key}</span>
                  </label>
                ))}
              </div>
            </div>
          </div>

          {/* ────── Home Sections ────── */}
          <div className="card bg-base-100 border border-base-200 shadow-sm">
            <div className="card-body p-4 sm:p-5">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-3">
                <div>
                  <div className="flex items-center gap-2 mb-0.5">
                    <Home className="h-5 w-5 text-primary" />
                    <h2 className="card-title text-lg">{t("settings.home_sections", "Home Page Sections")}</h2>
                  </div>
                  <p className="text-xs text-base-content/50">{t("settings.home_sections_desc", "Enable or disable sections on the library home page.")}</p>
                </div>
                <button
                  onClick={handleHomeSave}
                  disabled={isSaving("Home sections")}
                  className="btn btn-primary btn-sm gap-1 shrink-0 self-start sm:self-center"
                >
                  {isSaving("Home sections") ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                  {t("settings.save_home", "Save Home")}
                </button>
              </div>

              <div className="flex flex-col gap-2.5">
                <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary"
                    checked={homeSections.random_books}
                    onChange={(e) => setHomeSections({ ...homeSections, random_books: e.target.checked })}
                  />
                  <div>
                    <span className="text-sm font-medium">{t("settings.random_books", "Random Books")}</span>
                    <p className="text-xs text-base-content/50">{t("settings.random_books_desc", "Show random books section on the home page.")}</p>
                  </div>
                </label>
                <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary"
                    checked={homeSections.top_books}
                    onChange={(e) => setHomeSections({ ...homeSections, top_books: e.target.checked })}
                  />
                  <div>
                    <span className="text-sm font-medium">{t("settings.top_books", "Top Books")}</span>
                    <p className="text-xs text-base-content/50">{t("settings.top_books_desc", "Show top rated / most read books section.")}</p>
                  </div>
                </label>
              </div>
            </div>
          </div>

          {/* ────── Reader Features & Custom Fonts ────── */}
          <div className="card bg-base-100 border border-base-200 shadow-sm">
            <div className="card-body p-4 sm:p-5">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-3">
                <div>
                  <div className="flex items-center gap-2 mb-0.5">
                    <Layout className="h-5 w-5 text-primary" />
                    <h2 className="card-title text-lg">{t("settings.reader_features", "Reader Features & Fonts")}</h2>
                  </div>
                  <p className="text-xs text-base-content/50">{t("settings.reader_features_desc", "Toggle advanced reader search and server font uploads.")}</p>
                </div>
                <button
                  onClick={handleReaderFeaturesSave}
                  disabled={isSaving("Reader features")}
                  className="btn btn-primary btn-sm gap-1 shrink-0 self-start sm:self-center"
                >
                  {isSaving("Reader features") ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                  {t("settings.save_reader", "Save Reader Settings")}
                </button>
              </div>

              <div className="flex flex-col gap-2.5">
                <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary"
                    checked={inBookSearch}
                    onChange={(e) => setInBookSearch(e.target.checked)}
                  />
                  <div>
                    <span className="text-sm font-medium">{t("settings.in_book_search", "Enable In-Book Search")}</span>
                    <p className="text-xs text-base-content/50">{t("settings.in_book_search_desc", "Allow readers to search text inside open books (On-demand disk scan).")}</p>
                  </div>
                </label>
                <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary"
                    checked={customFontUpload}
                    onChange={(e) => setCustomFontUpload(e.target.checked)}
                  />
                  <div>
                    <span className="text-sm font-medium">{t("settings.custom_font_upload", "Enable Server Custom Font Uploads")}</span>
                    <p className="text-xs text-base-content/50">{t("settings.custom_font_upload_desc", "Allow authorized users to upload custom fonts to server (default is local IndexedDB font storage).")}</p>
                  </div>
                </label>
                <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary"
                    checked={anilistTracking}
                    onChange={(e) => setAnilistTracking(e.target.checked)}
                  />
                  <div>
                    <span className="text-sm font-medium">{t("settings.anilist_tracking", "Enable AniList Tracking")}</span>
                    <p className="text-xs text-base-content/50">{t("settings.anilist_tracking_desc", "Allow users to connect AniList and sync reading progress.")}</p>
                  </div>
                </label>
              </div>
            </div>
          </div>

          {/* ────── Registration & Guest ────── */}
          <div className="card bg-base-100 border border-base-200 shadow-sm">
            <div className="card-body p-4 sm:p-5">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-3">
                <div>
                  <div className="flex items-center gap-2 mb-0.5">
                    <UserPlus className="h-5 w-5 text-primary" />
                    <h2 className="card-title text-lg">{t("settings.registration_guest", "Registration & Guest Access")}</h2>
                  </div>
                  <p className="text-xs text-base-content/50">{t("settings.registration_guest_desc", "Control who can register and what guests can see.")}</p>
                </div>
                <button
                  onClick={handleRegistrationSave}
                  disabled={isSaving("Registration & Guest")}
                  className="btn btn-primary btn-sm gap-1 shrink-0 self-start sm:self-center"
                >
                  {isSaving("Registration & Guest") ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                  {t("settings.save_access", "Save Access")}
                </button>
              </div>

              <div className="flex flex-col gap-3">
                <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary"
                    checked={registration}
                    onChange={(e) => setRegistration(e.target.checked)}
                  />
                  <div>
                    <span className="text-sm font-medium">{t("settings.public_registration", "Enable Public Registration")}</span>
                    <p className="text-xs text-base-content/50">{t("settings.public_registration_desc", "Allow new users to create accounts.")}</p>
                  </div>
                </label>

                <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary"
                    checked={requireEmailVerify}
                    onChange={(e) => setRequireEmailVerify(e.target.checked)}
                    disabled={!adminSettings?.smtp?.enabled}
                  />
                  <div>
                    <span className="text-sm font-medium">{t("settings.require_email_verify", "Require Email Verification")}</span>
                    <p className="text-xs text-base-content/50">
                      {adminSettings?.smtp?.enabled
                        ? t("settings.require_email_verify_desc", "New accounts must confirm an emailed code before they are created.")
                        : t("settings.needs_smtp", "Configure SMTP below to enable this.")}
                    </p>
                  </div>
                </label>

                <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary"
                    checked={passwordResetEnabled}
                    onChange={(e) => setPasswordResetEnabled(e.target.checked)}
                    disabled={!adminSettings?.smtp?.enabled}
                  />
                  <div>
                    <span className="text-sm font-medium">{t("settings.password_reset_enabled", "Enable Password Reset by Email")}</span>
                    <p className="text-xs text-base-content/50">
                      {adminSettings?.smtp?.enabled
                        ? t("settings.password_reset_enabled_desc", "Shows a \"Forgot password?\" link and lets users reset with an emailed code.")
                        : t("settings.needs_smtp", "Configure SMTP below to enable this.")}
                    </p>
                  </div>
                </label>

                <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary"
                    checked={loginRequired}
                    onChange={(e) => setLoginRequired(e.target.checked)}
                  />
                  <div>
                    <span className="text-sm font-medium">{t("settings.login_required", "Require Login")}</span>
                    <p className="text-xs text-base-content/50">{t("settings.login_required_desc", "Force all users to sign in to access any content.")}</p>
                  </div>
                </label>

                <div className="flex flex-col gap-1.5">
                  <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1 flex items-center gap-1">
                    <Eye className="h-3 w-3" /> {t("settings.guest_mode", "Guest Access Mode")}
                  </label>
                  <select
                    className="select select-bordered w-full"
                    value={guestMode}
                    onChange={(e) => setGuestMode(e.target.value)}
                    disabled={loginRequired}
                  >
                    {GUEST_MODES.map((m) => (
                      <option key={m.value} value={m.value}>{t(m.labelKey)}</option>
                    ))}
                  </select>
                </div>
                {guestMode === "selected_libraries" && (
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">{t("settings.allowed_libraries", "Allowed Libraries")}</label>
                    <LibraryMultiSelect
                      ids={guestLibraryIds}
                      libraries={libraries}
                      onChange={setGuestLibraryIds}
                    />
                  </div>
                )}
              </div>
            </div>
          </div>

          <RuntimeLimitsCard />

          <div className="card bg-base-100 border border-base-200 shadow-sm">
            <div className="card-body p-4 sm:p-5">
              <SmtpSettingsTab settings={adminSettings} />
            </div>
          </div>

          {/* ────── Webhooks Integration ────── */}
          <div className="card bg-base-100 border border-base-200 shadow-sm">
            <div className="card-body p-4 sm:p-5">
              <WebhooksTab />
            </div>
          </div>
        </div>
      </div>

      {selectedCropImage && (
        <ImageCropperModal
          imageSrc={selectedCropImage}
          onCrop={handleCropApply}
          onCancel={() => setSelectedCropImage(null)}
          cropSize={512}
        />
      )}
    </div>
  );
}
