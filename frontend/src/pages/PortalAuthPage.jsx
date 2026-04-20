import { App, Button, Card, Form, Input, Space, Tabs, Typography } from "antd";
import { useNavigate } from "react-router-dom";
import { loginPortalUser, registerPortalUser } from "../api/client";
import usePortalAuthStore from "../store/portalAuthStore";

export default function PortalAuthPage() {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const setSession = usePortalAuthStore((state) => state.setSession);

  const handleLogin = async (values) => {
    try {
      const result = await loginPortalUser(values);
      setSession({ token: result.token, user: result.user, tenant: result.tenant || null });
      message.success("登录成功");
      navigate("/portal/keys", { replace: true });
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "登录失败");
    }
  };

  const handleRegister = async (values) => {
    try {
      const result = await registerPortalUser(values);
      setSession({ token: result.token, user: result.user, tenant: result.tenant || null });
      message.success("工作区已创建");
      navigate("/portal/keys", { replace: true });
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "注册失败");
    }
  };

  return (
    <div style={{ minHeight: "100vh", display: "grid", placeItems: "center", padding: 24 }}>
      <Card style={{ width: "100%", maxWidth: 720, borderRadius: 28 }}>
        <Space direction="vertical" size={20} style={{ width: "100%" }}>
          <div>
            <Typography.Text className="hero-eyebrow">用户入口</Typography.Text>
            <Typography.Title level={2} style={{ marginTop: 10, marginBottom: 6 }}>
              租户登录
            </Typography.Title>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              注册你的工作区，并自助管理客户端密钥、调用量和日志。
            </Typography.Paragraph>
          </div>

          <Tabs
            items={[
              {
                key: "login",
                label: "登录",
                children: (
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
                ),
              },
              {
                key: "register",
                label: "注册",
                children: (
                  <Form layout="vertical" onFinish={handleRegister}>
                    <Form.Item label="工作区名称" name="tenant_name" rules={[{ required: true }]}>
                      <Input />
                    </Form.Item>
                    <Form.Item
                      label="工作区 Slug"
                      name="tenant_slug"
                      extra="可选，仅允许小写字母、数字和连字符。"
                    >
                      <Input />
                    </Form.Item>
                    <Form.Item label="姓名" name="full_name" rules={[{ required: true }]}>
                      <Input />
                    </Form.Item>
                    <Form.Item label="邮箱" name="email" rules={[{ required: true, type: "email" }]}>
                      <Input />
                    </Form.Item>
                    <Form.Item label="密码" name="password" rules={[{ required: true, min: 8 }]}>
                      <Input.Password />
                    </Form.Item>
                    <Button type="primary" htmlType="submit">
                      创建工作区
                    </Button>
                  </Form>
                ),
              },
            ]}
          />
        </Space>
      </Card>
    </div>
  );
}
