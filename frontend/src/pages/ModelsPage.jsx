import { useEffect, useState } from "react";
import { App, Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag } from "antd";
import { createModel, fetchModels, fetchProviders, updateModel } from "../api/client";
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

export default function ModelsPage() {
  const { message } = App.useApp();
  const [models, setModels] = useState([]);
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [editingModel, setEditingModel] = useState(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [modelsResponse, providersResponse] = await Promise.all([fetchModels(), fetchProviders()]);
      setModels(modelsResponse.items || []);
      setProviders(providersResponse.items || []);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载模型失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const openCreateModal = () => {
    setEditingModel(null);
    form.setFieldsValue({
      route_strategy: "fixed",
      is_enabled: true,
      timeout_seconds: 120,
      temperature: 0.7,
      max_tokens: 0,
      cost_input_per_1m: 0,
      cost_output_per_1m: 0,
      sale_input_per_1m: 0,
      sale_output_per_1m: 0,
      metadata: "{}",
    });
    setOpen(true);
  };

  const openEditModal = (record) => {
    setEditingModel(record);
    form.setFieldsValue({
      public_name: record.public_name,
      provider_id: record.provider_id,
      upstream_model: record.upstream_model,
      route_strategy: record.route_strategy,
      is_enabled: record.is_enabled,
      max_tokens: record.max_tokens,
      temperature: record.temperature,
      timeout_seconds: record.timeout_seconds,
      cost_input_per_1m: record.cost_input_per_1m,
      cost_output_per_1m: record.cost_output_per_1m,
      sale_input_per_1m: record.sale_input_per_1m,
      sale_output_per_1m: record.sale_output_per_1m,
      metadata: formatJSON(record.metadata),
    });
    setOpen(true);
  };

  const closeModal = () => {
    setOpen(false);
    setEditingModel(null);
    form.resetFields();
  };

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      const payload = {
        ...values,
        metadata: parseJSONField(values.metadata),
      };

      if (editingModel) {
        await updateModel(editingModel.id, payload);
        message.success("模型映射已更新");
      } else {
        await createModel(payload);
        message.success("模型映射已创建");
      }

      closeModal();
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "保存失败");
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggleEnabled = async (record) => {
    try {
      await updateModel(record.id, {
        public_name: record.public_name,
        provider_id: record.provider_id,
        upstream_model: record.upstream_model,
        route_strategy: record.route_strategy,
        is_enabled: !record.is_enabled,
        max_tokens: record.max_tokens,
        temperature: record.temperature,
        timeout_seconds: record.timeout_seconds,
        cost_input_per_1m: record.cost_input_per_1m,
        cost_output_per_1m: record.cost_output_per_1m,
        sale_input_per_1m: record.sale_input_per_1m,
        sale_output_per_1m: record.sale_output_per_1m,
        metadata: record.metadata || {},
      });
      message.success("模型状态已更新");
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "状态更新失败");
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="模型映射"
        title="将对外模型名映射到真实上游模型"
        description="业务侧只需要知道公共模型名，真实上游模型、提供方、路由策略和启停状态都统一在这里维护。"
        actions={
          <Button type="primary" onClick={openCreateModal}>
            新建模型映射
          </Button>
        }
      />

      <section className="panel-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={models}
          pagination={false}
          columns={[
            { title: "公共模型名", dataIndex: "public_name", key: "public_name" },
            { title: "提供方", dataIndex: "provider_name", key: "provider_name" },
            { title: "上游模型", dataIndex: "upstream_model", key: "upstream_model" },
            { title: "策略", dataIndex: "route_strategy", key: "route_strategy" },
            {
              title: "输入成本价 $/1M",
              dataIndex: "cost_input_per_1m",
              key: "cost_input_per_1m",
              render: (value) => Number(value || 0).toFixed(4),
            },
            {
              title: "输出成本价 $/1M",
              dataIndex: "cost_output_per_1m",
              key: "cost_output_per_1m",
              render: (value) => Number(value || 0).toFixed(4),
            },
            {
              title: "输入售价 $/1M",
              dataIndex: "sale_input_per_1m",
              key: "sale_input_per_1m",
              render: (value) => Number(value || 0).toFixed(4),
            },
            {
              title: "输出售价 $/1M",
              dataIndex: "sale_output_per_1m",
              key: "sale_output_per_1m",
              render: (value) => Number(value || 0).toFixed(4),
            },
            {
              title: "状态",
              dataIndex: "is_enabled",
              key: "is_enabled",
              render: (value) => <Tag color={value ? "green" : "default"}>{value ? "启用" : "停用"}</Tag>,
            },
            {
              title: "操作",
              key: "actions",
              render: (_, record) => (
                <Space>
                  <Button size="small" onClick={() => openEditModal(record)}>
                    编辑
                  </Button>
                  <Popconfirm
                    title={record.is_enabled ? "确定停用这个模型映射吗？" : "确定启用这个模型映射吗？"}
                    onConfirm={() => handleToggleEnabled(record)}
                  >
                    <Button size="small">{record.is_enabled ? "停用" : "启用"}</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </section>

      <Modal
        open={open}
        title={editingModel ? "编辑模型映射" : "新建模型映射"}
        onCancel={closeModal}
        onOk={() => form.submit()}
        okButtonProps={{ loading: submitting }}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            route_strategy: "fixed",
            is_enabled: true,
            timeout_seconds: 120,
            temperature: 0.7,
            cost_input_per_1m: 0,
            cost_output_per_1m: 0,
            sale_input_per_1m: 0,
            sale_output_per_1m: 0,
          }}
        >
          <Form.Item label="公共模型名" name="public_name" rules={[{ required: true }]}>
            <Input placeholder="gpt-4o-mini" />
          </Form.Item>
          <Form.Item label="提供方" name="provider_id" rules={[{ required: true }]}>
            <Select options={providers.map((item) => ({ label: item.name, value: item.id }))} />
          </Form.Item>
          <Form.Item label="上游模型" name="upstream_model" rules={[{ required: true }]}>
            <Input placeholder="qwen-plus" />
          </Form.Item>
          <Form.Item label="路由策略" name="route_strategy">
            <Select
              options={[
                { label: "fixed", value: "fixed" },
                { label: "round_robin", value: "round_robin" },
                { label: "failover", value: "failover" },
              ]}
            />
          </Form.Item>
          <Form.Item label="启用" name="is_enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="最大 Tokens" name="max_tokens">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Temperature" name="temperature">
            <InputNumber min={0} max={2} step={0.1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="超时时间（秒）" name="timeout_seconds">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="每百万输入 Token 成本价" name="cost_input_per_1m" extra="你调用上游实际付出的单价，单位：美元 / 1,000,000 输入 tokens">
            <InputNumber min={0} step={0.0001} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="每百万输出 Token 成本价" name="cost_output_per_1m" extra="你调用上游实际付出的单价，单位：美元 / 1,000,000 输出 tokens">
            <InputNumber min={0} step={0.0001} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="每百万输入 Token 售价" name="sale_input_per_1m" extra="对客户计费使用的输入单价，单位：美元 / 1,000,000 输入 tokens">
            <InputNumber min={0} step={0.0001} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="每百万输出 Token 售价" name="sale_output_per_1m" extra="对客户计费使用的输出单价，单位：美元 / 1,000,000 输出 tokens">
            <InputNumber min={0} step={0.0001} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="元数据 JSON" name="metadata">
            <Input.TextArea rows={4} placeholder='{"tier":"default"}' />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
