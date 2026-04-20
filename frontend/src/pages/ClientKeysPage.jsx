import { useEffect, useState } from "react";
import { App, Button, Descriptions, Drawer, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, Typography } from "antd";
import { createClientKey, fetchClientKeys, fetchModels, updateClientKey } from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";

export default function ClientKeysPage() {
  const { message, modal } = App.useApp();
  const [clientKeys, setClientKeys] = useState([]);
  const [models, setModels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [selectedItem, setSelectedItem] = useState(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [clientKeysResponse, modelsResponse] = await Promise.all([fetchClientKeys(), fetchModels()]);
      setClientKeys(clientKeysResponse.items || []);
      setModels((modelsResponse.items || []).filter((item) => item.is_enabled));
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载客户端密钥失败");
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

  const openDetailDrawer = (record) => {
    setSelectedItem(record);
    setDetailOpen(true);
  };

  const closeDetailDrawer = () => {
    setDetailOpen(false);
    setSelectedItem(null);
  };

  const handleSubmit = async (values) => {
    setSubmitting(true);
    try {
      if (editingItem) {
        await updateClientKey(editingItem.id, values);
        message.success("客户端密钥已更新");
      } else {
        const created = await createClientKey(values);
        modal.success({
          title: "客户端密钥已创建",
          content: (
            <Space direction="vertical" size={12}>
              <Typography.Text>明文密钥只会展示一次，请在关闭弹窗前保存。</Typography.Text>
              <Typography.Paragraph copyable style={{ marginBottom: 0 }}>
                {created.plain_api_key}
              </Typography.Paragraph>
            </Space>
          ),
        });
        message.success("客户端密钥已创建");
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
      message.success("客户端密钥状态已更新");
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "状态更新失败");
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
        eyebrow="客户端接入"
        title="管理业务侧调用网关的 API Key"
        description="这些密钥用于你自己的应用调用 `/v1/*`，与上游提供方密钥分离，便于对调用方做归因和权限控制。"
        actions={
          <Button type="primary" onClick={openCreateModal}>
            新建客户端密钥
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
            { title: "名称", dataIndex: "name", key: "name" },
            { title: "脱敏密钥", dataIndex: "masked_key", key: "masked_key" },
            {
              title: "状态",
              dataIndex: "status",
              key: "status",
              render: (value) => <Tag color={value === "active" ? "green" : "default"}>{value}</Tag>,
            },
            {
              title: "授权模型",
              key: "allowed_models",
              render: (_, record) => (record.allowed_models?.length ? `${record.allowed_models.length} 个模型` : "全部模型"),
            },
            {
              title: "最近使用",
              dataIndex: "last_used_at",
              key: "last_used_at",
              render: (value) => (value ? new Date(value).toLocaleString() : "-"),
            },
            {
              title: "操作",
              key: "actions",
              render: (_, record) => (
                <Space>
                  <Button size="small" onClick={() => openDetailDrawer(record)}>
                    详情
                  </Button>
                  <Button size="small" onClick={() => openEditModal(record)}>
                    编辑
                  </Button>
                  <Popconfirm
                    title={record.status === "active" ? "确定停用这个客户端密钥吗？" : "确定启用这个客户端密钥吗？"}
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
        title={editingItem ? "编辑客户端密钥" : "新建客户端密钥"}
        onCancel={closeModal}
        onOk={() => form.submit()}
        okButtonProps={{ loading: submitting }}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ status: "active" }}>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="状态" name="status">
            <Select options={[{ label: "active", value: "active" }, { label: "disabled", value: "disabled" }]} />
          </Form.Item>
          <Form.Item label="RPM 限制" name="rpm_limit" extra="0 表示不限">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="每日请求上限" name="daily_request_limit" extra="0 表示不限">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="每日 Token 上限" name="daily_token_limit" extra="0 表示不限">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="每日预算上限" name="daily_cost_limit" extra="美元，0 表示不限">
            <InputNumber min={0} step={0.0001} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="每月预算上限" name="monthly_cost_limit" extra="美元，0 表示不限">
            <InputNumber min={0} step={0.0001} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="预警阈值" name="warning_threshold" extra="百分比，默认 80">
            <InputNumber min={0} max={100} step={1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="授权模型" name="allowed_model_ids" extra="留空表示允许全部已启用模型。">
            <Select
              mode="multiple"
              allowClear
              options={models.map((item) => ({
                label: `${item.public_name} (${item.provider_name})`,
                value: item.id,
              }))}
            />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        open={detailOpen}
        title={selectedItem ? `客户端密钥详情 - ${selectedItem.name}` : "客户端密钥详情"}
        width={720}
        onClose={closeDetailDrawer}
      >
        {selectedItem ? (
          <Space direction="vertical" size={24} style={{ width: "100%" }}>
            <Descriptions bordered column={1} size="small" title="基础信息">
              <Descriptions.Item label="名称">{selectedItem.name}</Descriptions.Item>
              <Descriptions.Item label="脱敏密钥">{selectedItem.masked_key}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={selectedItem.status === "active" ? "green" : "default"}>{selectedItem.status}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="描述">{selectedItem.description || "-"}</Descriptions.Item>
              <Descriptions.Item label="最近使用">
                {selectedItem.last_used_at ? new Date(selectedItem.last_used_at).toLocaleString() : "-"}
              </Descriptions.Item>
              <Descriptions.Item label="过期时间">
                {selectedItem.expires_at ? new Date(selectedItem.expires_at).toLocaleString() : "-"}
              </Descriptions.Item>
            </Descriptions>

            <Descriptions bordered column={1} size="small" title="访问策略">
              <Descriptions.Item label="RPM 限制">{selectedItem.rpm_limit || "不限"}</Descriptions.Item>
              <Descriptions.Item label="每日请求上限">{selectedItem.daily_request_limit || "不限"}</Descriptions.Item>
              <Descriptions.Item label="每日 Token 上限">{selectedItem.daily_token_limit || "不限"}</Descriptions.Item>
              <Descriptions.Item label="每日预算">
                {selectedItem.daily_cost_limit ? `$${Number(selectedItem.daily_cost_limit).toFixed(4)}` : "不限"}
              </Descriptions.Item>
              <Descriptions.Item label="每月预算">
                {selectedItem.monthly_cost_limit ? `$${Number(selectedItem.monthly_cost_limit).toFixed(4)}` : "不限"}
              </Descriptions.Item>
              <Descriptions.Item label="预警阈值">{selectedItem.warning_threshold}%</Descriptions.Item>
              <Descriptions.Item label="授权模型">
                {selectedItem.allowed_models?.length ? (
                  <Space wrap>
                    {selectedItem.allowed_models.map((item) => (
                      <Tag key={item}>{item}</Tag>
                    ))}
                  </Space>
                ) : (
                  "全部模型"
                )}
              </Descriptions.Item>
            </Descriptions>

            <Descriptions bordered column={1} size="small" title="配额使用">
              <Descriptions.Item label="当前 RPM">{selectedItem.usage?.current_rpm ?? 0}</Descriptions.Item>
              <Descriptions.Item label="今日已用请求数">{selectedItem.usage?.daily_requests_used ?? 0}</Descriptions.Item>
              <Descriptions.Item label="今日已用 Token">{selectedItem.usage?.daily_tokens_used ?? 0}</Descriptions.Item>
              <Descriptions.Item label="配额健康度">
                <Space wrap>
                  {renderUsageTag("RPM", selectedItem.usage?.rpm_usage_percent, selectedItem.usage?.is_rpm_limited)}
                  {renderUsageTag("请求", selectedItem.usage?.daily_request_usage_percent, selectedItem.usage?.is_daily_request_limited)}
                  {renderUsageTag("Token", selectedItem.usage?.daily_token_usage_percent, selectedItem.usage?.is_daily_token_limited)}
                </Space>
              </Descriptions.Item>
            </Descriptions>

            <Descriptions bordered column={1} size="small" title="预算使用">
              <Descriptions.Item label="今日已用成本">
                ${Number(selectedItem.cost_usage?.daily_cost_used || 0).toFixed(4)}
              </Descriptions.Item>
              <Descriptions.Item label="本月已用成本">
                ${Number(selectedItem.cost_usage?.monthly_cost_used || 0).toFixed(4)}
              </Descriptions.Item>
              <Descriptions.Item label="预算健康度">
                <Space wrap>
                  {renderUsageTag("日", selectedItem.cost_usage?.daily_cost_usage_percent, selectedItem.cost_usage?.is_daily_cost_limited)}
                  {renderUsageTag("月", selectedItem.cost_usage?.monthly_cost_usage_percent, selectedItem.cost_usage?.is_monthly_cost_limited)}
                  {selectedItem.cost_usage?.is_warning_triggered ? <Tag color="gold">预警</Tag> : null}
                </Space>
              </Descriptions.Item>
            </Descriptions>
          </Space>
        ) : null}
      </Drawer>
    </Space>
  );
}
