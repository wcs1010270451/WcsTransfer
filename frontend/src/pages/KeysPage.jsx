import { useEffect, useState } from "react";
import { App, Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag } from "antd";
import { createKey, fetchKeys, fetchProviders, updateKey } from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";

export default function KeysPage() {
  const { message } = App.useApp();
  const [keys, setKeys] = useState([]);
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [editingKey, setEditingKey] = useState(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [keysResponse, providersResponse] = await Promise.all([fetchKeys(), fetchProviders()]);
      setKeys(keysResponse.items || []);
      setProviders(providersResponse.items || []);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Failed to load keys");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const openCreateModal = () => {
    setEditingKey(null);
    form.setFieldsValue({
      status: "active",
      weight: 100,
      priority: 100,
      rpm_limit: 0,
      tpm_limit: 0,
      api_key: "",
    });
    setOpen(true);
  };

  const openEditModal = (record) => {
    setEditingKey(record);
    form.setFieldsValue({
      provider_id: record.provider_id,
      name: record.name,
      api_key: "",
      status: record.status,
      weight: record.weight,
      priority: record.priority,
      rpm_limit: record.rpm_limit,
      tpm_limit: record.tpm_limit,
    });
    setOpen(true);
  };

  const closeModal = () => {
    setOpen(false);
    setEditingKey(null);
    form.resetFields();
  };

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      if (editingKey) {
        const payload = {
          ...values,
          api_key: values.api_key ? values.api_key : undefined,
        };
        await updateKey(editingKey.id, payload);
        message.success("Key updated");
      } else {
        await createKey(values);
        message.success("Key created");
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
      await updateKey(record.id, {
        provider_id: record.provider_id,
        name: record.name,
        status: record.status === "active" ? "disabled" : "active",
        weight: record.weight,
        priority: record.priority,
        rpm_limit: record.rpm_limit,
        tpm_limit: record.tpm_limit,
      });
      message.success("Key status updated");
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Status update failed");
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="Key Pool"
        title="Manage each provider's API key pool"
        description="This page is the foundation for routing and failover. We can edit priority, weight, limits, and status without touching the calling applications."
        actions={
          <Button type="primary" onClick={openCreateModal}>
            New Key
          </Button>
        }
      />

      <section className="panel-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={keys}
          pagination={false}
          columns={[
            { title: "Name", dataIndex: "name", key: "name" },
            { title: "Provider", dataIndex: "provider_name", key: "provider_name" },
            { title: "Masked Key", dataIndex: "masked_api_key", key: "masked_api_key" },
            { title: "Weight", dataIndex: "weight", key: "weight" },
            { title: "Priority", dataIndex: "priority", key: "priority" },
            {
              title: "Status",
              dataIndex: "status",
              key: "status",
              render: (value) => <Tag color={value === "active" ? "green" : value === "disabled" ? "default" : "gold"}>{value}</Tag>,
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
                    title={record.status === "active" ? "Disable this key?" : "Set this key active?"}
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
        title={editingKey ? "Edit Key" : "New Key"}
        onCancel={closeModal}
        onOk={() => form.submit()}
        okButtonProps={{ loading: submitting }}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ status: "active", weight: 100, priority: 100 }}>
          <Form.Item label="Provider" name="provider_id" rules={[{ required: true }]}>
            <Select options={providers.map((item) => ({ label: item.name, value: item.id }))} />
          </Form.Item>
          <Form.Item label="Name" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item
            label={editingKey ? "API Key (leave empty to keep current value)" : "API Key"}
            name="api_key"
            rules={editingKey ? [] : [{ required: true }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item label="Status" name="status">
            <Select
              options={[
                { label: "active", value: "active" },
                { label: "disabled", value: "disabled" },
                { label: "rate_limited", value: "rate_limited" },
                { label: "invalid", value: "invalid" },
              ]}
            />
          </Form.Item>
          <Form.Item label="Weight" name="weight">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="Priority" name="priority">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="RPM Limit" name="rpm_limit">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="TPM Limit" name="tpm_limit">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
