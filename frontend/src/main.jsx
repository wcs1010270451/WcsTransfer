import React from "react";
import ReactDOM from "react-dom/client";
import { App as AntdApp, ConfigProvider } from "antd";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
import "./styles.css";

const appBasePath = (import.meta.env.VITE_APP_BASE_PATH || "/console/").trim().replace(/\/+$/, "") || "/console";

const theme = {
  token: {
    colorPrimary: "#0f766e",
    colorInfo: "#0f766e",
    colorSuccess: "#15803d",
    colorWarning: "#b45309",
    colorError: "#b91c1c",
    borderRadius: 18,
    fontFamily: `"Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif`,
    colorBgBase: "#f5f7f1",
  },
};

ReactDOM.createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <ConfigProvider theme={theme}>
      <AntdApp>
        <BrowserRouter basename={appBasePath}>
          <App />
        </BrowserRouter>
      </AntdApp>
    </ConfigProvider>
  </React.StrictMode>
);
