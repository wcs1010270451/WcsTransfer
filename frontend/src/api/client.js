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

const createFetchHeaders = () => {
  const { adminToken } = useSettingsStore.getState();
  const headers = {
    "Content-Type": "application/json",
  };

  if (adminToken) {
    headers.Authorization = `Bearer ${adminToken}`;
  }

  return headers;
};

export const fetchHealth = async () => {
  const response = await createClient().get("/healthz");
  return response.data;
};

export const fetchProviders = async () => {
  const response = await createClient().get("/admin/providers");
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
