import { Navigate, Route, Routes } from "react-router-dom";
import AppLayout from "./layouts/AppLayout";
import DashboardPage from "./pages/DashboardPage";
import ProvidersPage from "./pages/ProvidersPage";
import ClientKeysPage from "./pages/ClientKeysPage";
import KeysPage from "./pages/KeysPage";
import ModelsPage from "./pages/ModelsPage";
import LogsPage from "./pages/LogsPage";
import DebugPage from "./pages/DebugPage";
import ApiDocsPage from "./pages/ApiDocsPage";
import TenantsPage from "./pages/TenantsPage";
import PortalAuthPage from "./pages/PortalAuthPage";
import PortalKeysPage from "./pages/PortalKeysPage";
import AdminAuthPage from "./pages/AdminAuthPage";
import usePortalAuthStore from "./store/portalAuthStore";
import useAdminAuthStore from "./store/adminAuthStore";

function PortalGuard({ children }) {
  const token = usePortalAuthStore((state) => state.token);
  if (!token) {
    return <Navigate to="/portal/login" replace />;
  }
  return children;
}

function AdminGuard({ children }) {
  const token = useAdminAuthStore((state) => state.token);
  if (!token) {
    return <Navigate to="/admin/login" replace />;
  }
  return children;
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route path="/admin/login" element={<AdminAuthPage />} />
      <Route path="/portal/login" element={<PortalAuthPage />} />
      <Route
        path="/portal/keys"
        element={
          <PortalGuard>
            <PortalKeysPage />
          </PortalGuard>
        }
      />
      <Route
        element={
          <AdminGuard>
            <AppLayout />
          </AdminGuard>
        }
      >
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/providers" element={<ProvidersPage />} />
        <Route path="/tenants" element={<TenantsPage />} />
        <Route path="/client-keys" element={<ClientKeysPage />} />
        <Route path="/keys" element={<KeysPage />} />
        <Route path="/models" element={<ModelsPage />} />
        <Route path="/docs" element={<ApiDocsPage />} />
        <Route path="/debug" element={<DebugPage />} />
        <Route path="/logs" element={<LogsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  );
}
