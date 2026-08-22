import React, { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  X,
  Loader2,
  Eye,
  ChevronUp,
  ChevronDown,
  Palette,
  SlidersHorizontal,
  Bot,
  MessageSquare,
  Edit2,
  RotateCcw,
} from "lucide-react";
import { Webhook, CreateWebhookInput } from "@/types/admin";
import { toast } from "react-toastify";

type WebhookModalProps = {
  open: boolean;
  editingWebhook: Webhook | null;
  onClose: () => void;
  onSave: (input: CreateWebhookInput) => void;
  isSaving: boolean;
};

type FieldConfig = {
  id: string;
  defaultLabel: string;
  customLabel: string;
  enabled: boolean;
};

const DEFAULT_FIELDS: FieldConfig[] = [
  { id: "author", defaultLabel: "👤 Author", customLabel: "👤 Author", enabled: true },
  { id: "publisher", defaultLabel: "🏢 Publisher", customLabel: "🏢 Publisher", enabled: true },
  { id: "language", defaultLabel: "🌐 Language", customLabel: "🌐 Language", enabled: true },
  { id: "series", defaultLabel: "📖 Series", customLabel: "📖 Series", enabled: true },
  { id: "tags", defaultLabel: "🏷️ Tags", customLabel: "🏷️ Tags", enabled: true },
  { id: "cover", defaultLabel: "🖼️ Cover Image", customLabel: "🖼️ Cover Image", enabled: true },
  { id: "description", defaultLabel: "📝 Description", customLabel: "📝 Description", enabled: true },
  { id: "date", defaultLabel: "📅 Release Date", customLabel: "📅 Release Date", enabled: false },
];

const COLOR_SWATCHES = [
  { hex: "#5865F2", name: "Blurple" },
  { hex: "#3B82F6", name: "Blue" },
  { hex: "#10B981", name: "Emerald" },
  { hex: "#F59E0B", name: "Amber" },
  { hex: "#EF4444", name: "Red" },
  { hex: "#8B5CF6", name: "Purple" },
  { hex: "#EC4899", name: "Pink" },
  { hex: "#64748B", name: "Slate" },
];

const AVAILABLE_EVENTS = [
  { id: "book.created", label: "book.created" },
  { id: "book.deleted", label: "book.deleted" },
  { id: "metadata.updated", label: "metadata.updated" },
  { id: "reading.completed", label: "reading.completed" },
  { id: "job.failed", label: "job.failed" },
];

export const WebhookModal: React.FC<WebhookModalProps> = ({
  open,
  editingWebhook,
  onClose,
  onSave,
  isSaving,
}) => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<"general" | "customizer">("general");
  const [previewEvent, setPreviewEvent] = useState<string>("book.created");

  const [form, setForm] = useState<CreateWebhookInput>({
    name: "",
    url: "",
    template_type: "discord",
    secret: "",
    custom_headers: "",
    events: ["book.created", "book.deleted"],
    is_active: true,
  });
  const isEmailTemplate = form.template_type === "email";

  const [embedColor, setEmbedColor] = useState<string>("#5865F2");
  const [botName, setBotName] = useState<string>("NovelHub Bot");
  const [titleTemplate, setTitleTemplate] = useState<string>("📚 {title}");
  const [fields, setFields] = useState<FieldConfig[]>(DEFAULT_FIELDS);
  const [customHeadersJSON, setCustomHeadersJSON] = useState<string>("");

  useEffect(() => {
    if (editingWebhook) {
      setForm({
        name: editingWebhook.name,
        url: editingWebhook.url,
        template_type: editingWebhook.template_type,
        secret: editingWebhook.secret || "",
        custom_headers: editingWebhook.custom_headers || "",
        events: editingWebhook.events || ["book.created", "book.deleted"],
        is_active: editingWebhook.is_active,
      });

      if (editingWebhook.custom_headers) {
        try {
          const parsed = JSON.parse(editingWebhook.custom_headers);
          if (parsed._embed_color) setEmbedColor(parsed._embed_color);
          if (parsed._bot_name) setBotName(parsed._bot_name);
          if (parsed._title_template) setTitleTemplate(parsed._title_template);

          const savedLabels: Record<string, string> = parsed._field_labels || {};

          const httpHeadersOnly = { ...parsed };
          delete httpHeadersOnly._embed_color;
          delete httpHeadersOnly._bot_name;
          delete httpHeadersOnly._title_template;
          delete httpHeadersOnly._field_labels;
          delete httpHeadersOnly._fields;

          if (Object.keys(httpHeadersOnly).length > 0) {
            setCustomHeadersJSON(JSON.stringify(httpHeadersOnly, null, 2));
          } else {
            setCustomHeadersJSON("");
          }

          if (Array.isArray(parsed._fields)) {
            const savedFieldIds: string[] = parsed._fields;
            const updated = [...DEFAULT_FIELDS].map((f) => ({
              ...f,
              customLabel: savedLabels[f.id] || f.defaultLabel,
              enabled: savedFieldIds.includes(f.id),
            }));

            updated.sort((a, b) => {
              const idxA = savedFieldIds.indexOf(a.id);
              const idxB = savedFieldIds.indexOf(b.id);
              if (idxA === -1) return 1;
              if (idxB === -1) return -1;
              return idxA - idxB;
            });
            setFields(updated);
          }
        } catch {
          setCustomHeadersJSON(editingWebhook.custom_headers || "");
        }
      }
    } else {
      setForm({
        name: "",
        url: "",
        template_type: "discord",
        secret: "",
        custom_headers: "",
        events: ["book.created", "book.deleted"],
        is_active: true,
      });
      setEmbedColor("#5865F2");
      setBotName("NovelHub Bot");
      setTitleTemplate("📚 {title}");
      setFields(DEFAULT_FIELDS);
      setCustomHeadersJSON("");
    }
  }, [editingWebhook, open]);

  if (!open) return null;

  const toggleEvent = (evtId: string) => {
    setForm((prev) => {
      const current = prev.events || [];
      if (current.includes(evtId)) {
        return { ...prev, events: current.filter((e) => e !== evtId) };
      }
      return { ...prev, events: [...current, evtId] };
    });
  };

  const toggleFieldEnabled = (fieldId: string) => {
    setFields((prev) =>
      prev.map((f) => (f.id === fieldId ? { ...f, enabled: !f.enabled } : f))
    );
  };

  const updateFieldLabel = (fieldId: string, label: string) => {
    setFields((prev) =>
      prev.map((f) => (f.id === fieldId ? { ...f, customLabel: label } : f))
    );
  };

  const resetFieldLabel = (fieldId: string) => {
    setFields((prev) =>
      prev.map((f) => (f.id === fieldId ? { ...f, customLabel: f.defaultLabel } : f))
    );
  };

  const moveField = (index: number, direction: "up" | "down") => {
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= fields.length) return;
    const copy = [...fields];
    const temp = copy[index];
    copy[index] = copy[targetIndex];
    copy[targetIndex] = temp;
    setFields(copy);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim() || !form.url.trim()) {
      toast.error(t("admin.fill_required", "Please fill in all required fields"));
      return;
    }

    let headersObj: Record<string, any> = {};
    if (customHeadersJSON.trim()) {
      try {
        headersObj = JSON.parse(customHeadersJSON);
      } catch {
        toast.error("Invalid Custom HTTP Headers JSON format");
        return;
      }
    }

    headersObj._embed_color = embedColor;
    headersObj._bot_name = botName;
    headersObj._title_template = titleTemplate;

    const labelsMap: Record<string, string> = {};
    fields.forEach((f) => {
      if (f.customLabel.trim() && f.customLabel !== f.defaultLabel) {
        labelsMap[f.id] = f.customLabel.trim();
      }
    });
    if (Object.keys(labelsMap).length > 0) {
      headersObj._field_labels = labelsMap;
    }

    headersObj._fields = fields.filter((f) => f.enabled).map((f) => f.id);

    const finalInput: CreateWebhookInput = {
      ...form,
      custom_headers: JSON.stringify(headersObj),
    };

    onSave(finalInput);
  };

  const sampleData = {
    rawTitle: "Re:Zero − Starting Life in Another World, Vol. 9",
    author: "Tappei Nagatsuki",
    publisher: "Yen Press",
    language: "English",
    series: "Re:Zero (Vol. 9)",
    tags: ["Isekai", "Fantasy", "Light Novel"],
    description: "Subaru Natsuki confronts the Witch Cult once again to defend the Roswaal mansion and rescue everyone he holds dear...",
    cover_url: "https://images.unsplash.com/photo-1544716278-ca5e3f4abd8c?w=300&q=80",
    date: "2026-07-22",
  };

  const isFieldEnabled = (id: string) => fields.find((f) => f.id === id)?.enabled ?? true;
  const getFieldLabel = (id: string) => {
    const f = fields.find((item) => item.id === id);
    return f?.customLabel || f?.defaultLabel || id;
  };

  const formattedTitle = titleTemplate.replace("{title}", sampleData.rawTitle);

  return (
    <div className="modal modal-open">
      <div className="modal-box max-w-6xl w-11/12 p-0 rounded-3xl bg-base-100 shadow-2xl border border-base-300 overflow-hidden flex flex-col max-h-[90vh]">
        {/* Modal Header */}
        <div className="flex items-center justify-between px-6 py-4 bg-base-200/60 border-b border-base-300">
          <div className="flex items-center gap-3">
            <div className="p-2.5 bg-primary/10 text-primary rounded-xl">
              <Bot className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-bold text-lg leading-tight">
                {editingWebhook ? t("admin.edit_webhook", "Edit Webhook & Live Builder") : t("admin.create_webhook", "Configure New Webhook")}
              </h3>
              <p className="text-xs text-base-content/60">
                {t("admin.webhook_desc", "Customize titles, field labels, embed styles, and test live response preview")}
              </p>
            </div>
          </div>

          <button
            type="button"
            onClick={onClose}
            className="btn btn-sm btn-circle bg-base-200 hover:bg-base-300 text-base-content border border-base-300 shadow-sm flex items-center justify-center transition-all hover:scale-105"
            aria-label={t("common.close", "Close")}
          >
            <X className="w-4 h-4 text-base-content stroke-[2.5]" />
          </button>
        </div>

        {/* Dual Panel Body */}
        <div className="flex flex-col lg:flex-row flex-1 overflow-hidden min-h-0">
          {/* Left Panel: Configuration Form */}
          <form onSubmit={handleSubmit} className="w-full lg:w-1/2 p-6 overflow-y-auto flex flex-col gap-5 border-r border-base-300">
            {/* Tabs for Config vs Customizer */}
            <div className="flex bg-base-200/70 p-1 rounded-xl gap-1">
              <button
                type="button"
                onClick={() => setActiveTab("general")}
                className={`flex-1 py-1.5 px-3 rounded-lg text-xs font-bold transition-all flex items-center justify-center gap-2 ${
                  activeTab === "general" ? "bg-base-100 text-primary shadow-sm" : "text-base-content/60 hover:text-base-content"
                }`}
              >
                <SlidersHorizontal className="w-3.5 h-3.5" />
                {t("admin.general_config", "General Settings")}
              </button>
              <button
                type="button"
                onClick={() => setActiveTab("customizer")}
                className={`flex-1 py-1.5 px-3 rounded-lg text-xs font-bold transition-all flex items-center justify-center gap-2 ${
                  activeTab === "customizer" ? "bg-base-100 text-primary shadow-sm" : "text-base-content/60 hover:text-base-content"
                }`}
              >
                <Palette className="w-3.5 h-3.5" />
                {t("admin.embed_customizer", "Titles, Labels & Embed Styling")}
              </button>
            </div>

            {activeTab === "general" ? (
              <>
                {/* Name */}
                <div>
                  <label className="label text-xs font-bold text-base-content">{t("settings.webhook_name", "Webhook Name")} *</label>
                  <input
                    type="text"
                    required
                    placeholder="e.g. Discord Library Channel"
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    className="input input-bordered input-sm w-full font-medium"
                  />
                </div>

                {/* URL */}
                <div>
                  <label className="label text-xs font-bold text-base-content">
                    {isEmailTemplate
                      ? t("settings.email_recipients", "Recipients")
                      : t("settings.endpoint_url", "Endpoint URL")}{" "}
                    *
                  </label>
                  <input
                    type={isEmailTemplate ? "text" : "url"}
                    required
                    placeholder={isEmailTemplate ? "mailto:ops@example.com,team@example.com" : "https://discord.com/api/webhooks/..."}
                    value={form.url}
                    onChange={(e) => setForm({ ...form, url: e.target.value })}
                    className="input input-bordered input-sm w-full font-mono text-xs"
                  />
                  {isEmailTemplate && (
                    <p className="mt-1 text-[11px] text-base-content/50">
                      {t("settings.email_recipients_hint", "Comma-separated addresses after mailto:. Requires SMTP to be configured.")}
                    </p>
                  )}
                </div>

                {/* Platform Template */}
                <div>
                  <label className="label text-xs font-bold text-base-content">{t("settings.payload_format", "Payload Format Platform")}</label>
                  <select
                    value={form.template_type}
                    onChange={(e) => setForm({ ...form, template_type: e.target.value as any })}
                    className="select select-bordered select-sm w-full font-medium"
                  >
                    <option value="discord">Discord Webhook Embed</option>
                    <option value="telegram">Telegram Bot HTML</option>
                    <option value="slack">Slack Block Kit</option>
                    <option value="generic">Generic JSON (Custom / n8n / Zapier)</option>
                    <option value="email">{t("settings.template_email", "Email (SMTP)")}</option>
                  </select>
                </div>

                {/* Subscribed Events Selector */}
                <div>
                  <label className="label text-xs font-bold text-base-content">{t("settings.subscribed_events", "Subscribed Events")}</label>
                  <div className="flex flex-wrap items-center gap-2 mt-1">
                    {AVAILABLE_EVENTS.map((evt) => {
                      const isSelected = form.events.includes(evt.id);
                      return (
                        <button
                          type="button"
                          key={evt.id}
                          onClick={() => toggleEvent(evt.id)}
                          className={`px-3 py-1.5 rounded-xl text-xs font-mono font-bold transition-all flex items-center gap-1.5 border select-none ${
                            isSelected
                              ? "bg-primary text-primary-content border-primary shadow-sm ring-2 ring-primary/20 scale-[1.02]"
                              : "bg-base-200/70 text-base-content/70 border-base-300 hover:bg-base-300 hover:text-base-content"
                          }`}
                        >
                          <span className={`w-1.5 h-1.5 rounded-full ${isSelected ? "bg-primary-content" : "bg-base-content/40"}`} />
                          {evt.label}
                        </button>
                      );
                    })}
                  </div>
                </div>

                {/* Secret Key */}
                <div>
                  <label className="label text-xs font-bold text-base-content">{t("settings.secret_key", "Secret Key (HMAC SHA-256 Signature)")}</label>
                  <input
                    type="text"
                    placeholder={t("settings.secret_key_hint", "Optional secret key for X-NovelHub-Signature header")}
                    value={form.secret || ""}
                    onChange={(e) => setForm({ ...form, secret: e.target.value })}
                    className="input input-bordered input-sm w-full font-mono text-xs"
                  />
                </div>

                {/* Custom HTTP Headers */}
                <div>
                  <label className="label text-xs font-bold text-base-content">{t("settings.custom_headers", "Custom HTTP Headers (JSON format)")}</label>
                  <textarea
                    rows={2}
                    placeholder='{"Authorization": "Bearer token123", "X-Custom-Header": "value"}'
                    value={customHeadersJSON}
                    onChange={(e) => setCustomHeadersJSON(e.target.value)}
                    className="textarea textarea-bordered textarea-sm w-full font-mono text-xs"
                  />
                </div>

                {/* Enable Checkbox */}
                <div className="form-control">
                  <label className="label cursor-pointer justify-start gap-3">
                    <input
                      type="checkbox"
                      checked={form.is_active}
                      onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                      className="checkbox checkbox-primary checkbox-sm"
                    />
                    <span className="label-text font-semibold text-xs">{t("settings.enable_webhook", "Enable Webhook Dispatcher")}</span>
                  </label>
                </div>
              </>
            ) : (
              <>
                {/* Embed Title Template Format Input */}
                <div>
                  <label className="label text-xs font-bold text-base-content flex justify-between">
                    <span>{t("settings.embed_title_format", "Embed Title Format")}</span>
                    <span className="text-[11px] text-base-content/50 font-mono">Use {"{title}"} placeholder</span>
                  </label>
                  <input
                    type="text"
                    value={titleTemplate}
                    onChange={(e) => setTitleTemplate(e.target.value)}
                    placeholder="📚 {title}"
                    className="input input-bordered input-sm w-full text-xs font-medium font-mono"
                  />
                </div>

                {/* Embed Color Picker (for Discord) */}
                {form.template_type === "discord" && (
                  <div className="flex flex-col gap-2">
                    <label className="label text-xs font-bold text-base-content flex items-center justify-between">
                      <span>{t("settings.embed_accent_color", "Discord Embed Accent Color")}</span>
                      <span className="font-mono text-[11px] px-2 py-0.5 rounded bg-base-200" style={{ color: embedColor }}>
                        {embedColor}
                      </span>
                    </label>

                    <div className="flex items-center gap-2 flex-wrap">
                      {COLOR_SWATCHES.map((sw) => (
                        <button
                          type="button"
                          key={sw.hex}
                          onClick={() => setEmbedColor(sw.hex)}
                          className={`w-7 h-7 rounded-full border-2 transition-transform ${
                            embedColor.toLowerCase() === sw.hex.toLowerCase()
                              ? "border-base-content scale-110 shadow-md ring-2 ring-primary/40"
                              : "border-transparent hover:scale-105"
                          }`}
                          style={{ backgroundColor: sw.hex }}
                          title={sw.name}
                        />
                      ))}
                      <input
                        type="color"
                        value={embedColor}
                        onChange={(e) => setEmbedColor(e.target.value)}
                        className="w-7 h-7 rounded-lg cursor-pointer border border-base-300 p-0 bg-transparent"
                      />
                    </div>
                  </div>
                )}

                {/* Bot Name */}
                <div>
                  <label className="label text-xs font-bold text-base-content">Bot Display Name</label>
                  <input
                    type="text"
                    value={botName}
                    onChange={(e) => setBotName(e.target.value)}
                    placeholder="NovelHub Bot"
                    className="input input-bordered input-sm w-full text-xs font-medium"
                  />
                </div>

                {/* Edit Field Labels & Order */}
                <div className="flex flex-col gap-2">
                  <label className="label text-xs font-bold text-base-content flex justify-between">
                    <span>Edit Field Labels & Reorder</span>
                    <span className="text-[11px] text-base-content/60">Rename labels & reorder for embed</span>
                  </label>

                  <div className="flex flex-col gap-2 border border-base-300 rounded-xl p-2.5 bg-base-200/40 max-h-72 overflow-y-auto">
                    {fields.map((f, idx) => (
                      <div
                        key={f.id}
                        className={`flex items-center justify-between gap-2 p-2 rounded-lg text-xs transition-all ${
                          f.enabled ? "bg-base-100 shadow-sm border border-base-200" : "opacity-50 bg-base-200/50"
                        }`}
                      >
                        <div className="flex items-center gap-2 flex-1 min-w-0">
                          <input
                            type="checkbox"
                            checked={f.enabled}
                            onChange={() => toggleFieldEnabled(f.id)}
                            className="checkbox checkbox-primary checkbox-xs shrink-0"
                          />
                          <input
                            type="text"
                            value={f.customLabel}
                            onChange={(e) => updateFieldLabel(f.id, e.target.value)}
                            className="input input-bordered input-xs font-semibold text-xs text-base-content w-full focus:ring-1 focus:ring-primary"
                            placeholder={f.defaultLabel}
                          />
                          {f.customLabel !== f.defaultLabel && (
                            <button
                              type="button"
                              onClick={() => resetFieldLabel(f.id)}
                              className="btn btn-ghost btn-xs btn-square text-base-content/50 hover:text-base-content"
                              title="Reset to default label"
                            >
                              <RotateCcw className="w-3 h-3" />
                            </button>
                          )}
                        </div>

                        <div className="flex items-center gap-1 shrink-0">
                          <button
                            type="button"
                            disabled={idx === 0}
                            onClick={() => moveField(idx, "up")}
                            className="btn btn-ghost btn-xs btn-square hover:bg-base-200 disabled:opacity-20"
                          >
                            <ChevronUp className="w-3.5 h-3.5" />
                          </button>
                          <button
                            type="button"
                            disabled={idx === fields.length - 1}
                            onClick={() => moveField(idx, "down")}
                            className="btn btn-ghost btn-xs btn-square hover:bg-base-200 disabled:opacity-20"
                          >
                            <ChevronDown className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </>
            )}

            {/* Action Buttons */}
            <div className="flex items-center justify-end gap-2 mt-auto pt-4 border-t border-base-300">
              <button type="button" onClick={onClose} className="btn btn-ghost btn-sm">
                {t("common.cancel", "Cancel")}
              </button>
              <button type="submit" disabled={isSaving} className="btn btn-primary btn-sm px-5">
                {isSaving ? <Loader2 className="w-4 h-4 animate-spin" /> : t("common.save", "Save Webhook")}
              </button>
            </div>
          </form>

          {/* Right Panel: Live Real-Time Interactive Preview */}
          <div className="w-full lg:w-1/2 p-6 bg-base-200/60 flex flex-col gap-4 overflow-y-auto">
            {/* Preview Header Bar */}
            <div className="flex items-center justify-between flex-wrap gap-2 pb-2 border-b border-base-300">
              <div className="flex items-center gap-2">
                <Eye className="w-4 h-4 text-primary" />
                <span className="font-bold text-xs uppercase tracking-wider text-base-content/80">
                  Live Webhook Preview ({form.template_type})
                </span>
              </div>

              {/* Sample Event Switcher */}
              <div className="flex bg-base-300/60 p-0.5 rounded-lg gap-1 text-[11px] font-mono">
                {["book.created", "book.deleted"].map((evt) => (
                  <button
                    type="button"
                    key={evt}
                    onClick={() => setPreviewEvent(evt)}
                    className={`px-2 py-0.5 rounded-md transition-all ${
                      previewEvent === evt ? "bg-base-100 font-bold shadow-xs text-primary" : "opacity-60 hover:opacity-100"
                    }`}
                  >
                    {evt}
                  </button>
                ))}
              </div>
            </div>

            {/* Discord Real-Time Live Preview Box (Matches Image 2!) */}
            {form.template_type === "discord" && (
              <div className="bg-[#313338] text-[#dbdee1] p-5 rounded-2xl border border-[#232428] font-sans shadow-xl flex flex-col gap-2">
                {/* Bot Message Header */}
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-[#5865F2] flex items-center justify-center text-white font-bold shrink-0 shadow-sm">
                    📚
                  </div>
                  <div className="flex items-baseline gap-2">
                    <span className="font-bold text-white text-sm hover:underline cursor-pointer">
                      {botName || "NovelHub Bot"}
                    </span>
                    <span className="bg-[#5865F2] text-white text-[10px] px-1.5 py-0.2 rounded font-bold tracking-wider">
                      APP
                    </span>
                    <span className="text-[11px] text-[#949ba4]">Today at 13:10</span>
                  </div>
                </div>

                {/* Embed Container */}
                <div className="ml-13 flex flex-col">
                  <div
                    className="bg-[#2b2d31] rounded-lg p-4 border-l-4 shadow-md flex flex-col gap-3 max-w-lg"
                    style={{
                      borderLeftColor:
                        previewEvent === "book.deleted"
                          ? "#EF4444"
                          : embedColor || "#5865F2",
                    }}
                  >
                    {/* Header Title & Cover Thumbnail */}
                    <div className="flex justify-between items-start gap-4">
                      <div className="flex flex-col gap-1 min-w-0 flex-1">
                        <span className="text-[#00a8fc] font-bold text-base hover:underline cursor-pointer">
                          {formattedTitle}
                        </span>
                        {isFieldEnabled("description") && (
                          <p className="text-xs text-[#dbdee1]/90 leading-relaxed line-clamp-3">
                            {sampleData.description}
                          </p>
                        )}
                      </div>

                      {/* Thumbnail Cover Image */}
                      {isFieldEnabled("cover") && (
                        <img
                          src={sampleData.cover_url}
                          alt="Cover"
                          className="w-16 h-24 object-cover rounded-md shadow-md border border-[#1e1f22] shrink-0"
                        />
                      )}
                    </div>

                    {/* Fields Grid */}
                    <div className="grid grid-cols-2 gap-3 pt-2 text-xs border-t border-[#35373c]">
                      {fields.map((f) => {
                        if (!f.enabled) return null;
                        if (f.id === "author")
                          return (
                            <div key={f.id}>
                              <div className="text-[11px] font-bold text-[#949ba4] uppercase">{getFieldLabel("author")}</div>
                              <div className="font-medium text-white">{sampleData.author}</div>
                            </div>
                          );
                        if (f.id === "publisher")
                          return (
                            <div key={f.id}>
                              <div className="text-[11px] font-bold text-[#949ba4] uppercase">{getFieldLabel("publisher")}</div>
                              <div className="font-medium text-white">{sampleData.publisher}</div>
                            </div>
                          );
                        if (f.id === "language")
                          return (
                            <div key={f.id}>
                              <div className="text-[11px] font-bold text-[#949ba4] uppercase">{getFieldLabel("language")}</div>
                              <div className="font-medium text-white">{sampleData.language}</div>
                            </div>
                          );
                        if (f.id === "series")
                          return (
                            <div key={f.id}>
                              <div className="text-[11px] font-bold text-[#949ba4] uppercase">{getFieldLabel("series")}</div>
                              <div className="font-medium text-white">{sampleData.series}</div>
                            </div>
                          );
                        if (f.id === "tags")
                          return (
                            <div key={f.id} className="col-span-2">
                              <div className="text-[11px] font-bold text-[#949ba4] uppercase">{getFieldLabel("tags")}</div>
                              <div className="font-medium text-white">{sampleData.tags.join(", ")}</div>
                            </div>
                          );
                        if (f.id === "date")
                          return (
                            <div key={f.id}>
                              <div className="text-[11px] font-bold text-[#949ba4] uppercase">{getFieldLabel("date")}</div>
                              <div className="font-medium text-white">{sampleData.date}</div>
                            </div>
                          );
                        return null;
                      })}
                      <div>
                        <div className="text-[11px] font-bold text-[#949ba4] uppercase">⚡ Event</div>
                        <code className="text-[11px] bg-[#1e1f22] text-[#f2f3f5] px-1.5 py-0.5 rounded font-mono">
                          {previewEvent}
                        </code>
                      </div>
                    </div>

                    {/* Footer */}
                    <div className="text-[11px] text-[#949ba4] pt-1 flex items-center gap-1.5 border-t border-[#35373c]/50">
                      <span>NovelHub</span>
                      <span>•</span>
                      <span>Event: {previewEvent}</span>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Telegram Live Preview */}
            {form.template_type === "telegram" && (
              <div className="bg-[#0f172a] text-slate-200 p-5 rounded-2xl border border-slate-800 font-sans shadow-xl flex flex-col gap-3">
                <div className="flex items-center gap-2 pb-2 border-b border-slate-800 text-xs font-bold text-sky-400">
                  <MessageSquare className="w-4 h-4" /> Telegram Bot Live Message
                </div>
                <div className="bg-[#1e293b] p-4 rounded-xl text-xs flex flex-col gap-1.5 font-mono text-slate-100 border border-slate-700">
                  <div className="font-bold text-sky-400">📚 NovelHub Event: {previewEvent}</div>
                  <div><b>📖 Book:</b> {sampleData.rawTitle}</div>
                  {isFieldEnabled("author") && <div><b>{getFieldLabel("author")}:</b> {sampleData.author}</div>}
                  {isFieldEnabled("publisher") && <div><b>{getFieldLabel("publisher")}:</b> {sampleData.publisher}</div>}
                  {isFieldEnabled("language") && <div><b>{getFieldLabel("language")}:</b> {sampleData.language}</div>}
                  {isFieldEnabled("series") && <div><b>{getFieldLabel("series")}:</b> {sampleData.series}</div>}
                  {isFieldEnabled("description") && (
                    <div className="opacity-80 pt-1"><b>📝 Description:</b> {sampleData.description}</div>
                  )}
                </div>
              </div>
            )}

            {/* Slack Live Preview */}
            {form.template_type === "slack" && (
              <div className="bg-[#1a1d21] text-slate-200 p-5 rounded-2xl border border-slate-800 font-sans shadow-xl flex flex-col gap-3">
                <div className="flex items-center gap-2 pb-2 border-b border-slate-800 text-xs font-bold text-amber-400">
                  <MessageSquare className="w-4 h-4" /> Slack Block Kit Preview
                </div>
                <div className="bg-[#222529] p-4 rounded-xl text-xs flex justify-between items-start gap-4 border border-slate-700">
                  <div className="flex flex-col gap-1.5">
                    <div className="font-bold text-white text-sm">📚 NovelHub Event: {previewEvent}</div>
                    <div>*Book:* {sampleData.rawTitle}</div>
                    {isFieldEnabled("author") && <div>*{getFieldLabel("author")}:* {sampleData.author}</div>}
                    {isFieldEnabled("publisher") && <div>*{getFieldLabel("publisher")}:* {sampleData.publisher}</div>}
                    {isFieldEnabled("language") && <div>*{getFieldLabel("language")}:* {sampleData.language}</div>}
                  </div>
                  {isFieldEnabled("cover") && (
                    <img src={sampleData.cover_url} alt="Cover" className="w-14 h-20 object-cover rounded shadow-sm shrink-0" />
                  )}
                </div>
              </div>
            )}

            {/* Generic JSON Live Preview */}
            {form.template_type === "generic" && (
              <div className="bg-[#090d16] text-emerald-400 p-4 rounded-2xl border border-slate-800 font-mono text-xs shadow-xl flex flex-col gap-2">
                <div className="text-slate-400 text-[11px] font-bold uppercase tracking-wider">Raw HTTP POST Payload (JSON)</div>
                <pre className="overflow-x-auto p-3 bg-black/60 rounded-xl leading-relaxed">
                  {JSON.stringify(
                    {
                      event: previewEvent,
                      timestamp: Math.floor(Date.now() / 1000),
                      data: {
                        id: "book-uuid-12345",
                        title: sampleData.rawTitle,
                        author: isFieldEnabled("author") ? sampleData.author : undefined,
                        publisher: isFieldEnabled("publisher") ? sampleData.publisher : undefined,
                        language: isFieldEnabled("language") ? sampleData.language : undefined,
                        series: isFieldEnabled("series") ? sampleData.series : undefined,
                        description: isFieldEnabled("description") ? sampleData.description : undefined,
                        cover_url: isFieldEnabled("cover") ? sampleData.cover_url : undefined,
                        tags: isFieldEnabled("tags") ? sampleData.tags : undefined,
                      },
                    },
                    null,
                    2
                  )}
                </pre>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
