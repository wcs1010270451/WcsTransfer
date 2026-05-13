import { useEffect, useState } from "react";
import { Alert, Col, List, Progress, Row, Space, Table, Tag, Typography, Card } from "antd";
import {
  ArrowUpOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  DashboardOutlined,
  InfoCircleOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import MetricCard from "../components/MetricCard";
import PageHeaderCard from "../components/PageHeaderCard";
import { fetchHealth, fetchLogs, fetchStats } from "../api/client";

function formatCurrency(value) {
  return `$${Number(value || 0).toFixed(4)}`;
}

function formatNumber(value) {
  return new Intl.NumberFormat().format(value || 0);
}

export default function DashboardPage() {
  const [state, setState] = useState({
    loading: true,
    error: "",
    health: null,
    stats: null,
    logs: [],
  });

  useEffect(() => {
    let active = true;
    const load = async () => {
      try {
        const [health, stats, logs] = await Promise.all([fetchHealth(), fetchStats(), fetchLogs(10)]);
        if (!active) return;
        setState({
          loading: false,
          error: "",
          health,
          stats,
          logs: logs.items || [],
        });
      } catch (error) {
        if (!active) return;
        setState((prev) => ({
          ...prev,
          loading: false,
          error: error.message || "加载总览数据失败",
        }));
      }
    };

    load();
    const interval = setInterval(load, 30000); // Refresh every 30s
    return () => {
      active = false;
      clearInterval(interval);
    };
  }, []);

  const { stats, health, logs } = state;

  return (
    <Space direction="vertical" size={32} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="网关总览"
        title="集中查看网关健康度、流量质量和账单摘要"
        description="实时监控提供方状态、模型调用分布及营收数据。系统每 30 秒自动刷新一次。"
      />

      {state.error ? <Alert type="error" showIcon message={state.error} style={{ borderRadius: 12 }} /> : null}

      {/* ── Top Metrics ── */}
      <Row gutter={[20, 20]}>
        <Col xs={24} sm={12} xl={6}>
          <MetricCard
            title="累计请求 (24h)"
            value={formatNumber(stats?.request_count)}
            hint={`成功率 ${(stats?.success_rate || 0).toFixed(1)}%`}
          />
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <MetricCard
            title="总 Token (24h)"
            value={formatNumber(stats?.total_tokens)}
            hint={`平均延迟 ${Number(stats?.average_latency_ms || 0).toFixed(0)}ms`}
          />
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <MetricCard
            title="今日毛利"
            value={formatCurrency(stats?.today_gross_profit)}
            hint={`今日收入 ${formatCurrency(stats?.today_billable_amount)}`}
          />
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <MetricCard
            title="本月毛利"
            value={formatCurrency(stats?.month_gross_profit)}
            hint={`本月收入 ${formatCurrency(stats?.month_billable_amount)}`}
          />
        </Col>
      </Row>

      <Row gutter={[20, 20]}>
        {/* ── Health & Traffic ── */}
        <Col xs={24} xl={8}>
          <section className="panel-card">
            <div className="section-label">服务健康状态</div>
            {health ? (
              <Space direction="vertical" size={16} style={{ width: "100%" }}>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                  <Space>
                    <div
                      style={{
                        width: 10,
                        height: 10,
                        borderRadius: "50%",
                        backgroundColor: health.status === "ok" ? "#10b981" : "#f59e0b",
                        boxShadow: `0 0 10px ${health.status === "ok" ? "#10b981" : "#f59e0b"}`,
                      }}
                    />
                    <Typography.Text strong style={{ fontSize: 16 }}>
                      {health.status === "ok" ? "所有系统运行正常" : "系统存在异常"}
                    </Typography.Text>
                  </Space>
                  <Tag bordered={false} color="blue">{health.environment}</Tag>
                </div>

                <List
                  dataSource={Object.entries(health.dependencies || {})}
                  renderItem={([name, value]) => (
                    <List.Item style={{ padding: "8px 0" }}>
                      <Typography.Text>{name}</Typography.Text>
                      <Tag
                        bordered={false}
                        color={value.status === "up" ? "success" : value.status === "disabled" ? "default" : "error"}
                        icon={value.status === "up" ? <CheckCircleOutlined /> : <InfoCircleOutlined />}
                      >
                        {value.status.toUpperCase()}
                      </Tag>
                    </List.Item>
                  )}
                />
              </Space>
            ) : (
              <Typography.Text type="secondary">正在检查服务状态...</Typography.Text>
            )}
          </section>
        </Col>

        <Col xs={24} xl={16}>
          <section className="panel-card">
            <div className="section-label">资源压力监控 (Top 5)</div>
            <Row gutter={24}>
              <Col span={12}>
                <Typography.Text type="secondary" style={{ fontSize: 12, display: "block", marginBottom: 12 }}>
                  配额压力 (RPM/Daily)
                </Typography.Text>
                {stats?.quota_pressure?.length > 0 ? (
                  <List
                    dataSource={stats.quota_pressure}
                    renderItem={(item) => (
                      <div style={{ marginBottom: 12 }}>
                        <div style={{ display: "flex", justifyContent: "space-between", fontSize: 12, marginBottom: 4 }}>
                          <Typography.Text ellipsis style={{ maxWidth: "60%" }}>{item.client_api_key_name}</Typography.Text>
                          <Typography.Text type={item.highest_usage_percent > 80 ? "danger" : "secondary"}>
                            {item.highest_usage_percent.toFixed(0)}%
                          </Typography.Text>
                        </div>
                        <Progress
                          percent={item.highest_usage_percent}
                          size="small"
                          status={item.highest_usage_percent > 90 ? "exception" : "active"}
                          showInfo={false}
                          strokeColor={item.highest_usage_percent > 80 ? "#ef4444" : "#4f46e5"}
                        />
                      </div>
                    )}
                  />
                ) : (
                  <Typography.Text type="secondary" style={{ fontSize: 13 }}>暂无配额压力</Typography.Text>
                )}
              </Col>
              <Col span={12}>
                <Typography.Text type="secondary" style={{ fontSize: 12, display: "block", marginBottom: 12 }}>
                  财务预算压力
                </Typography.Text>
                {stats?.budget_pressure?.length > 0 ? (
                  <List
                    dataSource={stats.budget_pressure}
                    renderItem={(item) => (
                      <div style={{ marginBottom: 12 }}>
                        <div style={{ display: "flex", justifyContent: "space-between", fontSize: 12, marginBottom: 4 }}>
                          <Typography.Text ellipsis style={{ maxWidth: "60%" }}>{item.client_api_key_name}</Typography.Text>
                          <Typography.Text type={item.highest_usage_percent > 80 ? "danger" : "secondary"}>
                            {item.highest_usage_percent.toFixed(0)}%
                          </Typography.Text>
                        </div>
                        <Progress
                          percent={item.highest_usage_percent}
                          size="small"
                          showInfo={false}
                          strokeColor={item.is_warning_triggered ? "#f59e0b" : "#0ea5e9"}
                        />
                      </div>
                    )}
                  />
                ) : (
                  <Typography.Text type="secondary" style={{ fontSize: 13 }}>暂无预算压力</Typography.Text>
                )}
              </Col>
            </Row>
          </section>
        </Col>
      </Row>

      <Row gutter={[20, 20]}>
        {/* ── Top Usage ── */}
        <Col xs={24} lg={12}>
          <section className="panel-card">
            <div className="section-label">高频调用模型 (24h)</div>
            <Table
              size="small"
              dataSource={stats?.top_models}
              pagination={false}
              rowKey="model_public_name"
              columns={[
                { title: "模型名称", dataKey: "model_public_name", render: (text) => <Typography.Text strong>{text}</Typography.Text> },
                { title: "请求数", dataIndex: "request_count", align: "right", render: (v) => formatNumber(v) },
                { title: "成功率", dataIndex: "success_rate", align: "right", render: (v) => <Tag bordered={false} color={v > 98 ? "success" : "warning"}>{v.toFixed(1)}%</Tag> },
                { title: "平均延迟", dataIndex: "average_latency_ms", align: "right", render: (v) => `${v.toFixed(0)}ms` },
              ]}
            />
          </section>
        </Col>

        <Col xs={24} lg={12}>
          <section className="panel-card">
            <div className="section-label">主要营收客户端 (24h)</div>
            <Table
              size="small"
              dataSource={stats?.top_clients}
              pagination={false}
              rowKey="client_api_key_id"
              columns={[
                { title: "客户端", dataIndex: "client_api_key_name", render: (text) => <Typography.Text strong>{text}</Typography.Text> },
                { title: "请求数", dataIndex: "request_count", align: "right", render: (v) => formatNumber(v) },
                { title: "消费金额", dataIndex: "billable_amount", align: "right", render: (v) => formatCurrency(v) },
                { title: "毛利", render: (_, record) => (
                  <Typography.Text type="success">
                    {formatCurrency(record.billable_amount - record.cost_amount)}
                  </Typography.Text>
                )},
              ]}
            />
          </section>
        </Col>
      </Row>

      <Row gutter={[20, 20]}>
        <Col xs={24}>
          <section className="panel-card">
            <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 20 }}>
              <div className="section-label" style={{ margin: 0 }}>最近请求流水</div>
              <Typography.Link onClick={() => window.location.hash = "/logs"}>查看全部日志</Typography.Link>
            </div>
            <List
              dataSource={logs}
              locale={{ emptyText: "暂无请求日志" }}
              renderItem={(item) => (
                <List.Item style={{ borderBottom: "1px solid rgba(0,0,0,0.04)" }}>
                  <div className="log-list-item">
                    <Space size="large">
                      <div style={{ minWidth: 140 }}>
                        <Typography.Text strong style={{ fontSize: 14 }}>{item.model_public_name || "未知模型"}</Typography.Text>
                        <div className="log-subtitle">{item.request_type} · {item.client_api_key_name}</div>
                      </div>
                      <Tag bordered={false} color={item.success ? "success" : "error"} style={{ borderRadius: 6 }}>
                        {item.success ? "SUCCESS" : "FAILED"}
                      </Tag>
                    </Space>
                    <Space size="middle">
                      <div style={{ textAlign: "right" }}>
                        <Typography.Text style={{ fontSize: 13 }}>{formatNumber(item.total_tokens)} tokens</Typography.Text>
                        <div className="log-subtitle">{item.latency_ms}ms</div>
                      </div>
                      <div style={{ textAlign: "right", minWidth: 80 }}>
                        <Typography.Text strong style={{ fontSize: 13 }}>{formatCurrency(item.billable_amount)}</Typography.Text>
                        <div className="log-subtitle">BILLABLE</div>
                      </div>
                    </Space>
                  </div>
                </List.Item>
              )}
            />
          </section>
        </Col>
      </Row>
    </Space>
  );
}
