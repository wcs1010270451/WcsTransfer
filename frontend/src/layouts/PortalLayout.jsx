import { useEffect } from "react";
import { AppstoreOutlined, KeyOutlined, LogoutOutlined, UserOutlined } from "@ant-design/icons";
import { Avatar, Button, Layout, Menu, Space, Typography } from "antd";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { fetchPortalMe } from "../api/client";
import usePortalAuthStore from "../store/portalAuthStore";

const { Header, Content, Sider } = Layout;

const menuItems = [
  { key: "/portal/dashboard", icon: <AppstoreOutlined />, label: "仪表盘" },
  { key: "/portal/keys", icon: <KeyOutlined />, label: "Keys 管理" },
  { key: "/portal/profile", icon: <UserOutlined />, label: "个人中心" },
];

export default function PortalLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const currentUser = usePortalAuthStore((state) => state.user);
  const token = usePortalAuthStore((state) => state.token);
  const clearSession = usePortalAuthStore((state) => state.clearSession);
  const setSession = usePortalAuthStore((state) => state.setSession);

  useEffect(() => {
    fetchPortalMe()
      .then((res) => setSession({ token, user: res.user }))
      .catch(() => {});
  }, []);

  const handleLogout = () => {
    clearSession();
    navigate("/portal/login", { replace: true });
  };

  const avatarLetter = currentUser?.email?.[0]?.toUpperCase() || "U";

  return (
    <Layout className="portal-shell">
      <Header className="portal-header">
        <div className="portal-logo">
          <div className="portal-logo-mark">WT</div>
          <Typography.Text className="portal-logo-text">WcsTransfer</Typography.Text>
        </div>
        <Space align="center" size={12}>
          {currentUser?.email && (
            <>
              <Avatar className="portal-avatar">{avatarLetter}</Avatar>
              <Typography.Text className="portal-user-email">{currentUser.email}</Typography.Text>
            </>
          )}
          <Button
            type="text"
            icon={<LogoutOutlined />}
            onClick={handleLogout}
            className="portal-logout-btn"
          >
            退出登录
          </Button>
        </Space>
      </Header>

      <Layout className="portal-body">
        <Sider width={200} className="portal-sider" breakpoint="lg" collapsedWidth="0">
          <Menu
            mode="inline"
            selectedKeys={[location.pathname]}
            items={menuItems}
            onClick={({ key }) => navigate(key)}
            className="portal-menu"
          />
        </Sider>
        <Content className="portal-content">
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
