import { App, Button, Card, Form, Input, Space, Typography } from "antd";
import { useNavigate } from "react-router-dom";
import { loginPortalUser } from "../api/client";
import usePortalAuthStore from "../store/portalAuthStore";

export default function PortalAuthPage() {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const setSession = usePortalAuthStore((state) => state.setSession);

  const handleLogin = async (values) => {
    try {
      const result = await loginPortalUser(values);
      setSession({ token: result.token, user: result.user });
      message.success("登录成功");
      navigate("/portal/keys", { replace: true });
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "登录失败");
    }
  };

  return (
    <div style={{ minHeight: "100vh", display: "grid", placeItems: "center", padding: 24 }}>
      <Card style={{ width: "100%", maxWidth: 720, borderRadius: 28 }}>
        <Space direction="vertical" size={20} style={{ width: "100%" }}>
          <div>
            <Typography.Text className="hero-eyebrow">用户入口</Typography.Text>
            <Typography.Title level={2} style={{ marginTop: 10, marginBottom: 6 }}>
              用户登录
            </Typography.Title>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              请使用管理员分配的账号登录。账号由管理员后台统一创建与管理。
            </Typography.Paragraph>
          </div>

          <Form layout="vertical" onFinish={handleLogin}>
            <Form.Item label="邮箱" name="email" rules={[{ required: true, type: "email" }]}>
              <Input />
            </Form.Item>
            <Form.Item label="密码" name="password" rules={[{ required: true }]}>
              <Input.Password />
            </Form.Item>
            <Button type="primary" htmlType="submit">
              登录
            </Button>
          </Form>
        </Space>
      </Card>
    </div>
  );
}
