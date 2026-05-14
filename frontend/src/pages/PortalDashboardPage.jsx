import { useEffect, useState } from "react";
import { App, Card, Col, DatePicker, Row, Segmented, Space, Statistic, Tag, Typography } from "antd";
import dayjs from "dayjs";
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { fetchPortalDailyStats, fetchPortalStats } from "../api/client";

function fmt(value) {
  return <span style={{ fontVariantNumeric: "tabular-nums" }}>{new Intl.NumberFormat("zh-CN").format(Number(value || 0))}</span>;
}

function fmtCurrency(value) {
  const formatted = new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 4,
    maximumFractionDigits: 4,
  }).format(Number(value || 0));
  return <span style={{ fontVariantNumeric: "tabular-nums" }}>{formatted}</span>;
}

function StatCard({ title, value, suffix, formatter }) {
  return (
    <Card size="small" className="portal-stat-card">
      <Statistic title={title} value={value} suffix={suffix} formatter={formatter} />
    </Card>
  );
}

const RANGE_OPTIONS = [
  { label: "近 7 天", value: "7d" },
  { label: "近 1 个月", value: "30d" },
  { label: "近 3 个月", value: "90d" },
  { label: "自定义", value: "custom" },
];

const LINES = [
  { key: "request_count",     name: "请求数",      color: "#3b82f6", yAxis: "left" },
  { key: "prompt_tokens",     name: "输入 Token",  color: "#8b5cf6", yAxis: "right" },
  { key: "completion_tokens", name: "输出 Token",  color: "#06b6d4", yAxis: "right" },
  { key: "total_tokens",      name: "总 Token",    color: "#f59e0b", yAxis: "right" },
  { key: "billable_amount",   name: "消费金额($)", color: "#10b981", yAxis: "left" },
];

function getPresetRange(mode) {
  const today = dayjs().startOf("day");
  if (mode === "7d")  return [today.subtract(6,  "day"), today];
  if (mode === "30d") return [today.subtract(29, "day"), today];
  if (mode === "90d") return [today.subtract(89, "day"), today];
  return null;
}

export default function PortalDashboardPage() {
  const { message } = App.useApp();
  const [stats, setStats]           = useState(null);
  const [dailyData, setDailyData]   = useState([]);
  const [chartLoading, setChartLoading] = useState(true);
  const [rangeMode, setRangeMode]   = useState("30d");
  const [customRange, setCustomRange] = useState(null);
  const [calendarDates, setCalendarDates] = useState(null);

  useEffect(() => {
    fetchPortalStats()
      .then((res) => setStats(res || null))
      .catch((error) => message.error(error.response?.data?.error?.message || error.message || "加载统计失败"));
  }, []);

  useEffect(() => {
    let params = {};
    if (rangeMode === "custom") {
      if (!customRange) return;
      params = {
        from: customRange[0].format("YYYY-MM-DD"),
        to:   customRange[1].format("YYYY-MM-DD"),
      };
    } else {
      const [from, to] = getPresetRange(rangeMode);
      params = { from: from.format("YYYY-MM-DD"), to: to.format("YYYY-MM-DD") };
    }

    setChartLoading(true);
    fetchPortalDailyStats(params)
      .then((res) => setDailyData(res.items || []))
      .catch((error) => message.error(error.response?.data?.error?.message || error.message || "加载趋势数据失败"))
      .finally(() => setChartLoading(false));
  }, [rangeMode, customRange]);

  const s = stats || {};

  const chartData = dailyData.map((p) => ({ ...p, date: p.date.slice(5) }));

  const disabledDate = (current) => {
    if (current > dayjs().endOf("day")) return true;
    const anchor = calendarDates?.[0] || calendarDates?.[1];
    if (!anchor) return false;
    return current < anchor.subtract(3, "month") || current > anchor.add(3, "month");
  };

  const handleRangeChange = (val) => {
    setRangeMode(val);
    if (val !== "custom") setCustomRange(null);
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%", padding: 24 }}>
      <Typography.Title level={4} style={{ margin: 0 }}>仪表盘</Typography.Title>

      <section className="panel-card">
        <Typography.Title level={5} style={{ marginTop: 0, marginBottom: 12 }}>账户状态</Typography.Title>
        <Space wrap>
          <Tag bordered={false} className="tag-status" color={(s.wallet_balance || 0) > 0 ? "success" : "error"} style={{ fontSize: 14, padding: "4px 12px" }}>
            钱包余额：{fmtCurrency(s.wallet_balance || 0)}
          </Tag>
          <Tag bordered={false} className="tag-status" color="processing" style={{ fontSize: 14, padding: "4px 12px" }}>
            总密钥数：{fmt(s.client_key_count || 0)}
          </Tag>
          <Tag bordered={false} className="tag-status" color="processing" style={{ fontSize: 14, padding: "4px 12px" }}>
            活跃密钥：{fmt(s.active_client_keys || 0)}
          </Tag>
        </Space>
      </section>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={8}>
          <StatCard title="请求总数" value={s.request_count || 0} formatter={(v) => <span style={{ fontVariantNumeric: "tabular-nums" }}>{new Intl.NumberFormat("zh-CN").format(v)}</span>} />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <StatCard title="总 Token" value={s.total_tokens || 0} formatter={(v) => <span style={{ fontVariantNumeric: "tabular-nums" }}>{new Intl.NumberFormat("zh-CN").format(v)}</span>} />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <StatCard title="成功率" value={Number(s.success_rate || 0).toFixed(2)} suffix={<span style={{ fontVariantNumeric: "tabular-nums" }}>%</span>} />
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="输入 Token" value={s.prompt_tokens || 0} formatter={(v) => <span style={{ fontVariantNumeric: "tabular-nums" }}>{new Intl.NumberFormat("zh-CN").format(v)}</span>} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="输出 Token" value={s.completion_tokens || 0} formatter={(v) => <span style={{ fontVariantNumeric: "tabular-nums" }}>{new Intl.NumberFormat("zh-CN").format(v)}</span>} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="平均延迟" value={Number(s.average_latency_ms || 0).toFixed(0)} suffix={<span style={{ fontVariantNumeric: "tabular-nums" }}>ms</span>} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard title="活跃密钥数" value={s.active_client_keys || 0} formatter={(v) => <span style={{ fontVariantNumeric: "tabular-nums" }}>{new Intl.NumberFormat("zh-CN").format(v)}</span>} />
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={8}>
          <StatCard title="今日消费" value={s.today_billable_amount || 0} formatter={fmtCurrency} />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <StatCard title="本月消费" value={s.month_billable_amount || 0} formatter={fmtCurrency} />
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <StatCard title="总消费" value={s.billable_amount || 0} formatter={fmtCurrency} />
        </Col>
      </Row>

      <section className="panel-card">
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", flexWrap: "wrap", gap: 12, marginBottom: 20 }}>
          <Typography.Title level={5} style={{ margin: 0 }}>调用趋势</Typography.Title>
          <Space wrap size={8}>
            <Segmented
              size="small"
              options={RANGE_OPTIONS}
              value={rangeMode}
              onChange={handleRangeChange}
            />
            {rangeMode === "custom" && (
              <DatePicker.RangePicker
                size="small"
                disabledDate={disabledDate}
                onCalendarChange={(dates) => setCalendarDates(dates)}
                onOpenChange={(open) => { if (!open) setCalendarDates(null); }}
                onChange={(dates) => setCustomRange(dates)}
                value={customRange}
              />
            )}
          </Space>
        </div>

        <div style={{ position: "relative" }}>
          <ResponsiveContainer width="100%" height={320}>
            <LineChart data={chartData} margin={{ top: 4, right: 16, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(59,130,246,0.1)" />
              <XAxis dataKey="date" tick={{ fontSize: 12 }} />
              <YAxis yAxisId="left" tick={{ fontSize: 12 }} width={60} />
              <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 12 }} width={70} />
              <Tooltip
                contentStyle={{ borderRadius: 10, fontSize: 13 }}
                formatter={(value, name) => {
                  if (name === "消费金额($)") return [`$${Number(value).toFixed(4)}`, name];
                  return [new Intl.NumberFormat("zh-CN").format(value), name];
                }}
              />
              <Legend wrapperStyle={{ fontSize: 13, paddingTop: 12 }} />
              {LINES.map((l) => (
                <Line
                  key={l.key}
                  yAxisId={l.yAxis}
                  type="monotone"
                  dataKey={l.key}
                  name={l.name}
                  stroke={l.color}
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
          {!chartLoading && chartData.length === 0 && (
            <div style={{ position: "absolute", inset: 0, display: "flex", alignItems: "center", justifyContent: "center" }}>
              <Typography.Text type="secondary">暂无数据</Typography.Text>
            </div>
          )}
        </div>
      </section>
    </Space>
  );
}
