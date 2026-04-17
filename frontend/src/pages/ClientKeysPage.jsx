import { useEffect, useState } from "react";
import { App, Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, Typography } from "antd";
import { createClientKey, fetchClientKeys, fetchModels, updateClientKey } from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";

export default function ClientKeysPage() {
  const { message, modal } = App.useApp();
  const [clientKeys, setClientKeys] = useState([]);
  const [models, setModels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [clientKeysResponse, modelsResponse] = await Promise.all([fetchClientKeys(), fetchModels()]);
      setClientKeys(clientKeysResponse.items || []);
      setModels((modelsResponse.items || []).filter((item) => item.is_enabled));
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Failed to load client keys");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const openCreateModal = () => {
    setEditingItem(null);
    form.setFieldsValue({
      status: "active",
      description: "",
      rpm_limit: 0,
      daily_request_limit: 0,
      daily_token_limit: 0,
      daily_cost_limit: 0,
      monthly_cost_limit: 0,
      warning_threshold: 80,
      allowed_model_ids: [],
    });
    setOpen(true);
  };

  const openEditModal = (record) => {
    setEditingItem(record);
    form.setFieldsValue({
      name: record.name,
      status: record.status,
      description: record.description,
      rpm_limit: record.rpm_limit,
      daily_request_limit: record.daily_request_limit,
      daily_token_limit: record.daily_token_limit,
      daily_cost_limit: record.daily_cost_limit,
      monthly_cost_limit: record.monthly_cost_limit,
      warning_threshold: record.warning_threshold,
      allowed_model_ids: record.allowed_model_ids || [],
    });
    setOpen(true);
  };

  const closeModal = () => {
    setOpen(false);
    setEditingItem(null);
    form.resetFields();
  };

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      if (editingItem) {
        await updateClientKey(editingItem.id, values);
        message.success("Client key updated");
      } else {
        const created = await createClientKey(values);
        modal.success({
          title: "Client key created",
          content: (
            <Space direction="vertical" size={12}>
              <Typography.Text>This plain key is shown only once. Save it before closing this dialog.</Typography.Text>
              <Typography.Paragraph copyable style={{ marginBottom: 0 }}>
                {created.plain_api_key}
              </Typography.Paragraph>
            </Space>
          ),
        });
        message.success("Client key created");
      }

      closeModal();
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Save failed");
    } finally {
      setSubmitting(false);
    }
  };

  const handleToggleStatus = async (record) => {
    try {
      await updateClientKey(record.id, {
        name: record.name,
        status: record.status === "active" ? "disabled" : "active",
        description: record.description,
        rpm_limit: record.rpm_limit,
        daily_request_limit: record.daily_request_limit,
        daily_token_limit: record.daily_token_limit,
        daily_cost_limit: record.daily_cost_limit,
        monthly_cost_limit: record.monthly_cost_limit,
        warning_threshold: record.warning_threshold,
        allowed_model_ids: record.allowed_model_ids || [],
      });
      message.success("Client key status updated");
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Status update failed");
    }
  };

  const renderUsageTag = (label, value, limited) => {
    if (value === null || value === undefined) {
      return "-";
    }
    const color = limited ? "red" : value >= 80 ? "gold" : "blue";
    return <Tag color={color}>{label} {Number(value).toFixed(1)}%</Tag>;
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="Client Access"
        title="Manage business-side gateway API keys"
        description="These are the keys your own applications should use when calling `/v1/*`. They are separate from upstream provider keys and let us attribute traffic to each caller."
        actions={
          <Button type="primary" onClick={openCreateModal}>
            New Client Key
          </Button>
        }
      />

      <section className="panel-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={clientKeys}
          pagination={false}
          columns={[
            { title: "Name", dataIndex: "name", key: "name" },
            { title: "Masked Key", dataIndex: "masked_key", key: "masked_key" },
            {
              title: "Status",
              dataIndex: "status",
              key: "status",
              render: (value) => <Tag color={value === "active" ? "green" : "default"}>{value}</Tag>,
            },
            {
              title: "Last Used",
              dataIndex: "last_used_at",
              key: "last_used_at",
              render: (value) => (value ? new Date(value).toLocaleString() : "-"),
            },
            { title: "RPM", dataIndex: "rpm_limit", key: "rpm_limit" },
            { title: "Daily Requests", dataIndex: "daily_request_limit", key: "daily_request_limit" },
            { title: "Daily Tokens", dataIndex: "daily_token_limit", key: "daily_token_limit" },
            {
              title: "Daily Budget",
              dataIndex: "daily_cost_limit",
              key: "daily_cost_limit",
              render: (value) => (value ? `$${Number(value).toFixed(4)}` : "-"),
            },
            {
              title: "Monthly Budget",
              dataIndex: "monthly_cost_limit",
              key: "monthly_cost_limit",
              render: (value) => (value ? `$${Number(value).toFixed(4)}` : "-"),
            },
            {
              title: "Allowed Models",
              key: "allowed_models",
              render: (_, record) =>
                record.allowed_models?.length ? (
                  <Space wrap>
                    {record.allowed_models.map((item) => (
                      <Tag key={item}>{item}</Tag>
                    ))}
                  </Space>
                ) : (
                  <Typography.Text type="secondary">All models</Typography.Text>
                ),
            },
            {
              title: "Current RPM",
              key: "current_rpm",
              render: (_, record) => record.usage?.current_rpm ?? 0,
            },
            {
              title: "Daily Requests Used",
              key: "daily_requests_used",
              render: (_, record) => record.usage?.daily_requests_used ?? 0,
            },
            {
              title: "Daily Tokens Used",
              key: "daily_tokens_used",
              render: (_, record) => record.usage?.daily_tokens_used ?? 0,
            },
            {
              title: "Daily Cost Used",
              key: "daily_cost_used",
              render: (_, record) => `$${Number(record.cost_usage?.daily_cost_used || 0).toFixed(4)}`,
            },
            {
              title: "Monthly Cost Used",
              key: "monthly_cost_used",
              render: (_, record) => `$${Number(record.cost_usage?.monthly_cost_used || 0).toFixed(4)}`,
            },
            {
              title: "Quota Health",
              key: "quota_health",
              render: (_, record) => (
                <Space wrap>
                  {renderUsageTag("RPM", record.usage?.rpm_usage_percent, record.usage?.is_rpm_limited)}
                  {renderUsageTag("Req", record.usage?.daily_request_usage_percent, record.usage?.is_daily_request_limited)}
                  {renderUsageTag("Token", record.usage?.daily_token_usage_percent, record.usage?.is_daily_token_limited)}
                </Space>
              ),
            },
            {
              title: "Budget Health",
              key: "budget_health",
              render: (_, record) => (
                <Space wrap>
                  {renderUsageTag("Day", record.cost_usage?.daily_cost_usage_percent, record.cost_usage?.is_daily_cost_limited)}
                  {renderUsageTag("Month", record.cost_usage?.monthly_cost_usage_percent, record.cost_usage?.is_monthly_cost_limited)}
                  {record.cost_usage?.is_warning_triggered ? <Tag color="gold">Warn</Tag> : null}
                </Space>
              ),
            },
            {
              title: "Expires At",
              dataIndex: "expires_at",
              key: "expires_at",
              render: (value) => (value ? new Date(value).toLocaleString() : "-"),
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
                    title={record.status === "active" ? "Disable this client key?" : "Activate this client key?"}
                    onConfirm={() => handleToggleStatus(record)}
                  >
                    <Button size="small">{record.status === "active" ? "Disable" : "Activate"}</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </section>

      <Modal
        open={open}
        title={editingItem ? "Edit Client Key" : "New Client Key"}
        onCancel={closeModal}
        onOk={() => form.submit()}
        okButtonProps={{ loading: submitting }}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ status: "active" }}>
          <Form.Item label="Name" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="Status" name="status">
            <Select options={[{ label: "active", value: "active" }, { label: "disabled", value: "disabled" }]} />
          </Form.Item>
          <Form.Item label="RPM Limit" name="rpm_limit" extra="0 means unlimited">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Daily Request Limit" name="daily_request_limit" extra="0 means unlimited">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Daily Token Limit" name="daily_token_limit" extra="0 means unlimited">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Daily Cost Limit" name="daily_cost_limit" extra="USD, 0 means unlimited">
            <InputNumber min={0} step={0.0001} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Monthly Cost Limit" name="monthly_cost_limit" extra="USD, 0 means unlimited">
            <InputNumber min={0} step={0.0001} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Warning Threshold" name="warning_threshold" extra="Percent, default 80">
            <InputNumber min={0} max={100} step={1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Allowed Models" name="allowed_model_ids" extra="Leave empty to allow all enabled models.">
            <Select
              mode="multiple"
              allowClear
              options={models.map((item) => ({
                label: `${item.public_name} (${item.provider_name})`,
                value: item.id,
              }))}
            />
          </Form.Item>
          <Form.Item label="Description" name="description">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
