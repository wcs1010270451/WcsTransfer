import { useEffect, useMemo, useState } from "react";
import { Alert, App, Button, Card, Select, Space, Table, Tabs, Tag, Typography } from "antd";
import { fetchClientKeys, fetchModels } from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";
import useSettingsStore from "../store/settingsStore";

const errorRows = [
  { key: "unauthorized", code: 401, type: "unauthorized", meaning: "Client API Key missing or invalid" },
  { key: "forbidden", code: 403, type: "model_forbidden", meaning: "Client key is not allowed to access the model" },
  { key: "budget", code: 429, type: "budget_exceeded", meaning: "Daily or monthly budget has been exceeded" },
  { key: "quota", code: 429, type: "rpm_limit_exceeded", meaning: "Request rate or token quota exceeded" },
  { key: "notfound", code: 404, type: "not_found", meaning: "Model mapping does not exist or is disabled" },
  { key: "upstream", code: 502, type: "upstream_error", meaning: "Upstream provider request failed" },
];

function codeBlock(text) {
  return <pre className="json-preview">{text}</pre>;
}

export default function ApiDocsPage() {
  const { message } = App.useApp();
  const apiBaseUrl = useSettingsStore((state) => state.apiBaseUrl);
  const [models, setModels] = useState([]);
  const [clientKeys, setClientKeys] = useState([]);
  const [selectedModel, setSelectedModel] = useState("");
  const [selectedClientKeyName, setSelectedClientKeyName] = useState("");

  useEffect(() => {
    let active = true;

    const load = async () => {
      try {
        const [modelsResponse, clientKeysResponse] = await Promise.all([fetchModels(), fetchClientKeys()]);
        if (!active) {
          return;
        }

        const enabledModels = (modelsResponse.items || []).filter((item) => item.is_enabled);
        const clientKeyItems = clientKeysResponse.items || [];
        setModels(enabledModels);
        setClientKeys(clientKeyItems);
        setSelectedModel((previous) => previous || enabledModels[0]?.public_name || "gpt-4o-mini");
        setSelectedClientKeyName((previous) => previous || clientKeyItems[0]?.name || "your-client-key");
      } catch (error) {
        if (!active) {
          return;
        }
        message.error(error.response?.data?.error?.message || error.message || "Failed to load API docs data");
      }
    };

    load();
    return () => {
      active = false;
    };
  }, [message]);

  const selectedModelInfo = models.find((item) => item.public_name === selectedModel);
  const authHint = selectedClientKeyName || "your-client-key";
  const chatPayload = useMemo(
    () =>
      JSON.stringify(
        {
          model: selectedModel || "gpt-4o-mini",
          messages: [{ role: "user", content: "Hello from WcsTransfer" }],
        },
        null,
        2,
      ),
    [selectedModel],
  );
  const embeddingsPayload = useMemo(
    () =>
      JSON.stringify(
        {
          model: selectedModel || "text-embedding-3-small",
          input: "WcsTransfer embeddings example",
        },
        null,
        2,
      ),
    [selectedModel],
  );

  const curlChat = `curl.exe -X POST ${apiBaseUrl}/v1/chat/completions \\
  -H "Authorization: Bearer ${authHint}" \\
  -H "Content-Type: application/json" \\
  -d '${chatPayload}'`;

  const curlEmbeddings = `curl.exe -X POST ${apiBaseUrl}/v1/embeddings \\
  -H "Authorization: Bearer ${authHint}" \\
  -H "Content-Type: application/json" \\
  -d '${embeddingsPayload}'`;

  const jsExample = `const response = await fetch("${apiBaseUrl}/v1/chat/completions", {
  method: "POST",
  headers: {
    "Authorization": "Bearer ${authHint}",
    "Content-Type": "application/json"
  },
  body: JSON.stringify(${chatPayload})
});

const data = await response.json();
console.log(data);`;

  const pythonExample = `import requests

response = requests.post(
    "${apiBaseUrl}/v1/embeddings",
    headers={
        "Authorization": "Bearer ${authHint}",
        "Content-Type": "application/json",
    },
    json=${embeddingsPayload},
    timeout=60,
)

print(response.json())`;

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="Quickstart"
        title="Business API docs and copy-paste examples"
        description="Use this page to onboard application teams onto the gateway. It documents auth, supported routes, common errors, and concrete request examples against the current endpoint."
        actions={
          <Space wrap>
            <Button onClick={() => window.open(`${apiBaseUrl}/docs`, "_blank", "noopener,noreferrer")}>
              Open Swagger UI
            </Button>
            <Button onClick={() => window.open(`${apiBaseUrl}/redoc`, "_blank", "noopener,noreferrer")}>
              Open ReDoc
            </Button>
            <Button onClick={() => window.open(`${apiBaseUrl}/openapi.json`, "_blank", "noopener,noreferrer")}>
              View OpenAPI
            </Button>
            <Button
              type="primary"
              onClick={() => {
                const link = document.createElement("a");
                link.href = `${apiBaseUrl}/openapi.json`;
                link.download = "openapi.json";
                link.click();
              }}
            >
              Download OpenAPI
            </Button>
          </Space>
        }
      />

      <Alert
        type="info"
        showIcon
        message="Client keys are shown in plaintext only at creation time"
        description="The console stores only masked values afterward. The examples below use the selected client key name as a placeholder. Replace it with the actual plaintext key you saved when the key was created."
      />

      <section className="panel-card">
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          <Typography.Title level={4}>Environment</Typography.Title>
          <Space wrap>
            <Tag color="blue">Base URL: {apiBaseUrl}</Tag>
            <Tag color="green">Auth: Bearer client_api_key</Tag>
            <Tag color="gold">Selected model: {selectedModel || "-"}</Tag>
          </Space>

          <Space wrap size={16}>
            <div style={{ minWidth: 280 }}>
              <Typography.Text strong>Example model</Typography.Text>
              <Select
                style={{ width: "100%", marginTop: 8 }}
                value={selectedModel || undefined}
                onChange={setSelectedModel}
                options={models.map((item) => ({
                  label: `${item.public_name} (${item.provider_name})`,
                  value: item.public_name,
                }))}
              />
            </div>
            <div style={{ minWidth: 280 }}>
              <Typography.Text strong>Client key placeholder</Typography.Text>
              <Select
                style={{ width: "100%", marginTop: 8 }}
                value={selectedClientKeyName || undefined}
                onChange={setSelectedClientKeyName}
                options={clientKeys.map((item) => ({
                  label: `${item.name} (${item.masked_key})`,
                  value: item.name,
                }))}
              />
            </div>
          </Space>

          {selectedModelInfo ? (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              Current route target: {selectedModelInfo.public_name} {"->"} {selectedModelInfo.upstream_model} via{" "}
              {selectedModelInfo.provider_name}
            </Typography.Paragraph>
          ) : null}
        </Space>
      </section>

      <section className="panel-card">
        <Typography.Title level={4}>Authentication</Typography.Title>
        <Typography.Paragraph>
          All business-facing `/v1/*` routes require your own client key, not the upstream provider key.
        </Typography.Paragraph>
        {codeBlock(`Authorization: Bearer ${authHint}`)}
      </section>

      <section className="panel-card">
        <Typography.Title level={4}>Supported Routes</Typography.Title>
        <Table
          rowKey="path"
          pagination={false}
          dataSource={[
            { path: "/v1/models", method: "GET", note: "List models visible to the current client key" },
            { path: "/v1/chat/completions", method: "POST", note: "Proxy OpenAI-compatible chat completions" },
            { path: "/v1/embeddings", method: "POST", note: "Proxy OpenAI-compatible embeddings" },
          ]}
          columns={[
            { title: "Method", dataIndex: "method", key: "method", width: 120 },
            { title: "Path", dataIndex: "path", key: "path" },
            { title: "Description", dataIndex: "note", key: "note" },
          ]}
        />
      </section>

      <section className="panel-card">
        <Typography.Title level={4}>Examples</Typography.Title>
        <Tabs
          items={[
            { key: "curl-chat", label: "curl chat", children: codeBlock(curlChat) },
            { key: "curl-embeddings", label: "curl embeddings", children: codeBlock(curlEmbeddings) },
            { key: "javascript", label: "JavaScript", children: codeBlock(jsExample) },
            { key: "python", label: "Python", children: codeBlock(pythonExample) },
            { key: "chat-payload", label: "chat payload", children: codeBlock(chatPayload) },
            { key: "embeddings-payload", label: "embeddings payload", children: codeBlock(embeddingsPayload) },
          ]}
        />
      </section>

      <section className="panel-card">
        <Typography.Title level={4}>Error Types</Typography.Title>
        <Table
          rowKey="key"
          pagination={false}
          dataSource={errorRows}
          columns={[
            { title: "HTTP", dataIndex: "code", key: "code", width: 90 },
            { title: "Type", dataIndex: "type", key: "type", width: 220 },
            { title: "Meaning", dataIndex: "meaning", key: "meaning" },
          ]}
        />
      </section>
    </Space>
  );
}
