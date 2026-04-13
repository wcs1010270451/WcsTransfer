import { Card, Statistic, Typography } from "antd";

export default function MetricCard({ title, value, suffix, hint }) {
  return (
    <Card className="metric-card" bordered={false}>
      <Statistic title={title} value={value} suffix={suffix} />
      <Typography.Text className="metric-hint">{hint}</Typography.Text>
    </Card>
  );
}
