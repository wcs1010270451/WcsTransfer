import { useEffect, useState } from "react";
import { App, Button, Drawer, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, Typography } from "antd";
import {
  adjustUserWallet,
  correctUserWallet,
  createUser,
  exportUserBilling,
  fetchUserWalletLedger,
  fetchUsers,
  resetUserPassword,
  updateUserStatus,
} from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";

function formatCurrency(value) {
  return new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
  }).format(Number(value || 0));
}

function formatDateTime(value) {
  return value ? new Date(value).toLocaleString("zh-CN") : "-";
}

function downloadBlob(blob, filename) {
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
}

export default function UsersPage() {
  const { message } = App.useApp();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [walletOpen, setWalletOpen] = useState(false);
  const [correctionOpen, setCorrectionOpen] = useState(false);
  const [resetPasswordOpen, setResetPasswordOpen] = useState(false);
  const [ledgerOpen, setLedgerOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [walletSubmitting, setWalletSubmitting] = useState(false);
  const [correctionSubmitting, setCorrectionSubmitting] = useState(false);
  const [resetPasswordSubmitting, setResetPasswordSubmitting] = useState(false);
  const [ledgerLoading, setLedgerLoading] = useState(false);
  const [walletItem, setWalletItem] = useState(null);
  const [correctionItem, setCorrectionItem] = useState(null);
  const [ledgerItem, setLedgerItem] = useState(null);
  const [selectedUser, setSelectedUser] = useState(null);
  const [ledgerRows, setLedgerRows] = useState([]);
  const [ledgerPagination, setLedgerPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [createForm] = Form.useForm();
  const [walletForm] = Form.useForm();
  const [correctionForm] = Form.useForm();
  const [resetPasswordForm] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const response = await fetchUsers();
      setItems(response.items || []);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载用户失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const openCreateModal = () => {
    createForm.resetFields();
    createForm.setFieldsValue({ status: "active" });
    setCreateOpen(true);
  };

  const openWalletModal = (record) => {
    setWalletItem(record);
    walletForm.setFieldsValue({ amount: undefined, note: "" });
    setWalletOpen(true);
  };

  const openCorrectionModal = (record) => {
    setCorrectionItem(record);
    correctionForm.setFieldsValue({ amount: undefined, note: "" });
    setCorrectionOpen(true);
  };

  const openResetPasswordModal = (record) => {
    setSelectedUser(record);
    resetPasswordForm.setFieldsValue({ password: "" });
    setResetPasswordOpen(true);
  };

  const loadLedger = async (userId, page = 1, pageSize = 10) => {
    setLedgerLoading(true);
    try {
      const response = await fetchUserWalletLedger(userId, { page, page_size: pageSize });
      setLedgerRows(response.items || []);
      setLedgerPagination({
        current: response.page || page,
        pageSize: response.page_size || pageSize,
        total: response.total || 0,
      });
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载钱包流水失败");
    } finally {
      setLedgerLoading(false);
    }
  };

  const openLedgerDrawer = async (record) => {
    setLedgerItem(record);
    setLedgerOpen(true);
    await loadLedger(record.id, 1, 10);
  };

  const handleCreateUser = async (values) => {
    setCreating(true);
    try {
      await createUser(values);
      message.success("用户已创建");
      setCreateOpen(false);
      createForm.resetFields();
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "创建用户失败");
    } finally {
      setCreating(false);
    }
  };

  const handleToggleStatus = async (record) => {
    const nextStatus = record.status === "active" ? "disabled" : "active";
    try {
      await updateUserStatus(record.id, { status: nextStatus });
      message.success(nextStatus === "active" ? "用户已启用" : "用户已停用");
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "更新用户状态失败");
    }
  };

  const handleWalletSubmit = async (values) => {
    if (!walletItem) return;
    setWalletSubmitting(true);
    try {
      await adjustUserWallet(walletItem.id, values);
      message.success("充值成功");
      setWalletOpen(false);
      setWalletItem(null);
      walletForm.resetFields();
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "充值失败");
    } finally {
      setWalletSubmitting(false);
    }
  };

  const handleCorrectionSubmit = async (values) => {
    if (!correctionItem) return;
    setCorrectionSubmitting(true);
    try {
      await correctUserWallet(correctionItem.id, values);
      message.success("账务修正成功");
      setCorrectionOpen(false);
      setCorrectionItem(null);
      correctionForm.resetFields();
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "账务修正失败");
    } finally {
      setCorrectionSubmitting(false);
    }
  };

  const handleResetPassword = async (values) => {
    if (!selectedUser) return;
    setResetPasswordSubmitting(true);
    try {
      await resetUserPassword(selectedUser.id, values);
      message.success("密码已重置");
      setResetPasswordOpen(false);
      setSelectedUser(null);
      resetPasswordForm.resetFields();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "重置密码失败");
    } finally {
      setResetPasswordSubmitting(false);
    }
  };

  const handleExportBilling = async (record) => {
    try {
      const blob = await exportUserBilling(record.id);
      downloadBlob(blob, `user-billing-${record.email || record.id}.csv`);
      message.success("账单导出已开始");
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "导出账单失败");
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="用户管理"
        title="管理用户账号与钱包"
        description="创建和维护用户，管理钱包余额和账单，重置登录密码。"
        actions={
          <Button type="primary" onClick={openCreateModal}>
            新增用户
          </Button>
        }
      />

      <section className="panel-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={items}
          pagination={false}
          scroll={{ x: 1200 }}
          columns={[
            { title: "邮箱", dataIndex: "email", key: "email", width: 220 },
            { title: "姓名", dataIndex: "full_name", key: "full_name", width: 160 },
            {
              title: "状态",
              dataIndex: "status",
              key: "status",
              width: 100,
              render: (value) => (
                <Tag color={value === "active" ? "green" : "red"}>
                  {value === "active" ? "启用" : "停用"}
                </Tag>
              ),
            },
            {
              title: "钱包余额",
              dataIndex: "wallet_balance",
              key: "wallet_balance",
              width: 140,
              render: formatCurrency,
            },
            {
              title: "最低可调用余额",
              dataIndex: "min_available_balance",
              key: "min_available_balance",
              width: 160,
              render: formatCurrency,
            },
            { title: "最近登录", dataIndex: "last_login_at", key: "last_login_at", width: 180, render: formatDateTime },
            { title: "创建时间", dataIndex: "created_at", key: "created_at", width: 180, render: formatDateTime },
            {
              title: "操作",
              key: "actions",
              width: 380,
              fixed: "right",
              render: (_, record) => (
                <Space wrap>
                  <Button size="small" onClick={() => handleToggleStatus(record)}>
                    {record.status === "active" ? "停用" : "启用"}
                  </Button>
                  <Button size="small" onClick={() => openResetPasswordModal(record)}>
                    重置密码
                  </Button>
                  <Button size="small" type="primary" onClick={() => openWalletModal(record)}>
                    充值
                  </Button>
                  <Button size="small" danger onClick={() => openCorrectionModal(record)}>
                    修正
                  </Button>
                  <Button size="small" onClick={() => openLedgerDrawer(record)}>
                    流水
                  </Button>
                  <Button size="small" onClick={() => handleExportBilling(record)}>
                    导出账单
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </section>

      <Modal
        open={createOpen}
        title="新增用户"
        onCancel={() => { setCreateOpen(false); createForm.resetFields(); }}
        onOk={() => createForm.submit()}
        okButtonProps={{ loading: creating }}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical" onFinish={handleCreateUser}>
          <Form.Item label="邮箱" name="email" rules={[{ required: true, type: "email" }]}>
            <Input />
          </Form.Item>
          <Form.Item label="姓名" name="full_name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="初始密码" name="password" rules={[{ required: true, min: 8 }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item label="状态" name="status" initialValue="active">
            <Select
              options={[
                { label: "启用", value: "active" },
                { label: "停用", value: "disabled" },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={walletOpen}
        title={walletItem ? `充值 - ${walletItem.email}` : "用户充值"}
        onCancel={() => { setWalletOpen(false); setWalletItem(null); walletForm.resetFields(); }}
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
              <Input.TextArea rows={3} />
            </Form.Item>
          </Form>
        </Space>
      </Modal>

      <Modal
        open={correctionOpen}
        title={correctionItem ? `账务修正 - ${correctionItem.email}` : "账务修正"}
        onCancel={() => { setCorrectionOpen(false); setCorrectionItem(null); correctionForm.resetFields(); }}
        onOk={() => correctionForm.submit()}
        okButtonProps={{ loading: correctionSubmitting, danger: true }}
        destroyOnClose
      >
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          {correctionItem ? (
            <Typography.Text type="secondary">当前余额：{formatCurrency(correctionItem.wallet_balance)}</Typography.Text>
          ) : null}
          <Typography.Text type="warning">
            正数表示补账加余额，负数表示冲减余额。该操作用于人工修账，不是普通充值。
          </Typography.Text>
          <Form form={correctionForm} layout="vertical" onFinish={handleCorrectionSubmit}>
            <Form.Item label="修正金额" name="amount" rules={[{ required: true }]}>
              <InputNumber step={0.01} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item label="修正原因" name="note" rules={[{ required: true, whitespace: true }]}>
              <Input.TextArea rows={3} />
            </Form.Item>
          </Form>
        </Space>
      </Modal>

      <Modal
        open={resetPasswordOpen}
        title={selectedUser ? `重置密码 - ${selectedUser.email}` : "重置密码"}
        onCancel={() => { setResetPasswordOpen(false); setSelectedUser(null); resetPasswordForm.resetFields(); }}
        onOk={() => resetPasswordForm.submit()}
        okButtonProps={{ loading: resetPasswordSubmitting }}
        destroyOnClose
      >
        <Form form={resetPasswordForm} layout="vertical" onFinish={handleResetPassword}>
          <Form.Item label="新密码" name="password" rules={[{ required: true, min: 8 }]}>
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        open={ledgerOpen}
        width={960}
        title={ledgerItem ? `钱包流水 - ${ledgerItem.email}` : "钱包流水"}
        onClose={() => { setLedgerOpen(false); setLedgerItem(null); setLedgerRows([]); }}
      >
        <Table
          rowKey="id"
          loading={ledgerLoading}
          dataSource={ledgerRows}
          scroll={{ x: 1200 }}
          pagination={{
            current: ledgerPagination.current,
            pageSize: ledgerPagination.pageSize,
            total: ledgerPagination.total,
            showSizeChanger: true,
            pageSizeOptions: ["10", "20", "50", "100"],
            onChange: (page, pageSize) => {
              if (ledgerItem) loadLedger(ledgerItem.id, page, pageSize);
            },
          }}
          columns={[
            { title: "时间", dataIndex: "created_at", key: "created_at", width: 180, render: formatDateTime },
            {
              title: "方向",
              dataIndex: "direction",
              key: "direction",
              width: 100,
              render: (value) => (
                <Tag color={value === "credit" ? "green" : "red"}>{value === "credit" ? "充值" : "扣费"}</Tag>
              ),
            },
            { title: "金额", dataIndex: "amount", key: "amount", width: 120, render: formatCurrency },
            { title: "变动前", dataIndex: "balance_before", key: "balance_before", width: 120, render: formatCurrency },
            { title: "变动后", dataIndex: "balance_after", key: "balance_after", width: 120, render: formatCurrency },
            {
              title: "来源",
              dataIndex: "operator_type",
              key: "operator_type",
              width: 100,
              render: (value) => (value === "admin" ? "管理员" : value === "system" ? "系统" : value || "-"),
            },
            { title: "Trace ID", dataIndex: "trace_id", key: "trace_id", width: 180, render: (value) => value || "-" },
            { title: "模型", dataIndex: "model_public_name", key: "model_public_name", width: 140, render: (value) => value || "-" },
            { title: "Token", dataIndex: "total_tokens", key: "total_tokens", width: 100, render: (value) => value || 0 },
            { title: "预留金额", dataIndex: "reserved_amount", key: "reserved_amount", width: 120, render: formatCurrency },
            { title: "上游成本", dataIndex: "cost_amount", key: "cost_amount", width: 120, render: formatCurrency },
            { title: "对客计费", dataIndex: "billable_amount", key: "billable_amount", width: 120, render: formatCurrency },
            { title: "备注", dataIndex: "note", key: "note" },
          ]}
        />
      </Drawer>
    </Space>
  );
}
