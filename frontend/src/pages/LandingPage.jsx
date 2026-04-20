import { Button, Card, Col, Row, Space, Typography } from "antd";
import { Link } from "react-router-dom";

export default function LandingPage() {
  return (
    <div className="landing-shell">
      <div className="landing-inner">
        <Space direction="vertical" size={20} style={{ width: "100%" }}>
          <div>
            <Typography.Text className="hero-eyebrow">WcsTransfer</Typography.Text>
            <Typography.Title level={1} style={{ marginTop: 12, marginBottom: 12 }}>
              模型网关入口
            </Typography.Title>
            <Typography.Paragraph type="secondary" style={{ maxWidth: 720, marginBottom: 0 }}>
              这里先作为统一入口页使用。管理员进入控制台，租户进入用户工作台。
            </Typography.Paragraph>
          </div>

          <Row gutter={[16, 16]}>
            <Col xs={24} md={12}>
              <Card className="panel-card landing-card">
                <Space direction="vertical" size={16}>
                  <Typography.Title level={3} style={{ margin: 0 }}>
                    管理员入口
                  </Typography.Title>
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    进入管理控制台，维护提供方、密钥、模型、租户、日志和统计。
                  </Typography.Paragraph>
                  <Button type="primary">
                    <Link to="/dashboard">进入管理员控制台</Link>
                  </Button>
                </Space>
              </Card>
            </Col>
            <Col xs={24} md={12}>
              <Card className="panel-card landing-card">
                <Space direction="vertical" size={16}>
                  <Typography.Title level={3} style={{ margin: 0 }}>
                    租户入口
                  </Typography.Title>
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    进入租户登录页，注册工作区并管理客户端密钥、调用量和日志。
                  </Typography.Paragraph>
                  <Button type="primary">
                    <Link to="/portal/login">进入租户登录</Link>
                  </Button>
                </Space>
              </Card>
            </Col>
          </Row>
        </Space>
      </div>
    </div>
  );
}
