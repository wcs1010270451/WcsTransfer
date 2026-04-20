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
      message.error(error.response?.data?.error?.message || error.message || "加载上游密钥失败");
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
        message.success("上游密钥已更新");
      } else {
        await createKey(values);
        message.success("上游密钥已创建");
      }

      closeModal();
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "保存失败");
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
      message.success("上游密钥状态已更新");
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "状态更新失败");
    }
  };

  const renderGatewayHealth = (_, record) => {
    if (record.health_status === "cooldown") {
      return (
        <Space direction="vertical" size={2}>
          <Tag color="orange">冷却中</Tag>
          <span style={{ color: "#8c8c8c", fontSize: 12 }}>
            {record.cooldown_reason || "临时异常"}，截止 {record.cooldown_until ? new Date(record.cooldown_until).toLocaleString() : "-"}
          </span>
        </Space>
      );
    }

    return <Tag color="green">正常</Tag>;
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="上游密钥池"
        title="管理每个提供方的 API Key 池"
        description="这里是路由和故障切换的基础配置页，可以直接调整优先级、权重、限额和状态，而不影响业务侧调用方式。"
        actions={
          <Button type="primary" onClick={openCreateModal}>
            新建上游密钥
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
            { title: "名称", dataIndex: "name", key: "name" },
            { title: "提供方", dataIndex: "provider_name", key: "provider_name" },
            { title: "脱敏密钥", dataIndex: "masked_api_key", key: "masked_api_key" },
            { title: "权重", dataIndex: "weight", key: "weight" },
            { title: "优先级", dataIndex: "priority", key: "priority" },
            {
              title: "网关健康度",
              key: "health_status",
              render: renderGatewayHealth,
            },
            {
              title: "状态",
              dataIndex: "status",
              key: "status",
              render: (value) => <Tag color={value === "active" ? "green" : value === "disabled" ? "default" : "gold"}>{value}</Tag>,
            },
            {
              title: "最近错误",
              dataIndex: "last_error_message",
              key: "last_error_message",
              render: (value) => value || "-",
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
                    title={record.status === "active" ? "确定停用这把上游密钥吗？" : "确定启用这把上游密钥吗？"}
                    onConfirm={() => handleToggleStatus(record)}
                  >
                    <Button size="small">{record.status === "active" ? "停用" : "启用"}</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </section>

      <Modal
        open={open}
        title={editingKey ? "编辑上游密钥" : "新建上游密钥"}
        onCancel={closeModal}
        onOk={() => form.submit()}
        okButtonProps={{ loading: submitting }}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ status: "active", weight: 100, priority: 100 }}>
          <Form.Item label="提供方" name="provider_id" rules={[{ required: true }]}>
            <Select options={providers.map((item) => ({ label: item.name, value: item.id }))} />
          </Form.Item>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item
            label={editingKey ? "API Key（留空表示保持不变）" : "API Key"}
            name="api_key"
            rules={editingKey ? [] : [{ required: true }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item label="状态" name="status">
            <Select
              options={[
                { label: "active", value: "active" },
                { label: "disabled", value: "disabled" },
                { label: "rate_limited", value: "rate_limited" },
                { label: "invalid", value: "invalid" },
              ]}
            />
          </Form.Item>
          <Form.Item label="权重" name="weight">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="优先级" name="priority">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="RPM 限制" name="rpm_limit">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="TPM 限制" name="tpm_limit">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
