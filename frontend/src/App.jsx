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

export default function App() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/providers" element={<ProvidersPage />} />
        <Route path="/client-keys" element={<ClientKeysPage />} />
        <Route path="/keys" element={<KeysPage />} />
        <Route path="/models" element={<ModelsPage />} />
        <Route path="/docs" element={<ApiDocsPage />} />
        <Route path="/debug" element={<DebugPage />} />
        <Route path="/logs" element={<LogsPage />} />
      </Route>
    </Routes>
  );
}
