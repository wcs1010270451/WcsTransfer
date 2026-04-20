import { create } from "zustand";

const loadJSON = (key, fallback) => {
  try {
    const raw = window.localStorage.getItem(key);
    return raw ? JSON.parse(raw) : fallback;
  } catch {
    return fallback;
  }
};

const usePortalAuthStore = create((set) => ({
  token: window.localStorage.getItem("wcstransfer_portal_token") || "",
  user: loadJSON("wcstransfer_portal_user", null),
  tenant: loadJSON("wcstransfer_portal_tenant", null),
  setSession: ({ token, user, tenant }) => {
    window.localStorage.setItem("wcstransfer_portal_token", token || "");
    window.localStorage.setItem("wcstransfer_portal_user", JSON.stringify(user || null));
    window.localStorage.setItem("wcstransfer_portal_tenant", JSON.stringify(tenant || null));
    set({ token: token || "", user: user || null, tenant: tenant || null });
  },
  clearSession: () => {
    window.localStorage.removeItem("wcstransfer_portal_token");
    window.localStorage.removeItem("wcstransfer_portal_user");
    window.localStorage.removeItem("wcstransfer_portal_tenant");
    set({ token: "", user: null, tenant: null });
  },
}));

export default usePortalAuthStore;
