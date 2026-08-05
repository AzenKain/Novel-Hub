import React, { useState } from "react";
import { Cpu, Plus, Smartphone, Tablet, Trash2, HardDrive } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useCreateDeviceMutation, useDeleteDeviceMutation, useDevicesQuery } from "@/hooks";

function getDeviceIcon(type: string) {
  switch (type?.toLowerCase()) {
    case "kindle":
      return <Tablet className="h-5 w-5 text-warning" />;
    case "pocketbook":
      return <Smartphone className="h-5 w-5 text-info" />;
    case "koreader":
      return <Cpu className="h-5 w-5 text-success" />;
    default:
      return <HardDrive className="h-5 w-5 text-primary" />;
  }
}

export const UserDevicesCard: React.FC = () => {
  const { t } = useTranslation();
  const [showAddForm, setShowAddForm] = useState(false);
  const [name, setName] = useState("");
  const [deviceType, setDeviceType] = useState("kindle");
  const [targetAddress, setTargetAddress] = useState("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const { data: devices = [], isLoading } = useDevicesQuery(true, { limit: 50 });
  const createMutation = useCreateDeviceMutation();
  const deleteMutation = useDeleteDeviceMutation();

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);

    if (!name.trim()) {
      setErrorMsg("Please enter a device name.");
      return;
    }
    if (!targetAddress.trim()) {
      setErrorMsg("Please enter a target email or endpoint address.");
      return;
    }

    createMutation.mutate(
      { name: name.trim(), device_type: deviceType, target_address: targetAddress.trim() },
      {
        onSuccess: (res) => {
          if (!res.status) {
            setErrorMsg(res.message || "Failed to create device.");
            return;
          }
          setName("");
          setTargetAddress("");
          setShowAddForm(false);
        },
        onError: (err: any) => {
          setErrorMsg(err?.message || "Failed to create device.");
        },
      }
    );
  };

  const handleDelete = (id: string) => {
    deleteMutation.mutate(id);
  };

  return (
    <div className="rounded-2xl border border-base-300 bg-base-100 p-6 shadow-sm space-y-4">
      <div className="flex items-center justify-between border-b border-base-200 pb-3">
        <div className="flex items-center gap-3">
          <div className="grid h-10 w-10 place-items-center rounded-xl bg-primary/10 text-primary">
            <HardDrive className="h-5 w-5" />
          </div>
          <div>
            <h3 className="text-base font-bold">
              {t("device.management", "Multi-Device Delivery Center")}
            </h3>
            <p className="text-xs text-base-content/60">
              {t("device.management_desc", "Manage reading devices for 1-click book delivery (Kindle, PocketBook, KOReader).")}
            </p>
          </div>
        </div>
        <button
          className="btn btn-primary btn-sm gap-1.5 rounded-xl"
          onClick={() => setShowAddForm(!showAddForm)}
        >
          <Plus className="h-4 w-4" />
          {t("device.add_new", "Add Device")}
        </button>
      </div>

      {showAddForm && (
        <form onSubmit={handleCreate} className="p-4 rounded-xl border border-base-200 bg-base-200/40 space-y-3">
          <h4 className="text-xs font-bold uppercase tracking-wider text-base-content/70">
            {t("device.register_device", "Register New Reading Device")}
          </h4>

          {errorMsg && (
            <div className="alert alert-error text-xs py-2 px-3 rounded-lg">
              <span>{errorMsg}</span>
            </div>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <div>
              <label className="block text-[11px] font-semibold text-base-content/70 mb-1">
                {t("device.name_label", "Device Name")}
              </label>
              <input
                type="text"
                placeholder="e.g. My Paperwhite 11"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="input input-bordered input-sm w-full text-xs"
                required
              />
            </div>
            <div>
              <label className="block text-[11px] font-semibold text-base-content/70 mb-1">
                {t("device.type_label", "Device Type")}
              </label>
              <select
                value={deviceType}
                onChange={(e) => setDeviceType(e.target.value)}
                className="select select-bordered select-sm w-full text-xs"
              >
                <option value="kindle">Kindle (Email)</option>
                <option value="pocketbook">PocketBook (@pbsync.com)</option>
                <option value="koreader">KOReader (HTTP Endpoint)</option>
              </select>
            </div>
            <div>
              <label className="block text-[11px] font-semibold text-base-content/70 mb-1">
                {deviceType === "koreader"
                  ? t("device.endpoint_label", "HTTP Endpoint")
                  : t("device.email_label", "Device Email")}
              </label>
              <input
                type={deviceType === "koreader" ? "text" : "email"}
                placeholder={
                  deviceType === "koreader"
                    ? "http://192.168.1.50:8080/push"
                    : "user@kindle.com / user@pbsync.com"
                }
                value={targetAddress}
                onChange={(e) => setTargetAddress(e.target.value)}
                className="input input-bordered input-sm w-full text-xs"
                required
              />
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-1">
            <button
              type="button"
              className="btn btn-ghost btn-xs"
              onClick={() => setShowAddForm(false)}
            >
              {t("admin.cancel", "Cancel")}
            </button>
            <button
              type="submit"
              className="btn btn-primary btn-xs"
              disabled={createMutation.isPending}
            >
              {createMutation.isPending && <span className="loading loading-spinner loading-xs" />}
              {t("common.save", "Save Device")}
            </button>
          </div>
        </form>
      )}

      {isLoading ? (
        <div className="flex justify-center py-6">
          <span className="loading loading-spinner loading-md text-primary" />
        </div>
      ) : devices.length === 0 ? (
        <div className="text-center py-8 border border-dashed border-base-200 rounded-xl">
          <p className="text-xs text-base-content/60">
            {t("device.no_saved_devices", "No reading devices registered yet.")}
          </p>
          <p className="text-[11px] text-base-content/40 mt-1">
            {t("device.register_tip", "Click 'Add Device' above to connect your Kindle, PocketBook, or KOReader.")}
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {devices.map((device) => (
            <div
              key={device.id}
              className="flex items-center justify-between p-3.5 border border-base-200 rounded-xl bg-base-100 hover:border-primary/30 transition-all"
            >
              <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-xl bg-base-200/60">
                  {getDeviceIcon(device.device_type)}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-semibold text-xs text-base-content">{device.name}</span>
                    <span className="badge badge-xs uppercase font-mono tracking-wider font-semibold opacity-70">
                      {device.device_type}
                    </span>
                  </div>
                  <div className="text-xs font-mono text-base-content/60 mt-0.5 truncate max-w-xs sm:max-w-md">
                    {device.target_address}
                  </div>
                </div>
              </div>

              <button
                className="btn btn-ghost btn-sm text-error hover:bg-error/10 btn-square rounded-lg"
                title={t("common.delete", "Delete")}
                onClick={() => handleDelete(device.id)}
                disabled={deleteMutation.isPending}
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
