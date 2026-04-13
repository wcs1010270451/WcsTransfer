import { useEffect, useState } from "react";
import { App, Button, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag } from "antd";
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

  const loadProviders = async () => {
    setLoading(true);
    try {
      const response = await fetchProviders();
      setProviders(response.items || []);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Failed to load providers");
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
      status: "active",
      extra_config: "{}",
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
      const payload = {
        ...values,
        extra_config: parseJSONField(values.extra_config),
      };

      if (editingProvider) {
        await updateProvider(editingProvider.id, payload);
        message.success("Provider updated");
      } else {
        await createProvider(payload);
        message.success("Provider created");
      }

      closeModal();
      await loadProviders();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Save failed");
    } finally {
      setSubmitting(false);
    }
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
      message.success("Provider status updated");
      await loadProviders();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Status update failed");
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="Provider Registry"
        title="Manage upstream providers and compatibility endpoints"
        description="Each provider represents one real upstream destination. Split regions, vendors, or compatibility layers into separate provider records so routing stays explicit."
        actions={
          <Button type="primary" onClick={openCreateModal}>
            New Provider
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
            { title: "Name", dataIndex: "name", key: "name" },
            { title: "Slug", dataIndex: "slug", key: "slug" },
            { title: "Type", dataIndex: "provider_type", key: "provider_type" },
            { title: "Base URL", dataIndex: "base_url", key: "base_url" },
            {
              title: "Status",
              dataIndex: "status",
              key: "status",
              render: (value) => <Tag color={value === "active" ? "green" : "default"}>{value}</Tag>,
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
                    title={record.status === "active" ? "Disable this provider?" : "Activate this provider?"}
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
        title={editingProvider ? "Edit Provider" : "New Provider"}
        onCancel={closeModal}
        onOk={() => form.submit()}
        okButtonProps={{ loading: submitting }}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ provider_type: "openai_compatible", status: "active" }}>
          <Form.Item label="Name" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="Slug" name="slug" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="Type" name="provider_type">
            <Select
              options={[
                { label: "OpenAI Compatible", value: "openai_compatible" },
                { label: "OpenAI", value: "openai" },
                { label: "Azure OpenAI", value: "azure_openai" },
                { label: "Custom", value: "custom" },
              ]}
            />
          </Form.Item>
          <Form.Item label="Base URL" name="base_url" rules={[{ required: true }]}>
            <Input placeholder="https://api.example.com/v1" />
          </Form.Item>
          <Form.Item label="Status" name="status">
            <Select options={[{ label: "active", value: "active" }, { label: "disabled", value: "disabled" }]} />
          </Form.Item>
          <Form.Item label="Description" name="description">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item label="Extra Config JSON" name="extra_config">
            <Input.TextArea rows={4} placeholder='{"region":"cn"}' />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
