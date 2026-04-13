import axios from "axios";
import useSettingsStore from "../store/settingsStore";

const createClient = () => {
  const { apiBaseUrl, adminToken } = useSettingsStore.getState();
  const headers = {};

  if (adminToken) {
    headers.Authorization = `Bearer ${adminToken}`;
  }

  return axios.create({
    baseURL: apiBaseUrl,
    headers,
  });
};

export const fetchHealth = async () => {
  const response = await createClient().get("/healthz");
  return response.data;
};

export const fetchProviders = async () => {
  const response = await createClient().get("/admin/providers");
  return response.data;
};

export const createProvider = async (payload) => {
  const response = await createClient().post("/admin/providers", payload);
  return response.data;
};

export const updateProvider = async (id, payload) => {
  const response = await createClient().put(`/admin/providers/${id}`, payload);
  return response.data;
};

export const fetchKeys = async () => {
  const response = await createClient().get("/admin/keys");
  return response.data;
};

export const createKey = async (payload) => {
  const response = await createClient().post("/admin/keys", payload);
  return response.data;
};

export const updateKey = async (id, payload) => {
  const response = await createClient().put(`/admin/keys/${id}`, payload);
  return response.data;
};

export const fetchModels = async () => {
  const response = await createClient().get("/admin/models");
  return response.data;
};

export const createModel = async (payload) => {
  const response = await createClient().post("/admin/models", payload);
  return response.data;
};

export const updateModel = async (id, payload) => {
  const response = await createClient().put(`/admin/models/${id}`, payload);
  return response.data;
};

export const fetchLogs = async (input = 50) => {
  const params = typeof input === "number" ? { page: 1, page_size: input } : input;
  const response = await createClient().get("/admin/logs", {
    params,
  });
  return response.data;
};

export const fetchStats = async () => {
  const response = await createClient().get("/admin/stats");
  return response.data;
};

export const fetchLogDetail = async (id) => {
  const response = await createClient().get(`/admin/logs/${id}`);
  return response.data;
};

export const exportLogs = async (params = {}) => {
  const response = await createClient().get("/admin/logs/export", {
    params,
    responseType: "blob",
  });
  return response.data;
};
