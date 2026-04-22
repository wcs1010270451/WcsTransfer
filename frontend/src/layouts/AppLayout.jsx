import { useState } from "react";
import {
  ApiOutlined,
  AppstoreOutlined,
  BookOutlined,
  ContactsOutlined,
  DatabaseOutlined,
  ExperimentOutlined,
  FileTextOutlined,
  KeyOutlined,
  SettingOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { Button, Layout, Menu, Space, Tag, Typography } from "antd";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import SettingsDrawer from "../components/SettingsDrawer";
import useSettingsStore from "../store/settingsStore";
import useAdminAuthStore from "../store/adminAuthStore";

const { Header, Content, Sider } = Layout;

const menuItems = [
  { key: "/dashboard", icon: <AppstoreOutlined />, label: "总览" },
  { key: "/providers", icon: <DatabaseOutlined />, label: "提供方" },
  { key: "/tenants", icon: <TeamOutlined />, label: "租户" },
  { key: "/client-keys", icon: <ContactsOutlined />, label: "客户端密钥" },
  { key: "/keys", icon: <KeyOutlined />, label: "上游密钥" },
  { key: "/models", icon: <ApiOutlined />, label: "模型" },
  { key: "/docs", icon: <BookOutlined />, label: "接口文档" },
  { key: "/debug", icon: <ExperimentOutlined />, label: "调试" },
  { key: "/logs", icon: <FileTextOutlined />, label: "日志" },
];

export default function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const apiBaseUrl = useSettingsStore((state) => state.apiBaseUrl);
  const adminUser = useAdminAuthStore((state) => state.user);
  const clearAdminSession = useAdminAuthStore((state) => state.clearSession);
  const [settingsOpen, setSettingsOpen] = useState(false);

  const handleLogout = () => {
    clearAdminSession();
    navigate("/admin/login", { replace: true });
  };

  return (
    <Layout className="app-shell">
      <Sider breakpoint="lg" collapsedWidth="0" width={280} className="app-sider">
        <div className="brand-block">
          <div className="brand-mark">WT</div>
          <div>
            <Typography.Text className="brand-label">WcsTransfer</Typography.Text>
            <Typography.Title level={4} className="brand-title">
              模型网关控制台
            </Typography.Title>
          </div>
        </div>

        <div className="sider-panel">
          <Typography.Text className="sider-panel-label">当前接口地址</Typography.Text>
          <Typography.Paragraph className="sider-panel-value">{apiBaseUrl}</Typography.Paragraph>
          <Tag color="green" bordered={false}>
            管理接口已连接
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
              <Typography.Text className="header-kicker">内部控制台</Typography.Text>
              <Typography.Title level={3} className="header-title">
                网关控制中心
              </Typography.Title>
            </div>
          </Space>
          <Space>
            {adminUser?.display_name || adminUser?.username ? (
              <Tag color="green">{adminUser.display_name || adminUser.username}</Tag>
            ) : null}
            <Button icon={<SettingOutlined />} onClick={() => setSettingsOpen(true)}>
              连接设置
            </Button>
            <Button onClick={handleLogout}>退出</Button>
          </Space>
        </Header>

        <Content className="app-content">
          <Outlet />
        </Content>
      </Layout>

      <SettingsDrawer open={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </Layout>
  );
}
