import { ImageCropperModal } from "@/components/common/ImageCropperModal";
import { PasswordStrength } from "@/components/common/PasswordStrength";
import { ReadingHeatmap } from "@/components/profile/ReadingHeatmap";
import { OPDSSyncCard } from "@/components/profile/OPDSSyncCard";
import { VBookSyncCard } from "@/components/profile/VBookSyncCard";
import { KoboSyncCard } from "@/components/profile/KoboSyncCard";
import { TwoFactorCard } from "@/components/profile/TwoFactorCard";
import { TrackerConnectCard } from "@/components/profile/TrackerConnectCard";
import { UserDevicesCard } from "@/components/profile/UserDevicesCard";
import { useChangePasswordMutation, useUpdateProfileMutation } from "@/hooks";
import { useAuthStore } from "@/stores";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";
import { useShallow } from "zustand/react/shallow";
import { X, User, Key, Activity } from "lucide-react";

export const UserProfile = () => {
  const { user, isProfileModalOpen, setProfileModalOpen } = useAuthStore(
    useShallow((state) => ({
      user: state.user,
      isProfileModalOpen: state.isProfileModalOpen,
      setProfileModalOpen: state.setProfileModalOpen,
    }))
  );

  const updateProfileMutation = useUpdateProfileMutation();
  const { t } = useTranslation();
  
  const [fullName, setFullName] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [success, setSuccess] = useState(false);

  // URL input states
  const [urlInputOpen, setUrlInputOpen] = useState(false);
  const [tempUrl, setTempUrl] = useState("");

  // Crop states
  const [selectedImage, setSelectedImage] = useState<string | null>(null);

  // Password states
  const changePasswordMutation = useChangePasswordMutation();
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordError, setPasswordError] = useState("");

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

  const handleSave = (e: React.SyntheticEvent) => {
    e.preventDefault();
    setSuccess(false);
    updateProfileMutation.mutate(
      { full_name: fullName, avatar_url: avatarUrl },
      {
        onSuccess: () => {
          setSuccess(true);
          setTimeout(() => setSuccess(false), 3000);
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : t('user.profile_update_failed', 'Could not save your profile'));
        },
      }
    );
  };

  const handleChangePassword = (e: React.SyntheticEvent) => {
    e.preventDefault();
    setPasswordError("");
    if (newPassword !== confirmPassword) {
      setPasswordError(t('user.password_mismatch', 'New passwords do not match'));
      return;
    }
    changePasswordMutation.mutate(
      { old_password: oldPassword, new_password: newPassword },
      {
        onSuccess: () => {
          setProfileModalOpen(false);
        },
        onError: (err) => {
          setPasswordError(err instanceof Error ? err.message : t('user.password_change_failed', 'Could not change your password'));
        },
      }
    );
  };

  const closeModal = () => {
    setProfileModalOpen(false);
    setSelectedImage(null);
    setUrlInputOpen(false);
  };

  return (
    <dialog className={`modal ${isProfileModalOpen && user ? "modal-open" : ""}`}>
      {user && (
        <div className="modal-box max-w-4xl w-11/12 max-h-[90vh] flex flex-col p-0 overflow-hidden rounded-2xl shadow-2xl border border-base-300 bg-base-100">
          {/* Sticky Header */}
          <div className="sticky top-0 z-30 flex items-center justify-between px-6 py-4 bg-base-100/95 backdrop-blur-md border-b border-base-200">
            <div className="flex items-center gap-3">
              <div className="grid h-9 w-9 place-items-center rounded-xl bg-primary/10 text-primary">
                <User className="h-5 w-5" />
              </div>
              <h3 className="font-bold text-lg text-base-content">
                {t('user.profile_title', 'Your Profile')}
              </h3>
            </div>
            <button 
              type="button"
              onClick={closeModal}
              className="btn btn-md btn-circle btn-ghost text-base-content/70 hover:text-base-content hover:bg-base-200"
              title={t('common.close', 'Close')}
            >
              <X className="h-6 w-6" />
            </button>
          </div>

          {/* Scrollable Body Content */}
          <div className="flex-1 overflow-y-auto p-6 space-y-4 custom-scrollbar">
            {updateProfileMutation.error && (
              <div className="alert alert-error py-2 text-sm rounded-xl">
                <span>{updateProfileMutation.error instanceof Error ? updateProfileMutation.error.message : String(updateProfileMutation.error)}</span>
              </div>
            )}
            
            {success && (
              <div className="alert alert-success py-2 text-sm rounded-xl">
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
                {/* Account Details & Main Settings Card */}
                <div className="rounded-2xl border border-base-300 bg-base-200/40 p-5 space-y-4">
                  <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 border-b border-base-200 pb-4">
                    <div className="flex items-center gap-4">
                      <div className="avatar">
                        <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center border border-primary/20 text-primary font-bold text-xl overflow-hidden shadow-sm">
                          {avatarUrl ? (
                            <img src={avatarUrl} alt="Avatar" loading="lazy" className="object-cover w-full h-full" />
                          ) : (
                            <span>{fullName ? fullName.charAt(0).toUpperCase() : user.email.charAt(0).toUpperCase()}</span>
                          )}
                        </div>
                      </div>
                      <div>
                        <div className="font-bold text-base text-base-content flex items-center gap-2">
                          {user.email}
                          <span className="badge badge-ghost badge-sm uppercase font-semibold text-base-content/70">{user.auth_provider}</span>
                        </div>
                        <div className="flex flex-wrap gap-2 mt-2">
                          <label className="btn btn-xs btn-primary !text-white cursor-pointer">
                            <span className="!text-white font-medium">{t('user.upload_avatar', 'Upload Photo')}</span>
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
                          <div className="flex gap-1.5 items-center mt-2">
                            <input
                              type="text"
                              placeholder="https://example.com/avatar.png"
                              value={tempUrl}
                              onChange={(e) => setTempUrl(e.target.value)}
                              className="input input-bordered input-xs w-56 focus:input-primary font-mono text-xs"
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

                    {user.auth_provider === "LOCAL" && (
                      <button
                        type="button"
                        onClick={() => {
                          setPasswordOpen((v) => !v);
                          setPasswordError("");
                          setOldPassword("");
                          setNewPassword("");
                          setConfirmPassword("");
                        }}
                        className="btn btn-sm btn-outline gap-1.5 shrink-0"
                      >
                        <Key className="w-4 h-4" />
                        {t('user.change_password', 'Change Password')}
                      </button>
                    )}
                  </div>

                  {/* Profile Edit Form */}
                  <form onSubmit={handleSave} className="space-y-3">
                    <div className="flex flex-col sm:flex-row gap-3 items-start sm:items-center">
                      <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 w-28 shrink-0">
                        {t('user.full_name', 'Full Name')}
                      </label>
                      <div className="flex-1 flex gap-2 w-full">
                        <input
                          type="text"
                          className="input input-bordered input-sm flex-1 focus:input-primary"
                          value={fullName}
                          onChange={(e) => setFullName(e.target.value)}
                          placeholder={t('user.full_name_placeholder', 'Your full name')}
                        />
                        <button
                          type="submit"
                          disabled={updateProfileMutation.isPending}
                          className="btn btn-primary btn-sm shrink-0"
                        >
                          {updateProfileMutation.isPending ? <span className="loading loading-spinner loading-xs"></span> : t('user.save_changes', 'Save Changes')}
                        </button>
                      </div>
                    </div>
                  </form>

                  {/* Password Drawer */}
                  {passwordOpen && (
                    <form onSubmit={handleChangePassword} className="flex flex-col gap-3 pt-3 border-t border-base-200">
                      {passwordError && (
                        <div className="alert alert-error py-2 text-xs rounded-lg">
                          <span>{passwordError}</span>
                        </div>
                      )}
                      {changePasswordMutation.error && (
                        <div className="alert alert-error py-2 text-xs rounded-lg">
                          <span>{changePasswordMutation.error instanceof Error ? changePasswordMutation.error.message : String(changePasswordMutation.error)}</span>
                        </div>
                      )}
                      <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                        <input
                          type="password"
                          autoComplete="current-password"
                          className="input input-bordered input-sm w-full focus:input-primary"
                          value={oldPassword}
                          onChange={(e) => setOldPassword(e.target.value)}
                          placeholder={t('user.current_password', 'Current password')}
                          required
                        />
                        <input
                          type="password"
                          autoComplete="new-password"
                          className="input input-bordered input-sm w-full focus:input-primary"
                          value={newPassword}
                          onChange={(e) => setNewPassword(e.target.value)}
                          placeholder={t('user.new_password', 'New password')}
                          required
                        />
                        <input
                          type="password"
                          autoComplete="new-password"
                          className="input input-bordered input-sm w-full focus:input-primary"
                          value={confirmPassword}
                          onChange={(e) => setConfirmPassword(e.target.value)}
                          placeholder={t('user.confirm_password', 'Confirm new password')}
                          required
                        />
                      </div>
                      {newPassword.length > 0 && <PasswordStrength password={newPassword} />}
                      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-2 pt-1">
                        <p className="text-xs text-base-content/60">
                          {t('user.password_relogin_note', 'You will be signed out after changing your password.')}
                        </p>
                        <button
                          type="submit"
                          disabled={changePasswordMutation.isPending}
                          className="btn btn-primary btn-xs self-end sm:self-auto"
                        >
                          {changePasswordMutation.isPending ? <span className="loading loading-spinner loading-xs"></span> : t('user.update_password', 'Update Password')}
                        </button>
                      </div>
                    </form>
                  )}
                </div>

                {/* Two-Factor Authentication Card */}
                <TwoFactorCard />

                {/* Reading Heatmap Card */}
                <div className="rounded-2xl border border-base-300 bg-base-100 p-6 shadow-sm space-y-4">
                  <div className="flex items-center gap-3 border-b border-base-200 pb-3">
                    <div className="grid h-10 w-10 place-items-center rounded-xl bg-primary/10 text-primary">
                      <Activity className="h-5 w-5" />
                    </div>
                    <h3 className="text-base font-bold">
                      {t("analytics.activity_grid", "Annual Reading Activity")}
                    </h3>
                  </div>
                  <ReadingHeatmap showTitle={false} />
                </div>

                {/* OPDS 2.0 & Progress Sync Card */}
                <OPDSSyncCard />

                {/* VBook Plugin Card */}
                <VBookSyncCard />

                {/* Kobo Native Wi-Fi Sync Card */}
                <KoboSyncCard />

                {/* Multi-Device Push Center Card */}
                <UserDevicesCard />

                {/* AniList Tracker Connect Card */}
                <TrackerConnectCard />
              </>
            )}
          </div>
        </div>
      )}
      <form method="dialog" className="modal-backdrop">
        <button onClick={closeModal}>close</button>
      </form>
    </dialog>
  );
};
