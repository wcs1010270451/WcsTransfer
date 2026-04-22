import { useEffect, useState } from "react";
import { App, Button, Drawer, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, Typography } from "antd";
import {
  adjustTenantWallet,
  correctTenantWallet,
  createTenant,
  createTenantUser,
  exportTenantBilling,
  fetchTenants,
  fetchTenantUsers,
  fetchTenantWalletLedger,
  resetTenantUserPassword,
  updateTenant,
  updateTenantUserStatus,
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

export default function TenantsPage() {
  const { message } = App.useApp();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [walletOpen, setWalletOpen] = useState(false);
  const [correctionOpen, setCorrectionOpen] = useState(false);
  const [usersOpen, setUsersOpen] = useState(false);
  const [resetPasswordOpen, setResetPasswordOpen] = useState(false);
  const [ledgerOpen, setLedgerOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState(false);
  const [walletSubmitting, setWalletSubmitting] = useState(false);
  const [correctionSubmitting, setCorrectionSubmitting] = useState(false);
  const [usersSubmitting, setUsersSubmitting] = useState(false);
  const [resetPasswordSubmitting, setResetPasswordSubmitting] = useState(false);
  const [ledgerLoading, setLedgerLoading] = useState(false);
  const [usersLoading, setUsersLoading] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [walletItem, setWalletItem] = useState(null);
  const [correctionItem, setCorrectionItem] = useState(null);
  const [ledgerItem, setLedgerItem] = useState(null);
  const [usersItem, setUsersItem] = useState(null);
  const [selectedUser, setSelectedUser] = useState(null);
  const [ledgerRows, setLedgerRows] = useState([]);
  const [userRows, setUserRows] = useState([]);
  const [ledgerPagination, setLedgerPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [walletForm] = Form.useForm();
  const [correctionForm] = Form.useForm();
  const [userForm] = Form.useForm();
  const [resetPasswordForm] = Form.useForm();

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

  const openCreateModal = () => {
    createForm.setFieldsValue({
      name: "",
      slug: "",
      status: "pending",
      max_client_keys: 0,
      min_available_balance: 0.01,
      notes: "",
    });
    setCreateOpen(true);
  };

  const openEditModal = (record) => {
    setEditingItem(record);
    editForm.setFieldsValue({
      name: record.name,
      slug: record.slug,
      status: record.status,
      max_client_keys: record.max_client_keys,
      min_available_balance: record.min_available_balance,
      notes: record.notes,
    });
    setEditOpen(true);
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

  const loadLedger = async (tenantId, page = 1, pageSize = 10) => {
    setLedgerLoading(true);
    try {
      const response = await fetchTenantWalletLedger(tenantId, { page, page_size: pageSize });
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

  const loadUsers = async (tenantId) => {
    setUsersLoading(true);
    try {
      const response = await fetchTenantUsers(tenantId);
      setUserRows(response.items || []);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载租户用户失败");
    } finally {
      setUsersLoading(false);
    }
  };

  const openUsersDrawer = async (record) => {
    setUsersItem(record);
    userForm.setFieldsValue({
      email: "",
      full_name: "",
      password: "",
      status: "active",
    });
    setUsersOpen(true);
    await loadUsers(record.id);
  };

  const handleCreateTenant = async (values) => {
    setCreating(true);
    try {
      await createTenant(values);
      message.success("租户已创建");
      setCreateOpen(false);
      createForm.resetFields();
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "创建租户失败");
    } finally {
      setCreating(false);
    }
  };

  const handleUpdateTenant = async (values) => {
    if (!editingItem) {
      return;
    }
    setEditing(true);
    try {
      await updateTenant(editingItem.id, values);
      message.success("租户已更新");
      setEditOpen(false);
      setEditingItem(null);
      editForm.resetFields();
      await load();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "保存租户失败");
    } finally {
      setEditing(false);
    }
  };

  const handleWalletSubmit = async (values) => {
    if (!walletItem) {
      return;
    }
    setWalletSubmitting(true);
    try {
      await adjustTenantWallet(walletItem.id, values);
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
    if (!correctionItem) {
      return;
    }
    setCorrectionSubmitting(true);
    try {
      await correctTenantWallet(correctionItem.id, values);
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

  const handleCreateTenantUser = async (values) => {
    if (!usersItem) {
      return;
    }
    setUsersSubmitting(true);
    try {
      await createTenantUser(usersItem.id, values);
      message.success("租户用户已创建");
      userForm.setFieldsValue({
        email: "",
        full_name: "",
        password: "",
        status: "active",
      });
      await loadUsers(usersItem.id);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "创建租户用户失败");
    } finally {
      setUsersSubmitting(false);
    }
  };

  const handleToggleTenantUserStatus = async (record) => {
    if (!usersItem) {
      return;
    }
    const nextStatus = record.status === "active" ? "disabled" : "active";
    try {
      await updateTenantUserStatus(usersItem.id, record.id, { status: nextStatus });
      message.success(nextStatus === "active" ? "租户用户已启用" : "租户用户已停用");
      await loadUsers(usersItem.id);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "更新用户状态失败");
    }
  };

  const openResetPasswordModal = (record) => {
    setSelectedUser(record);
    resetPasswordForm.setFieldsValue({ password: "" });
    setResetPasswordOpen(true);
  };

  const handleResetTenantUserPassword = async (values) => {
    if (!usersItem || !selectedUser) {
      return;
    }
    setResetPasswordSubmitting(true);
    try {
      await resetTenantUserPassword(usersItem.id, selectedUser.id, values);
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
      const blob = await exportTenantBilling(record.id);
      downloadBlob(blob, `tenant-billing-${record.slug || record.id}.csv`);
      message.success("账单导出已开始");
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "导出账单失败");
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="租户管理"
        title="审核租户、维护钱包和管理租户用户"
        description="这里负责租户状态、钱包充值与修正、账单导出，以及为租户创建和维护登录用户。"
        actions={
          <Button type="primary" onClick={openCreateModal}>
            新增租户
          </Button>
        }
      />

      <section className="panel-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={items}
          pagination={false}
          scroll={{ x: 1400 }}
          columns={[
            { title: "名称", dataIndex: "name", key: "name", width: 180 },
            { title: "Slug", dataIndex: "slug", key: "slug", width: 180 },
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
              title: "最低可调用余额",
              dataIndex: "min_available_balance",
              key: "min_available_balance",
              width: 160,
              render: formatCurrency,
            },
            {
              title: "钱包余额",
              dataIndex: "wallet_balance",
              key: "wallet_balance",
              width: 140,
              render: formatCurrency,
            },
            { title: "备注", dataIndex: "notes", key: "notes" },
            {
              title: "操作",
              key: "actions",
              width: 380,
              fixed: "right",
              render: (_, record) => (
                <Space wrap>
                  <Button size="small" onClick={() => openEditModal(record)}>
                    编辑
                  </Button>
                  <Button size="small" type="primary" onClick={() => openWalletModal(record)}>
                    充值
                  </Button>
                  <Button size="small" danger onClick={() => openCorrectionModal(record)}>
                    修正
                  </Button>
                  <Button size="small" onClick={() => openUsersDrawer(record)}>
                    用户
                  </Button>
                  <Button size="small" onClick={() => handleExportBilling(record)}>
                    导出账单
                  </Button>
                  <Button size="small" onClick={() => openLedgerDrawer(record)}>
                    流水
                  </Button>
                </Space>
              ),
            },
          ]}
        />
      </section>

      <Modal
        open={createOpen}
        title="新增租户"
        onCancel={() => {
          setCreateOpen(false);
          createForm.resetFields();
        }}
        onOk={() => createForm.submit()}
        okButtonProps={{ loading: creating }}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical" onFinish={handleCreateTenant}>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="Slug" name="slug" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="状态" name="status" initialValue="pending">
            <Select
              options={[
                { label: "待审核", value: "pending" },
                { label: "启用", value: "active" },
                { label: "停用", value: "disabled" },
              ]}
            />
          </Form.Item>
          <Form.Item label="可建客户端密钥上限" name="max_client_keys" initialValue={0} extra="0 表示该租户不允许自助创建客户端密钥。">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="最低可调用余额" name="min_available_balance" initialValue={0.01}>
            <InputNumber min={0} step={0.01} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="备注" name="notes">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={editOpen}
        title="编辑租户"
        onCancel={() => {
          setEditOpen(false);
          setEditingItem(null);
          editForm.resetFields();
        }}
        onOk={() => editForm.submit()}
        okButtonProps={{ loading: editing }}
        destroyOnClose
      >
        <Form form={editForm} layout="vertical" onFinish={handleUpdateTenant}>
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
          <Form.Item label="可建客户端密钥上限" name="max_client_keys" extra="0 表示该租户不允许自助创建客户端密钥。">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="最低可调用余额" name="min_available_balance" extra="钱包余额低于此值时，业务接口会直接拒绝请求。">
            <InputNumber min={0} step={0.01} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item label="备注" name="notes">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={walletOpen}
        title={walletItem ? `租户充值 - ${walletItem.name}` : "租户充值"}
        onCancel={() => {
          setWalletOpen(false);
          setWalletItem(null);
          walletForm.resetFields();
        }}
        onOk={() => walletForm.submit()}
        okButtonProps={{ loading: walletSubmitting }}
        destroyOnClose
      >
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          {walletItem ? <Typography.Text type="secondary">当前余额：{formatCurrency(walletItem.wallet_balance)}</Typography.Text> : null}
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
        title={correctionItem ? `账务修正 - ${correctionItem.name}` : "账务修正"}
        onCancel={() => {
          setCorrectionOpen(false);
          setCorrectionItem(null);
          correctionForm.resetFields();
        }}
        onOk={() => correctionForm.submit()}
        okButtonProps={{ loading: correctionSubmitting, danger: true }}
        destroyOnClose
      >
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          {correctionItem ? <Typography.Text type="secondary">当前余额：{formatCurrency(correctionItem.wallet_balance)}</Typography.Text> : null}
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

      <Drawer
        open={usersOpen}
        width={980}
        title={usersItem ? `租户用户 - ${usersItem.name}` : "租户用户"}
        onClose={() => {
          setUsersOpen(false);
          setUsersItem(null);
          setUserRows([]);
          userForm.resetFields();
        }}
      >
        <Space direction="vertical" size={20} style={{ width: "100%" }}>
          <section className="panel-card">
            <Typography.Title level={5} style={{ marginTop: 0 }}>
              新增租户用户
            </Typography.Title>
            <Form form={userForm} layout="vertical" onFinish={handleCreateTenantUser}>
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
              <Button type="primary" htmlType="submit" loading={usersSubmitting}>
                创建用户
              </Button>
            </Form>
          </section>

          <section className="panel-card">
            <Table
              rowKey="id"
              loading={usersLoading}
              dataSource={userRows}
              pagination={false}
              scroll={{ x: 980 }}
              columns={[
                { title: "邮箱", dataIndex: "email", key: "email", width: 240 },
                { title: "姓名", dataIndex: "full_name", key: "full_name", width: 160 },
                {
                  title: "状态",
                  dataIndex: "status",
                  key: "status",
                  width: 100,
                  render: (value) => <Tag color={value === "active" ? "green" : "red"}>{value === "active" ? "启用" : "停用"}</Tag>,
                },
                { title: "最近登录", dataIndex: "last_login_at", key: "last_login_at", width: 180, render: formatDateTime },
                { title: "创建时间", dataIndex: "created_at", key: "created_at", width: 180, render: formatDateTime },
                {
                  title: "操作",
                  key: "actions",
                  width: 220,
                  render: (_, record) => (
                    <Space wrap>
                      <Button size="small" onClick={() => handleToggleTenantUserStatus(record)}>
                        {record.status === "active" ? "停用" : "启用"}
                      </Button>
                      <Button size="small" onClick={() => openResetPasswordModal(record)}>
                        重置密码
                      </Button>
                    </Space>
                  ),
                },
              ]}
            />
          </section>
        </Space>
      </Drawer>

      <Modal
        open={resetPasswordOpen}
        title={selectedUser ? `重置密码 - ${selectedUser.email}` : "重置密码"}
        onCancel={() => {
          setResetPasswordOpen(false);
          setSelectedUser(null);
          resetPasswordForm.resetFields();
        }}
        onOk={() => resetPasswordForm.submit()}
        okButtonProps={{ loading: resetPasswordSubmitting }}
        destroyOnClose
      >
        <Form form={resetPasswordForm} layout="vertical" onFinish={handleResetTenantUserPassword}>
          <Form.Item label="新密码" name="password" rules={[{ required: true, min: 8 }]}>
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        open={ledgerOpen}
        width={960}
        title={ledgerItem ? `钱包流水 - ${ledgerItem.name}` : "钱包流水"}
        onClose={() => {
          setLedgerOpen(false);
          setLedgerItem(null);
          setLedgerRows([]);
        }}
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
              if (ledgerItem) {
                loadLedger(ledgerItem.id, page, pageSize);
              }
            },
          }}
          columns={[
            { title: "时间", dataIndex: "created_at", key: "created_at", width: 180, render: formatDateTime },
            {
              title: "方向",
              dataIndex: "direction",
              key: "direction",
              width: 100,
              render: (value) => <Tag color={value === "credit" ? "green" : "red"}>{value === "credit" ? "充值" : "扣费"}</Tag>,
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
