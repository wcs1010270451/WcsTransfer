import { Button, Drawer, Form, Input, Space, Typography } from "antd";
import useSettingsStore from "../store/settingsStore";

export default function SettingsDrawer({ open, onClose }) {
  const apiBaseUrl = useSettingsStore((state) => state.apiBaseUrl);
  const adminToken = useSettingsStore((state) => state.adminToken);
  const setApiBaseUrl = useSettingsStore((state) => state.setApiBaseUrl);
  const setAdminToken = useSettingsStore((state) => state.setAdminToken);
  const [form] = Form.useForm();

  const handleFinish = (values) => {
    setApiBaseUrl(values.apiBaseUrl);
    setAdminToken(values.adminToken);
    onClose();
  };

  return (
    <Drawer
      title="连接设置"
      placement="right"
      width={420}
      open={open}
      onClose={onClose}
      destroyOnClose
    >
      <Typography.Paragraph type="secondary">
        修改后端地址和后台令牌后，会保存到浏览器本地存储，刷新页面也会保留。
      </Typography.Paragraph>
      <Form
        layout="vertical"
        form={form}
        initialValues={{ apiBaseUrl, adminToken }}
        onFinish={handleFinish}
      >
        <Form.Item
          label="API Base URL"
          name="apiBaseUrl"
          rules={[{ required: true, message: "请输入后端地址" }]}
        >
          <Input placeholder="http://localhost:8080" />
        </Form.Item>
        <Form.Item
          label="Admin Token"
          name="adminToken"
          rules={[{ required: true, message: "请输入后台令牌" }]}
        >
          <Input.Password placeholder="change-me" />
        </Form.Item>
        <Space>
          <Button onClick={onClose}>取消</Button>
          <Button type="primary" htmlType="submit">
            保存
          </Button>
        </Space>
      </Form>
    </Drawer>
  );
}
