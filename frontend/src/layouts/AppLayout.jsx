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
  LogoutOutlined,
  SettingOutlined,
  TeamOutlined,
  UserOutlined,
} from "@ant-design/icons";
import { Avatar, Button, Dropdown, Layout, Menu, Space, Tag, Typography } from "antd";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import SettingsDrawer from "../components/SettingsDrawer";
import useSettingsStore from "../store/settingsStore";
import useAdminAuthStore from "../store/adminAuthStore";

const { Header, Content, Sider } = Layout;

const menuItems = [
  { key: "/dashboard", icon: <AppstoreOutlined />, label: "总览" },
  { key: "/providers", icon: <DatabaseOutlined />, label: "提供方" },
  { key: "/users", icon: <TeamOutlined />, label: "用户" },
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

  const currentMenuItem = menuItems.find((item) => item.key === location.pathname) || menuItems[0];

  const userMenuItems = [
    {
      key: "settings",
      icon: <SettingOutlined />,
      label: "连接设置",
      onClick: () => setSettingsOpen(true),
    },
    { type: "divider" },
    {
      key: "logout",
      icon: <LogoutOutlined />,
      label: "退出登录",
      danger: true,
      onClick: handleLogout,
    },
  ];

  return (
    <Layout className="app-shell">
      <Sider breakpoint="lg" collapsedWidth="0" width={280} className="app-sider">
        <div className="brand-block">
          <div className="brand-mark">WT</div>
          <div>
            <Typography.Text className="brand-label">WcsTransfer</Typography.Text>
            <Typography.Title level={4} className="brand-title">
              模型网关
            </Typography.Title>
          </div>
        </div>

        <div className="sider-panel">
          <Typography.Text className="sider-panel-label">API BASE URL</Typography.Text>
          <Typography.Paragraph ellipsis={{ rows: 1 }} className="sider-panel-value">
            {apiBaseUrl}
          </Typography.Paragraph>
          <Tag color="cyan" bordered={false} style={{ borderRadius: 6, fontSize: 11 }}>
            ADMIN CONNECTED
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
          <div className="fade-in">
            <Typography.Text className="header-kicker">管理控制台</Typography.Text>
            <Typography.Title level={3} className="header-title">
              {currentMenuItem.label}
            </Typography.Title>
          </div>

          <Space size="large">
            <Button
              type="text"
              icon={<SettingOutlined />}
              onClick={() => setSettingsOpen(true)}
              style={{ color: "var(--text-muted)" }}
            >
              连接设置
            </Button>

            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight" arrow>
              <Space className="user-profile-trigger" style={{ cursor: "pointer" }}>
                <div style={{ textAlign: "right", lineHeight: 1 }}>
                  <Typography.Text strong style={{ display: "block", fontSize: 13 }}>
                    {adminUser?.display_name || adminUser?.username || "Admin"}
                  </Typography.Text>
                  <Typography.Text type="secondary" style={{ fontSize: 11 }}>
                    平台管理员
                  </Typography.Text>
                </div>
                <Avatar
                  icon={<UserOutlined />}
                  style={{ backgroundColor: "var(--accent-primary)", boxShadow: "0 4px 8px rgba(13, 148, 136, 0.2)" }}
                />
              </Space>
            </Dropdown>
          </Space>
        </Header>

        <Content className="app-content">
          <div className="fade-in">
            <Outlet />
          </div>
        </Content>
      </Layout>

      <SettingsDrawer open={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </Layout>
  );
}
