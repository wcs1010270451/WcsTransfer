import { Card, Statistic, Typography, Space } from "antd";
import { ArrowDownOutlined, ArrowUpOutlined } from "@ant-design/icons";

export default function MetricCard({ title, value, suffix, hint, trend, trendValue }) {
  const renderTrend = () => {
    if (!trend) return null;
    const isUp = trend === "up";
    const color = isUp ? "#10b981" : "#ef4444";
    const Icon = isUp ? ArrowUpOutlined : ArrowDownOutlined;

    return (
      <Space size={4} style={{ fontSize: 12, color }}>
        <Icon />
        <Typography.Text style={{ color, fontSize: 12, fontWeight: 600 }}>
          {trendValue}
        </Typography.Text>
      </Space>
    );
  };

  return (
    <Card className="metric-card" bordered={false}>
      <Statistic
        title={
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <span>{title}</span>
            {renderTrend()}
          </div>
        }
        value={value}
        suffix={suffix}
        valueStyle={{
          fontFamily: "ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, sans-serif",
          fontVariantNumeric: "tabular-nums",
          fontWeight: 700,
          letterSpacing: "-0.01em",
        }}
      />
      <Typography.Text className="metric-hint" style={{ marginTop: 12 }}>
        {hint}
      </Typography.Text>
    </Card>
  );
}
