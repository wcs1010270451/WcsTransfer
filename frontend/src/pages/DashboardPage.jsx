import { useEffect, useState } from "react";
import { Alert, Col, List, Progress, Row, Space, Tag, Typography } from "antd";
import MetricCard from "../components/MetricCard";
import PageHeaderCard from "../components/PageHeaderCard";
import { fetchHealth, fetchLogs, fetchStats } from "../api/client";

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

        if (!active) {
          return;
        }

        setState({
          loading: false,
          error: "",
          health,
          stats,
          logs: logs.items || [],
        });
      } catch (error) {
        if (!active) {
          return;
        }

        setState((previous) => ({
          ...previous,
          loading: false,
          error: error.message || "Failed to load dashboard data",
        }));
      }
    };

    load();
    return () => {
      active = false;
    };
  }, []);

  return (
    <Space direction="vertical" size={24} style={{ width: "100%" }}>
      <PageHeaderCard
        eyebrow="Gateway Overview"
        title="See gateway health, traffic quality, and capacity in one place"
        description="This page now aggregates the last 24 hours of request activity, resource counts, and recent traffic so we can judge whether the gateway is healthy and stable at a glance."
      />

      {state.error ? <Alert type="error" showIcon message={state.error} /> : null}

      <Row gutter={[18, 18]}>
        <Col xs={24} md={12} xl={6}>
          <MetricCard
            title="Providers"
            value={state.stats?.provider_count ?? 0}
            hint={`Configured ${state.stats?.provider_count ?? 0}, active keys ${state.stats?.active_key_count ?? 0}`}
          />
        </Col>
        <Col xs={24} md={12} xl={6}>
          <MetricCard
            title="Requests (24h)"
            value={state.stats?.request_count ?? 0}
            hint={`Success ${state.stats?.success_count ?? 0}, failed ${state.stats?.failed_count ?? 0}`}
          />
        </Col>
        <Col xs={24} md={12} xl={6}>
          <MetricCard
            title="Models"
            value={state.stats?.enabled_model_count ?? 0}
            hint={`Total ${state.stats?.model_count ?? 0}, enabled ${state.stats?.enabled_model_count ?? 0}`}
          />
        </Col>
        <Col xs={24} md={12} xl={6}>
          <MetricCard
            title="Tokens (24h)"
            value={state.stats?.total_tokens ?? 0}
            hint={
              state.loading
                ? "Loading aggregated usage"
                : `Prompt ${state.stats?.prompt_tokens ?? 0} / Completion ${state.stats?.completion_tokens ?? 0}`
            }
          />
        </Col>
      </Row>

      <Row gutter={[18, 18]}>
        <Col xs={24} xl={8}>
          <section className="panel-card">
            <Typography.Title level={4}>Service Health</Typography.Title>
            {state.health ? (
              <Space direction="vertical" size={12} style={{ width: "100%" }}>
                <Space wrap>
                  <Tag color={state.health.status === "ok" ? "green" : "gold"}>{state.health.status}</Tag>
                  <Tag>{state.health.environment}</Tag>
                </Space>
                <List
                  dataSource={Object.entries(state.health.dependencies || {})}
                  renderItem={([name, value]) => (
                    <List.Item>
                      <Space>
                        <Typography.Text strong>{name}</Typography.Text>
                        <Tag color={value.status === "up" ? "green" : value.status === "disabled" ? "default" : "red"}>
                          {value.status}
                        </Tag>
                      </Space>
                    </List.Item>
                  )}
                />
              </Space>
            ) : (
              <Typography.Text type="secondary">Waiting for health check...</Typography.Text>
            )}
          </section>
        </Col>

        <Col xs={24} xl={8}>
          <section className="panel-card">
            <Typography.Title level={4}>Traffic Snapshot</Typography.Title>
            <Space direction="vertical" size={16} style={{ width: "100%" }}>
              <div>
                <div className="section-label">Success rate in the last {state.stats?.window_hours ?? 24} hours</div>
                <Progress percent={Number((state.stats?.success_rate ?? 0).toFixed(1))} strokeColor="#0f766e" />
              </div>
              <div className="log-list-item">
                <div>
                  <Typography.Text strong>Average latency</Typography.Text>
                  <Typography.Paragraph type="secondary" className="log-subtitle">
                    Mean latency across logged requests
                  </Typography.Paragraph>
                </div>
                <Typography.Text>{Number(state.stats?.average_latency_ms ?? 0).toFixed(1)} ms</Typography.Text>
              </div>
              <div className="log-list-item">
                <div>
                  <Typography.Text strong>Estimated cost</Typography.Text>
                  <Typography.Paragraph type="secondary" className="log-subtitle">
                    Aggregated from request_logs.estimated_cost
                  </Typography.Paragraph>
                </div>
                <Typography.Text>${Number(state.stats?.estimated_cost ?? 0).toFixed(4)}</Typography.Text>
              </div>
            </Space>
          </section>
        </Col>

        <Col xs={24} xl={8}>
          <section className="panel-card">
            <Typography.Title level={4}>Top Models</Typography.Title>
            <List
              dataSource={state.stats?.top_models || []}
              locale={{ emptyText: "No model traffic in the last 24 hours" }}
              renderItem={(item) => (
                <List.Item>
                  <div className="log-list-item">
                    <div>
                      <Typography.Text strong>{item.model_public_name}</Typography.Text>
                      <Typography.Paragraph type="secondary" className="log-subtitle">
                        {item.request_count} requests, {item.total_tokens} tokens
                      </Typography.Paragraph>
                    </div>
                    <Space direction="vertical" size={4} align="end">
                      <Tag color={item.success_rate >= 95 ? "green" : item.success_rate >= 80 ? "gold" : "red"}>
                        {Number(item.success_rate).toFixed(1)}%
                      </Tag>
                      <Typography.Text type="secondary">{Number(item.average_latency_ms).toFixed(1)} ms</Typography.Text>
                    </Space>
                  </div>
                </List.Item>
              )}
            />
          </section>
        </Col>
      </Row>

      <Row gutter={[18, 18]}>
        <Col xs={24} xl={10}>
          <section className="panel-card">
            <Typography.Title level={4}>Top Providers</Typography.Title>
            <List
              dataSource={state.stats?.top_providers || []}
              locale={{ emptyText: "No provider traffic in the last 24 hours" }}
              renderItem={(item) => (
                <List.Item>
                  <div className="log-list-item">
                    <div>
                      <Typography.Text strong>{item.provider_name}</Typography.Text>
                      <Typography.Paragraph type="secondary" className="log-subtitle">
                        {item.request_count} requests, {item.total_tokens} tokens
                      </Typography.Paragraph>
                    </div>
                    <Space direction="vertical" size={4} align="end">
                      <Tag color={item.success_rate >= 95 ? "green" : item.success_rate >= 80 ? "gold" : "red"}>
                        {Number(item.success_rate).toFixed(1)}%
                      </Tag>
                      <Typography.Text type="secondary">{Number(item.average_latency_ms).toFixed(1)} ms</Typography.Text>
                    </Space>
                  </div>
                </List.Item>
              )}
            />
          </section>
        </Col>

        <Col xs={24} xl={14}>
          <section className="panel-card">
            <Typography.Title level={4}>Recent Requests</Typography.Title>
            <List
              dataSource={state.logs}
              locale={{ emptyText: "No request logs yet" }}
              renderItem={(item) => (
                <List.Item>
                  <div className="log-list-item">
                    <div>
                      <Typography.Text strong>{item.model_public_name || "unknown-model"}</Typography.Text>
                      <Typography.Paragraph type="secondary" className="log-subtitle">
                        {item.request_type} | {item.request_path || "n/a"}
                      </Typography.Paragraph>
                    </div>
                    <Space direction="vertical" size={4} align="end">
                      <Tag color={item.success ? "green" : "red"}>{item.success ? "success" : "failed"}</Tag>
                      <Typography.Text type="secondary">
                        {item.latency_ms} ms | {item.total_tokens || 0} tokens
                      </Typography.Text>
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
