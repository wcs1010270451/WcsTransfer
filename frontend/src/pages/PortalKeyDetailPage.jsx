import { useEffect, useRef, useState } from "react";
import {
  App,
  Button,
  Checkbox,
  Descriptions,
  Drawer,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from "antd";
import { ArrowLeftOutlined, SendOutlined } from "@ant-design/icons";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { fetchPortalClientKeys, fetchPortalKeyModelStats, fetchPortalLogDetail, fetchPortalLogs, fetchPortalModels, sendPortalDebugChat } from "../api/client";

function fmtCurrency(value) {
  return new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 4,
    maximumFractionDigits: 4,
  }).format(Number(value || 0));
}

function fmtJSON(value) {
  try {
    return JSON.stringify(typeof value === "string" ? JSON.parse(value) : value, null, 2);
  } catch {
    return String(value || "");
  }
}

export default function PortalKeyDetailPage() {
  const { id } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  const { message } = App.useApp();

  const [keyInfo, setKeyInfo] = useState(location.state?.key || null);
  const [modelStats, setModelStats] = useState([]);
  const [modelStatsLoading, setModelStatsLoading] = useState(true);
  const [logs, setLogs] = useState([]);
  const [logLoading, setLogLoading] = useState(false);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 });
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [selectedLog, setSelectedLog] = useState(null);

  const [debugOpen, setDebugOpen] = useState(false);
  const [models, setModels] = useState([]);
  const [debugModel, setDebugModel] = useState("");
  const [debugMessage, setDebugMessage] = useState("");
  const [debugStream, setDebugStream] = useState(true);
  const [debugLoading, setDebugLoading] = useState(false);
  const [debugResult, setDebugResult] = useState(null);
  const abortCtrlRef = useRef(null);

  useEffect(() => {
    if (!keyInfo) {
      fetchPortalClientKeys()
        .then((res) => {
          const found = (res.items || []).find((k) => String(k.id) === id);
          if (found) setKeyInfo(found);
        })
        .catch(() => {});
    }

    setModelStatsLoading(true);
    fetchPortalKeyModelStats(id)
      .then((res) => setModelStats(res.items || []))
      .catch(() => {})
      .finally(() => setModelStatsLoading(false));

    fetchPortalModels()
      .then((res) => {
        const list = res.items || [];
        setModels(list);
        if (list.length > 0 && !debugModel) setDebugModel(list[0].public_name);
      })
      .catch(() => {});

    loadLogs(1, 20);
  }, [id]);

  const loadLogs = async (page, pageSize) => {
    setLogLoading(true);
    try {
      const res = await fetchPortalLogs({ page, page_size: pageSize, client_api_key_id: id });
      setLogs(res.items || []);
      setPagination({ current: res.page || page, pageSize: res.page_size || pageSize, total: res.total || 0 });
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载调用日志失败");
    } finally {
      setLogLoading(false);
    }
  };

  const openDetail = async (record) => {
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

  const sendDebug = async () => {
    if (!debugMessage.trim() || !debugModel) return;
    if (abortCtrlRef.current) abortCtrlRef.current.abort();
    const ctrl = new AbortController();
    abortCtrlRef.current = ctrl;
    setDebugLoading(true);
    setDebugResult(null);
    try {
      const payload = { model: debugModel, messages: [{ role: "user", content: debugMessage.trim() }], stream: debugStream };
      const result = await sendPortalDebugChat(id, payload, {
        signal: ctrl.signal,
        onUpdate: (update) => setDebugResult({ ...update }),
      });
      setDebugResult(result);
    } catch (err) {
      if (err.name === "AbortError") return;
      const errMsg = err.response?.data?.error?.message || err.message || "调试请求失败";
      message.error(errMsg);
      setDebugResult({ error: errMsg });
    } finally {
      setDebugLoading(false);
      abortCtrlRef.current = null;
    }
  };

  const logColumns = [
    {
      title: "时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 180,
      render: (v) => (v ? new Date(v).toLocaleString("zh-CN") : "-"),
    },
    { title: "模型", dataIndex: "model_public_name", key: "model_public_name", width: 200, render: (v) => v || "-" },
    { title: "请求类型", dataIndex: "request_type", key: "request_type", width: 120 },
    {
      title: "状态",
      dataIndex: "success",
      key: "success",
      width: 80,
      render: (v) => <Tag color={v ? "blue" : "red"}>{v ? "成功" : "失败"}</Tag>,
    },
    { title: "HTTP", dataIndex: "http_status", key: "http_status", width: 70 },
    {
      title: "Token",
      dataIndex: "total_tokens",
      key: "total_tokens",
      width: 90,
      render: (v) => new Intl.NumberFormat("zh-CN").format(v || 0),
    },
    { title: "延迟", dataIndex: "latency_ms", key: "latency_ms", width: 90, render: (v) => `${v || 0} ms` },
    {
      title: "错误信息",
      dataIndex: "error_message",
      key: "error_message",
      width: 200,
      ellipsis: true,
      render: (v) => v ? (
        <Tooltip title={v} placement="topLeft">
          <span>{v}</span>
        </Tooltip>
      ) : "-",
    },
  ];

  return (
    <Space direction="vertical" size={24} style={{ width: "100%", padding: 24 }}>
      <Space>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate("/portal/keys")}
          style={{ paddingLeft: 0 }}
        >
          返回
        </Button>
        <Typography.Title level={4} style={{ margin: 0 }}>密钥详情</Typography.Title>
      </Space>

      <section className="panel-card">
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 12 }}>
          <Typography.Title level={5} style={{ margin: 0 }}>基本信息</Typography.Title>
          <Button type="primary" ghost size="small" onClick={() => setDebugOpen(true)}>API 调试</Button>
        </div>
        <div style={{ overflowX: "auto" }}>
          <Descriptions bordered column={3} size="small" style={{ minWidth: 600 }}>
            <Descriptions.Item label="名称">{keyInfo?.name || "-"}</Descriptions.Item>
            <Descriptions.Item label="状态">
              {keyInfo ? (
                <Tag color={keyInfo.status === "active" ? "blue" : "default"}>
                  {keyInfo.status === "active" ? "启用" : "停用"}
                </Tag>
              ) : "-"}
            </Descriptions.Item>
            <Descriptions.Item label="脱敏密钥">{keyInfo?.masked_key || "-"}</Descriptions.Item>
            <Descriptions.Item label="今日消费">{fmtCurrency(keyInfo?.cost_usage?.daily_cost_used)}</Descriptions.Item>
            <Descriptions.Item label="本月消费">{fmtCurrency(keyInfo?.cost_usage?.monthly_cost_used)}</Descriptions.Item>
            <Descriptions.Item label="累计消费">{fmtCurrency(keyInfo?.cost_usage?.total_cost_used)}</Descriptions.Item>
            <Descriptions.Item label="最近使用">
              {keyInfo?.last_used_at ? new Date(keyInfo.last_used_at).toLocaleString("zh-CN") : "-"}
            </Descriptions.Item>
            <Descriptions.Item label="过期时间">
              {keyInfo?.expires_at ? new Date(keyInfo.expires_at).toLocaleString("zh-CN") : "永不过期"}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {keyInfo?.created_at ? new Date(keyInfo.created_at).toLocaleString("zh-CN") : "-"}
            </Descriptions.Item>
          </Descriptions>
        </div>
      </section>

      <section className="panel-card">
        <Typography.Title level={5} style={{ marginTop: 0, marginBottom: 16 }}>模型用量</Typography.Title>
        <Table
          rowKey="model_public_name"
          loading={modelStatsLoading}
          dataSource={modelStats}
          pagination={false}
          scroll={{ x: "max-content" }}
          locale={{ emptyText: "暂无调用记录" }}
          columns={[
            { title: "模型", dataIndex: "model_public_name", key: "model_public_name", minWidth: 160 },
            { title: "请求数", dataIndex: "request_count", key: "request_count", width: 100, render: (v) => new Intl.NumberFormat("zh-CN").format(v) },
            { title: "输入 Token", dataIndex: "prompt_tokens", key: "prompt_tokens", width: 130, render: (v) => new Intl.NumberFormat("zh-CN").format(v) },
            { title: "输出 Token", dataIndex: "completion_tokens", key: "completion_tokens", width: 130, render: (v) => new Intl.NumberFormat("zh-CN").format(v) },
            { title: "总 Token", dataIndex: "total_tokens", key: "total_tokens", width: 120, render: (v) => new Intl.NumberFormat("zh-CN").format(v) },
            { title: "消费金额", dataIndex: "billable_amount", key: "billable_amount", width: 140, render: (v) => fmtCurrency(v) },
          ]}
        />
      </section>

      <section className="panel-card">
        <Typography.Title level={5} style={{ marginTop: 0, marginBottom: 16 }}>调用日志</Typography.Title>
        <Table
          rowKey="id"
          loading={logLoading}
          dataSource={logs}
          locale={{ emptyText: "暂无调用日志" }}
          scroll={{ x: 1000 }}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            pageSizeOptions: ["10", "20", "50", "100"],
          }}
          onRow={(record) => ({ onClick: () => openDetail(record), style: { cursor: "pointer" } })}
          onChange={(p) => loadLogs(p.current, p.pageSize)}
          columns={logColumns}
        />
      </section>

      <Modal
        open={debugOpen}
        title="API 调试"
        width={640}
        footer={null}
        onCancel={() => {
          abortCtrlRef.current?.abort();
          setDebugOpen(false);
          setDebugResult(null);
          setDebugMessage("");
        }}
        destroyOnClose
      >
        <Space direction="vertical" style={{ width: "100%" }} size={12}>
          <Space wrap align="center">
            <Select
              value={debugModel}
              onChange={setDebugModel}
              style={{ width: 240 }}
              placeholder="选择模型"
              options={models.map((m) => ({ value: m.public_name, label: m.public_name }))}
            />
            <Checkbox checked={debugStream} onChange={(e) => setDebugStream(e.target.checked)}>
              流式输出
            </Checkbox>
          </Space>
          <Input.TextArea
            value={debugMessage}
            onChange={(e) => setDebugMessage(e.target.value)}
            rows={4}
            placeholder="输入消息内容，Enter 发送，Shift+Enter 换行"
            onPressEnter={(e) => { if (!e.shiftKey) { e.preventDefault(); sendDebug(); } }}
          />
          <Space>
            <Button
              type="primary"
              icon={<SendOutlined />}
              loading={debugLoading}
              onClick={sendDebug}
              disabled={!debugMessage.trim() || !debugModel}
            >
              发送
            </Button>
            {debugLoading && (
              <Button onClick={() => abortCtrlRef.current?.abort()}>停止</Button>
            )}
          </Space>
          {debugResult && (
            <div>
              {debugResult.error ? (
                <Typography.Text type="danger">{debugResult.error}</Typography.Text>
              ) : (
                <>
                  <pre className="json-preview" style={{ minHeight: 60 }}>
                    {debugResult.assistantText || "（等待响应...）"}
                  </pre>
                  {debugResult.usage && (
                    <Space size={16} style={{ marginTop: 8, flexWrap: "wrap" }}>
                      <Typography.Text type="secondary">输入 Token: {debugResult.usage.prompt_tokens ?? "-"}</Typography.Text>
                      <Typography.Text type="secondary">输出 Token: {debugResult.usage.completion_tokens ?? "-"}</Typography.Text>
                      <Typography.Text type="secondary">总 Token: {debugResult.usage.total_tokens ?? "-"}</Typography.Text>
                    </Space>
                  )}
                </>
              )}
            </div>
          )}
        </Space>
      </Modal>

      <Drawer
        open={detailOpen}
        width={720}
        title="日志详情"
        onClose={() => { setDetailOpen(false); setSelectedLog(null); }}
      >
        {detailLoading || !selectedLog ? (
          <Typography.Text type="secondary">正在加载详情...</Typography.Text>
        ) : (
          <Space direction="vertical" size={20} style={{ width: "100%" }}>
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="Trace ID" span={2}>{selectedLog.trace_id || "-"}</Descriptions.Item>
              <Descriptions.Item label="模型">{selectedLog.model_public_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="上游模型">{selectedLog.upstream_model || "-"}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={selectedLog.success ? "blue" : "red"}>{selectedLog.success ? "成功" : "失败"}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="HTTP">{selectedLog.http_status}</Descriptions.Item>
              <Descriptions.Item label="延迟">{selectedLog.latency_ms} ms</Descriptions.Item>
              <Descriptions.Item label="输入 Tokens">{selectedLog.prompt_tokens}</Descriptions.Item>
              <Descriptions.Item label="输出 Tokens">{selectedLog.completion_tokens}</Descriptions.Item>
              <Descriptions.Item label="总 Tokens">{selectedLog.total_tokens}</Descriptions.Item>
              <Descriptions.Item label="消费金额">{fmtCurrency(selectedLog.billable_amount || 0)}</Descriptions.Item>
              <Descriptions.Item label="错误类型">{selectedLog.error_type || "-"}</Descriptions.Item>
              <Descriptions.Item label="错误信息" span={2}>{selectedLog.error_message || "-"}</Descriptions.Item>
            </Descriptions>
            <div>
              <Typography.Title level={5}>请求体</Typography.Title>
              <pre className="json-preview">{fmtJSON(selectedLog.request_payload)}</pre>
            </div>
            <div>
              <Typography.Title level={5}>响应体</Typography.Title>
              <pre className="json-preview">{fmtJSON(selectedLog.response_payload)}</pre>
            </div>
            <div>
              <Typography.Title level={5}>元数据</Typography.Title>
              <pre className="json-preview">{fmtJSON(selectedLog.metadata)}</pre>
            </div>
          </Space>
        )}
      </Drawer>
    </Space>
  );
}
