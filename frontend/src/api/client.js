import axios from "axios";
import useSettingsStore from "../store/settingsStore";
import usePortalAuthStore from "../store/portalAuthStore";
import useAdminAuthStore from "../store/adminAuthStore";

const createClient = () => {
  const { apiBaseUrl } = useSettingsStore.getState();
  const { token } = useAdminAuthStore.getState();
  const headers = {};

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  return axios.create({
    baseURL: apiBaseUrl,
    headers,
  });
};

const createFetchHeaders = () => {
  const { token } = useAdminAuthStore.getState();
  const headers = {
    "Content-Type": "application/json",
  };

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  return headers;
};

const createPortalClient = () => {
  const { apiBaseUrl } = useSettingsStore.getState();
  const { token } = usePortalAuthStore.getState();
  const headers = {};

  if (token) {
    headers.Authorization = `Bearer ${token}`;
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

export const loginAdminUser = async (payload) => {
  const { apiBaseUrl } = useSettingsStore.getState();
  const response = await axios.post(`${apiBaseUrl}/admin/auth/login`, payload);
  return response.data;
};

export const fetchAdminMe = async () => {
  const response = await createClient().get("/admin/me");
  return response.data;
};

export const fetchProviders = async () => {
  const response = await createClient().get("/admin/providers");
  return response.data;
};

export const fetchUsers = async () => {
  const response = await createClient().get("/admin/users");
  return response.data;
};

export const createUser = async (payload) => {
  const response = await createClient().post("/admin/users", payload);
  return response.data;
};

export const updateUserStatus = async (id, payload) => {
  const response = await createClient().put(`/admin/users/${id}/status`, payload);
  return response.data;
};

export const resetUserPassword = async (id, payload) => {
  const response = await createClient().post(`/admin/users/${id}/reset-password`, payload);
  return response.data;
};

export const adjustUserWallet = async (id, payload) => {
  const response = await createClient().post(`/admin/users/${id}/wallet/adjust`, payload);
  return response.data;
};

export const correctUserWallet = async (id, payload) => {
  const response = await createClient().post(`/admin/users/${id}/wallet/correct`, payload);
  return response.data;
};

export const fetchUserWalletLedger = async (id, params = { page: 1, page_size: 20 }) => {
  const response = await createClient().get(`/admin/users/${id}/wallet/ledger`, { params });
  return response.data;
};

export const exportUserBilling = async (id, params = {}) => {
  const response = await createClient().get(`/admin/users/${id}/billing/export`, {
    params,
    responseType: "blob",
  });
  return response.data;
};

export const fetchClientKeys = async () => {
  const response = await createClient().get("/admin/client-keys");
  return response.data;
};

export const createClientKey = async (payload) => {
  const response = await createClient().post("/admin/client-keys", payload);
  return response.data;
};

export const updateClientKey = async (id, payload) => {
  const response = await createClient().put(`/admin/client-keys/${id}`, payload);
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

export const debugChatCompletion = async (payload) => {
  const response = await createClient().post("/admin/debug/chat/completions", payload);
  return {
    data: response.data,
    headers: response.headers,
    status: response.status,
  };
};

export const debugEmbeddings = async (payload) => {
  const response = await createClient().post("/admin/debug/embeddings", payload);
  return {
    data: response.data,
    headers: response.headers,
    status: response.status,
  };
};

export const debugAnthropicMessages = async (payload) => {
  const response = await createClient().post("/admin/debug/messages", payload);
  return {
    data: response.data,
    headers: response.headers,
    status: response.status,
  };
};

export const debugChatCompletionStream = async (payload, options = {}) => {
  const { apiBaseUrl } = useSettingsStore.getState();
  const response = await fetch(`${apiBaseUrl}/admin/debug/chat/completions`, {
    method: "POST",
    headers: createFetchHeaders(),
    body: JSON.stringify(payload),
    signal: options.signal,
  });

  const headers = Object.fromEntries(response.headers.entries());
  const contentType = response.headers.get("content-type") || "";

  if (!contentType.toLowerCase().includes("text/event-stream")) {
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      const error = new Error(data?.error?.message || `Request failed with status ${response.status}`);
      error.response = { data, headers, status: response.status };
      throw error;
    }
    return {
      data,
      headers,
      status: response.status,
      assistantText: data?.choices?.[0]?.message?.content || data?.error?.message || "",
      rawText: JSON.stringify(data, null, 2),
      usage: data?.usage || null,
    };
  }

  const reader = response.body?.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let rawText = "";
  let assistantText = "";
  let usage = null;

  const flushEvent = (eventText) => {
    const lines = eventText.split("\n");
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed.startsWith("data:")) {
        continue;
      }

      const payloadText = trimmed.slice(5).trim();
      if (!payloadText || payloadText === "[DONE]") {
        continue;
      }

      try {
        const parsed = JSON.parse(payloadText);
        const delta = parsed?.choices?.[0]?.delta;
        const content = typeof delta?.content === "string" ? delta.content : "";
        if (content) {
          assistantText += content;
        }
        if (parsed?.usage) {
          usage = parsed.usage;
        }
        options.onUpdate?.({
          assistantText,
          rawText,
          lastEvent: parsed,
          usage,
          headers,
          status: response.status,
        });
      } catch {
        // Ignore invalid stream fragments and keep reading.
      }
    }
  };

  while (reader) {
    const { value, done } = await reader.read();
    if (done) {
      break;
    }

    const chunkText = decoder.decode(value, { stream: true });
    rawText += chunkText;
    buffer += chunkText;

    let separatorIndex = buffer.indexOf("\n\n");
    while (separatorIndex >= 0) {
      const eventText = buffer.slice(0, separatorIndex);
      buffer = buffer.slice(separatorIndex + 2);
      flushEvent(eventText);
      separatorIndex = buffer.indexOf("\n\n");
    }
  }

  const tail = decoder.decode();
  if (tail) {
    rawText += tail;
    buffer += tail;
  }
  if (buffer.trim()) {
    flushEvent(buffer);
  }

  if (!response.ok) {
    const error = new Error(`Request failed with status ${response.status}`);
    error.response = {
      data: { error: { message: error.message } },
      headers,
      status: response.status,
    };
    throw error;
  }

  return {
    data: { stream: true, assistant_text: assistantText, usage },
    headers,
    status: response.status,
    assistantText,
    rawText,
    usage,
  };
};

export const debugAnthropicMessagesStream = async (payload, options = {}) => {
  const { apiBaseUrl } = useSettingsStore.getState();
  const response = await fetch(`${apiBaseUrl}/admin/debug/messages`, {
    method: "POST",
    headers: createFetchHeaders(),
    body: JSON.stringify(payload),
    signal: options.signal,
  });

  const headers = Object.fromEntries(response.headers.entries());
  const contentType = response.headers.get("content-type") || "";

  if (!contentType.toLowerCase().includes("text/event-stream")) {
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      const error = new Error(data?.error?.message || `Request failed with status ${response.status}`);
      error.response = { data, headers, status: response.status };
      throw error;
    }
    const assistantText =
      data?.content?.filter?.((item) => item?.type === "text").map((item) => item.text).join("") ||
      data?.error?.message ||
      "";
    const usage = data?.usage
      ? {
          prompt_tokens: data.usage.input_tokens ?? 0,
          completion_tokens: data.usage.output_tokens ?? 0,
          total_tokens: (data.usage.input_tokens ?? 0) + (data.usage.output_tokens ?? 0),
        }
      : null;
    return {
      data,
      headers,
      status: response.status,
      assistantText,
      rawText: JSON.stringify(data, null, 2),
      usage,
    };
  }

  const reader = response.body?.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let rawText = "";
  let assistantText = "";
  let promptTokens = 0;
  let completionTokens = 0;

  const flushEvent = (eventText) => {
    const lines = eventText.split("\n");
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed.startsWith("data:")) {
        continue;
      }

      const payloadText = trimmed.slice(5).trim();
      if (!payloadText) {
        continue;
      }

      try {
        const parsed = JSON.parse(payloadText);
        if (parsed?.type === "content_block_delta" && parsed?.delta?.type === "text_delta") {
          assistantText += parsed.delta.text || "";
        }
        if (parsed?.message?.usage?.input_tokens !== undefined) {
          promptTokens = parsed.message.usage.input_tokens || 0;
        }
        if (parsed?.usage?.output_tokens !== undefined) {
          completionTokens = parsed.usage.output_tokens || 0;
        }
        options.onUpdate?.({
          assistantText,
          rawText,
          lastEvent: parsed,
          usage: {
            prompt_tokens: promptTokens,
            completion_tokens: completionTokens,
            total_tokens: promptTokens + completionTokens,
          },
          headers,
          status: response.status,
        });
      } catch {
        // Ignore invalid stream fragments and keep reading.
      }
    }
  };

  while (reader) {
    const { value, done } = await reader.read();
    if (done) {
      break;
    }

    const chunkText = decoder.decode(value, { stream: true });
    rawText += chunkText;
    buffer += chunkText;

    let separatorIndex = buffer.indexOf("\n\n");
    while (separatorIndex >= 0) {
      const eventText = buffer.slice(0, separatorIndex);
      buffer = buffer.slice(separatorIndex + 2);
      flushEvent(eventText);
      separatorIndex = buffer.indexOf("\n\n");
    }
  }

  const tail = decoder.decode();
  if (tail) {
    rawText += tail;
    buffer += tail;
  }
  if (buffer.trim()) {
    flushEvent(buffer);
  }

  if (!response.ok) {
    const error = new Error(`Request failed with status ${response.status}`);
    error.response = {
      data: { error: { message: error.message } },
      headers,
      status: response.status,
    };
    throw error;
  }

  return {
    data: {
      stream: true,
      assistant_text: assistantText,
      usage: {
        prompt_tokens: promptTokens,
        completion_tokens: completionTokens,
        total_tokens: promptTokens + completionTokens,
      },
    },
    headers,
    status: response.status,
    assistantText,
    rawText,
    usage: {
      prompt_tokens: promptTokens,
      completion_tokens: completionTokens,
      total_tokens: promptTokens + completionTokens,
    },
  };
};

export const loginPortalUser = async (payload) => {
  const { apiBaseUrl } = useSettingsStore.getState();
  const response = await axios.post(`${apiBaseUrl}/portal/auth/login`, payload);
  return response.data;
};

export const fetchPortalMe = async () => {
  const response = await createPortalClient().get("/portal/me");
  return response.data;
};

export const fetchPortalClientKeys = async () => {
  const response = await createPortalClient().get("/portal/client-keys");
  return response.data;
};

export const fetchPortalStats = async () => {
  const response = await createPortalClient().get("/portal/stats");
  return response.data;
};

export const fetchPortalWalletLedger = async (params = { page: 1, page_size: 20 }) => {
  const response = await createPortalClient().get("/portal/wallet/ledger", { params });
  return response.data;
};

export const exportPortalBilling = async (params = {}) => {
  const response = await createPortalClient().get("/portal/billing/export", {
    params,
    responseType: "blob",
  });
  return response.data;
};

export const fetchPortalModels = async () => {
  const response = await createPortalClient().get("/portal/models");
  return response.data;
};

export const fetchPortalLogs = async (params = { page: 1, page_size: 20 }) => {
  const response = await createPortalClient().get("/portal/logs", { params });
  return response.data;
};

export const fetchPortalLogDetail = async (id) => {
  const response = await createPortalClient().get(`/portal/logs/${id}`);
  return response.data;
};

export const createPortalClientKey = async (payload) => {
  const response = await createPortalClient().post("/portal/client-keys", payload);
  return response.data;
};

export const disablePortalClientKey = async (id) => {
  const response = await createPortalClient().post(`/portal/client-keys/${id}/disable`);
  return response.data;
};
