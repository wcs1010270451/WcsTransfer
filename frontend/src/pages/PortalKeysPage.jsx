import { useEffect, useMemo, useState } from "react";
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Descriptions,
  Drawer,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
} from "antd";
import {
  createPortalClientKey,
  disablePortalClientKey,
  exportPortalBilling,
  fetchPortalClientKeys,
  fetchPortalLogDetail,
  fetchPortalLogs,
  fetchPortalMe,
  fetchPortalModels,
  fetchPortalStats,
  fetchPortalWalletLedger,
} from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";
import usePortalAuthStore from "../store/portalAuthStore";

function formatNumber(value) {
  return new Intl.NumberFormat("zh-CN").format(Number(value || 0));
}

function formatCurrency(value) {
  return new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 4,
    maximumFractionDigits: 4,
  }).format(Number(value || 0));
}

function formatJSON(value) {
  try {
    return JSON.stringify(typeof value === "string" ? JSON.parse(value) : value, null, 2);
  } catch {
    return String(value || "");
  }
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

export default function PortalKeysPage() {
  const { message, modal } = App.useApp();
  const [logFilterForm] = Form.useForm();
  const clearSession = usePortalAuthStore((state) => state.clearSession);
  const tenant = usePortalAuthStore((state) => state.tenant);
  const setSession = usePortalAuthStore((state) => state.setSession);
  const [items, setItems] = useState([]);
  const [models, setModels] = useState([]);
  const [stats, setStats] = useState(null);
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [logLoading, setLogLoading] = useState(false);
  const [logPagination, setLogPagination] = useState({ current: 1, pageSize: 20, total: 0 });
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [selectedLog, setSelectedLog] = useState(null);
  const [walletRows, setWalletRows] = useState([]);
  const [walletLoading, setWalletLoading] = useState(false);
  const [walletPagination, setWalletPagination] = useState({ current: 1, pageSize: 10, total: 0 });

  const loadBase = async () => {
    setLoading(true);
    try {
      const [me, keys, portalStats, portalModels] = await Promise.all([
        fetchPortalMe(),
        fetchPortalClientKeys(),
        fetchPortalStats(),
        fetchPortalModels(),
      ]);
      setSession({
        token: usePortalAuthStore.getState().token,
        user: me.user,
        tenant: me.tenant || null,
      });
      setItems(keys.items || []);
      setModels(portalModels.items || []);
      setStats(portalStats || null);
    } catch (error) {
      if (error.response?.status === 401) {
        clearSession();
      }
      message.error(error.response?.data?.error?.message || error.message || "加载租户工作台失败");
    } finally {
      setLoading(false);
    }
  };

  const buildLogParams = (page = logPagination.current, pageSize = logPagination.pageSize) => {
    const values = logFilterForm.getFieldsValue();
    return {
      page,
      page_size: pageSize,
      model_public_name: values.model_public_name || undefined,
      success: values.success === "all" || values.success == null ? undefined : values.success,
      http_status: values.http_status || undefined,
      trace_id: values.trace_id || undefined,
      created_from: values.created_at?.[0] ? values.created_at[0].toISOString() : undefined,
      created_to: values.created_at?.[1] ? values.created_at[1].toISOString() : undefined,
    };
  };

  const loadLogs = async (page = 1, pageSize = logPagination.pageSize) => {
    setLogLoading(true);
    try {
      const response = await fetchPortalLogs(buildLogParams(page, pageSize));
      setLogs(response.items || []);
      setLogPagination({
        current: response.page || page,
        pageSize: response.page_size || pageSize,
        total: response.total || 0,
      });
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载调用日志失败");
    } finally {
      setLogLoading(false);
    }
  };

  const loadWalletLedger = async (page = 1, pageSize = walletPagination.pageSize) => {
    setWalletLoading(true);
    try {
      const response = await fetchPortalWalletLedger({ page, page_size: pageSize });
      setWalletRows(response.items || []);
      setWalletPagination({
        current: response.page || page,
        pageSize: response.page_size || pageSize,
        total: response.total || 0,
      });
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载钱包流水失败");
    } finally {
      setWalletLoading(false);
    }
  };

  useEffect(() => {
    logFilterForm.setFieldsValue({ success: "all" });
    loadBase();
    loadLogs(1, 20);
    loadWalletLedger(1, 10);
  }, []);

  const handleCreate = async () => {
    const name = `app-${Date.now()}`;
    try {
      const created = await createPortalClientKey({
        name,
        description: "用户自助创建的客户端密钥",
      });
      modal.success({
        title: "客户端密钥已创建",
        content: (
          <Space direction="vertical" size={12}>
            <Typography.Text>明文密钥只展示一次，关闭后无法再次查看。</Typography.Text>
            <Typography.Paragraph copyable style={{ marginBottom: 0 }}>
              {created.plain_api_key}
            </Typography.Paragraph>
          </Space>
        ),
      });
      await loadBase();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "创建客户端密钥失败");
    }
  };

  const handleDisable = async (record) => {
    try {
      await disablePortalClientKey(record.id);
      message.success("客户端密钥已停用");
      await loadBase();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "停用客户端密钥失败");
    }
  };

  const handleLogout = () => {
    clearSession();
    window.location.href = `${window.location.origin}${window.location.pathname.replace(/\/portal\/keys.*$/, "/portal/login")}`;
  };

  const handleLogSearch = async () => {
    await loadLogs(1, logPagination.pageSize);
  };

  const handleLogReset = async () => {
    logFilterForm.resetFields();
    logFilterForm.setFieldsValue({ success: "all" });
    await loadLogs(1, logPagination.pageSize);
  };

  const handleExportBilling = async () => {
    try {
      const blob = await exportPortalBilling(buildLogParams(1, logPagination.pageSize));
      downloadBlob(blob, "portal-billing.csv");
      message.success("账单导出已开始");
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "导出账单失败");
    }
  };

  const handleLogTableChange = async (nextPagination) => {
    await loadLogs(nextPagination.current, nextPagination.pageSize);
  };

  const openLogDetail = async (record) => {
    setDetailOpen(true);
    setDetailLoading(true);
    try {
      const detail = await fetchPortalLogDetail(record.id);
      setSelectedLog(detail);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载日志详情失败");
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  const tenantActive = tenant?.status === "active";
  const modelOptions = useMemo(
    () => models.map((item) => ({ label: item.public_name, value: item.public_name })),
    [models],
  );

  const keyColumns = [
    { title: "名称", dataIndex: "name", key: "name", width: 180 },
    { title: "脱敏密钥", dataIndex: "masked_key", key: "masked_key", width: 180 },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (value) => <Tag color={value === "active" ? "green" : "default"}>{value === "active" ? "启用" : value}</Tag>,
    },
    {
      title: "最近使用",
      dataIndex: "last_used_at",
      key: "last_used_at",
      width: 180,
      render: (value) => (value ? new Date(value).toLocaleString("zh-CN") : "-"),
    },
    {
      title: "过期时间",
      dataIndex: "expires_at",
      key: "expires_at",
      width: 180,
      render: (value) => (value ? new Date(value).toLocaleString("zh-CN") : "-"),
    },
    {
      title: "操作",
      key: "actions",
      width: 120,
      fixed: "right",
      render: (_, record) => (
        <Popconfirm title="确定停用这把客户端密钥吗？" onConfirm={() => handleDisable(record)} disabled={record.status !== "active"}>
          <Button size="small" disabled={record.status !== "active"}>
            停用
          </Button>
        </Popconfirm>
      ),
    },
  ];

  const logColumns = [
    {
      title: "时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 180,
      render: (value) => (value ? new Date(value).toLocaleString("zh-CN") : "-"),
    },
    { title: "模型", dataIndex: "model_public_name", key: "model_public_name", width: 150, render: (value) => value || "-" },
    { title: "请求类型", dataIndex: "request_type", key: "request_type", width: 140 },
    {
      title: "状态",
      dataIndex: "success",
      key: "success",
      width: 100,
      render: (value) => <Tag color={value ? "green" : "red"}>{value ? "成功" : "失败"}</Tag>,
    },
    { title: "HTTP", dataIndex: "http_status", key: "http_status", width: 90 },
    {
      title: "Token",
      dataIndex: "total_tokens",
      key: "total_tokens",
      width: 110,
      render: (value) => formatNumber(value),
    },
    {
      title: "预留金额",
      dataIndex: "reserved_amount",
      key: "reserved_amount",
      width: 120,
      render: (value) => formatCurrency(value),
    },
    {
      title: "延迟",
      dataIndex: "latency_ms",
      key: "latency_ms",
      width: 110,
      render: (value) => `${value || 0} ms`,
    },
    { title: "错误信息", dataIndex: "error_message", key: "error_message", render: (value) => value || "-" },
    {
      title: "操作",
      key: "actions",
      width: 100,
      render: (_, record) => (
        <Button size="small" onClick={() => openLogDetail(record)}>
          详情
        </Button>
      ),
    },
  ];

  return (
    <Space direction="vertical" size={24} style={{ width: "100%", padding: 24 }}>
      <PageHeaderCard
        eyebrow="用户工作台"
        title={`客户端密钥${tenant?.name ? ` - ${tenant.name}` : ""}`}
        description="这里可以管理当前工作区的客户端密钥，并查看调用量、成本、可用模型和调用日志。"
        actions={
          <Space>
            <Button onClick={handleLogout}>退出登录</Button>
            <Button type="primary" onClick={handleCreate} disabled={!tenantActive}>
              创建客户端密钥
            </Button>
          </Space>
        }
      />

      {!tenantActive ? (
        <Alert
          type={tenant?.status === "disabled" ? "error" : "warning"}
          showIcon
          message={tenant?.status === "disabled" ? "工作区已停用" : "工作区待激活"}
          description={
            tenant?.status === "disabled"
              ? "当前工作区已被管理员停用，如需恢复请联系平台管理员。"
              : "当前工作区已注册成功，但还需要管理员激活并设置客户端密钥上限。"
          }
        />
      ) : null}

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="请求总数" value={stats?.request_count || 0} formatter={(value) => formatNumber(value)} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="总 Token" value={stats?.total_tokens || 0} formatter={(value) => formatNumber(value)} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="成功率" value={Number(stats?.success_rate || 0).toFixed(2)} suffix="%" /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="计费收入" value={stats?.billable_amount || 0} formatter={(value) => formatCurrency(value)} /></Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="输入 Token" value={stats?.prompt_tokens || 0} formatter={(value) => formatNumber(value)} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="输出 Token" value={stats?.completion_tokens || 0} formatter={(value) => formatNumber(value)} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="平均延迟" value={Number(stats?.average_latency_ms || 0).toFixed(0)} suffix="ms" /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="可用密钥数" value={stats?.active_client_keys || 0} formatter={(value) => formatNumber(value)} /></Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="总成本" value={stats?.cost_amount || 0} formatter={(value) => formatCurrency(value)} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="总毛利" value={stats?.gross_profit || 0} formatter={(value) => formatCurrency(value)} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="今日收入" value={stats?.today_billable_amount || 0} formatter={(value) => formatCurrency(value)} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="今日毛利" value={stats?.today_gross_profit || 0} formatter={(value) => formatCurrency(value)} /></Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="今日成本" value={stats?.today_cost_amount || 0} formatter={(value) => formatCurrency(value)} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="本月收入" value={stats?.month_billable_amount || 0} formatter={(value) => formatCurrency(value)} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="本月成本" value={stats?.month_cost_amount || 0} formatter={(value) => formatCurrency(value)} /></Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card><Statistic title="本月毛利" value={stats?.month_gross_profit || 0} formatter={(value) => formatCurrency(value)} /></Card>
        </Col>
      </Row>

      <section className="panel-card">
        <Typography.Title level={5} style={{ marginTop: 0 }}>客户端密钥</Typography.Title>
        <Table rowKey="id" loading={loading} dataSource={items} pagination={false} scroll={{ x: 900 }} columns={keyColumns} />
      </section>

      <section className="panel-card">
        <Typography.Title level={5} style={{ marginTop: 0 }}>可用模型</Typography.Title>
        <Space wrap>
          {models.length > 0 ? models.map((model) => <Tag key={model.id} color="blue">{model.public_name}</Tag>) : "暂无可用模型"}
        </Space>
      </section>

      <section className="panel-card">
        <Space direction="vertical" size={18} style={{ width: "100%" }}>
          <Space style={{ width: "100%", justifyContent: "space-between" }} wrap>
            <Typography.Title level={5} style={{ margin: 0 }}>调用日志</Typography.Title>
            <Space>
              <Button onClick={handleLogReset}>重置</Button>
              <Button onClick={handleExportBilling}>导出账单</Button>
              <Button type="primary" onClick={handleLogSearch}>查询</Button>
            </Space>
          </Space>

          <Form form={logFilterForm} layout="vertical">
            <Space wrap size={16} style={{ width: "100%" }}>
              <Form.Item label="模型" name="model_public_name" style={{ minWidth: 180, marginBottom: 0 }}>
                <Select allowClear options={modelOptions} placeholder="全部模型" />
              </Form.Item>
              <Form.Item label="状态" name="success" style={{ minWidth: 140, marginBottom: 0 }}>
                <Select options={[{ label: "全部", value: "all" }, { label: "成功", value: true }, { label: "失败", value: false }]} />
              </Form.Item>
              <Form.Item label="HTTP 状态码" name="http_status" style={{ width: 140, marginBottom: 0 }}>
                <InputNumber min={0} style={{ width: "100%" }} />
              </Form.Item>
              <Form.Item label="Trace ID" name="trace_id" style={{ minWidth: 220, marginBottom: 0 }}>
                <Input placeholder="支持模糊匹配" />
              </Form.Item>
              <Form.Item label="创建时间" name="created_at" style={{ minWidth: 320, marginBottom: 0 }}>
                <DatePicker.RangePicker showTime style={{ width: "100%" }} />
              </Form.Item>
            </Space>
          </Form>

          <Table
            rowKey="id"
            loading={logLoading}
            dataSource={logs}
            locale={{ emptyText: "暂无调用日志" }}
            scroll={{ x: 1200 }}
            pagination={{
              current: logPagination.current,
              pageSize: logPagination.pageSize,
              total: logPagination.total,
              showSizeChanger: true,
              pageSizeOptions: ["10", "20", "50", "100"],
            }}
            onChange={handleLogTableChange}
            columns={logColumns}
          />
        </Space>
      </section>

      {tenant ? (
        <section className="panel-card">
          <Typography.Title level={5} style={{ marginTop: 0 }}>工作区状态</Typography.Title>
          <Space wrap>
            <Tag color={tenantActive ? "green" : tenant?.status === "disabled" ? "red" : "gold"}>
              {tenant?.status === "active" ? "已激活" : tenant?.status === "disabled" ? "已停用" : "待激活"}
            </Tag>
            <Tag color="blue">客户端密钥上限：{tenant.max_client_keys}</Tag>
            <Tag>总密钥数：{formatNumber(stats?.client_key_count || 0)}</Tag>
            <Tag color={(tenant.wallet_balance || 0) > 0 ? "green" : "red"}>钱包余额：{formatCurrency(tenant.wallet_balance || 0)}</Tag>
          </Space>
          {tenant.notes ? <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 12 }}>{tenant.notes}</Typography.Paragraph> : null}
        </section>
      ) : null}

      <section className="panel-card">
        <Space direction="vertical" size={18} style={{ width: "100%" }}>
          <Typography.Title level={5} style={{ margin: 0 }}>钱包流水</Typography.Title>
          <Table
            rowKey="id"
            loading={walletLoading}
            dataSource={walletRows}
            locale={{ emptyText: "暂无钱包流水" }}
            scroll={{ x: 1100 }}
            pagination={{
              current: walletPagination.current,
              pageSize: walletPagination.pageSize,
              total: walletPagination.total,
              showSizeChanger: true,
              pageSizeOptions: ["10", "20", "50", "100"],
            }}
            onChange={(nextPagination) => loadWalletLedger(nextPagination.current, nextPagination.pageSize)}
            columns={[
              { title: "时间", dataIndex: "created_at", key: "created_at", width: 180, render: (value) => (value ? new Date(value).toLocaleString("zh-CN") : "-") },
              { title: "方向", dataIndex: "direction", key: "direction", width: 100, render: (value) => <Tag color={value === "credit" ? "green" : "red"}>{value === "credit" ? "充值" : "扣费"}</Tag> },
              { title: "金额", dataIndex: "amount", key: "amount", width: 120, render: (value) => formatCurrency(value) },
              { title: "变动前", dataIndex: "balance_before", key: "balance_before", width: 120, render: (value) => formatCurrency(value) },
              { title: "变动后", dataIndex: "balance_after", key: "balance_after", width: 120, render: (value) => formatCurrency(value) },
              { title: "来源", dataIndex: "operator_type", key: "operator_type", width: 100, render: (value) => (value === "admin" ? "管理员" : value === "system" ? "系统" : value || "-") },
              { title: "Trace ID", dataIndex: "trace_id", key: "trace_id", width: 180, render: (value) => value || "-" },
              { title: "模型", dataIndex: "model_public_name", key: "model_public_name", width: 140, render: (value) => value || "-" },
              { title: "Token", dataIndex: "total_tokens", key: "total_tokens", width: 100, render: (value) => formatNumber(value) },
              { title: "预留金额", dataIndex: "reserved_amount", key: "reserved_amount", width: 120, render: (value) => formatCurrency(value) },
              { title: "上游成本", dataIndex: "cost_amount", key: "cost_amount", width: 120, render: (value) => formatCurrency(value) },
              { title: "对客计费", dataIndex: "billable_amount", key: "billable_amount", width: 120, render: (value) => formatCurrency(value) },
              { title: "备注", dataIndex: "note", key: "note" },
            ]}
          />
        </Space>
      </section>

      <Drawer
        open={detailOpen}
        width={720}
        title="日志详情"
        onClose={() => {
          setDetailOpen(false);
          setSelectedLog(null);
        }}
      >
        {detailLoading || !selectedLog ? (
          <Typography.Text type="secondary">正在加载详情...</Typography.Text>
        ) : (
          <Space direction="vertical" size={20} style={{ width: "100%" }}>
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="Trace ID" span={2}>{selectedLog.trace_id || "-"}</Descriptions.Item>
              <Descriptions.Item label="模型">{selectedLog.model_public_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="上游模型">{selectedLog.upstream_model || "-"}</Descriptions.Item>
              <Descriptions.Item label="状态"><Tag color={selectedLog.success ? "green" : "red"}>{selectedLog.success ? "成功" : "失败"}</Tag></Descriptions.Item>
              <Descriptions.Item label="HTTP">{selectedLog.http_status}</Descriptions.Item>
              <Descriptions.Item label="延迟">{selectedLog.latency_ms} ms</Descriptions.Item>
              <Descriptions.Item label="输入 Tokens">{selectedLog.prompt_tokens}</Descriptions.Item>
              <Descriptions.Item label="输出 Tokens">{selectedLog.completion_tokens}</Descriptions.Item>
              <Descriptions.Item label="总 Tokens">{selectedLog.total_tokens}</Descriptions.Item>
              <Descriptions.Item label="预留金额">{formatCurrency(selectedLog.reserved_amount || 0)}</Descriptions.Item>
              <Descriptions.Item label="上游成本">{formatCurrency(selectedLog.cost_amount || 0)}</Descriptions.Item>
              <Descriptions.Item label="对客计费">{formatCurrency(selectedLog.billable_amount || 0)}</Descriptions.Item>
              <Descriptions.Item label="错误类型">{selectedLog.error_type || "-"}</Descriptions.Item>
              <Descriptions.Item label="错误信息" span={2}>{selectedLog.error_message || "-"}</Descriptions.Item>
            </Descriptions>

            <div>
              <Typography.Title level={5}>请求体</Typography.Title>
              <pre className="json-preview">{formatJSON(selectedLog.request_payload)}</pre>
            </div>
            <div>
              <Typography.Title level={5}>响应体</Typography.Title>
              <pre className="json-preview">{formatJSON(selectedLog.response_payload)}</pre>
            </div>
            <div>
              <Typography.Title level={5}>元数据</Typography.Title>
              <pre className="json-preview">{formatJSON(selectedLog.metadata)}</pre>
            </div>
          </Space>
        )}
      </Drawer>
    </Space>
  );
}
