import { useEffect, useState } from "react";
import { App, Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, Typography } from "antd";
import { createClientKey, fetchClientKeys, updateClientKey } from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";

export default function ClientKeysPage() {
  const { message, modal } = App.useApp();
  const [clientKeys, setClientKeys] = useState([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const response = await fetchClientKeys();
      setClientKeys(response.items || []);
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
      });
      message.success("Client key status updated");
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Status update failed");
    }
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
          <Form.Item label="Description" name="description">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
