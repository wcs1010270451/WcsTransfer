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
          error: error.message || "加载总览数据失败",
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
        eyebrow="网关总览"
        title="集中查看网关健康度、流量质量和资源状态"
        description="这里汇总最近 24 小时的请求活动、资源数量和最新流量情况，方便快速判断网关是否稳定。"
      />

      {state.error ? <Alert type="error" showIcon message={state.error} /> : null}

      <Row gutter={[18, 18]}>
        <Col xs={24} md={12} xl={6}>
          <MetricCard title="提供方" value={state.stats?.provider_count ?? 0} hint={`已配置 ${state.stats?.provider_count ?? 0} 个，活跃密钥 ${state.stats?.active_key_count ?? 0} 个`} />
        </Col>
        <Col xs={24} md={12} xl={6}>
          <MetricCard title="请求数（24 小时）" value={state.stats?.request_count ?? 0} hint={`成功 ${state.stats?.success_count ?? 0}，失败 ${state.stats?.failed_count ?? 0}`} />
        </Col>
        <Col xs={24} md={12} xl={6}>
          <MetricCard title="模型" value={state.stats?.enabled_model_count ?? 0} hint={`总数 ${state.stats?.model_count ?? 0}，启用 ${state.stats?.enabled_model_count ?? 0}`} />
        </Col>
        <Col xs={24} md={12} xl={6}>
          <MetricCard title="客户端密钥" value={state.stats?.active_client_key_count ?? 0} hint={`总数 ${state.stats?.client_key_count ?? 0}，活跃 ${state.stats?.active_client_key_count ?? 0}`} />
        </Col>
      </Row>

      <Row gutter={[18, 18]}>
        <Col xs={24} xl={8}>
          <section className="panel-card">
            <Typography.Title level={4}>服务健康</Typography.Title>
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
                        <Tag color={value.status === "up" ? "green" : value.status === "disabled" ? "default" : "red"}>{value.status}</Tag>
                      </Space>
                    </List.Item>
                  )}
                />
              </Space>
            ) : (
              <Typography.Text type="secondary">正在获取健康检查结果...</Typography.Text>
            )}
          </section>
        </Col>

        <Col xs={24} xl={8}>
          <section className="panel-card">
            <Typography.Title level={4}>流量快照</Typography.Title>
            <Space direction="vertical" size={16} style={{ width: "100%" }}>
              <div>
                <div className="section-label">最近 {state.stats?.window_hours ?? 24} 小时成功率</div>
                <Progress percent={Number((state.stats?.success_rate ?? 0).toFixed(1))} strokeColor="#0f766e" />
              </div>
              <div className="log-list-item">
                <div>
                  <Typography.Text strong>平均延迟</Typography.Text>
                  <Typography.Paragraph type="secondary" className="log-subtitle">
                    已记录请求的平均耗时
                  </Typography.Paragraph>
                </div>
                <Typography.Text>{Number(state.stats?.average_latency_ms ?? 0).toFixed(1)} ms</Typography.Text>
              </div>
              <div className="log-list-item">
                <div>
                  <Typography.Text strong>预估成本</Typography.Text>
                  <Typography.Paragraph type="secondary" className="log-subtitle">
                    汇总自 request_logs.estimated_cost
                  </Typography.Paragraph>
                </div>
                <Typography.Text>${Number(state.stats?.billable_amount ?? 0).toFixed(4)}</Typography.Text>
              </div>
            </Space>
          </section>
        </Col>

        <Col xs={24} xl={8}>
          <section className="panel-card">
            <Typography.Title level={4}>最近请求</Typography.Title>
            <List
              dataSource={state.logs}
              locale={{ emptyText: "暂无请求日志" }}
              renderItem={(item) => (
                <List.Item>
                  <div className="log-list-item">
                    <div>
                      <Typography.Text strong>{item.model_public_name || "unknown-model"}</Typography.Text>
                      <Typography.Paragraph type="secondary" className="log-subtitle">
                        {item.request_type} | {item.request_path || "-"}
                      </Typography.Paragraph>
                    </div>
                    <Space direction="vertical" size={4} align="end">
                      <Tag color={item.success ? "green" : "red"}>{item.success ? "成功" : "失败"}</Tag>
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
