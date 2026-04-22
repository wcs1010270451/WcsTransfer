import { App, Button, Card, Form, Input, Space, Typography } from "antd";
import { useNavigate } from "react-router-dom";
import { loginAdminUser } from "../api/client";
import useAdminAuthStore from "../store/adminAuthStore";

export default function AdminAuthPage() {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const setSession = useAdminAuthStore((state) => state.setSession);

  const handleFinish = async (values) => {
    try {
      const result = await loginAdminUser(values);
      setSession({ token: result.token, user: result.user });
      message.success("管理员登录成功");
      navigate("/dashboard", { replace: true });
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "管理员登录失败");
    }
  };

  return (
    <div className="landing-shell">
      <div className="landing-inner" style={{ maxWidth: 520 }}>
        <Card className="panel-card landing-card">
          <Space direction="vertical" size={18} style={{ width: "100%" }}>
            <div>
              <Typography.Text className="hero-eyebrow">Admin Portal</Typography.Text>
              <Typography.Title level={2} style={{ marginTop: 12, marginBottom: 8 }}>
                管理员登录
              </Typography.Title>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                使用管理员账号登录控制台。登录成功后，会话仅保存在当前浏览器会话中。
              </Typography.Paragraph>
            </div>

            <Form layout="vertical" onFinish={handleFinish}>
              <Form.Item label="用户名" name="username" rules={[{ required: true, message: "请输入用户名" }]}>
                <Input autoComplete="username" />
              </Form.Item>
              <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
                <Input.Password autoComplete="current-password" />
              </Form.Item>
              <Space>
                <Button onClick={() => navigate("/", { replace: true })}>返回首页</Button>
                <Button type="primary" htmlType="submit">
                  登录
                </Button>
              </Space>
            </Form>
          </Space>
        </Card>
      </div>
    </div>
  );
}
