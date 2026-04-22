import { Button, Drawer, Form, Input, Space, Typography } from "antd";
import useSettingsStore from "../store/settingsStore";

export default function SettingsDrawer({ open, onClose }) {
  const apiBaseUrl = useSettingsStore((state) => state.apiBaseUrl);
  const setApiBaseUrl = useSettingsStore((state) => state.setApiBaseUrl);
  const [form] = Form.useForm();

  const handleFinish = (values) => {
    setApiBaseUrl(values.apiBaseUrl);
    onClose();
  };

  return (
    <Drawer title="连接设置" placement="right" width={420} open={open} onClose={onClose} destroyOnClose>
      <Typography.Paragraph type="secondary">
        管理端登录会话与接口地址仅保存在当前浏览器会话中，不会打包进前端静态资源。
      </Typography.Paragraph>
      <Form layout="vertical" form={form} initialValues={{ apiBaseUrl }} onFinish={handleFinish}>
        <Form.Item label="后端地址" name="apiBaseUrl" rules={[{ required: true, message: "请输入后端地址" }]}>
          <Input placeholder="http://localhost:3210" />
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
