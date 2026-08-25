import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useSearchParams, useNavigate } from "react-router-dom";
import { getMediaUrl } from "@/config/api";
import {
  User,
  Shield,
  Smartphone,
  BarChart3,
  Palette,
  Camera,
  Key,
  Check,
  Globe,
  ArrowLeft,
} from "lucide-react";
import { toast } from "react-toastify";
import { useShallow } from "zustand/react/shallow";

import { useAuthStore } from "@/stores";
import {
  useChangePasswordMutation,
  useUpdateProfileMutation,
  useUploadAvatarMutation,
} from "@/hooks";
import { ImageCropperModal } from "@/components/common/ImageCropperModal";
import { PasswordStrength } from "@/components/common/PasswordStrength";
import { TopNav } from "@/components/common/TopNav";

// Profile Cards
import { TwoFactorCard } from "@/components/profile/TwoFactorCard";
import { KidsModePinCard } from "@/components/profile/KidsModePinCard";
import { EReaderMagicCodeCard } from "@/components/profile/EReaderMagicCodeCard";
import { UserDevicesCard } from "@/components/profile/UserDevicesCard";
import { OPDSSyncCard } from "@/components/profile/OPDSSyncCard";
import { KoboSyncCard } from "@/components/profile/KoboSyncCard";
import { VBookSyncCard } from "@/components/profile/VBookSyncCard";
import { ReadingHeatmap } from "@/components/profile/ReadingHeatmap";
import { TrackerConnectCard } from "@/components/profile/TrackerConnectCard";
import { HardcoverTrackerCard } from "@/components/profile/HardcoverTrackerCard";
import { ReadwiseConnectCard } from "@/components/profile/ReadwiseConnectCard";
import { SoundscapesCard } from "@/components/profile/SoundscapesCard";
import { FontsCard } from "@/components/profile/FontsCard";
import { CustomThemesCard } from "@/components/profile/CustomThemesCard";
import { CustomCSSCard } from "@/components/profile/CustomCSSCard";

type ProfileTab = "account" | "devices" | "trackers" | "customization";

export const ProfilePage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const currentTab = (searchParams.get("tab") as ProfileTab) || "account";

  const { user } = useAuthStore(
    useShallow((state) => ({
      user: state.user,
    }))
  );

  const updateProfileMutation = useUpdateProfileMutation();
  const uploadAvatarMutation = useUploadAvatarMutation();
  const changePasswordMutation = useChangePasswordMutation();

  const [fullName, setFullName] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  const [selectedImage, setSelectedImage] = useState<string | null>(null);

  // Password change state
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordError, setPasswordError] = useState("");
  const [passwordSuccess, setPasswordSuccess] = useState(false);

  useEffect(() => {
    if (!user) {
      navigate("/login", { replace: true });
      return;
    }
    setFullName(user.full_name || "");
    setAvatarUrl(user.avatar_url || "");
  }, [user, navigate]);

  if (!user) return null;

  const setTab = (tab: ProfileTab) => {
    setSearchParams({ tab }, { replace: true });
  };

  const handleBack = () => {
    if (window.history.length > 1) {
      navigate(-1);
    } else {
      navigate("/");
    }
  };

  const base64ToBlob = (base64: string): Blob => {
    const parts = base64.split(";base64,");
    const contentType = parts[0].split(":")[1];
    const raw = window.atob(parts[1]);
    const rawLength = raw.length;
    const uInt8Array = new Uint8Array(rawLength);
    for (let i = 0; i < rawLength; ++i) {
      uInt8Array[i] = raw.charCodeAt(i);
    }
    return new Blob([uInt8Array], { type: contentType });
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (event) => {
        if (event.target?.result) {
          setSelectedImage(event.target.result as string);
        }
      };
      reader.readAsDataURL(file);
    }
  };

  const handleCropApply = async (base64: string) => {
    setSelectedImage(null);
    try {
      const blob = base64ToBlob(base64);
      const file = new File([blob], "avatar.png", { type: blob.type });

      const uploadedUrl = await uploadAvatarMutation.mutateAsync(file);
      setAvatarUrl(uploadedUrl);

      await updateProfileMutation.mutateAsync({
        full_name: fullName,
        avatar_url: uploadedUrl,
      });
      toast.success(t("profile.avatar_updated", "Avatar updated successfully"));
    } catch {
      toast.error(t("profile.avatar_failed", "Failed to update avatar"));
    }
  };

  const handleSaveProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await updateProfileMutation.mutateAsync({
        full_name: fullName.trim(),
        avatar_url: avatarUrl,
      });
      toast.success(t("profile.save_success", "Profile updated successfully"));
    } catch {
      toast.error(t("profile.save_failed", "Failed to update profile"));
    }
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordError("");
    setPasswordSuccess(false);

    if (newPassword !== confirmPassword) {
      setPasswordError(t("profile.passwords_do_not_match", "New passwords do not match"));
      return;
    }

    try {
      await changePasswordMutation.mutateAsync({
        old_password: oldPassword,
        new_password: newPassword,
      });
      setPasswordSuccess(true);
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
      toast.success(t("profile.password_updated", "Password changed successfully"));
    } catch (err: any) {
      setPasswordError(err?.response?.data?.message || t("profile.password_change_failed", "Failed to change password"));
    }
  };

  const tabs: { id: ProfileTab; label: string; icon: React.ReactNode; badge?: string }[] = [
    {
      id: "account",
      label: t("profile.tab_account", "Account & Security"),
      icon: <Shield className="w-4 h-4" />,
    },
    {
      id: "devices",
      label: t("profile.tab_devices", "Devices & Sync"),
      icon: <Smartphone className="w-4 h-4" />,
    },
    {
      id: "trackers",
      label: t("profile.tab_trackers", "Reading & Trackers"),
      icon: <BarChart3 className="w-4 h-4" />,
    },
    {
      id: "customization",
      label: t("profile.tab_customization", "Customization & Audio"),
      icon: <Palette className="w-4 h-4" />,
      badge: "NEW",
    },
  ];

  return (
    <div className="min-h-screen bg-base-200/50 pb-16">
      <TopNav showSidebarToggle={false} />

      {/* Header Banner */}
      <div className="bg-base-100 border-b border-base-content/10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div className="flex items-center gap-4">
              <button
                type="button"
                onClick={handleBack}
                className="btn btn-circle btn-sm btn-ghost"
                title={t("common.back", "Back")}
              >
                <ArrowLeft className="w-5 h-5" />
              </button>

              <div className="relative group">
                <div className="avatar">
                  <div className="w-16 h-16 rounded-2xl ring-2 ring-primary ring-offset-2 ring-offset-base-100 overflow-hidden bg-base-300">
                    {avatarUrl ? (
                      <img src={getMediaUrl(avatarUrl, undefined, user?.updated_at)} alt={fullName || user?.email} />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center font-bold text-xl text-base-content/60">
                        {(fullName || user?.email || "U").charAt(0).toUpperCase()}
                      </div>
                    )}
                  </div>
                </div>
                <label className="absolute -bottom-1 -right-1 p-1.5 rounded-lg bg-primary text-primary-content cursor-pointer shadow-md hover:scale-105 transition-transform">
                  <Camera className="w-3.5 h-3.5" />
                  <input
                    type="file"
                    className="hidden"
                    accept="image/png,image/jpeg,image/webp"
                    onChange={handleFileChange}
                  />
                </label>
              </div>

              <div>
                <h1 className="text-xl sm:text-2xl font-bold flex items-center gap-2">
                  {fullName || user?.email || t("profile.title", "User Profile")}
                  {user?.auth_provider && (
                    <span className="badge badge-sm badge-ghost uppercase text-[10px]">
                      {user.auth_provider}
                    </span>
                  )}
                </h1>
                <p className="text-xs sm:text-sm opacity-60 flex items-center gap-2">
                  <span>{user?.email}</span>
                  {user?.roles && user.roles.length > 0 && (
                    <span className="badge badge-primary badge-outline badge-xs text-[10px]">
                      {user.roles.map((r) => r.name).join(", ")}
                    </span>
                  )}
                </p>
              </div>
            </div>

            {/* Quick action: Return to Library */}
            <button
              type="button"
              onClick={() => navigate("/")}
              className="btn btn-outline btn-sm rounded-xl gap-2 self-start sm:self-center"
            >
              <Globe className="w-4 h-4" />
              {t("profile.return_home", "Go to Library")}
            </button>
          </div>

          {/* Navigation Tabs Bar */}
          <div className="flex items-center gap-1 mt-6 border-b border-base-content/10 overflow-x-auto custom-scrollbar">
            {tabs.map((tab) => {
              const isActive = currentTab === tab.id;
              return (
                <button
                  key={tab.id}
                  type="button"
                  onClick={() => setTab(tab.id)}
                  className={`flex items-center gap-2 py-3 px-4 border-b-2 font-medium text-xs sm:text-sm transition-all whitespace-nowrap ${
                    isActive
                      ? "border-primary text-primary font-bold"
                      : "border-transparent text-base-content/70 hover:text-base-content hover:border-base-content/20"
                  }`}
                >
                  {tab.icon}
                  <span>{tab.label}</span>
                  {tab.badge && (
                    <span className="badge badge-primary badge-xs text-[9px] font-bold">
                      {tab.badge}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>
      </div>

      {/* Main Tab Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-6">
        {currentTab === "account" && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* General Profile Info Card */}
            <div className="card bg-base-100 shadow-xl border border-base-content/10">
              <div className="card-body p-5 sm:p-6">
                <div className="flex items-center gap-3 border-b border-base-content/10 pb-4">
                  <div className="p-2.5 rounded-xl bg-primary/10 text-primary">
                    <User className="w-5 h-5" />
                  </div>
                  <div>
                    <h2 className="card-title text-base sm:text-lg">
                      {t("profile.basic_info", "Basic Information")}
                    </h2>
                    <p className="text-xs opacity-60">
                      {t("profile.basic_info_desc", "Update your display name and public credentials.")}
                    </p>
                  </div>
                </div>

                <form onSubmit={handleSaveProfile} className="space-y-4 mt-4">
                  <div>
                    <label className="label label-text text-xs">{t("profile.full_name", "Full Name")}</label>
                    <input
                      type="text"
                      value={fullName}
                      onChange={(e) => setFullName(e.target.value)}
                      className="input input-bordered input-sm w-full rounded-xl"
                      placeholder="e.g. John Doe"
                    />
                  </div>

                  <div>
                    <label className="label label-text text-xs">{t("profile.email", "Email Address")}</label>
                    <input
                      type="email"
                      value={user?.email || ""}
                      disabled
                      className="input input-bordered input-sm w-full rounded-xl bg-base-200 opacity-70"
                    />
                    <span className="text-[11px] opacity-50 block mt-1">
                      {t("profile.email_immutable", "Email address is tied to your account authentication.")}
                    </span>
                  </div>

                  <div className="flex justify-end pt-2">
                    <button
                      type="submit"
                      disabled={updateProfileMutation.isPending}
                      className="btn btn-primary btn-sm rounded-xl gap-2"
                    >
                      {updateProfileMutation.isPending && <span className="loading loading-spinner loading-xs" />}
                      {t("common.save", "Save Changes")}
                    </button>
                  </div>
                </form>
              </div>
            </div>

            {/* Change Password Card (Only for LOCAL accounts) */}
            {user?.auth_provider === "local" || !user?.auth_provider ? (
              <div className="card bg-base-100 shadow-xl border border-base-content/10">
                <div className="card-body p-5 sm:p-6">
                  <div className="flex items-center gap-3 border-b border-base-content/10 pb-4">
                    <div className="p-2.5 rounded-xl bg-warning/10 text-warning">
                      <Key className="w-5 h-5" />
                    </div>
                    <div>
                      <h2 className="card-title text-base sm:text-lg">
                        {t("profile.change_password", "Change Password")}
                      </h2>
                      <p className="text-xs opacity-60">
                        {t("profile.change_password_desc", "Keep your account secure with a strong password.")}
                      </p>
                    </div>
                  </div>

                  <form onSubmit={handleChangePassword} className="space-y-3 mt-4">
                    {passwordError && (
                      <div className="alert alert-error text-xs p-2 rounded-xl">
                        {passwordError}
                      </div>
                    )}
                    {passwordSuccess && (
                      <div className="alert alert-success text-xs p-2 rounded-xl flex items-center gap-2">
                        <Check className="w-4 h-4" />
                        {t("profile.password_updated", "Password changed successfully")}
                      </div>
                    )}

                    <div>
                      <label className="label label-text text-xs">{t("profile.current_password", "Current Password")}</label>
                      <input
                        type="password"
                        value={oldPassword}
                        onChange={(e) => setOldPassword(e.target.value)}
                        className="input input-bordered input-sm w-full rounded-xl"
                        required
                      />
                    </div>

                    <div>
                      <label className="label label-text text-xs">{t("profile.new_password", "New Password")}</label>
                      <input
                        type="password"
                        value={newPassword}
                        onChange={(e) => setNewPassword(e.target.value)}
                        className="input input-bordered input-sm w-full rounded-xl"
                        required
                      />
                      <PasswordStrength password={newPassword} />
                    </div>

                    <div>
                      <label className="label label-text text-xs">{t("profile.confirm_password", "Confirm New Password")}</label>
                      <input
                        type="password"
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                        className="input input-bordered input-sm w-full rounded-xl"
                        required
                      />
                    </div>

                    <div className="flex justify-end pt-2">
                      <button
                        type="submit"
                        disabled={changePasswordMutation.isPending}
                        className="btn btn-warning btn-sm rounded-xl gap-2"
                      >
                        {changePasswordMutation.isPending && <span className="loading loading-spinner loading-xs" />}
                        {t("profile.update_password_btn", "Update Password")}
                      </button>
                    </div>
                  </form>
                </div>
              </div>
            ) : null}

            {/* Security: 2FA & Parental PIN */}
            <TwoFactorCard />
            <KidsModePinCard />
          </div>
        )}

        {currentTab === "devices" && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <EReaderMagicCodeCard />
              <UserDevicesCard />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <OPDSSyncCard />
              <KoboSyncCard />
              <VBookSyncCard />
            </div>
          </div>
        )}

        {currentTab === "trackers" && (
          <div className="space-y-6">
            <ReadingHeatmap />
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <TrackerConnectCard />
              <HardcoverTrackerCard />
              <ReadwiseConnectCard />
            </div>
          </div>
        )}

        {currentTab === "customization" && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <SoundscapesCard />
              <FontsCard />
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <CustomThemesCard />
              <CustomCSSCard />
            </div>
          </div>
        )}
      </main>

      {/* Image Cropper Modal */}
      {selectedImage && (
        <ImageCropperModal
          imageSrc={selectedImage}
          onCancel={() => setSelectedImage(null)}
          onCrop={handleCropApply}
          cropSize={256}
        />
      )}
    </div>
  );
};
