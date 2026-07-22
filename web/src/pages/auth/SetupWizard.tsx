import { SIDEBAR_LABELS } from "@/constants";
import { invalidatePublicSettings } from "@/hooks/useSettings";
import { settingsService } from "@/services";
import { BookOpen, Loader2, Image as ImageIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ImageCropperModal, PasswordStrength } from "@/components/common";

export function SetupWizard() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [setupRequired, setSetupRequired] = useState(false);

  const [form, setForm] = useState({
    username: "",
    email: "",
    password: "",
    site_title: "NovelHub",
    site_description: "Local novel library manager",
    logo: "/logo.svg",
    favicon: "/favicon.ico",
    registration: true,
    guest_mode: "all" as string,
    download_mode: "all" as string,
    bookmark_mode: "all" as string,
    collection_mode: "all" as string,
    review_mode: "all" as string,
    share_mode: "all" as string,
    read_mode: "all" as string,
    stats_mode: "all" as string,
    sidebar_visible_items: Object.keys(SIDEBAR_LABELS),
  });

  const [uploadingLogo, setUploadingLogo] = useState(false);
  const [uploadingFavicon, setUploadingFavicon] = useState(false);
  const [selectedCropImage, setSelectedCropImage] = useState<string | null>(null);
  const [cropTarget, setCropTarget] = useState<"logo" | "favicon" | null>(null);

  useEffect(() => {
    settingsService.getSetupStatus().then((res) => {
      if (res.status && res.data?.required) {
        setSetupRequired(true);
      } else {
        navigate("/", { replace: true });
      }
      setLoading(false);
    }).catch(() => {
      setLoading(false);
    });
  }, [navigate]);

  const handleSubmit = async (e: React.SyntheticEvent) => {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const res = await settingsService.submitSetup({
        username: form.username,
        email: form.email,
        password: form.password,
        site_title: form.site_title,
        site_description: form.site_description,
        logo: form.logo,
        favicon: form.favicon,
        registration: form.registration,
        guest_mode: form.guest_mode,
        read_mode: form.read_mode,
        download_mode: form.download_mode,
        bookmark_mode: form.bookmark_mode,
        collection_mode: form.collection_mode,
        review_mode: form.review_mode,
        share_mode: form.share_mode,
        stats_mode: form.stats_mode,
        sidebar_visible_items: form.sidebar_visible_items,
      });
      if (res.status) {
        // Refresh the cached public settings so the SetupGuard no longer sees
        // the stale `setup_completed: false` and bounces us back to /setup.
        await invalidatePublicSettings();
        navigate("/", { replace: true });
      } else {
        setError(res.message || "Setup failed");
      }
    } catch {
      setError("An unexpected error occurred");
    } finally {
      setSubmitting(false);
    }
  };

  const handleUploadLink = async (type: "logo" | "favicon") => {
    const url = form[type];
    if (!url || !url.startsWith("http")) return;
    
    if (type === "logo") setUploadingLogo(true);
    else setUploadingFavicon(true);
    
    const fd = new FormData();
    fd.append("url", url);
    fd.append("target", type);
    try {
      const res = await settingsService.uploadSetupLogo(fd);
      if (res.status && res.data?.url) {
        setForm({ ...form, [type]: res.data.url });
      } else {
        setError(res.message || `Failed to fetch ${type}`);
      }
    } catch {
      setError(`Failed to fetch ${type}`);
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
    e.target.value = ""; // Reset input
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
      
      const res = await settingsService.uploadSetupLogo(fd);
      if (res.status && res.data?.url) {
        setForm({ ...form, [target]: res.data.url });
      } else {
        setError(res.message || `Failed to upload ${target}`);
      }
    } catch {
      setError(`Failed to upload ${target}`);
    } finally {
      if (target === "logo") setUploadingLogo(false);
      else setUploadingFavicon(false);
    }
  };


  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center bg-base-200">
        <span className="loading loading-spinner loading-lg text-primary" />
      </div>
    );
  }

  if (!setupRequired) return null;

  return (
    <div className="min-h-screen bg-base-200 flex items-center justify-center p-4">
      <div className="card w-full max-w-2xl bg-base-100 shadow-xl">
        <div className="card-body">
          <div className="flex flex-col items-center gap-2 mb-4">
            <div className="w-14 h-14 rounded-2xl bg-primary/10 flex items-center justify-center">
              <BookOpen size={28} className="text-primary" />
            </div>
            <h2 className="text-2xl font-bold">NovelHub Setup</h2>
            <p className="text-sm text-base-content/60">
              Create your root administrator account and configure initial settings.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <fieldset className="fieldset">
              <legend className="fieldset-legend font-semibold">Admin Account</legend>
              <input
                type="text"
                placeholder="Username"
                className="input input-bordered w-full"
                value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })}
                required
              />
              <input
                type="email"
                placeholder="Email"
                className="input input-bordered w-full"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                required
              />
              <div className="flex flex-col gap-1 w-full">
                <input
                  type="password"
                  placeholder="Password (min 8 characters)"
                  className="input input-bordered w-full"
                  value={form.password}
                  onChange={(e) => setForm({ ...form, password: e.target.value })}
                  required
                  minLength={8}
                />
                {form.password.length > 0 && <PasswordStrength password={form.password} />}
              </div>
            </fieldset>

            <fieldset className="fieldset">
              <legend className="fieldset-legend font-semibold">Site Info & Branding</legend>
              <input
                type="text"
                placeholder="Site title"
                className="input input-bordered w-full"
                value={form.site_title}
                onChange={(e) => setForm({ ...form, site_title: e.target.value })}
              />
              <input
                type="text"
                placeholder="Site description"
                className="input input-bordered w-full"
                value={form.site_description}
                onChange={(e) => setForm({ ...form, site_description: e.target.value })}
              />
              <div className="flex flex-col gap-4">
                <div className="flex gap-2 items-start">
                  <div className="flex flex-col gap-2 flex-1">
                    <div className="join w-full">
                      <input
                        type="text"
                        placeholder="Logo URL (e.g. /pwa-192x192.png or https://...)"
                        className="input input-bordered join-item w-full h-12"
                        value={form.logo}
                        onChange={(e) => setForm({ ...form, logo: e.target.value })}
                      />
                      <button 
                        type="button" 
                        className="btn btn-primary join-item h-12"
                        onClick={() => handleUploadLink("logo")}
                        disabled={!form.logo.startsWith('http') || uploadingLogo}
                      >
                        {uploadingLogo ? <Loader2 className="w-4 h-4 animate-spin" /> : "Fetch"}
                      </button>
                    </div>
                    
                    <div className="flex items-center gap-2">
                      <label className="btn btn-sm btn-outline cursor-pointer font-normal">
                        Upload Logo
                        <input 
                          type="file" 
                          className="hidden" 
                          accept="image/*"
                          onChange={(e) => handleFileUpload(e, "logo")} 
                        />
                      </label>
                      <button 
                        type="button"
                        className="btn btn-sm btn-ghost font-normal"
                        onClick={() => setForm({ ...form, logo: '/logo.svg' })}
                      >
                        Use Default
                      </button>
                    </div>
                  </div>
                  
                  <div className="w-auto h-12 px-3 rounded-lg bg-base-100 border border-base-300 flex items-center justify-center overflow-hidden shrink-0">
                    {form.logo ? (
                      <div className="flex items-center gap-2">
                        <img src={form.logo} alt="Logo preview" className="w-8 h-8 rounded" onError={(e) => { e.currentTarget.style.display = 'none'; e.currentTarget.nextElementSibling?.classList.remove('hidden') }} onLoad={(e) => { e.currentTarget.style.display = 'block'; e.currentTarget.nextElementSibling?.classList.add('hidden') }} />
                        <ImageIcon className={`w-5 h-5 opacity-40 hidden`} />
                        <span className="font-bold whitespace-nowrap">{form.site_title || "NovelHub"}</span>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2">
                        <ImageIcon className="w-5 h-5 opacity-40" />
                        <span className="font-bold whitespace-nowrap">{form.site_title || "NovelHub"}</span>
                      </div>
                    )}
                  </div>
                </div>

                <div className="flex gap-2 items-start">
                  <div className="flex flex-col gap-2 flex-1">
                    <div className="join w-full">
                      <input
                        type="text"
                        placeholder="Favicon URL (e.g. /favicon.ico or https://...)"
                        className="input input-bordered join-item w-full h-12"
                        value={form.favicon}
                        onChange={(e) => setForm({ ...form, favicon: e.target.value })}
                      />
                      <button 
                        type="button" 
                        className="btn btn-primary join-item h-12"
                        onClick={() => handleUploadLink("favicon")}
                        disabled={!form.favicon.startsWith('http') || uploadingFavicon}
                      >
                        {uploadingFavicon ? <Loader2 className="w-4 h-4 animate-spin" /> : "Fetch"}
                      </button>
                    </div>
                    
                    <div className="flex items-center gap-2">
                      <label className="btn btn-sm btn-outline cursor-pointer font-normal">
                        Upload Favicon
                        <input 
                          type="file" 
                          className="hidden" 
                          accept="image/*"
                          onChange={(e) => handleFileUpload(e, "favicon")} 
                        />
                      </label>
                      <button 
                        type="button"
                        className="btn btn-sm btn-ghost font-normal"
                        onClick={() => setForm({ ...form, favicon: '/favicon.ico' })}
                      >
                        Use Default
                      </button>
                    </div>
                  </div>
                  
                  <div className="w-12 h-12 rounded-lg bg-base-100 border border-base-300 flex items-center justify-center overflow-hidden shrink-0">
                    {form.favicon ? (
                      <img src={form.favicon} alt="Favicon preview" className="w-8 h-8 rounded" onError={(e) => { e.currentTarget.style.display = 'none'; e.currentTarget.nextElementSibling?.classList.remove('hidden') }} onLoad={(e) => { e.currentTarget.style.display = 'block'; e.currentTarget.nextElementSibling?.classList.add('hidden') }} />
                    ) : null}
                    <ImageIcon className={`w-5 h-5 opacity-40 ${form.favicon ? 'hidden' : ''}`} />
                  </div>
                </div>
              </div>
            </fieldset>
            
            <fieldset className="fieldset">
              <legend className="fieldset-legend font-semibold">Sidebar Navigation</legend>
              <div className="text-xs text-base-content/60 mb-2">Select which navigation links are visible on the library sidebar.</div>
              <div className="grid grid-cols-2 gap-y-3 gap-x-4 max-h-56 overflow-y-auto pb-1">
                {Object.keys(SIDEBAR_LABELS).map((key) => (
                  <label key={key} className="flex items-center gap-2 cursor-pointer hover:opacity-80 transition-opacity">
                    <input
                      type="checkbox"
                      className="checkbox checkbox-xs checkbox-primary"
                      checked={form.sidebar_visible_items.includes(key)}
                      onChange={(e) => {
                        const checked = e.target.checked;
                        setForm((prev) => ({
                          ...prev,
                          sidebar_visible_items: checked
                            ? [...prev.sidebar_visible_items, key]
                            : prev.sidebar_visible_items.filter((i) => i !== key)
                        }));
                      }}
                    />
                    <span className="text-xs font-medium">{SIDEBAR_LABELS[key]}</span>
                  </label>
                ))}
              </div>
            </fieldset>

            <fieldset className="fieldset">
              <legend className="fieldset-legend font-semibold">Policies & Access Control</legend>
              <div className="flex items-center gap-2 mb-2">
                <input
                  type="checkbox"
                  className="toggle toggle-primary toggle-sm"
                  checked={form.registration}
                  onChange={(e) => setForm({ ...form, registration: e.target.checked })}
                />
                <span className="text-sm font-medium">Enable public registration</span>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 my-1">
                <div className="form-control w-full">
                  <label className="label py-0.5"><span className="label-text text-xs font-semibold">Guest access</span></label>
                  <select className="select select-bordered select-sm w-full" value={form.guest_mode}
                    onChange={(e) => setForm({ ...form, guest_mode: e.target.value })}>
                    <option value="all">All libraries</option>
                    <option value="selected_libraries">Selected libraries</option>
                    <option value="login_required">Login required</option>
                  </select>
                </div>

                {["read", "download", "bookmark", "collection", "review", "share", "stats"].map((policy) => (
                  <div key={policy} className="form-control w-full">
                    <label className="label py-0.5">
                      <span className="label-text text-xs font-semibold capitalize">{policy} policy</span>
                    </label>
                    <select className="select select-bordered select-sm w-full"
                      value={form[`${policy}_mode` as keyof typeof form] as string}
                      onChange={(e) => setForm({ ...form, [`${policy}_mode`]: e.target.value })}>
                      <option value="all">All</option>
                      <option value="disabled">Disabled</option>
                      <option value="selected_libraries">Selected libraries</option>
                    </select>
                  </div>
                ))}
              </div>
            </fieldset>

            {error && (
              <div className="alert alert-error py-2 text-sm rounded-lg">{error}</div>
            )}

            <button className="btn btn-primary mt-2" disabled={submitting}>
              {submitting ? <Loader2 className="animate-spin" size={20} /> : null}
              Complete Setup
            </button>
          </form>
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
