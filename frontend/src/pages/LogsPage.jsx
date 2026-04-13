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
      message.error(error.response?.data?.error?.message || error.message || "Failed to load filter options");
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
      message.error(error.response?.data?.error?.message || error.message || "Failed to load logs");
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
      message.success("CSV export started");
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "Failed to export logs");
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
      message.error(error.response?.data?.error?.message || error.message || "Failed to load log detail");
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="Observability"
        title="Search logs, narrow failures, and inspect request details"
        description="This page is now a real troubleshooting entry point. Filter by provider, model, status, trace id, and HTTP code, then open any row to inspect the captured payload summary."
        actions={
          <Space>
            <Button onClick={handleExport}>Export CSV</Button>
            <Button onClick={handleReset}>Reset</Button>
            <Button type="primary" onClick={handleSearch}>
              Search
            </Button>
          </Space>
        }
      />

      <section className="panel-card">
        <Form form={form} layout="vertical">
          <Space wrap size={16} style={{ width: "100%" }}>
            <Form.Item label="Provider" name="provider_id" style={{ minWidth: 180, marginBottom: 0 }}>
              <Select allowClear options={providerOptions} placeholder="All providers" />
            </Form.Item>
            <Form.Item label="Model" name="model_public_name" style={{ minWidth: 180, marginBottom: 0 }}>
              <Select allowClear options={modelOptions} placeholder="All models" />
            </Form.Item>
            <Form.Item label="Status" name="success" style={{ minWidth: 140, marginBottom: 0 }}>
              <Select
                options={[
                  { label: "All", value: "all" },
                  { label: "Success", value: true },
                  { label: "Failed", value: false },
                ]}
              />
            </Form.Item>
            <Form.Item label="HTTP Status" name="http_status" style={{ width: 140, marginBottom: 0 }}>
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item label="Trace ID" name="trace_id" style={{ minWidth: 220, marginBottom: 0 }}>
              <Input placeholder="partial trace id" />
            </Form.Item>
            <Form.Item label="Created At" name="created_at" style={{ minWidth: 320, marginBottom: 0 }}>
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
            { title: "Provider", dataIndex: "provider_name", key: "provider_name", width: 140 },
            { title: "Key", dataIndex: "provider_key_name", key: "provider_key_name", width: 140 },
            { title: "Model", dataIndex: "model_public_name", key: "model_public_name", width: 160 },
            { title: "Request Type", dataIndex: "request_type", key: "request_type", width: 150 },
            {
              title: "Status",
              dataIndex: "success",
              key: "success",
              width: 110,
              render: (value) => <Tag color={value ? "green" : "red"}>{value ? "success" : "failed"}</Tag>,
            },
            { title: "HTTP", dataIndex: "http_status", key: "http_status", width: 90 },
            { title: "Latency", dataIndex: "latency_ms", key: "latency_ms", width: 100, render: (value) => `${value} ms` },
            { title: "Tokens", dataIndex: "total_tokens", key: "total_tokens", width: 100 },
            {
              title: "Created At",
              dataIndex: "created_at",
              key: "created_at",
              width: 180,
              render: (value) => (value ? new Date(value).toLocaleString() : "-"),
            },
            {
              title: "Actions",
              key: "actions",
              width: 100,
              render: (_, record) => (
                <Button size="small" onClick={() => openDetail(record)}>
                  Detail
                </Button>
              ),
            },
          ]}
        />
      </section>

      <Drawer
        open={detailOpen}
        width={720}
        title="Log Detail"
        onClose={() => {
          setDetailOpen(false);
          setSelectedLog(null);
        }}
      >
        {detailLoading || !selectedLog ? (
          <Typography.Text type="secondary">Loading detail...</Typography.Text>
        ) : (
          <Space direction="vertical" size={20} style={{ width: "100%" }}>
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="Trace ID" span={2}>
                {selectedLog.trace_id || "-"}
              </Descriptions.Item>
              <Descriptions.Item label="Provider">{selectedLog.provider_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="Key">{selectedLog.provider_key_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="Model">{selectedLog.model_public_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="Upstream Model">{selectedLog.upstream_model || "-"}</Descriptions.Item>
              <Descriptions.Item label="Status">
                <Tag color={selectedLog.success ? "green" : "red"}>{selectedLog.success ? "success" : "failed"}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="HTTP">{selectedLog.http_status}</Descriptions.Item>
              <Descriptions.Item label="Latency">{selectedLog.latency_ms} ms</Descriptions.Item>
              <Descriptions.Item label="Prompt Tokens">{selectedLog.prompt_tokens}</Descriptions.Item>
              <Descriptions.Item label="Completion Tokens">{selectedLog.completion_tokens}</Descriptions.Item>
              <Descriptions.Item label="Total Tokens">{selectedLog.total_tokens}</Descriptions.Item>
              <Descriptions.Item label="Error Type">{selectedLog.error_type || "-"}</Descriptions.Item>
              <Descriptions.Item label="Error Message" span={2}>
                {selectedLog.error_message || "-"}
              </Descriptions.Item>
            </Descriptions>

            <div>
              <Typography.Title level={5}>Request Payload</Typography.Title>
              <pre className="json-preview">{formatJSON(selectedLog.request_payload)}</pre>
            </div>
            <div>
              <Typography.Title level={5}>Response Payload</Typography.Title>
              <pre className="json-preview">{formatJSON(selectedLog.response_payload)}</pre>
            </div>
            <div>
              <Typography.Title level={5}>Metadata</Typography.Title>
              <pre className="json-preview">{formatJSON(selectedLog.metadata)}</pre>
            </div>
          </Space>
        )}
      </Drawer>
    </Space>
  );
}
