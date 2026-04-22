import { create } from "zustand";

const loadJSON = (key, fallback) => {
  try {
    const raw = window.sessionStorage.getItem(key);
    return raw ? JSON.parse(raw) : fallback;
  } catch {
    return fallback;
  }
};

const useAdminAuthStore = create((set) => ({
  token: window.sessionStorage.getItem("wcstransfer_admin_session_token") || "",
  user: loadJSON("wcstransfer_admin_session_user", null),
  setSession: ({ token, user }) => {
    window.sessionStorage.setItem("wcstransfer_admin_session_token", token || "");
    window.sessionStorage.setItem("wcstransfer_admin_session_user", JSON.stringify(user || null));
    set({ token: token || "", user: user || null });
  },
  clearSession: () => {
    window.sessionStorage.removeItem("wcstransfer_admin_session_token");
    window.sessionStorage.removeItem("wcstransfer_admin_session_user");
    set({ token: "", user: null });
  },
}));

export default useAdminAuthStore;
