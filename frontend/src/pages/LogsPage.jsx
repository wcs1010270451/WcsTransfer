import { useEffect, useMemo, useState } from "react";
import {
  App,
  Button,
  DatePicker,
  Descriptions,
  Drawer,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from "antd";
import { exportLogs, fetchLogDetail, fetchLogs, fetchModels, fetchProviders } from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";

function formatJSON(value) {
  try {
    return JSON.stringify(typeof value === "string" ? JSON.parse(value) : value, null, 2);
  } catch {
    return String(value || "");
  }
}

export default function LogsPage() {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [providers, setProviders] = useState([]);
  const [models, setModels] = useState([]);
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 });
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [selectedLog, setSelectedLog] = useState(null);

  const filterValues = Form.useWatch([], form) || {};

  const loadOptions = async () => {
    try {
      const [providersResponse, modelsResponse] = await Promise.all([fetchProviders(), fetchModels()]);
      setProviders(providersResponse.items || []);
      setModels(modelsResponse.items || []);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载筛选项失败");
    }
  };

  const buildParams = (page = pagination.current, pageSize = pagination.pageSize, values = filterValues) => ({
    page,
    page_size: pageSize,
    provider_id: values.provider_id || undefined,
    model_public_name: values.model_public_name || undefined,
    success: values.success === "all" || values.success == null ? undefined : values.success,
    http_status: values.http_status || undefined,
    trace_id: values.trace_id || undefined,
    created_from: values.created_at?.[0] ? values.created_at[0].toISOString() : undefined,
    created_to: values.created_at?.[1] ? values.created_at[1].toISOString() : undefined,
  });

  const loadLogs = async (page = pagination.current, pageSize = pagination.pageSize, values = filterValues) => {
    setLoading(true);
    try {
      const response = await fetchLogs(buildParams(page, pageSize, values));
      setLogs(response.items || []);
      setPagination({
        current: response.page || page,
        pageSize: response.page_size || pageSize,
        total: response.total || 0,
      });
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载日志失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    form.setFieldsValue({ success: "all" });
    loadOptions();
    loadLogs(1, 20, { success: "all" });
  }, []);

  const providerOptions = useMemo(
    () => providers.map((item) => ({ label: item.name, value: item.id })),
    [providers],
  );

  const modelOptions = useMemo(
    () => models.map((item) => ({ label: item.public_name, value: item.public_name })),
    [models],
  );

  const handleSearch = async () => {
    await loadLogs(1, pagination.pageSize, form.getFieldsValue());
  };

  const handleReset = async () => {
    form.resetFields();
    const nextValues = { success: "all" };
    form.setFieldsValue(nextValues);
    await loadLogs(1, pagination.pageSize, nextValues);
  };

  const handleExport = async () => {
    try {
      const blob = await exportLogs(buildParams(1, pagination.pageSize, form.getFieldsValue()));
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "request_logs.csv";
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      message.success("CSV 导出已开始");
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "导出日志失败");
    }
  };

  const handleTableChange = async (nextPagination) => {
    await loadLogs(nextPagination.current, nextPagination.pageSize, form.getFieldsValue());
  };

  const openDetail = async (record) => {
    setDetailOpen(true);
    setDetailLoading(true);
    try {
      const detail = await fetchLogDetail(record.id);
      setSelectedLog(detail);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载日志详情失败");
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="可观测性"
        title="检索日志、缩小故障范围并查看请求详情"
        description="这里是实际排障入口。可以按提供方、模型、状态、Trace ID 和 HTTP 状态码筛选，再打开单条日志查看请求和响应摘要。"
        actions={
          <Space>
            <Button onClick={handleExport}>导出 CSV</Button>
            <Button onClick={handleReset}>重置</Button>
            <Button type="primary" onClick={handleSearch}>
              查询
            </Button>
          </Space>
        }
      />

      <section className="panel-card">
        <Form form={form} layout="vertical">
          <Space wrap size={16} style={{ width: "100%" }}>
            <Form.Item label="提供方" name="provider_id" style={{ minWidth: 180, marginBottom: 0 }}>
              <Select allowClear options={providerOptions} placeholder="全部提供方" />
            </Form.Item>
            <Form.Item label="模型" name="model_public_name" style={{ minWidth: 180, marginBottom: 0 }}>
              <Select allowClear options={modelOptions} placeholder="全部模型" />
            </Form.Item>
            <Form.Item label="状态" name="success" style={{ minWidth: 140, marginBottom: 0 }}>
              <Select
                options={[
                  { label: "全部", value: "all" },
                  { label: "成功", value: true },
                  { label: "失败", value: false },
                ]}
              />
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
      </section>

      <section className="panel-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={logs}
          scroll={{ x: 1400 }}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            pageSizeOptions: ["10", "20", "50", "100"],
          }}
          onChange={handleTableChange}
          columns={[
            { title: "Trace ID", dataIndex: "trace_id", key: "trace_id", width: 220, ellipsis: true },
            { title: "客户端", dataIndex: "client_api_key_name", key: "client_api_key_name", width: 160 },
            { title: "提供方", dataIndex: "provider_name", key: "provider_name", width: 140 },
            { title: "密钥", dataIndex: "provider_key_name", key: "provider_key_name", width: 140 },
            { title: "模型", dataIndex: "model_public_name", key: "model_public_name", width: 160 },
            { title: "请求类型", dataIndex: "request_type", key: "request_type", width: 150 },
            {
              title: "状态",
              dataIndex: "success",
              key: "success",
              width: 110,
              render: (value) => <Tag color={value ? "green" : "red"}>{value ? "成功" : "失败"}</Tag>,
            },
            { title: "HTTP", dataIndex: "http_status", key: "http_status", width: 90 },
            { title: "延迟", dataIndex: "latency_ms", key: "latency_ms", width: 100, render: (value) => `${value} ms` },
            { title: "Tokens", dataIndex: "total_tokens", key: "total_tokens", width: 100 },
            {
              title: "创建时间",
              dataIndex: "created_at",
              key: "created_at",
              width: 180,
              render: (value) => (value ? new Date(value).toLocaleString() : "-"),
            },
            {
              title: "操作",
              key: "actions",
              width: 100,
              render: (_, record) => (
                <Button size="small" onClick={() => openDetail(record)}>
                  详情
                </Button>
              ),
            },
          ]}
        />
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
              <Descriptions.Item label="Trace ID" span={2}>
                {selectedLog.trace_id || "-"}
              </Descriptions.Item>
              <Descriptions.Item label="提供方">{selectedLog.provider_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="密钥">{selectedLog.provider_key_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="客户端">{selectedLog.client_api_key_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="模型">{selectedLog.model_public_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="上游模型">{selectedLog.upstream_model || "-"}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={selectedLog.success ? "green" : "red"}>{selectedLog.success ? "成功" : "失败"}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="HTTP">{selectedLog.http_status}</Descriptions.Item>
              <Descriptions.Item label="延迟">{selectedLog.latency_ms} ms</Descriptions.Item>
              <Descriptions.Item label="输入 Tokens">{selectedLog.prompt_tokens}</Descriptions.Item>
              <Descriptions.Item label="输出 Tokens">{selectedLog.completion_tokens}</Descriptions.Item>
              <Descriptions.Item label="总 Tokens">{selectedLog.total_tokens}</Descriptions.Item>
              <Descriptions.Item label="错误类型">{selectedLog.error_type || "-"}</Descriptions.Item>
              <Descriptions.Item label="错误信息" span={2}>
                {selectedLog.error_message || "-"}
              </Descriptions.Item>
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
