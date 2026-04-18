import { create } from "zustand";

const envBaseUrl = import.meta.env.VITE_API_BASE_URL?.trim();

const loadInitialValue = (key, fallback) => {
  const stored = window.sessionStorage.getItem(key);
  if (stored && stored.trim()) {
    return stored.trim();
  }

  return fallback;
};

const useSettingsStore = create((set) => ({
  apiBaseUrl: loadInitialValue("wcstransfer_api_base_url", envBaseUrl || "http://localhost:8080"),
  adminToken: loadInitialValue("wcstransfer_admin_token", ""),
  setApiBaseUrl: (value) => {
    const nextValue = value.trim();
    window.sessionStorage.setItem("wcstransfer_api_base_url", nextValue);
    set({ apiBaseUrl: nextValue });
  },
  setAdminToken: (value) => {
    const nextValue = value.trim();
    if (nextValue) {
      window.sessionStorage.setItem("wcstransfer_admin_token", nextValue);
    } else {
      window.sessionStorage.removeItem("wcstransfer_admin_token");
    }
    set({ adminToken: nextValue });
  },
}));

export default useSettingsStore;
