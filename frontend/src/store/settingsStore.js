import { create } from "zustand";

const envBaseUrl = import.meta.env.VITE_API_BASE_URL?.trim();
const envAdminToken = import.meta.env.VITE_ADMIN_TOKEN?.trim();

const loadInitialValue = (key, fallback) => {
  const stored = window.localStorage.getItem(key);
  if (stored && stored.trim()) {
    return stored.trim();
  }

  return fallback;
};

const useSettingsStore = create((set) => ({
  apiBaseUrl: loadInitialValue("wcstransfer_api_base_url", envBaseUrl || "http://localhost:8080"),
  adminToken: loadInitialValue("wcstransfer_admin_token", envAdminToken || "change-me"),
  setApiBaseUrl: (value) => {
    const nextValue = value.trim();
    window.localStorage.setItem("wcstransfer_api_base_url", nextValue);
    set({ apiBaseUrl: nextValue });
  },
  setAdminToken: (value) => {
    const nextValue = value.trim();
    window.localStorage.setItem("wcstransfer_admin_token", nextValue);
    set({ adminToken: nextValue });
  },
}));

export default useSettingsStore;
