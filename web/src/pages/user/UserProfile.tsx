import { useAuthStore } from "@/stores";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { ImageCropperModal } from "@/components/common/ImageCropperModal";

export const UserProfile = () => {
  const navigate = useNavigate();
  const { user, updateProfile, logout, loading, error, clearError, isProfileModalOpen, setProfileModalOpen } = useAuthStore();
  const { t } = useTranslation();
  
  const [fullName, setFullName] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [success, setSuccess] = useState(false);

  // URL input states
  const [urlInputOpen, setUrlInputOpen] = useState(false);
  const [tempUrl, setTempUrl] = useState("");

  // Crop states
  const [selectedImage, setSelectedImage] = useState<string | null>(null);

  useEffect(() => {
    if (user) {
      setFullName(user.full_name || "");
      setAvatarUrl(user.avatar_url || "");
    }
  }, [user]);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (event) => {
        if (event.target?.result) {
          setSelectedImage(event.target.result as string);
          setUrlInputOpen(false);
        }
      };
      reader.readAsDataURL(file);
    }
  };

  const handleCropApply = (base64: string) => {
    setAvatarUrl(base64);
    setSelectedImage(null);
  };


  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    clearError();
    setSuccess(false);
    try {
      await updateProfile({ full_name: fullName, avatar_url: avatarUrl });
      setSuccess(true);
      setTimeout(() => setSuccess(false), 3000);
    } catch (err) {
      // Error handled by store
    }
  };


  return (
    <dialog className={`modal ${isProfileModalOpen && user ? "modal-open" : ""}`}>
      {user && (
      <div className="modal-box">
        <button 
          onClick={() => {
            setProfileModalOpen(false);
            setSelectedImage(null);
            setUrlInputOpen(false);
          }}
          className="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
        >
          ✕
        </button>
        <h3 className="font-bold text-lg mb-6">{t('user.profile_title', 'Your Profile')}</h3>

        {error && (
          <div className="alert alert-error py-2 text-sm rounded-lg mb-4">
            <span>{error}</span>
          </div>
        )}
        
        {success && (
          <div className="alert alert-success py-2 text-sm rounded-lg mb-4">
            <span>{t('user.profile_success', 'Profile updated successfully!')}</span>
          </div>
        )}

        {selectedImage ? (
          <ImageCropperModal
            imageSrc={selectedImage}
            onCrop={handleCropApply}
            onCancel={() => setSelectedImage(null)}
            cropSize={200}
          />
        ) : (
          <>
            <div className="flex items-center gap-4 mb-6">
              <div className="avatar">
                <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center border border-primary/20 text-primary font-bold text-xl overflow-hidden">
                  {avatarUrl ? (
                    <img src={avatarUrl} alt="Avatar" loading="lazy" className="object-cover w-full h-full" />
                  ) : (
                    <span>{fullName ? fullName.charAt(0).toUpperCase() : user.email.charAt(0).toUpperCase()}</span>
                  )}
                </div>
              </div>
              <div>
                <div className="font-bold text-lg leading-tight">{user.email}</div>
                <div className="flex flex-col gap-2 mt-2">
                  <div className="flex gap-2">
                    <label className="btn btn-xs btn-outline btn-primary cursor-pointer">
                      {t('user.upload_avatar', 'Upload Photo')}
                      <input
                        type="file"
                        accept="image/*"
                        className="hidden"
                        onChange={handleFileChange}
                      />
                    </label>
                    <button
                      type="button"
                      onClick={() => setUrlInputOpen(!urlInputOpen)}
                      className="btn btn-xs btn-outline"
                    >
                      {t('user.load_url', 'From URL')}
                    </button>
                    {avatarUrl && (
                      <button
                        type="button"
                        onClick={() => setAvatarUrl("")}
                        className="btn btn-xs btn-ghost text-error"
                      >
                        {t('user.remove_avatar', 'Remove')}
                      </button>
                    )}
                  </div>
                  
                  {urlInputOpen && (
                    <div className="flex gap-1 items-center">
                      <input
                        type="text"
                        placeholder="https://example.com/avatar.png"
                        value={tempUrl}
                        onChange={(e) => setTempUrl(e.target.value)}
                        className="input input-bordered input-xs w-48 focus:input-primary"
                      />
                      <button
                        type="button"
                        onClick={() => {
                          if (tempUrl.trim()) {
                            setSelectedImage(tempUrl.trim());
                            setUrlInputOpen(false);
                          }
                        }}
                        className="btn btn-xs btn-primary font-bold"
                      >
                        {t('common.ok', 'OK')}
                      </button>
                    </div>
                  )}
                </div>
              </div>
            </div>

            <form onSubmit={handleSave} className="flex flex-col gap-4">
              <div className="flex flex-col gap-1.5 w-full">
                <label className="text-sm font-semibold pl-1">
                  {t('user.full_name', 'Full Name')}
                </label>
                <input
                  type="text"
                  className="input input-bordered w-full focus:input-primary"
                  value={fullName}
                  onChange={(e) => setFullName(e.target.value)}
                  placeholder={t('user.full_name_placeholder', 'Your full name')}
                />
              </div>
              
              <div className="modal-action mt-2">
                <button
                  type="button"
                  onClick={() => {
                    setProfileModalOpen(false);
                    setSelectedImage(null);
                  }}
                  className="btn btn-outline border-base-300"
                >
                  {t('common.cancel', 'Cancel')}
                </button>
                <button
                  type="submit"
                  disabled={loading}
                  className="btn btn-primary"
                >
                  {loading ? <span className="loading loading-spinner"></span> : t('user.save_changes', 'Save Changes')}
                </button>
              </div>
            </form>
          </>
        )}
      </div>
      )}
      <form method="dialog" className="modal-backdrop">
        <button onClick={() => {
          setProfileModalOpen(false);
          setSelectedImage(null);
        }}>close</button>
      </form>
    </dialog>
  );
};

