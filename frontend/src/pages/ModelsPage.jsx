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
      message.error(error.response?.data?.error?.message || error.message || "Failed to load models");
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
      input_cost_per_1m: 0,
      output_cost_per_1m: 0,
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
      input_cost_per_1m: record.input_cost_per_1m,
      output_cost_per_1m: record.output_cost_per_1m,
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
        message.success("Model mapping updated");
      } else {
        await createModel(payload);
        message.success("Model mapping created");
      }

      closeModal();
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Save failed");
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
        input_cost_per_1m: record.input_cost_per_1m,
        output_cost_per_1m: record.output_cost_per_1m,
        metadata: record.metadata || {},
      });
      message.success("Model status updated");
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Status update failed");
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="Model Mapping"
        title="Map public model names to real upstream targets"
        description="Applications only need to know the public model name. The actual upstream model, provider, strategy, and enablement all stay centralized here."
        actions={
          <Button type="primary" onClick={openCreateModal}>
            New Model Mapping
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
            { title: "Public Model", dataIndex: "public_name", key: "public_name" },
            { title: "Provider", dataIndex: "provider_name", key: "provider_name" },
            { title: "Upstream Model", dataIndex: "upstream_model", key: "upstream_model" },
            { title: "Strategy", dataIndex: "route_strategy", key: "route_strategy" },
            {
              title: "Input $/1M",
              dataIndex: "input_cost_per_1m",
              key: "input_cost_per_1m",
              render: (value) => Number(value || 0).toFixed(4),
            },
            {
              title: "Output $/1M",
              dataIndex: "output_cost_per_1m",
              key: "output_cost_per_1m",
              render: (value) => Number(value || 0).toFixed(4),
            },
            {
              title: "Enabled",
              dataIndex: "is_enabled",
              key: "is_enabled",
              render: (value) => <Tag color={value ? "green" : "default"}>{value ? "enabled" : "disabled"}</Tag>,
            },
            {
              title: "Actions",
              key: "actions",
              render: (_, record) => (
                <Space>
                  <Button size="small" onClick={() => openEditModal(record)}>
                    Edit
                  </Button>
                  <Popconfirm
                    title={record.is_enabled ? "Disable this model mapping?" : "Enable this model mapping?"}
                    onConfirm={() => handleToggleEnabled(record)}
                  >
                    <Button size="small">{record.is_enabled ? "Disable" : "Enable"}</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </section>

      <Modal
        open={open}
        title={editingModel ? "Edit Model Mapping" : "New Model Mapping"}
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
            input_cost_per_1m: 0,
            output_cost_per_1m: 0,
          }}
        >
          <Form.Item label="Public Model" name="public_name" rules={[{ required: true }]}>
            <Input placeholder="gpt-4o-mini" />
          </Form.Item>
          <Form.Item label="Provider" name="provider_id" rules={[{ required: true }]}>
            <Select options={providers.map((item) => ({ label: item.name, value: item.id }))} />
          </Form.Item>
          <Form.Item label="Upstream Model" name="upstream_model" rules={[{ required: true }]}>
            <Input placeholder="qwen-plus" />
          </Form.Item>
          <Form.Item label="Route Strategy" name="route_strategy">
            <Select
              options={[
                { label: "fixed", value: "fixed" },
                { label: "round_robin", value: "round_robin" },
                { label: "failover", value: "failover" },
              ]}
            />
          </Form.Item>
          <Form.Item label="Enabled" name="is_enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label="Max Tokens" name="max_tokens">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Temperature" name="temperature">
            <InputNumber min={0} max={2} step={0.1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Timeout Seconds" name="timeout_seconds">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Input Cost Per 1M Tokens" name="input_cost_per_1m" extra="USD per 1,000,000 prompt tokens">
            <InputNumber min={0} step={0.0001} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Output Cost Per 1M Tokens" name="output_cost_per_1m" extra="USD per 1,000,000 completion tokens">
            <InputNumber min={0} step={0.0001} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Metadata JSON" name="metadata">
            <Input.TextArea rows={4} placeholder='{"tier":"default"}' />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
