import { useEffect, useRef, useState } from "react";
import { App, Button, Card, Form, Input, InputNumber, Select, Space, Switch, Tag, Typography } from "antd";
import { debugChatCompletion, debugChatCompletionStream, debugEmbeddings, fetchKeys, fetchModels } from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";

const routeStrategyOptions = [
  { label: "跟随模型策略", value: "" },
  { label: "fixed", value: "fixed" },
  { label: "failover", value: "failover" },
  { label: "round_robin", value: "round_robin" },
];

const requestTypeOptions = [
  { label: "chat_completions", value: "chat" },
  { label: "embeddings", value: "embeddings" },
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
  const requestType = Form.useWatch("request_type", form);
  const selectedModelName = Form.useWatch("model", form);
  const selectedProviderKeyID = Form.useWatch("provider_key_id", form);

  const load = async () => {
    setLoading(true);
    try {
      const [modelsResponse, keysResponse] = await Promise.all([fetchModels(), fetchKeys()]);
      setModels((modelsResponse.items || []).filter((item) => item.is_enabled));
      setKeys(keysResponse.items || []);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载调试资源失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    form.setFieldsValue({
      route_strategy: "",
      request_type: "chat",
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
      const payload = { model: values.model };
      if (values.request_type === "chat") {
        payload.stream = Boolean(values.stream);
        payload.messages = [];
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
      } else {
        payload.input = values.embedding_input;
      }

      const requestPayload = {
        payload,
        provider_key_id: values.provider_key_id || undefined,
        route_strategy: values.provider_key_id ? undefined : values.route_strategy || undefined,
      };

      let response;
      if (values.request_type === "chat" && values.stream) {
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
      } else if (values.request_type === "embeddings") {
        response = await debugEmbeddings(requestPayload);
      } else {
        response = await debugChatCompletion(requestPayload);
      }

      setResult(response.data);
      setResultHeaders(response.headers || {});
      setResultStatus(response.status);
      setStreamText(response.assistantText || "");
      setStreamRaw(response.rawText || "");
      setStreamUsage(response.usage || null);
      message.success("调试请求已完成");
    } catch (error) {
      const response = error.response;
      setResult(response?.data || { error: { message: error.message || "请求失败" } });
      setResultHeaders(response?.headers || {});
      setResultStatus(response?.status || 0);
      setStreamText("");
      setStreamRaw("");
      setStreamUsage(null);
      if (error.name !== "AbortError") {
        message.error(response?.data?.error?.message || error.message || "调试请求失败");
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
    message.info("已停止流式输出");
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
        eyebrow="调试"
        title="交互式路由调试器"
        description="选择模型，可选强制指定某把上游密钥，并对比当前策略和手动覆盖策略的效果，便于压测前先确认路由行为。"
      />

      <section className="panel-card">
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Space direction="vertical" size={18} style={{ width: "100%" }}>
            <Form.Item label="模型" name="model" rules={[{ required: true, message: "请选择模型" }]}>
              <Select
                loading={loading}
                options={models.map((item) => ({
                  label: `${item.public_name} (${item.provider_name})`,
                  value: item.public_name,
                }))}
                onChange={() => form.setFieldValue("provider_key_id", undefined)}
              />
            </Form.Item>

            <Form.Item label="请求类型" name="request_type">
              <Select options={requestTypeOptions} />
            </Form.Item>

            <Form.Item label="上游密钥" name="provider_key_id" extra="留空表示按模型当前策略自动选择。指定密钥后会强制走该密钥。">
              <Select
                allowClear
                placeholder={selectedModel ? "按策略自动选择" : "请先选择模型"}
                disabled={!selectedModel}
                options={availableKeys.map((item) => ({
                  label: `${item.name} (${item.masked_api_key || "已脱敏"})${item.health_status === "cooldown" ? " [冷却中]" : ""}`,
                  value: item.id,
                }))}
              />
            </Form.Item>

            <Form.Item label="路由策略覆盖" name="route_strategy" extra="仅在未强制指定上游密钥时生效。">
              <Select disabled={Boolean(selectedProviderKeyID)} options={routeStrategyOptions} />
            </Form.Item>

            <Form.Item label="System Prompt" name="system_prompt" hidden={requestType !== "chat"}>
              <Input.TextArea rows={3} placeholder="可选的系统提示词" />
            </Form.Item>

            <Form.Item label="流式返回" name="stream" valuePropName="checked" hidden={requestType !== "chat"} extra="开启后可测试网关的实时 SSE 转发能力。">
              <Switch />
            </Form.Item>

            <Form.Item label="用户消息" name="user_message" hidden={requestType !== "chat"} rules={requestType === "chat" ? [{ required: true, message: "请输入用户消息" }] : []}>
              <Input.TextArea rows={6} placeholder="输入一段内容，用来验证路由行为是否符合预期" />
            </Form.Item>

            <Form.Item label="Embedding 输入" name="embedding_input" hidden={requestType !== "embeddings"} rules={requestType === "embeddings" ? [{ required: true, message: "请输入向量化内容" }] : []}>
              <Input.TextArea rows={6} placeholder="待向量化的文本" />
            </Form.Item>

            <Space size={16} wrap hidden={requestType !== "chat"}>
              <Form.Item label="Temperature" name="temperature" style={{ minWidth: 180 }}>
                <InputNumber min={0} max={2} step={0.1} style={{ width: "100%" }} />
              </Form.Item>
              <Form.Item label="最大 Tokens" name="max_tokens" style={{ minWidth: 180 }}>
                <InputNumber min={1} max={32768} style={{ width: "100%" }} />
              </Form.Item>
            </Space>

            <Space>
              <Button type="primary" htmlType="submit" loading={submitting}>
                发送调试请求
              </Button>
              <Button danger onClick={stopStream} disabled={!submitting || !form.getFieldValue("stream") || requestType !== "chat"}>
                停止流式
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
                重置
              </Button>
            </Space>
          </Space>
        </Form>
      </section>

      <section className="panel-card">
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          <div className="section-label">最终路由结果</div>
          <Space wrap>
            <Tag color="blue">状态: {resultStatus ?? "-"}</Tag>
            <Tag color="cyan">策略: {resultHeaders["x-wcs-debug-route-strategy"] || "-"}</Tag>
            <Tag color="green">密钥 ID: {resultHeaders["x-wcs-debug-provider-key-id"] || "-"}</Tag>
            <Tag color="gold">密钥名称: {resultHeaders["x-wcs-debug-provider-key-name"] || "-"}</Tag>
            <Tag color="purple">重试: {resultHeaders["x-wcs-debug-retry-count"] || "0"}</Tag>
            <Tag color="magenta">切换: {resultHeaders["x-wcs-debug-failover-count"] || "0"}</Tag>
          </Space>

          <Card size="small" title="响应预览">
            <Typography.Paragraph style={{ marginBottom: 0, whiteSpace: "pre-wrap" }}>
              {requestType === "embeddings"
                ? JSON.stringify(result?.data?.[0]?.embedding ? result.data[0].embedding.slice(0, 16) : result, null, 2)
                : assistantMessage || "暂无响应"}
            </Typography.Paragraph>
          </Card>

          {streamUsage ? (
            <Card size="small" title="流式用量">
              <Space wrap>
                <Tag color="blue">输入: {streamUsage.prompt_tokens ?? 0}</Tag>
                <Tag color="green">输出: {streamUsage.completion_tokens ?? 0}</Tag>
                <Tag color="purple">总计: {streamUsage.total_tokens ?? 0}</Tag>
              </Space>
            </Card>
          ) : null}

          <div className="section-label">原始响应</div>
          <pre className="json-preview">{form.getFieldValue("stream") ? streamRaw || JSON.stringify(result, null, 2) : JSON.stringify(result, null, 2)}</pre>
        </Space>
      </section>
    </Space>
  );
}
