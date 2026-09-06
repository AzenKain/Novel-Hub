import { ConfirmModal, DiscordMarkdown } from "@/components/common";
import { useBulkSendUserEmailMutation } from "@/hooks/useAdminQueries";
import type { User } from "@/types";
import {
  AlertCircle,
  Bold,
  ChevronDown,
  Code,
  Eye,
  FileText,
  Globe,
  Heading2,
  Italic,
  Link2,
  List,
  ListOrdered,
  Mail,
  Minus,
  PenLine,
  Quote,
  Send,
  Sparkles,
  Strikethrough,
  Users,
  X,
} from "lucide-react";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

function computeTextHash(text: string): string {
  let hash = 5381;
  for (let i = 0; i < text.length; i++) {
    hash = ((hash << 5) + hash) + text.charCodeAt(i);
    hash = hash & hash;
  }
  return hash.toString(36);
}

const EMAIL_LANGUAGES: { code: string; label: string }[] = [
  { code: "en", label: "English" },
  { code: "vi", label: "Tiếng Việt" },
  { code: "ja", label: "日本語" },
  { code: "ko", label: "한국어" },
  { code: "zh-CN", label: "简体中文" },
  { code: "zh-TW", label: "繁體中文" },
  { code: "es", label: "Español" },
  { code: "fr", label: "Français" },
  { code: "de", label: "Deutsch" },
  { code: "pt", label: "Português" },
  { code: "ru", label: "Русский" },
  { code: "ar", label: "العربية" },
  { code: "hi", label: "हिन्दी" },
  { code: "id", label: "Bahasa Indonesia" },
  { code: "th", label: "ไทย" },
  { code: "it", label: "Italiano" },
];

interface BulkSendEmailModalProps {
  isOpen: boolean;
  selectedUsers: User[];
  onClose: () => void;
  onSuccess: () => void;
}

export const BulkSendEmailModal: React.FC<BulkSendEmailModalProps> = ({
  isOpen,
  selectedUsers,
  onClose,
  onSuccess,
}) => {
  const { t, i18n } = useTranslation();
  const sendEmailMutation = useBulkSendUserEmailMutation();

  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [activeTab, setActiveTab] = useState<"write" | "preview">("write");
  const [templateMenuOpen, setTemplateMenuOpen] = useState(false);
  const [langMenuOpen, setLangMenuOpen] = useState(false);
  const [selectedLang, setSelectedLang] = useState<string>(
    i18n.language || "en",
  );
  const [selectedTemplateId, setSelectedTemplateId] = useState<string | null>(
    null,
  );
  const [templateContentHash, setTemplateContentHash] = useState<string | null>(
    null,
  );
  const [confirmOverwriteModal, setConfirmOverwriteModal] = useState<{
    open: boolean;
    type: "template" | "language";
    templateId?: string;
    lang?: string;
  }>({
    open: false,
    type: "template",
  });

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const langDropdownRef = useRef<HTMLDivElement>(null);

  // Filter out deleted accounts and accounts with empty emails
  const { eligibleUsers, skippedCount } = useMemo(() => {
    const eligible: User[] = [];
    let skipped = 0;

    for (const user of selectedUsers) {
      if (user.is_deleted || !user.email || !user.email.trim()) {
        skipped++;
        continue;
      }
      eligible.push(user);
    }

    return { eligibleUsers: eligible, skippedCount: skipped };
  }, [selectedUsers]);

  useEffect(() => {
    if (isOpen) {
      setSubject("");
      setBody("");
      setActiveTab("write");
      setTemplateMenuOpen(false);
      setLangMenuOpen(false);
      setSelectedTemplateId(null);
      setTemplateContentHash(null);
      setConfirmOverwriteModal({ open: false, type: "template" });
      setSelectedLang(i18n.language || "en");
    }
  }, [isOpen, i18n.language]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent | TouchEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setTemplateMenuOpen(false);
      }
      if (
        langDropdownRef.current &&
        !langDropdownRef.current.contains(event.target as Node)
      ) {
        setLangMenuOpen(false);
      }
    };
    if (templateMenuOpen || langMenuOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      document.addEventListener("touchstart", handleClickOutside);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("touchstart", handleClickOutside);
    };
  }, [templateMenuOpen, langMenuOpen]);

  const ensureLanguageLoaded = async (lang: string) => {
    if (i18n.hasResourceBundle && i18n.hasResourceBundle(lang, "translation")) {
      return;
    }
    try {
      if (typeof i18n.loadLanguages === "function") {
        await i18n.loadLanguages(lang);
      }
    } catch {
      // ignore
    }
    try {
      const res = await fetch(`/locales/${lang}.json`);
      if (res.ok) {
        const bundle = await res.json();
        if (i18n.addResourceBundle) {
          i18n.addResourceBundle(lang, "translation", bundle, true, true);
        }
      }
    } catch {
      // ignore
    }
  };

  const isContentModified = useCallback(() => {
    const currentHash = computeTextHash(`${subject}:::${body}`);
    if (templateContentHash !== null) {
      return currentHash !== templateContentHash;
    }
    return subject.trim().length > 0 || body.trim().length > 0;
  }, [subject, body, templateContentHash]);

  const doApplyTemplate = async (templateId: string, lang: string) => {
    await ensureLanguageLoaded(lang);
    const subjectKey = `admin.template_${templateId}_subject`;
    const bodyKey = `admin.template_${templateId}_body`;

    const generatedSubject = i18n.t(subjectKey, {
      lng: lang,
      name: "{{name}}",
      email: "{{email}}",
      defaultValue: "",
    });
    const generatedBody = i18n.t(bodyKey, {
      lng: lang,
      name: "{{name}}",
      email: "{{email}}",
      defaultValue: "",
    });

    if (generatedSubject) setSubject(generatedSubject);
    if (generatedBody) setBody(generatedBody);
    setSelectedTemplateId(templateId);
    setTemplateContentHash(
      computeTextHash(`${generatedSubject}:::${generatedBody}`),
    );
    setTemplateMenuOpen(false);
    setActiveTab("write");
  };

  const handleSelectTemplate = (templateId: string) => {
    setTemplateMenuOpen(false);
    if (isContentModified()) {
      setConfirmOverwriteModal({
        open: true,
        type: "template",
        templateId,
        lang: selectedLang,
      });
    } else {
      void doApplyTemplate(templateId, selectedLang);
    }
  };

  const doChangeLanguage = async (newLang: string) => {
    setSelectedLang(newLang);
    setLangMenuOpen(false);
    await ensureLanguageLoaded(newLang);

    if (selectedTemplateId) {
      await doApplyTemplate(selectedTemplateId, newLang);
    }
  };

  const handleSelectLang = (newLang: string) => {
    setLangMenuOpen(false);
    if (newLang === selectedLang) return;

    if (selectedTemplateId && isContentModified()) {
      setConfirmOverwriteModal({
        open: true,
        type: "language",
        lang: newLang,
      });
    } else {
      void doChangeLanguage(newLang);
    }
  };

  const handleConfirmOverwrite = async () => {
    const { type, templateId, lang } = confirmOverwriteModal;
    setConfirmOverwriteModal({ open: false, type: "template" });

    if (type === "template" && templateId) {
      await doApplyTemplate(templateId, lang || selectedLang);
    } else if (type === "language" && lang) {
      await doChangeLanguage(lang);
    }
  };

  const handleCancelOverwrite = () => {
    setConfirmOverwriteModal({ open: false, type: "template" });
  };

  const currentLangLabel = useMemo(() => {
    return (
      EMAIL_LANGUAGES.find((l) => l.code === selectedLang)?.label ||
      selectedLang.toUpperCase()
    );
  }, [selectedLang]);

  const templateList = useMemo(
    () => [
      {
        id: "system_notice",
        label: i18n.t("admin.template_system_notice", {
          lng: selectedLang,
          defaultValue: t(
            "admin.template_system_notice",
            "System Announcement",
          ),
        }),
      },
      {
        id: "welcome",
        label: i18n.t("admin.template_welcome", {
          lng: selectedLang,
          defaultValue: t("admin.template_welcome", "Welcome New Member"),
        }),
      },
      {
        id: "support",
        label: i18n.t("admin.template_support", {
          lng: selectedLang,
          defaultValue: t("admin.template_support", "Account Support"),
        }),
      },
      {
        id: "policy",
        label: i18n.t("admin.template_policy", {
          lng: selectedLang,
          defaultValue: t(
            "admin.template_policy",
            "Terms of Service Notice",
          ),
        }),
      },
    ],
    [selectedLang, i18n, t],
  );

  const insertFormatting = (
    prefix: string,
    suffix: string = "",
    placeholder: string = "",
  ) => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const selectedText = body.substring(start, end);
    const replacement = selectedText
      ? `${prefix}${selectedText}${suffix}`
      : `${prefix}${placeholder}${suffix}`;

    const newBody =
      body.substring(0, start) + replacement + body.substring(end);
    setBody(newBody);

    setTimeout(() => {
      textarea.focus();
      if (selectedText) {
        textarea.setSelectionRange(
          start + prefix.length,
          end + prefix.length,
        );
      } else {
        textarea.setSelectionRange(
          start + prefix.length,
          start + prefix.length + placeholder.length,
        );
      }
    }, 0);
  };

  const insertVariable = (variable: string) => {
    const textarea = textareaRef.current;
    if (!textarea) {
      setBody((prev) => `${prev} ${variable} `);
      return;
    }

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const replacement = ` ${variable} `;
    const newBody =
      body.substring(0, start) + replacement + body.substring(end);
    setBody(newBody);

    setTimeout(() => {
      textarea.focus();
      textarea.setSelectionRange(
        start + replacement.length,
        start + replacement.length,
      );
    }, 0);
  };

  const wordCount = useMemo(() => {
    const trimmed = body.trim();
    if (!trimmed) return 0;
    return trimmed.split(/\s+/).filter(Boolean).length;
  }, [body]);

  const charCount = body.length;

  const previewBody = useMemo(() => {
    const sampleName = eligibleUsers[0]?.full_name || "John Doe";
    const sampleEmail = eligibleUsers[0]?.email || "user@example.com";
    return body
      .replace(/\{\{name\}\}/g, sampleName)
      .replace(/\{\{email\}\}/g, sampleEmail);
  }, [body, eligibleUsers]);

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!subject.trim()) {
      toast.warn(t("admin.email_subject_required", "Subject cannot be empty"));
      return;
    }
    if (!body.trim()) {
      toast.warn(t("admin.email_body_required", "Email body cannot be empty"));
      return;
    }
    if (eligibleUsers.length === 0) return;

    try {
      const res = await sendEmailMutation.mutateAsync({
        user_ids: eligibleUsers.map((u) => u.id),
        subject: subject.trim(),
        body: body.trim(),
      });

      if (res.affected_count > 0) {
        toast.success(
          t(
            "admin.bulk_email_queued",
            "Queued {{count}} emails for background delivery",
            {
              count: res.affected_count,
            },
          ),
        );
      }
      if (res.skipped_count > 0 && res.skipped_reasons?.length) {
        toast.info(
          t(
            "admin.bulk_skip_notice",
            "Skipped {{count}} users due to safety constraints",
            {
              count: res.skipped_count,
            },
          ),
        );
      }

      onSuccess();
      onClose();
    } catch (err: unknown) {
      const msg =
        err instanceof Error
          ? err.message
          : t("admin.email_send_failed", "Failed to send emails");
      toast.error(msg);
    }
  };

  if (!isOpen) return null;

  return (
    <dialog className="modal modal-open z-50">
      <div className="modal-box max-w-2xl w-11/12 p-0 rounded-2xl shadow-2xl border border-base-300 bg-base-100 flex flex-col max-h-[90vh] overflow-hidden animate-in fade-in zoom-in-95 duration-200">
        {/* Window Header */}
        <div className="flex items-center justify-between px-5 py-3.5 bg-base-200/50 border-b border-base-200">
          <div className="flex items-center gap-2.5">
            <div className="grid h-8 w-8 place-items-center rounded-lg bg-primary/10 text-primary">
              <Mail className="h-4 w-4" />
            </div>
            <div>
              <h3 className="font-bold text-sm text-base-content leading-tight">
                {t(
                  "admin.bulk_send_email_title",
                  "Send Email to Multiple Users",
                )}
              </h3>
              <p className="text-[11px] text-base-content/60">
                {t(
                  "admin.bulk_send_email_desc",
                  "Broadcast an email message to selected active users via background queue",
                )}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={sendEmailMutation.isPending}
            className="btn btn-sm btn-circle btn-ghost text-base-content/60 hover:text-base-content"
            title={t("common.close", "Close")}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <form onSubmit={handleSend} className="flex-1 flex flex-col min-h-0">
          {/* Recipients Row */}
          <div className="flex items-center justify-between px-5 py-2.5 border-b border-base-200 bg-base-100 gap-3 flex-wrap">
            <div className="flex items-center gap-2 text-xs flex-wrap">
              <span className="text-xs font-semibold text-base-content/50 w-12 shrink-0">
                {t("admin.email_to", "To:")}
              </span>
              <div className="inline-flex items-center gap-2 px-2.5 py-1 rounded-lg bg-base-200/70 border border-base-300/60 text-xs">
                <Users className="w-3.5 h-3.5 text-primary shrink-0" />
                <span className="font-semibold text-base-content">
                  {t(
                    "admin.recipients_count",
                    "Recipients: {{count}} active users",
                    {
                      count: eligibleUsers.length,
                    },
                  )}
                </span>
              </div>
              {skippedCount > 0 && (
                <span className="badge badge-xs badge-warning">
                  {t(
                    "admin.skipped_deleted_badge",
                    "{{count}} deleted/invalid skipped",
                    {
                      count: skippedCount,
                    },
                  )}
                </span>
              )}
            </div>

            {/* Placeholders Quick Insertion */}
            <div className="flex items-center gap-1.5 text-xs">
              <span className="text-base-content/50 text-[11px] mr-0.5 hidden sm:inline flex items-center gap-1">
                <Sparkles className="w-3 h-3 text-primary" />
                <span>{t("admin.insert_placeholder", "Insert:")}</span>
              </span>
              <button
                type="button"
                onClick={() => insertVariable("{{name}}")}
                className="badge badge-sm badge-outline hover:badge-primary cursor-pointer transition-colors font-mono text-[11px]"
                title={t("admin.insert_name_hint", "Insert recipient full name")}
              >
                {"{{name}}"}
              </button>
              <button
                type="button"
                onClick={() => insertVariable("{{email}}")}
                className="badge badge-sm badge-outline hover:badge-primary cursor-pointer transition-colors font-mono text-[11px]"
                title={t("admin.insert_email_hint", "Insert recipient email")}
              >
                {"{{email}}"}
              </button>
            </div>
          </div>

          {/* Subject Row */}
          <div className="flex items-center gap-3 px-5 py-2.5 border-b border-base-200 bg-base-100">
            <span className="text-xs font-semibold text-base-content/50 w-12 shrink-0">
              {t("admin.email_subject", "Subject:")}
            </span>
            <input
              type="text"
              required
              maxLength={200}
              placeholder={t(
                "admin.email_subject_placeholder",
                "Subject of your message...",
              )}
              value={subject}
              onChange={(e) => {
                const newSubject = e.target.value;
                setSubject(newSubject);
                if (!newSubject.trim() && !body.trim()) {
                  setSelectedTemplateId(null);
                  setTemplateContentHash(null);
                }
              }}
              disabled={sendEmailMutation.isPending}
              className="w-full bg-transparent text-sm font-medium text-base-content placeholder:text-base-content/30 focus:outline-none py-1"
            />
          </div>

          {/* Toolbar & View Tabs */}
          <div className="border-b border-base-200 bg-base-200/40 relative z-20">
            {/* Row 1: Formatting Tools (Scrollable on small screens, no wrapping) */}
            <div
              className={`flex items-center gap-0.5 px-3 py-1.5 overflow-x-auto no-scrollbar border-b border-base-200/60 flex-nowrap transition-opacity ${
                activeTab !== "write" ? "opacity-30 pointer-events-none" : ""
              }`}
            >
              <button
                type="button"
                onClick={() =>
                  insertFormatting(
                    "**",
                    "**",
                    t("admin.placeholder_bold", "bold text"),
                  )
                }
                disabled={activeTab !== "write" || sendEmailMutation.isPending}
                className="btn btn-ghost btn-xs btn-square shrink-0 text-base-content/70 hover:text-base-content"
                title={t("admin.format_bold", "Bold (Ctrl+B)")}
              >
                <Bold className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() =>
                  insertFormatting(
                    "*",
                    "*",
                    t("admin.placeholder_italic", "italic text"),
                  )
                }
                disabled={activeTab !== "write" || sendEmailMutation.isPending}
                className="btn btn-ghost btn-xs btn-square shrink-0 text-base-content/70 hover:text-base-content"
                title={t("admin.format_italic", "Italic (Ctrl+I)")}
              >
                <Italic className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() =>
                  insertFormatting(
                    "~~",
                    "~~",
                    t(
                      "admin.placeholder_strikethrough",
                      "strikethrough text",
                    ),
                  )
                }
                disabled={activeTab !== "write" || sendEmailMutation.isPending}
                className="btn btn-ghost btn-xs btn-square shrink-0 text-base-content/70 hover:text-base-content"
                title={t("admin.format_strikethrough", "Strikethrough")}
              >
                <Strikethrough className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() =>
                  insertFormatting(
                    "### ",
                    "",
                    t("admin.placeholder_heading", "Heading"),
                  )
                }
                disabled={activeTab !== "write" || sendEmailMutation.isPending}
                className="btn btn-ghost btn-xs btn-square shrink-0 text-base-content/70 hover:text-base-content"
                title={t("admin.format_heading", "Heading")}
              >
                <Heading2 className="w-3.5 h-3.5" />
              </button>

              <div className="divider divider-horizontal mx-1 my-0.5 opacity-40 shrink-0"></div>

              <button
                type="button"
                onClick={() =>
                  insertFormatting(
                    "- ",
                    "",
                    t("admin.placeholder_list", "List item"),
                  )
                }
                disabled={activeTab !== "write" || sendEmailMutation.isPending}
                className="btn btn-ghost btn-xs btn-square shrink-0 text-base-content/70 hover:text-base-content"
                title={t("admin.format_bullet_list", "Bullet List")}
              >
                <List className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() =>
                  insertFormatting(
                    "1. ",
                    "",
                    t("admin.placeholder_numbered", "Item 1"),
                  )
                }
                disabled={activeTab !== "write" || sendEmailMutation.isPending}
                className="btn btn-ghost btn-xs btn-square shrink-0 text-base-content/70 hover:text-base-content"
                title={t("admin.format_numbered_list", "Numbered List")}
              >
                <ListOrdered className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() =>
                  insertFormatting(
                    "> ",
                    "",
                    t("admin.placeholder_quote", "Quote text"),
                  )
                }
                disabled={activeTab !== "write" || sendEmailMutation.isPending}
                className="btn btn-ghost btn-xs btn-square shrink-0 text-base-content/70 hover:text-base-content"
                title={t("admin.format_quote", "Quote")}
              >
                <Quote className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() =>
                  insertFormatting(
                    "`",
                    "`",
                    t("admin.placeholder_code", "code snippet"),
                  )
                }
                disabled={activeTab !== "write" || sendEmailMutation.isPending}
                className="btn btn-ghost btn-xs btn-square shrink-0 text-base-content/70 hover:text-base-content"
                title={t("admin.format_code", "Code")}
              >
                <Code className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() =>
                  insertFormatting(
                    "[",
                    "](https://example.com)",
                    t("admin.placeholder_link", "link text"),
                  )
                }
                disabled={activeTab !== "write" || sendEmailMutation.isPending}
                className="btn btn-ghost btn-xs btn-square shrink-0 text-base-content/70 hover:text-base-content"
                title={t("admin.format_link", "Link")}
              >
                <Link2 className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() => insertFormatting("\n---\n", "", "")}
                disabled={activeTab !== "write" || sendEmailMutation.isPending}
                className="btn btn-ghost btn-xs btn-square shrink-0 text-base-content/70 hover:text-base-content"
                title={t("admin.format_hr", "Horizontal Line")}
              >
                <Minus className="w-3.5 h-3.5" />
              </button>
            </div>

            {/* Row 2: Template & Language Selectors */}
            <div className="relative z-30 flex items-center gap-2 px-3 py-1.5 border-b border-base-200/60 bg-base-200/20">
              {/* Template Picker Dropdown */}
              <div
                className="relative inline-block text-left shrink-0"
                ref={dropdownRef}
              >
                <button
                  type="button"
                  onClick={() => {
                    setLangMenuOpen(false);
                    setTemplateMenuOpen((prev) => !prev);
                  }}
                  disabled={sendEmailMutation.isPending}
                  className="btn btn-ghost btn-xs gap-1.5 text-xs text-base-content/70 hover:text-base-content font-medium px-2 shrink-0 border border-base-300/60 bg-base-100/50"
                >
                  <FileText className="w-3.5 h-3.5 text-primary shrink-0" />
                  <span>{t("admin.template_select", "Templates")}</span>
                  <ChevronDown className="w-3 h-3 opacity-60 shrink-0" />
                </button>

                {templateMenuOpen && (
                  <div className="absolute left-0 top-full mt-1 w-64 max-w-[calc(100vw-3rem)] rounded-xl border border-base-200 bg-base-100 shadow-2xl z-50 p-1.5 space-y-1 animate-in fade-in zoom-in-95 duration-150">
                    <div className="text-[10px] font-bold uppercase tracking-wider text-base-content/40 px-2 py-1">
                      {t("admin.template_select", "Choose Template")}
                    </div>
                    {templateList.map((tmpl) => (
                      <button
                        key={tmpl.id}
                        type="button"
                        onClick={() => handleSelectTemplate(tmpl.id)}
                        className={`w-full text-left px-2.5 py-1.5 rounded-lg text-xs transition-colors flex items-center gap-2 ${
                          selectedTemplateId === tmpl.id
                            ? "bg-primary/10 text-primary font-semibold"
                            : "hover:bg-base-200 text-base-content"
                        }`}
                      >
                        <Sparkles className="w-3.5 h-3.5 text-primary/70 shrink-0" />
                        <span className="font-medium truncate">
                          {tmpl.label}
                        </span>
                      </button>
                    ))}
                  </div>
                )}
              </div>

              {/* Template Language Selector */}
              <div
                className="relative inline-block text-left shrink-0"
                ref={langDropdownRef}
              >
                <button
                  type="button"
                  onClick={() => {
                    setTemplateMenuOpen(false);
                    setLangMenuOpen((prev) => !prev);
                  }}
                  disabled={sendEmailMutation.isPending}
                  className="btn btn-ghost btn-xs gap-1.5 text-xs text-base-content/70 hover:text-base-content font-medium px-2 shrink-0 border border-base-300/60 bg-base-100/50"
                  title={t("admin.template_language", "Template Language")}
                >
                  <Globe className="w-3.5 h-3.5 text-info shrink-0" />
                  <span>{currentLangLabel}</span>
                  <ChevronDown className="w-3 h-3 opacity-60 shrink-0" />
                </button>

                {langMenuOpen && (
                  <div className="absolute right-0 sm:right-auto sm:left-0 top-full mt-1 w-48 max-h-60 overflow-y-auto rounded-xl border border-base-200 bg-base-100 shadow-2xl z-50 p-1.5 space-y-0.5 animate-in fade-in zoom-in-95 duration-150">
                    <div className="text-[10px] font-bold uppercase tracking-wider text-base-content/40 px-2 py-1 sticky top-0 bg-base-100">
                      {t("admin.template_language", "Template Language")}
                    </div>
                    {EMAIL_LANGUAGES.map((lang) => (
                      <button
                        key={lang.code}
                        type="button"
                        onClick={() => handleSelectLang(lang.code)}
                        className={`w-full text-left px-2.5 py-1.5 rounded-lg text-xs transition-colors flex items-center justify-between ${
                          selectedLang === lang.code
                            ? "bg-primary/10 text-primary font-semibold"
                            : "hover:bg-base-200 text-base-content"
                        }`}
                      >
                        <span className="truncate">{lang.label}</span>
                        <span className="text-[10px] uppercase font-mono opacity-50 ml-1">
                          {lang.code}
                        </span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Row 3: View Mode Toggle (Write vs Preview on its own line) */}
            <div className="flex items-center justify-between px-3 py-1.5 bg-base-100/80 border-b border-base-200">
              <div className="join bg-base-300/40 p-0.5 rounded-lg shrink-0">
                <button
                  type="button"
                  onClick={() => setActiveTab("write")}
                  className={`btn btn-xs join-item font-medium gap-1.5 ${
                    activeTab === "write"
                      ? "btn-primary shadow-xs"
                      : "btn-ghost text-base-content/60"
                  }`}
                >
                  <PenLine className="w-3 h-3" />
                  <span>{t("admin.email_write_tab", "Write")}</span>
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab("preview")}
                  className={`btn btn-xs join-item font-medium gap-1.5 ${
                    activeTab === "preview"
                      ? "btn-primary shadow-xs"
                      : "btn-ghost text-base-content/60"
                  }`}
                >
                  <Eye className="w-3 h-3" />
                  <span>{t("admin.email_preview_tab", "Preview")}</span>
                </button>
              </div>

              <div className="text-[11px] text-base-content/40 hidden sm:flex items-center gap-1.5">
                {activeTab === "write" ? (
                  <span>{t("admin.markdown_supported", "Markdown supported")}</span>
                ) : (
                  <span className="flex items-center gap-1 text-primary/80 font-medium">
                    <Sparkles className="w-3 h-3" />
                    <span>{t("admin.preview_mode_active", "Live preview active")}</span>
                  </span>
                )}
              </div>
            </div>
          </div>

          {/* Editor / Preview Area */}
          <div className="flex-1 overflow-y-auto min-h-55 max-h-90 bg-base-100 relative">
            {activeTab === "write" ? (
              <textarea
                ref={textareaRef}
                required
                maxLength={10000}
                placeholder={t(
                  "admin.email_body_placeholder",
                  "Type your message here... Markdown formatting is supported.",
                )}
                value={body}
                onChange={(e) => {
                  const newBody = e.target.value;
                  setBody(newBody);
                  if (!subject.trim() && !newBody.trim()) {
                    setSelectedTemplateId(null);
                    setTemplateContentHash(null);
                  }
                }}
                disabled={sendEmailMutation.isPending}
                className="w-full h-full min-h-55 p-5 text-sm leading-relaxed text-base-content placeholder:text-base-content/30 bg-transparent focus:outline-none resize-none font-sans"
              />
            ) : (
              <div className="p-5 min-h-55 text-sm leading-relaxed select-text">
                {previewBody.trim() ? (
                  <div className="space-y-3">
                    <div className="text-xs text-base-content/50 italic bg-base-200/50 p-2 rounded-lg">
                      {t(
                        "admin.preview_variable_hint",
                        "Sample preview showing placeholders replaced with first recipient's info:",
                      )}
                    </div>
                    <div className="prose prose-sm max-w-none text-base-content">
                      <DiscordMarkdown content={previewBody} />
                    </div>
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center py-12 text-center text-base-content/40 space-y-2">
                    <PenLine className="w-8 h-8 opacity-30" />
                    <p className="text-xs">
                      {t(
                        "admin.email_preview_empty",
                        "Nothing to preview yet. Start typing your message.",
                      )}
                    </p>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="px-5 py-3 bg-base-200/40 border-t border-base-200 flex items-center justify-between gap-3 flex-wrap">
            {/* Meta stats */}
            <div className="flex items-center gap-3 text-xs text-base-content/50">
              <span className="font-mono text-[11px]">
                {wordCount} {t("admin.words", "words")} • {charCount}{" "}
                {t("admin.characters", "characters")}
              </span>
              <span className="hidden sm:inline-flex items-center gap-1 text-[11px] text-base-content/40">
                <Sparkles className="w-3 h-3 text-primary/70" />
                {t(
                  "admin.bulk_email_queue_notice",
                  "Emails will be queued and sent asynchronously via background queue.",
                )}
              </span>
            </div>

            {/* Actions */}
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={onClose}
                disabled={sendEmailMutation.isPending}
                className="btn btn-ghost btn-sm"
              >
                {t("common.cancel", "Cancel")}
              </button>
              <button
                type="submit"
                disabled={
                  sendEmailMutation.isPending ||
                  !subject.trim() ||
                  !body.trim() ||
                  eligibleUsers.length === 0
                }
                className="btn btn-primary btn-sm gap-2 font-semibold shadow-sm"
              >
                {sendEmailMutation.isPending ? (
                  <>
                    <span className="loading loading-spinner loading-xs" />
                    <span>{t("common.loading", "Sending...")}</span>
                  </>
                ) : (
                  <>
                    <Send className="w-3.5 h-3.5" />
                    <span>
                      {t("admin.bulk_send_action", "Send ({{count}})", {
                        count: eligibleUsers.length,
                      })}
                    </span>
                  </>
                )}
              </button>
            </div>
          </div>
        </form>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose} disabled={sendEmailMutation.isPending}>
          close
        </button>
      </form>

      {/* Confirmation modal before overwriting modified email content */}
      <ConfirmModal
        open={confirmOverwriteModal.open}
        title={
          confirmOverwriteModal.type === "template"
            ? t(
                "admin.template_overwrite_confirm_title",
                "Overwrite Email Content?",
              )
            : t(
                "admin.template_lang_overwrite_confirm_title",
                "Change Template Language?",
              )
        }
        message={
          confirmOverwriteModal.type === "template"
            ? t(
                "admin.template_overwrite_confirm_desc",
                "You have modified the email content. Applying a new template will overwrite your custom changes. Are you sure you want to continue?",
              )
            : t(
                "admin.template_lang_overwrite_confirm_desc",
                "You have modified the email content. Changing the template language will reload the template and overwrite your custom changes. Are you sure you want to continue?",
              )
        }
        variant="warning"
        confirmText={
          confirmOverwriteModal.type === "template"
            ? t(
                "admin.template_overwrite_confirm_btn",
                "Overwrite & Apply",
              )
            : t(
                "admin.template_lang_overwrite_confirm_btn",
                "Change Language & Overwrite",
              )
        }
        cancelText={t("common.cancel", "Cancel")}
        onConfirm={() => void handleConfirmOverwrite()}
        onClose={handleCancelOverwrite}
      />
    </dialog>
  );
};
