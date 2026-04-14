import { useEffect, useRef, useState } from "react";
import { App, Button, Card, Form, Input, InputNumber, Select, Space, Switch, Tag, Typography } from "antd";
import { debugChatCompletion, debugChatCompletionStream, fetchKeys, fetchModels } from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";

const routeStrategyOptions = [
  { label: "Follow model strategy", value: "" },
  { label: "fixed", value: "fixed" },
  { label: "failover", value: "failover" },
  { label: "round_robin", value: "round_robin" },
];

export default function DebugPage() {
  const { message } = App.useApp();
  const [models, setModels] = useState([]);
  const [keys, setKeys] = useState([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState(null);
  const [resultHeaders, setResultHeaders] = useState({});
  const [resultStatus, setResultStatus] = useState(null);
  const [streamText, setStreamText] = useState("");
  const [streamRaw, setStreamRaw] = useState("");
  const [streamUsage, setStreamUsage] = useState(null);
  const abortRef = useRef(null);
  const [form] = Form.useForm();
  const selectedModelName = Form.useWatch("model", form);
  const selectedProviderKeyID = Form.useWatch("provider_key_id", form);

  const load = async () => {
    setLoading(true);
    try {
      const [modelsResponse, keysResponse] = await Promise.all([fetchModels(), fetchKeys()]);
      setModels((modelsResponse.items || []).filter((item) => item.is_enabled));
      setKeys(keysResponse.items || []);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Failed to load debug resources");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    form.setFieldsValue({
      route_strategy: "",
      provider_key_id: undefined,
      temperature: undefined,
      max_tokens: undefined,
      stream: false,
    });
  }, []);

  const selectedModel = models.find((item) => item.public_name === selectedModelName);
  const availableKeys = keys.filter((item) => item.provider_id === selectedModel?.provider_id && item.status === "active");

  const handleSubmit = async (values) => {
    setSubmitting(true);
    setResult(null);
    setResultHeaders({});
    setResultStatus(null);
    setStreamText("");
    setStreamRaw("");
    setStreamUsage(null);

    try {
      const payload = {
        model: values.model,
        stream: Boolean(values.stream),
        messages: [],
      };

      if (values.system_prompt) {
        payload.messages.push({ role: "system", content: values.system_prompt });
      }
      payload.messages.push({ role: "user", content: values.user_message });

      if (values.temperature !== undefined && values.temperature !== null) {
        payload.temperature = values.temperature;
      }
      if (values.max_tokens !== undefined && values.max_tokens !== null) {
        payload.max_tokens = values.max_tokens;
      }

      const requestPayload = {
        payload,
        provider_key_id: values.provider_key_id || undefined,
        route_strategy: values.provider_key_id ? undefined : values.route_strategy || undefined,
      };

      let response;
      if (values.stream) {
        const controller = new AbortController();
        abortRef.current = controller;
        response = await debugChatCompletionStream(requestPayload, {
          signal: controller.signal,
          onUpdate: (update) => {
            setStreamText(update.assistantText || "");
            setStreamRaw(update.rawText || "");
            setStreamUsage(update.usage || null);
            setResultHeaders(update.headers || {});
            setResultStatus(update.status || 0);
          },
        });
      } else {
        response = await debugChatCompletion(requestPayload);
      }

      setResult(response.data);
      setResultHeaders(response.headers || {});
      setResultStatus(response.status);
      setStreamText(response.assistantText || "");
      setStreamRaw(response.rawText || "");
      setStreamUsage(response.usage || null);
      message.success("Debug request completed");
    } catch (error) {
      const response = error.response;
      setResult(response?.data || { error: { message: error.message || "Request failed" } });
      setResultHeaders(response?.headers || {});
      setResultStatus(response?.status || 0);
      setStreamText("");
      setStreamRaw("");
      setStreamUsage(null);
      if (error.name !== "AbortError") {
        message.error(response?.data?.error?.message || error.message || "Debug request failed");
      }
    } finally {
      abortRef.current = null;
      setSubmitting(false);
    }
  };

  const stopStream = () => {
    abortRef.current?.abort();
    abortRef.current = null;
    setSubmitting(false);
    message.info("Stream stopped");
  };

  const assistantMessage =
    streamText ||
    result?.choices?.[0]?.message?.content ||
    result?.choices?.[0]?.delta?.content ||
    result?.error?.message ||
    "";

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="Debug"
        title="Interactive routing debugger"
        description="Pick a model, optionally force a specific key, and compare the gateway's current routing strategy against a manual override before you do broader pressure tests."
      />

      <section className="panel-card">
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Space direction="vertical" size={18} style={{ width: "100%" }}>
            <Form.Item label="Model" name="model" rules={[{ required: true, message: "Select a model" }]}>
              <Select
                loading={loading}
                options={models.map((item) => ({
                  label: `${item.public_name} (${item.provider_name})`,
                  value: item.public_name,
                }))}
                onChange={() => form.setFieldValue("provider_key_id", undefined)}
              />
            </Form.Item>

            <Form.Item label="Provider Key" name="provider_key_id" extra="Leave empty to follow the model's current routing strategy. Choosing a key forces the request to that key.">
              <Select
                allowClear
                placeholder={selectedModel ? "Auto select by strategy" : "Select a model first"}
                disabled={!selectedModel}
                options={availableKeys.map((item) => ({
                  label: `${item.name} (${item.masked_api_key || "masked"})${item.health_status === "cooldown" ? " [cooldown]" : ""}`,
                  value: item.id,
                }))}
              />
            </Form.Item>

            <Form.Item label="Route Strategy Override" name="route_strategy" extra="Only used when no provider key is forced.">
              <Select disabled={Boolean(selectedProviderKeyID)} options={routeStrategyOptions} />
            </Form.Item>

            <Form.Item label="System Prompt" name="system_prompt">
              <Input.TextArea rows={3} placeholder="Optional system instruction" />
            </Form.Item>

            <Form.Item label="Stream" name="stream" valuePropName="checked" extra="Turn this on to test real-time SSE forwarding from the gateway.">
              <Switch />
            </Form.Item>

            <Form.Item label="User Message" name="user_message" rules={[{ required: true, message: "Enter a user message" }]}>
              <Input.TextArea rows={6} placeholder="Ask the model something to validate routing behavior" />
            </Form.Item>

            <Space size={16} wrap>
              <Form.Item label="Temperature" name="temperature" style={{ minWidth: 180 }}>
                <InputNumber min={0} max={2} step={0.1} style={{ width: "100%" }} />
              </Form.Item>
              <Form.Item label="Max Tokens" name="max_tokens" style={{ minWidth: 180 }}>
                <InputNumber min={1} max={32768} style={{ width: "100%" }} />
              </Form.Item>
            </Space>

            <Space>
              <Button type="primary" htmlType="submit" loading={submitting}>
                Send Debug Request
              </Button>
              <Button danger onClick={stopStream} disabled={!submitting || !form.getFieldValue("stream")}>
                Stop Stream
              </Button>
              <Button
                onClick={() => {
                  form.resetFields();
                  setResult(null);
                  setResultHeaders({});
                  setResultStatus(null);
                  setStreamText("");
                  setStreamRaw("");
                  setStreamUsage(null);
                }}
              >
                Reset
              </Button>
            </Space>
          </Space>
        </Form>
      </section>

      <section className="panel-card">
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          <div className="section-label">Resolved route</div>
          <Space wrap>
            <Tag color="blue">status: {resultStatus ?? "-"}</Tag>
            <Tag color="cyan">strategy: {resultHeaders["x-wcs-debug-route-strategy"] || "-"}</Tag>
            <Tag color="green">key id: {resultHeaders["x-wcs-debug-provider-key-id"] || "-"}</Tag>
            <Tag color="gold">key name: {resultHeaders["x-wcs-debug-provider-key-name"] || "-"}</Tag>
            <Tag color="purple">retry: {resultHeaders["x-wcs-debug-retry-count"] || "0"}</Tag>
            <Tag color="magenta">failover: {resultHeaders["x-wcs-debug-failover-count"] || "0"}</Tag>
          </Space>

          <Card size="small" title="Assistant preview">
            <Typography.Paragraph style={{ marginBottom: 0, whiteSpace: "pre-wrap" }}>
              {assistantMessage || "No response yet"}
            </Typography.Paragraph>
          </Card>

          {streamUsage ? (
            <Card size="small" title="Stream usage">
              <Space wrap>
                <Tag color="blue">prompt: {streamUsage.prompt_tokens ?? 0}</Tag>
                <Tag color="green">completion: {streamUsage.completion_tokens ?? 0}</Tag>
                <Tag color="purple">total: {streamUsage.total_tokens ?? 0}</Tag>
              </Space>
            </Card>
          ) : null}

          <div className="section-label">Raw response</div>
          <pre className="json-preview">{form.getFieldValue("stream") ? streamRaw || JSON.stringify(result, null, 2) : JSON.stringify(result, null, 2)}</pre>
        </Space>
      </section>
    </Space>
  );
}
