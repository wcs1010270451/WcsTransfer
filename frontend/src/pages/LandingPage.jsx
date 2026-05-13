import { Button, Card, Space, Typography } from "antd";
import { Link } from "react-router-dom";

export default function LandingPage() {
  return (
    <div className="landing-shell">
      <div className="landing-inner fade-in">
        <Space direction="vertical" size={32} style={{ width: "100%" }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
            <Card className="landing-card" hoverable>
              <Space direction="vertical" size={16}>
                <div style={{ fontSize: 40 }}>🛠️</div>
                <Typography.Title level={3} style={{ margin: 0 }}>
                  管理员入口
                </Typography.Title>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
                  维护系统配置、监控流量与管理用户
                </Typography.Paragraph>
                <Link to="/dashboard" style={{ width: "100%" }}>
                  <Button type="primary" size="large" block>
                    进入管理控制台
                  </Button>
                </Link>
              </Space>
            </Card>

            <Card className="landing-card" hoverable>
              <Space direction="vertical" size={16}>
                <div style={{ fontSize: 40 }}>👤</div>
                <Typography.Title level={3} style={{ margin: 0 }}>
                  用户入口
                </Typography.Title>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
                  管理 API 密钥、查看账单与调用统计
                </Typography.Paragraph>
                <Link to="/portal/login" style={{ width: "100%" }}>
                  <Button type="primary" size="large" block>
                    进入用户工作台
                  </Button>
                </Link>
              </Space>
            </Card>
          </div>

          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            WcsTransfer Model Gateway © 2026
          </Typography.Text>
        </Space>
      </div>
    </div>
  );
}
