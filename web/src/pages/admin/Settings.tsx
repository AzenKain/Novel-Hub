import { LibraryMultiSelect } from "@/components/admin/settings/LibraryMultiSelect";
import { GUEST_MODES, POLICY_MODES, SIDEBAR_LABELS } from "@/constants";
import { useAdminSettingsQuery, useLibrariesQuery } from "@/hooks";
import { invalidatePublicSettings } from "@/hooks/useSettings";
import { adminService } from "@/services";
import { useSettingsAdminStore } from "@/stores";
import { useQueryClient } from "@tanstack/react-query";
import {
  Bookmark,
  Download,
  Eye,
  Globe,
  Home,
  Layout,
  Library,
  Loader2,
  MessageSquareText,
  RefreshCw,
  Save,
  UserPlus,
} from "lucide-react";
import { FormEvent, useEffect, useState } from "react";
import { toast } from "react-toastify";
import { useShallow } from "zustand/react/shallow";

import { ImageCropperModal } from "@/components/common/ImageCropperModal";

export function Settings() {
  const queryClient = useQueryClient();
  const { data: adminSettings, isLoading: settingsLoading, refetch: refetchSettings } = useAdminSettingsQuery();
  const { data: librariesList, isLoading: librariesLoading, refetch: refetchLibraries } = useLibrariesQuery();

  const {
    settings, setSettings,
    libraries, setLibraries,
    loading, setLoading,
    saving, setSaving,
    site, setSite,
    sidebarItems, setSidebarItems,
    homeSections, setHomeSections,
    registration, setRegistration,
    guestMode, setGuestMode,
    guestLibraryIds, setGuestLibraryIds,
    downloadMode, setDownloadMode,
    downloadLibraryIds, setDownloadLibraryIds,
    bookmarkMode, setBookmarkMode,
    bookmarkLibraryIds, setBookmarkLibraryIds,
    collectionMode, setCollectionMode,
    collectionLibraryIds, setCollectionLibraryIds,
    reviewMode, setReviewMode,
    reviewLibraryIds, setReviewLibraryIds,
    reset
  } = useSettingsAdminStore(useShallow((state) => ({
    settings: state.settings, setSettings: state.setSettings,
    libraries: state.libraries, setLibraries: state.setLibraries,
    loading: state.loading, setLoading: state.setLoading,
    saving: state.saving, setSaving: state.setSaving,
    site: state.site, setSite: state.setSite,
    sidebarItems: state.sidebarItems, setSidebarItems: state.setSidebarItems,
    homeSections: state.homeSections, setHomeSections: state.setHomeSections,
    registration: state.registration, setRegistration: state.setRegistration,
    guestMode: state.guestMode, setGuestMode: state.setGuestMode,
    guestLibraryIds: state.guestLibraryIds, setGuestLibraryIds: state.setGuestLibraryIds,
    downloadMode: state.downloadMode, setDownloadMode: state.setDownloadMode,
    downloadLibraryIds: state.downloadLibraryIds, setDownloadLibraryIds: state.setDownloadLibraryIds,
    bookmarkMode: state.bookmarkMode, setBookmarkMode: state.setBookmarkMode,
    bookmarkLibraryIds: state.bookmarkLibraryIds, setBookmarkLibraryIds: state.setBookmarkLibraryIds,
    collectionMode: state.collectionMode, setCollectionMode: state.setCollectionMode,
    collectionLibraryIds: state.collectionLibraryIds, setCollectionLibraryIds: state.setCollectionLibraryIds,
    reviewMode: state.reviewMode, setReviewMode: state.setReviewMode,
    reviewLibraryIds: state.reviewLibraryIds, setReviewLibraryIds: state.setReviewLibraryIds,
    reset: state.reset
  })));

  const [uploadingLogo, setUploadingLogo] = useState(false);
  const [uploadingFavicon, setUploadingFavicon] = useState(false);
  const [selectedCropImage, setSelectedCropImage] = useState<string | null>(null);
  const [cropTarget, setCropTarget] = useState<"logo" | "favicon" | null>(null);

  useEffect(() => {
    if (adminSettings) {
      setSettings(adminSettings);
      setSite(adminSettings.site || { title: "", description: "", favicon: "", logo: "", meta_description: "" });
      setSidebarItems(adminSettings.sidebar_visible_items || []);
      setHomeSections(adminSettings.home_sections || { random_books: true, top_books: true });
      setRegistration(adminSettings.registration_enabled);
      setGuestMode(adminSettings.guest_access.mode);
      setGuestLibraryIds(adminSettings.guest_access.library_ids || []);
      setDownloadMode(adminSettings.download.mode);
      setDownloadLibraryIds(adminSettings.download.library_ids || []);
      setBookmarkMode(adminSettings.bookmark.mode);
      setBookmarkLibraryIds(adminSettings.bookmark.library_ids || []);
      setCollectionMode(adminSettings.collection.mode);
      setCollectionLibraryIds(adminSettings.collection.library_ids || []);
      setReviewMode(adminSettings.review.mode);
      setReviewLibraryIds(adminSettings.review.library_ids || []);
    }
  }, [adminSettings]);

  useEffect(() => {
    if (librariesList) {
      setLibraries(librariesList);
    }
  }, [librariesList]);

  useEffect(() => {
    setLoading(settingsLoading || librariesLoading);
  }, [settingsLoading, librariesLoading, setLoading]);

  useEffect(() => {
    return () => {
      reset();
    };
  }, [reset]);

  async function saveSection(section: string, data: Record<string, unknown>) {
    setSaving(section);
    try {
      await adminService.updateSettings(data);
      toast.success(`${section} saved`);
      await invalidatePublicSettings();
      await queryClient.invalidateQueries({ queryKey: ["admin", "settings"] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(null);
    }
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

  function handleSiteSave(e: FormEvent) {
    e.preventDefault();
    void saveSection("Site settings", {
      "site.title": site.title,
      "site.description": site.description,
      "site.favicon": site.favicon,
      "site.logo": site.logo,
      "site.meta_description": site.meta_description,
    });
  }

  function handleSidebarSave() {
    void saveSection("Sidebar", { "sidebar.visible_items": sidebarItems });
  }

  function handleHomeSave() {
    void saveSection("Home sections", { "home.sections": homeSections });
  }

  function handleRegistrationSave() {
    void saveSection("Registration & Guest", {
      "auth.registration_enabled": registration,
      "guest_access.mode": guestMode,
      "guest_access.library_ids": guestMode === "selected_libraries" ? guestLibraryIds : [],
    });
  }

  function handlePolicySave(prefix: string) {
    const modeMap: Record<string, string> = {
      download: downloadMode,
      bookmark: bookmarkMode,
      collection: collectionMode,
      review: reviewMode,
    };
    const idsMap: Record<string, string[]> = {
      download: downloadLibraryIds,
      bookmark: bookmarkLibraryIds,
      collection: collectionLibraryIds,
      review: reviewLibraryIds,
    };
    const mode = modeMap[prefix];
    void saveSection(`${prefix} policy`, {
      [`${prefix}.mode`]: mode,
      [`${prefix}.library_ids`]: mode === "selected_libraries" ? idsMap[prefix] : [],
    });
  }

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center bg-base-100">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }



  const isSaving = (s: string) => saving === s;

  return (
    <div className="flex flex-col h-full bg-base-100">
      {/* Header */}
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex items-center justify-between bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
          <p className="text-sm text-base-content/60 mt-1">Website customization, policies, and access control</p>
        </div>
        <button
          onClick={() => {
            void refetchSettings();
            void refetchLibraries();
          }}
          className="btn btn-square btn-ghost btn-sm sm:btn-md"
          title="Refresh"
        >
          <RefreshCw className={`h-5 w-5 ${loading ? "animate-spin" : ""}`} />
        </button>
      </header>

      <div className="flex-1 overflow-auto p-4 sm:p-6 lg:p-8">
        <div className="max-w-4xl mx-auto space-y-4">
          {/* ────── Site Settings ────── */}
          <div className="card bg-base-100 border border-base-200 shadow-sm">
            <div className="card-body p-5 sm:p-6">
              <div className="flex items-center gap-2 mb-1">
                <Globe className="h-5 w-5 text-primary" />
                <h2 className="card-title text-lg">Site Information</h2>
              </div>
              <p className="text-xs text-base-content/50 mb-4">Customize how your library appears to visitors.</p>
              <form onSubmit={handleSiteSave} className="flex flex-col gap-4">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Title</label>
                    <input
                      type="text"
                      className="input input-bordered w-full"
                      value={site.title}
                      onChange={(e) => setSite({ ...site, title: e.target.value })}
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Meta Description</label>
                    <input
                      type="text"
                      className="input input-bordered w-full"
                      value={site.meta_description}
                      onChange={(e) => setSite({ ...site, meta_description: e.target.value })}
                    />
                  </div>
                  <div className="flex flex-col gap-1.5 col-span-1 sm:col-span-2">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Logo URL</label>
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
                            {uploadingLogo ? <Loader2 className="w-4 h-4 animate-spin" /> : "Fetch"}
                          </button>
                        </div>
                        <div className="flex items-center gap-2">
                          <label className="btn btn-sm btn-outline cursor-pointer font-normal border-base-300">
                            Upload Logo
                            <input type="file" accept="image/*" className="hidden" onChange={(e) => handleFileUpload(e, "logo")} />
                          </label>
                          <button 
                            type="button"
                            className="btn btn-sm btn-ghost font-normal"
                            onClick={() => setSite({ ...site, logo: '/logo.svg' })}
                          >
                            Use Default
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
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Favicon URL</label>
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
                            {uploadingFavicon ? <Loader2 className="w-4 h-4 animate-spin" /> : "Fetch"}
                          </button>
                        </div>
                        <div className="flex items-center gap-2">
                          <label className="btn btn-sm btn-outline cursor-pointer font-normal border-base-300">
                            Upload Favicon
                            <input type="file" accept="image/*" className="hidden" onChange={(e) => handleFileUpload(e, "favicon")} />
                          </label>
                          <button 
                            type="button"
                            className="btn btn-sm btn-ghost font-normal"
                            onClick={() => setSite({ ...site, favicon: '/favicon.ico' })}
                          >
                            Use Default
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
                  <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Description</label>
                  <textarea
                    className="textarea textarea-bordered w-full"
                    value={site.description}
                    onChange={(e) => setSite({ ...site, description: e.target.value })}
                    rows={2}
                  />
                </div>
                <div className="flex justify-end">
                  <button
                    type="submit"
                    disabled={isSaving("Site settings")}
                    className="btn btn-primary btn-sm gap-1"
                  >
                    {isSaving("Site settings") ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                    Save Site
                  </button>
                </div>
              </form>
            </div>
          </div>

          {/* ────── Sidebar Visibility ────── */}
          <div className="card bg-base-100 border border-base-200 shadow-sm">
            <div className="card-body p-5 sm:p-6">
              <div className="flex items-center gap-2 mb-1">
                <Layout className="h-5 w-5 text-primary" />
                <h2 className="card-title text-lg">Sidebar Navigation</h2>
              </div>
              <p className="text-xs text-base-content/50 mb-4">Choose which navigation items appear in the library sidebar.</p>
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2 mb-4">
                {(settings?.available_sidebar_items || []).map((key) => (
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
                    <span className="text-xs font-medium">{SIDEBAR_LABELS[key] || key}</span>
                  </label>
                ))}
              </div>
              <div className="flex justify-end">
                <button
                  onClick={handleSidebarSave}
                  disabled={isSaving("Sidebar")}
                  className="btn btn-primary btn-sm gap-1"
                >
                  {isSaving("Sidebar") ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                  Save Sidebar
                </button>
              </div>
            </div>
          </div>

          {/* ────── Home Sections ────── */}
          <div className="card bg-base-100 border border-base-200 shadow-sm">
            <div className="card-body p-5 sm:p-6">
              <div className="flex items-center gap-2 mb-1">
                <Home className="h-5 w-5 text-primary" />
                <h2 className="card-title text-lg">Home Page Sections</h2>
              </div>
              <p className="text-xs text-base-content/50 mb-4">Enable or disable sections on the library home page.</p>
              <div className="flex flex-col gap-3 mb-4">
                <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary"
                    checked={homeSections.random_books}
                    onChange={(e) => setHomeSections({ ...homeSections, random_books: e.target.checked })}
                  />
                  <div>
                    <span className="text-sm font-medium">Random Books</span>
                    <p className="text-xs text-base-content/50">Show random books section on the home page.</p>
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
                    <span className="text-sm font-medium">Top Books</span>
                    <p className="text-xs text-base-content/50">Show top rated / most read books section.</p>
                  </div>
                </label>
              </div>
              <div className="flex justify-end">
                <button
                  onClick={handleHomeSave}
                  disabled={isSaving("Home sections")}
                  className="btn btn-primary btn-sm gap-1"
                >
                  {isSaving("Home sections") ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                  Save Home
                </button>
              </div>
            </div>
          </div>

          {/* ────── Registration & Guest ────── */}
          <div className="card bg-base-100 border border-base-200 shadow-sm">
            <div className="card-body p-5 sm:p-6">
              <div className="flex items-center gap-2 mb-1">
                <UserPlus className="h-5 w-5 text-primary" />
                <h2 className="card-title text-lg">Registration & Guest Access</h2>
              </div>
              <p className="text-xs text-base-content/50 mb-4">Control who can register and what guests can see.</p>
              <div className="flex flex-col gap-4">
                <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary"
                    checked={registration}
                    onChange={(e) => setRegistration(e.target.checked)}
                  />
                  <div>
                    <span className="text-sm font-medium">Enable Public Registration</span>
                    <p className="text-xs text-base-content/50">Allow new users to create accounts.</p>
                  </div>
                </label>
                <div className="flex flex-col gap-1.5">
                  <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1 flex items-center gap-1">
                    <Eye className="h-3 w-3" /> Guest Access Mode
                  </label>
                  <select
                    className="select select-bordered w-full"
                    value={guestMode}
                    onChange={(e) => setGuestMode(e.target.value)}
                  >
                    {GUEST_MODES.map((m) => (
                      <option key={m.value} value={m.value}>{m.label}</option>
                    ))}
                  </select>
                </div>
                {guestMode === "selected_libraries" && (
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Allowed Libraries</label>
                    <LibraryMultiSelect
                      ids={guestLibraryIds}
                      libraries={libraries}
                      onChange={setGuestLibraryIds}
                    />
                  </div>
                )}
              </div>
              <div className="flex justify-end mt-4">
                <button
                  onClick={handleRegistrationSave}
                  disabled={isSaving("Registration & Guest")}
                  className="btn btn-primary btn-sm gap-1"
                >
                  {isSaving("Registration & Guest") ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                  Save Access
                </button>
              </div>
            </div>
          </div>

          {/* ────── Feature Policies ────── */}
          {[
            {
              key: "download",
              icon: Download,
              title: "Download Policy",
              desc: "Control who can download book files.",
              mode: downloadMode,
              setMode: setDownloadMode,
              ids: downloadLibraryIds,
              setIds: setDownloadLibraryIds,
            },
            {
              key: "bookmark",
              icon: Bookmark,
              title: "Bookmark Policy",
              desc: "Control who can bookmark books.",
              mode: bookmarkMode,
              setMode: setBookmarkMode,
              ids: bookmarkLibraryIds,
              setIds: setBookmarkLibraryIds,
            },
            {
              key: "collection",
              icon: Library,
              title: "Collection Policy",
              desc: "Control who can create and manage collections.",
              mode: collectionMode,
              setMode: setCollectionMode,
              ids: collectionLibraryIds,
              setIds: setCollectionLibraryIds,
            },
            {
              key: "review",
              icon: MessageSquareText,
              title: "Review Policy",
              desc: "Control who can submit and read reviews.",
              mode: reviewMode,
              setMode: setReviewMode,
              ids: reviewLibraryIds,
              setIds: setReviewLibraryIds,
            },
          ].map((policy) => (
            <div key={policy.key} className="card bg-base-100 border border-base-200 shadow-sm">
              <div className="card-body p-5 sm:p-6">
                <div className="flex items-center gap-2 mb-1">
                  <policy.icon className="h-5 w-5 text-primary" />
                  <h2 className="card-title text-lg">{policy.title}</h2>
                </div>
                <p className="text-xs text-base-content/50 mb-4">{policy.desc}</p>
                <div className="flex flex-col gap-3">
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Mode</label>
                    <select
                      className="select select-bordered w-full"
                      value={policy.mode}
                      onChange={(e) => policy.setMode(e.target.value)}
                    >
                      {POLICY_MODES.map((m) => (
                        <option key={m.value} value={m.value}>{m.label}</option>
                      ))}
                    </select>
                  </div>
                  {policy.mode === "selected_libraries" && (
                    <div className="flex flex-col gap-1.5">
                      <label className="text-xs font-bold uppercase tracking-wider opacity-60 pl-1">Allowed Libraries</label>
                      <LibraryMultiSelect
                        ids={policy.ids}
                        libraries={libraries}
                        onChange={policy.setIds}
                      />
                    </div>
                  )}
                </div>
                <div className="flex justify-end mt-4">
                  <button
                    onClick={() => handlePolicySave(policy.key)}
                    disabled={isSaving(`${policy.key} policy`)}
                    className="btn btn-primary btn-sm gap-1"
                  >
                    {isSaving(`${policy.key} policy`) ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                    Save {policy.key}
                  </button>
                </div>
              </div>
            </div>
          ))}
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
