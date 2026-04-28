import { create } from "zustand";

const loadJSON = (key, fallback) => {
  try {
    const raw = window.sessionStorage.getItem(key);
    return raw ? JSON.parse(raw) : fallback;
  } catch {
    return fallback;
  }
};

const loadToken = () => {
  const sessionValue = window.sessionStorage.getItem("wcstransfer_portal_token");
  if (sessionValue && sessionValue.trim()) {
    return sessionValue.trim();
  }

  const legacyValue = window.localStorage.getItem("wcstransfer_portal_token");
  if (legacyValue && legacyValue.trim()) {
    const trimmed = legacyValue.trim();
    window.sessionStorage.setItem("wcstransfer_portal_token", trimmed);
    window.localStorage.removeItem("wcstransfer_portal_token");
    return trimmed;
  }

  return "";
};

const migrateJSON = (key) => {
  const sessionValue = window.sessionStorage.getItem(key);
  if (sessionValue) return;
  const legacyValue = window.localStorage.getItem(key);
  if (legacyValue) {
    window.sessionStorage.setItem(key, legacyValue);
    window.localStorage.removeItem(key);
  }
};

migrateJSON("wcstransfer_portal_user");

const usePortalAuthStore = create((set) => ({
  token: loadToken(),
  user: loadJSON("wcstransfer_portal_user", null),
  setSession: ({ token, user }) => {
    window.sessionStorage.setItem("wcstransfer_portal_token", token || "");
    window.sessionStorage.setItem("wcstransfer_portal_user", JSON.stringify(user || null));
    window.localStorage.removeItem("wcstransfer_portal_token");
    window.localStorage.removeItem("wcstransfer_portal_user");
    window.localStorage.removeItem("wcstransfer_portal_tenant");
    set({ token: token || "", user: user || null });
  },
  clearSession: () => {
    window.sessionStorage.removeItem("wcstransfer_portal_token");
    window.sessionStorage.removeItem("wcstransfer_portal_user");
    window.localStorage.removeItem("wcstransfer_portal_token");
    window.localStorage.removeItem("wcstransfer_portal_user");
    window.localStorage.removeItem("wcstransfer_portal_tenant");
    set({ token: "", user: null });
  },
}));

export default usePortalAuthStore;
