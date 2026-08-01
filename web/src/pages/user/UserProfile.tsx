import { ImageCropperModal } from "@/components/common/ImageCropperModal";
import { PasswordStrength } from "@/components/common/PasswordStrength";
import { ReadingHeatmap } from '@/components/profile/ReadingHeatmap';
import { TrackerConnectCard } from "@/components/profile/TrackerConnectCard";
import { useChangePasswordMutation, useUpdateProfileMutation } from "@/hooks";
import { useAuthStore } from "@/stores";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useShallow } from "zustand/react/shallow";

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
    // Đổi mật khẩu thu hồi session -> hook clear auth, đóng modal để user đăng nhập lại.
    changePasswordMutation.mutate(
      { old_password: oldPassword, new_password: newPassword },
      {
        onSuccess: () => {
          setProfileModalOpen(false);
        },
      }
    );
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

        {updateProfileMutation.error && (
          <div className="alert alert-error py-2 text-sm rounded-lg mb-4">
            <span>{updateProfileMutation.error instanceof Error ? updateProfileMutation.error.message : String(updateProfileMutation.error)}</span>
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
                    <label className="btn btn-xs btn-neutral cursor-pointer">
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
                  disabled={updateProfileMutation.isPending}
                  className="btn btn-primary"
                >
                  {updateProfileMutation.isPending ? <span className="loading loading-spinner"></span> : t('user.save_changes', 'Save Changes')}
                </button>
              </div>
            </form>

            {user.auth_provider === "LOCAL" && (
              <div className="mt-6 border-t border-base-300 pt-6">
                <button
                  type="button"
                  onClick={() => {
                    setPasswordOpen((v) => !v);
                    setPasswordError("");
                    setOldPassword("");
                    setNewPassword("");
                    setConfirmPassword("");
                  }}
                  className="btn btn-sm btn-outline w-full"
                >
                  {t('user.change_password', 'Change Password')}
                </button>

                {passwordOpen && (
                  <form onSubmit={handleChangePassword} className="flex flex-col gap-3 mt-4">
                    {passwordError && (
                      <div className="alert alert-error py-2 text-sm rounded-lg">
                        <span>{passwordError}</span>
                      </div>
                    )}
                    {changePasswordMutation.error && (
                      <div className="alert alert-error py-2 text-sm rounded-lg">
                        <span>{changePasswordMutation.error instanceof Error ? changePasswordMutation.error.message : String(changePasswordMutation.error)}</span>
                      </div>
                    )}
                    <input
                      type="password"
                      autoComplete="current-password"
                      className="input input-bordered w-full focus:input-primary"
                      value={oldPassword}
                      onChange={(e) => setOldPassword(e.target.value)}
                      placeholder={t('user.current_password', 'Current password')}
                      required
                    />
                    <input
                      type="password"
                      autoComplete="new-password"
                      className="input input-bordered w-full focus:input-primary"
                      value={newPassword}
                      onChange={(e) => setNewPassword(e.target.value)}
                      placeholder={t('user.new_password', 'New password')}
                      required
                    />
                    {newPassword.length > 0 && <PasswordStrength password={newPassword} />}
                    <input
                      type="password"
                      autoComplete="new-password"
                      className="input input-bordered w-full focus:input-primary"
                      value={confirmPassword}
                      onChange={(e) => setConfirmPassword(e.target.value)}
                      placeholder={t('user.confirm_password', 'Confirm new password')}
                      required
                    />
                    <p className="text-xs text-base-content/60">
                      {t('user.password_relogin_note', 'You will be signed out after changing your password.')}
                    </p>
                    <button
                      type="submit"
                      disabled={changePasswordMutation.isPending}
                      className="btn btn-primary btn-sm"
                    >
                      {changePasswordMutation.isPending ? <span className="loading loading-spinner"></span> : t('user.update_password', 'Update Password')}
                    </button>
                  </form>
                )}
              </div>
            )}

            <div className="mt-6 border-t border-base-300 pt-6">
              <ReadingHeatmap />
            </div>

            <div className="mt-6 border-t border-base-300 pt-6">
              <TrackerConnectCard />
            </div>

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
