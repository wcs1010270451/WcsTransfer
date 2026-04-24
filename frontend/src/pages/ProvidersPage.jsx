import { useEffect, useMemo, useState } from "react";
import { Alert, App, Button, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, Typography } from "antd";
import { createProvider, fetchProviders, updateProvider } from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";

function parseJSONField(value, fallback = {}) {
  if (!value || !String(value).trim()) {
    return fallback;
  }
  return JSON.parse(value);
}

function formatJSON(value) {
  if (!value) {
    return "{}";
  }
  try {
    return JSON.stringify(typeof value === "string" ? JSON.parse(value) : value, null, 2);
  } catch {
    return "{}";
  }
}

export default function ProvidersPage() {
  const { message } = App.useApp();
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [editingProvider, setEditingProvider] = useState(null);
  const [form] = Form.useForm();
  const providerType = Form.useWatch("provider_type", form);

  const providerHints = useMemo(
    () => ({
      openai_compatible: {
        baseURL: "https://api.openai.com/v1",
        extraConfig: "{}",
        note: "这里只填写服务根路径，不要附加 /chat/completions 或 /embeddings。",
      },
      anthropic: {
        baseURL: "https://api.anthropic.com",
        extraConfig: JSON.stringify({ anthropic_version: "2023-06-01" }, null, 2),
        note: "这里只填写 API 主机地址，网关会自动拼接 /v1/messages。",
      },
      gemini: {
        baseURL: "https://generativelanguage.googleapis.com",
        extraConfig: JSON.stringify({ gemini_api_version: "v1beta" }, null, 2),
        note: "这里只填写 Gemini 官方 API 主机地址，网关会自动拼接 /v1beta/models/{model}:generateContent。",
      },
      openai: {
        baseURL: "https://api.openai.com/v1",
        extraConfig: "{}",
        note: "这里只填写 API 根路径，不要附加具体接口路径。",
      },
      azure_openai: {
        baseURL: "https://your-resource.openai.azure.com/openai",
        extraConfig: "{}",
        note: "请填写 Azure OpenAI 根路径，不要附加具体接口路径。",
      },
      custom: {
        baseURL: "https://api.example.com/v1",
        extraConfig: "{}",
        note: "这里只填写服务根路径，网关会按所选协议自动拼接接口路径。",
      },
    }),
    [],
  );

  const validateBaseURL = async (_, value) => {
    const normalized = String(value || "").trim();
    if (
      normalized.includes("/chat/completions") ||
      normalized.includes("/embeddings") ||
      normalized.includes("/messages") ||
      normalized.includes(":generateContent") ||
      normalized.includes(":streamGenerateContent")
    ) {
      throw new Error("这里只能填写提供方根地址，不能直接填写完整接口路径。");
    }
  };

  const loadProviders = async () => {
    setLoading(true);
    try {
      const response = await fetchProviders();
      setProviders(response.items || []);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载提供方失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProviders();
  }, []);

  const openCreateModal = () => {
    setEditingProvider(null);
    form.setFieldsValue({
      provider_type: "openai_compatible",
      base_url: providerHints.openai_compatible.baseURL,
      status: "active",
      extra_config: providerHints.openai_compatible.extraConfig,
    });
    setOpen(true);
  };

  const openEditModal = (record) => {
    setEditingProvider(record);
    form.setFieldsValue({
      name: record.name,
      slug: record.slug,
      provider_type: record.provider_type,
      base_url: record.base_url,
      status: record.status,
      description: record.description,
      extra_config: formatJSON(record.extra_config),
    });
    setOpen(true);
  };

  const closeModal = () => {
    setOpen(false);
    setEditingProvider(null);
    form.resetFields();
  };

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      const normalizedBaseURL = String(values.base_url || "").trim();
      await validateBaseURL(null, normalizedBaseURL);

      const payload = {
        ...values,
        base_url: normalizedBaseURL,
        extra_config: parseJSONField(values.extra_config),
      };

      if (editingProvider) {
        await updateProvider(editingProvider.id, payload);
        message.success("提供方已更新");
      } else {
        await createProvider(payload);
        message.success("提供方已创建");
      }

      closeModal();
      await loadProviders();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "保存失败");
    } finally {
      setSubmitting(false);
    }
  };

  const handleProviderTypeChange = (value) => {
    const hint = providerHints[value];
    if (!hint || editingProvider) {
      return;
    }
    form.setFieldsValue({
      base_url: hint.baseURL,
      extra_config: hint.extraConfig,
    });
  };

  const handleToggleStatus = async (record) => {
    try {
      await updateProvider(record.id, {
        name: record.name,
        slug: record.slug,
        provider_type: record.provider_type,
        base_url: record.base_url,
        status: record.status === "active" ? "disabled" : "active",
        description: record.description,
        extra_config: record.extra_config || {},
      });
      message.success("提供方状态已更新");
      await loadProviders();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "状态更新失败");
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="提供方管理"
        title="管理上游提供方和协议类型"
        description="每个提供方代表一个真实上游目标。不同厂商、区域或兼容层建议拆成独立记录，便于后续路由和运维。"
        actions={
          <Button type="primary" onClick={openCreateModal}>
            新建提供方
          </Button>
        }
      />

      <section className="panel-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={providers}
          pagination={false}
          columns={[
            { title: "名称", dataIndex: "name", key: "name" },
            { title: "Slug", dataIndex: "slug", key: "slug" },
            { title: "类型", dataIndex: "provider_type", key: "provider_type" },
            { title: "Base URL", dataIndex: "base_url", key: "base_url" },
            {
              title: "状态",
              dataIndex: "status",
              key: "status",
              render: (value) => <Tag color={value === "active" ? "green" : "default"}>{value === "active" ? "启用" : "停用"}</Tag>,
            },
            {
              title: "操作",
              key: "actions",
              render: (_, record) => (
                <Space>
                  <Button size="small" onClick={() => openEditModal(record)}>
                    编辑
                  </Button>
                  <Popconfirm title={record.status === "active" ? "确定停用这个提供方吗？" : "确定启用这个提供方吗？"} onConfirm={() => handleToggleStatus(record)}>
                    <Button size="small">{record.status === "active" ? "停用" : "启用"}</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </section>

      <Modal open={open} title={editingProvider ? "编辑提供方" : "新建提供方"} onCancel={closeModal} onOk={() => form.submit()} okButtonProps={{ loading: submitting }} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ provider_type: "openai_compatible", status: "active" }}>
          {providerType ? (
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
              message={`推荐 Base URL：${providerHints[providerType]?.baseURL || "-"}`}
              description={providerHints[providerType]?.note || "请使用提供方根地址。"}
            />
          ) : null}

          <Form.Item label="名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="Slug" name="slug" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="类型" name="provider_type">
            <Select
              onChange={handleProviderTypeChange}
              options={[
                { label: "OpenAI 兼容", value: "openai_compatible" },
                { label: "Anthropic", value: "anthropic" },
                { label: "Gemini", value: "gemini" },
                { label: "OpenAI", value: "openai" },
                { label: "Azure OpenAI", value: "azure_openai" },
                { label: "自定义", value: "custom" },
              ]}
            />
          </Form.Item>
          <Form.Item
            label="Base URL"
            name="base_url"
            rules={[{ required: true }, { validator: validateBaseURL }]}
            extra={
              providerType === "anthropic"
                ? "Anthropic 示例：https://api.anthropic.com"
                : providerType === "gemini"
                  ? "Gemini 示例：https://generativelanguage.googleapis.com"
                  : "OpenAI 兼容示例：https://api.openai.com/v1"
            }
          >
            <Input placeholder="https://api.example.com/v1" />
          </Form.Item>
          <Form.Item label="状态" name="status">
            <Select options={[{ label: "启用", value: "active" }, { label: "停用", value: "disabled" }]} />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item label="附加配置 JSON" name="extra_config">
            <Input.TextArea rows={4} placeholder='{"region":"cn"}' />
          </Form.Item>

          {providerType === "anthropic" ? (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              Anthropic 附加配置支持 `anthropic_version` 和可选的 `anthropic_beta`。
              <br />
              <code>{`{"anthropic_version":"2023-06-01","anthropic_beta":["prompt-caching-2024-07-31"]}`}</code>
            </Typography.Paragraph>
          ) : null}

          {providerType === "gemini" ? (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              Gemini 附加配置当前支持 `gemini_api_version`，默认 `v1beta`。
              <br />
              <code>{`{"gemini_api_version":"v1beta"}`}</code>
            </Typography.Paragraph>
          ) : null}
        </Form>
      </Modal>
    </Space>
  );
}
