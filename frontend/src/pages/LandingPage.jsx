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
              管理员进入控制台，用户进入工作台管理密钥和查看调用数据。
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
                    进入管理控制台，维护提供方、密钥、模型、用户、日志和统计。
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
                    用户入口
                  </Typography.Title>
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    进入用户登录页，管理客户端密钥、查看调用量和日志。
                  </Typography.Paragraph>
                  <Button type="primary">
                    <Link to="/portal/login">进入用户登录</Link>
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
