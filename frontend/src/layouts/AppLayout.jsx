import { useState } from "react";
import {
  ApiOutlined,
  AppstoreOutlined,
  DatabaseOutlined,
  FileTextOutlined,
  KeyOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import { Button, Layout, Menu, Space, Tag, Typography } from "antd";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import SettingsDrawer from "../components/SettingsDrawer";
import useSettingsStore from "../store/settingsStore";

const { Header, Content, Sider } = Layout;

const menuItems = [
  { key: "/dashboard", icon: <AppstoreOutlined />, label: "总览" },
  { key: "/providers", icon: <DatabaseOutlined />, label: "Providers" },
  { key: "/keys", icon: <KeyOutlined />, label: "Keys" },
  { key: "/models", icon: <ApiOutlined />, label: "Models" },
  { key: "/logs", icon: <FileTextOutlined />, label: "Logs" },
];

export default function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const apiBaseUrl = useSettingsStore((state) => state.apiBaseUrl);
  const [settingsOpen, setSettingsOpen] = useState(false);

  return (
    <Layout className="app-shell">
      <Sider breakpoint="lg" collapsedWidth="0" width={280} className="app-sider">
        <div className="brand-block">
          <div className="brand-mark">WT</div>
          <div>
            <Typography.Text className="brand-label">WcsTransfer</Typography.Text>
            <Typography.Title level={4} className="brand-title">
              Model Gateway Console
            </Typography.Title>
          </div>
        </div>

        <div className="sider-panel">
          <Typography.Text className="sider-panel-label">当前连接</Typography.Text>
          <Typography.Paragraph className="sider-panel-value">{apiBaseUrl}</Typography.Paragraph>
          <Tag color="green" bordered={false}>
            Admin API Ready
          </Tag>
        </div>

        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          className="sider-menu"
        />
      </Sider>

      <Layout>
        <Header className="app-header">
          <Space size="middle">
            <div>
              <Typography.Text className="header-kicker">内部管理台</Typography.Text>
              <Typography.Title level={3} className="header-title">
                Gateway Control Room
              </Typography.Title>
            </div>
          </Space>
          <Button icon={<SettingOutlined />} onClick={() => setSettingsOpen(true)}>
            连接设置
          </Button>
        </Header>

        <Content className="app-content">
          <Outlet />
        </Content>
      </Layout>

      <SettingsDrawer open={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </Layout>
  );
}
