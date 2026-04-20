import { useEffect, useState } from "react";
import { App, Button, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, Typography } from "antd";
import { adjustTenantWallet, fetchTenants, updateTenant } from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";

function formatCurrency(value) {
  return new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
  }).format(Number(value || 0));
}

export default function TenantsPage() {
  const { message } = App.useApp();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [walletOpen, setWalletOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [walletSubmitting, setWalletSubmitting] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [walletItem, setWalletItem] = useState(null);
  const [form] = Form.useForm();
  const [walletForm] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const response = await fetchTenants();
      setItems(response.items || []);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载租户失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const openEditModal = (record) => {
    setEditingItem(record);
    form.setFieldsValue({
      name: record.name,
      slug: record.slug,
      status: record.status,
      max_client_keys: record.max_client_keys,
      notes: record.notes,
    });
    setOpen(true);
  };

  const closeModal = () => {
    setOpen(false);
    setEditingItem(null);
    form.resetFields();
  };

  const openWalletModal = (record) => {
    setWalletItem(record);
    walletForm.setFieldsValue({
      amount: undefined,
      note: "",
    });
    setWalletOpen(true);
  };

  const closeWalletModal = () => {
    setWalletOpen(false);
    setWalletItem(null);
    walletForm.resetFields();
  };

  const handleSubmit = async (values) => {
    if (!editingItem) {
      return;
    }
    setSubmitting(true);
    try {
      await updateTenant(editingItem.id, values);
      message.success("租户已更新");
      closeModal();
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "保存失败");
    } finally {
      setSubmitting(false);
    }
  };

  const handleWalletSubmit = async (values) => {
    if (!walletItem) {
      return;
    }
    setWalletSubmitting(true);
    try {
      await adjustTenantWallet(walletItem.id, values);
      message.success("钱包充值成功");
      closeWalletModal();
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "充值失败");
    } finally {
      setWalletSubmitting(false);
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="租户管理"
        title="审核、启用并为租户充值"
        description="新注册租户默认待激活。管理员审核后可以设置状态、客户端密钥上限，并手动充值租户钱包余额。"
      />

      <section className="panel-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={items}
          pagination={false}
          scroll={{ x: 1100 }}
          columns={[
            { title: "名称", dataIndex: "name", key: "name", width: 180 },
            { title: "Slug", dataIndex: "slug", key: "slug", width: 160 },
            {
              title: "状态",
              dataIndex: "status",
              key: "status",
              width: 110,
              render: (value) => (
                <Tag color={value === "active" ? "green" : value === "pending" ? "gold" : "red"}>
                  {value === "active" ? "启用" : value === "pending" ? "待审核" : "停用"}
                </Tag>
              ),
            },
            { title: "密钥上限", dataIndex: "max_client_keys", key: "max_client_keys", width: 120 },
            {
              title: "钱包余额",
              dataIndex: "wallet_balance",
              key: "wallet_balance",
              width: 140,
              render: (value) => formatCurrency(value),
            },
            { title: "备注", dataIndex: "notes", key: "notes" },
            {
              title: "操作",
              key: "actions",
              width: 180,
              fixed: "right",
              render: (_, record) => (
                <Space>
                  <Button size="small" onClick={() => openEditModal(record)}>
                    编辑
                  </Button>
                  <Button size="small" type="primary" onClick={() => openWalletModal(record)}>
                    充值
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </section>

      <Modal
        open={open}
        title="编辑租户"
        onCancel={closeModal}
        onOk={() => form.submit()}
        okButtonProps={{ loading: submitting }}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="Slug" name="slug" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="状态" name="status" rules={[{ required: true }]}>
            <Select
              options={[
                { label: "待审核", value: "pending" },
                { label: "启用", value: "active" },
                { label: "停用", value: "disabled" },
              ]}
            />
          </Form.Item>
          <Form.Item
            label="可建客户端密钥上限"
            name="max_client_keys"
            extra="0 表示该租户不能自助创建客户端密钥。"
          >
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="备注" name="notes">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={walletOpen}
        title={walletItem ? `租户充值 - ${walletItem.name}` : "租户充值"}
        onCancel={closeWalletModal}
        onOk={() => walletForm.submit()}
        okButtonProps={{ loading: walletSubmitting }}
        destroyOnClose
      >
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          {walletItem ? (
            <Typography.Text type="secondary">当前余额：{formatCurrency(walletItem.wallet_balance)}</Typography.Text>
          ) : null}
          <Form form={walletForm} layout="vertical" onFinish={handleWalletSubmit}>
            <Form.Item label="充值金额" name="amount" rules={[{ required: true }]}>
              <InputNumber min={0.0001} step={1} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item label="备注" name="note">
              <Input.TextArea rows={3} placeholder="例如：人工充值、测试额度、补发余额" />
            </Form.Item>
          </Form>
        </Space>
      </Modal>
    </Space>
  );
}
