import { useEffect, useMemo, useState } from "react";
import { Alert, App, Button, Card, Select, Space, Table, Tabs, Tag, Typography } from "antd";
import { fetchClientKeys, fetchModels } from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";
import useSettingsStore from "../store/settingsStore";

const errorRows = [
  { key: "unauthorized", code: 401, type: "unauthorized", meaning: "客户端 API Key 缺失或无效" },
  { key: "forbidden", code: 403, type: "model_forbidden", meaning: "当前客户端密钥无权访问该模型" },
  { key: "budget", code: 429, type: "budget_exceeded", meaning: "已超出每日或每月预算" },
  { key: "quota", code: 429, type: "rpm_limit_exceeded", meaning: "已超出请求速率或 Token 配额" },
  { key: "notfound", code: 404, type: "not_found", meaning: "模型映射不存在或已停用" },
  { key: "upstream", code: 502, type: "upstream_error", meaning: "上游提供方请求失败" },
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
        message.error(error.response?.data?.error?.message || error.message || "加载接口文档数据失败");
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
  const geminiPayload = useMemo(
    () =>
      JSON.stringify(
        {
          model: selectedModel || "gemini-2.5-pro",
          contents: [
            {
              role: "user",
              parts: [{ text: "Hello from WcsTransfer Gemini" }],
            },
          ],
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

  const curlGemini = `curl.exe -X POST ${apiBaseUrl}/v1/gemini/generate-content \\
  -H "Authorization: Bearer ${authHint}" \\
  -H "Content-Type: application/json" \\
  -d '${geminiPayload}'`;

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
        eyebrow="快速开始"
        title="业务 API 文档和可直接复制的示例"
        description="这个页面用于帮助业务团队接入网关，包含认证方式、支持的接口、常见错误以及基于当前环境的请求示例。"
        actions={
          <Space wrap>
            <Button onClick={() => window.open(`${apiBaseUrl}/docs`, "_blank", "noopener,noreferrer")}>
              打开 Swagger UI
            </Button>
            <Button onClick={() => window.open(`${apiBaseUrl}/redoc`, "_blank", "noopener,noreferrer")}>
              打开 ReDoc
            </Button>
            <Button onClick={() => window.open(`${apiBaseUrl}/openapi.json`, "_blank", "noopener,noreferrer")}>
              查看 OpenAPI
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
              下载 OpenAPI
            </Button>
          </Space>
        }
      />

      <Alert
        type="info"
        showIcon
        message="客户端密钥明文只会在创建时展示一次"
        description="控制台后续只保存脱敏值。下面的示例会用你选择的客户端密钥名称作为占位符，请替换成你创建时保存下来的真实明文密钥。"
      />

      <section className="panel-card">
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          <Typography.Title level={4}>环境信息</Typography.Title>
          <Space wrap>
            <Tag color="blue">Base URL: {apiBaseUrl}</Tag>
            <Tag color="green">认证：Bearer client_api_key</Tag>
            <Tag color="gold">当前模型：{selectedModel || "-"}</Tag>
          </Space>

          <Space wrap size={16}>
            <div style={{ minWidth: 280 }}>
              <Typography.Text strong>示例模型</Typography.Text>
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
              <Typography.Text strong>客户端密钥占位符</Typography.Text>
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
              当前路由目标：{selectedModelInfo.public_name} {"->"} {selectedModelInfo.upstream_model}，提供方 {selectedModelInfo.provider_name}
            </Typography.Paragraph>
          ) : null}
        </Space>
      </section>

      <section className="panel-card">
        <Typography.Title level={4}>认证方式</Typography.Title>
        <Typography.Paragraph>
          所有业务侧 `/v1/*` 接口都要求使用你自己的客户端密钥，而不是上游提供方密钥。
        </Typography.Paragraph>
        {codeBlock(`Authorization: Bearer ${authHint}`)}
      </section>

      <section className="panel-card">
        <Typography.Title level={4}>支持的接口</Typography.Title>
        <Table
          rowKey="path"
          pagination={false}
          dataSource={[
            { path: "/v1/models", method: "GET", note: "列出当前客户端密钥可见的模型" },
            { path: "/v1/chat/completions", method: "POST", note: "代理 OpenAI 兼容的对话补全接口" },
            { path: "/v1/embeddings", method: "POST", note: "代理 OpenAI 兼容的向量接口" },
            { path: "/v1/messages", method: "POST", note: "代理 Anthropic 官方原生 Messages API" },
            { path: "/v1/gemini/generate-content", method: "POST", note: "代理 Gemini 官方原生 generateContent API" },
            { path: "/v1/gemini/stream-generate-content", method: "POST", note: "代理 Gemini 官方原生 streamGenerateContent API" },
          ]}
          columns={[
            { title: "方法", dataIndex: "method", key: "method", width: 120 },
            { title: "路径", dataIndex: "path", key: "path" },
            { title: "说明", dataIndex: "note", key: "note" },
          ]}
        />
      </section>

      <section className="panel-card">
        <Typography.Title level={4}>示例</Typography.Title>
        <Tabs
          items={[
            { key: "curl-chat", label: "curl 对话", children: codeBlock(curlChat) },
            { key: "curl-embeddings", label: "curl 向量", children: codeBlock(curlEmbeddings) },
            { key: "curl-gemini", label: "curl Gemini", children: codeBlock(curlGemini) },
            { key: "javascript", label: "JavaScript", children: codeBlock(jsExample) },
            { key: "python", label: "Python", children: codeBlock(pythonExample) },
            { key: "chat-payload", label: "对话请求体", children: codeBlock(chatPayload) },
            { key: "embeddings-payload", label: "向量请求体", children: codeBlock(embeddingsPayload) },
            { key: "gemini-payload", label: "Gemini 请求体", children: codeBlock(geminiPayload) },
          ]}
        />
      </section>

      <section className="panel-card">
        <Typography.Title level={4}>错误类型</Typography.Title>
        <Table
          rowKey="key"
          pagination={false}
          dataSource={errorRows}
          columns={[
            { title: "HTTP", dataIndex: "code", key: "code", width: 90 },
            { title: "类型", dataIndex: "type", key: "type", width: 220 },
            { title: "含义", dataIndex: "meaning", key: "meaning" },
          ]}
        />
      </section>
    </Space>
  );
}
