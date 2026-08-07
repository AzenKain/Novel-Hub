import i18n from "i18next";
import httpBackend from "i18next-http-backend";
import { initReactI18next } from "react-i18next";
import { useSettingsStore } from "@/stores/settingsStore";

const savedLanguage = useSettingsStore.getState().language || "en";

i18n
  .use(httpBackend)
  .use(initReactI18next)
  .init({
    lng: savedLanguage,
    fallbackLng: "en",
    load: "currentOnly",
    debug: false,
    interpolation: {
      escapeValue: false,
    },
    backend: {
      loadPath: `/locales/{{lng}}.json`,
    },
  });

export default i18n;
