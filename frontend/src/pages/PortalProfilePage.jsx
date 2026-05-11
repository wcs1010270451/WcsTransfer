import { useEffect, useState } from "react";
import { App, Descriptions, Space, Table, Tag, Typography } from "antd";
import { fetchPortalMe, fetchPortalWalletLedger } from "../api/client";
import usePortalAuthStore from "../store/portalAuthStore";

function fmtCurrency(value) {
  return new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 4,
    maximumFractionDigits: 4,
  }).format(Number(value || 0));
}

function fmt(value) {
  return new Intl.NumberFormat("zh-CN").format(Number(value || 0));
}

export default function PortalProfilePage() {
  const { message } = App.useApp();
  const token = usePortalAuthStore((state) => state.token);
  const setSession = usePortalAuthStore((state) => state.setSession);
  const [userInfo, setUserInfo] = useState(null);
  const [walletRows, setWalletRows] = useState([]);
  const [walletLoading, setWalletLoading] = useState(false);
  const [walletPagination, setWalletPagination] = useState({ current: 1, pageSize: 20, total: 0 });

  useEffect(() => {
    fetchPortalMe()
      .then((res) => {
        setUserInfo(res.user);
        setSession({ token, user: res.user });
      })
      .catch((error) => {
        message.error(error.response?.data?.error?.message || error.message || "加载用户信息失败");
      });

    loadWallet(1, 20);
  }, []);

  const loadWallet = async (page, pageSize) => {
    setWalletLoading(true);
    try {
      const response = await fetchPortalWalletLedger({ page, page_size: pageSize });
      setWalletRows(response.items || []);
      setWalletPagination({
        current: response.page || page,
        pageSize: response.page_size || pageSize,
        total: response.total || 0,
      });
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载钱包流水失败");
    } finally {
      setWalletLoading(false);
    }
  };

  return (
    <Space direction="vertical" size={24} style={{ width: "100%", padding: 24 }}>
      <Typography.Title level={4} style={{ margin: 0 }}>个人中心</Typography.Title>

      <section className="panel-card">
        <Typography.Title level={5} style={{ marginTop: 0 }}>账户信息</Typography.Title>
        <Descriptions bordered column={1} size="small" style={{ maxWidth: 600 }}>
          <Descriptions.Item label="邮箱">{userInfo?.email || "-"}</Descriptions.Item>
          <Descriptions.Item label="钱包余额">
            {userInfo ? (
              <Tag color={(userInfo.wallet_balance || 0) > 0 ? "green" : "red"}>
                {fmtCurrency(userInfo.wallet_balance || 0)}
              </Tag>
            ) : "-"}
          </Descriptions.Item>
          <Descriptions.Item label="最低可用余额">
            {userInfo ? fmtCurrency(userInfo.min_available_balance || 0) : "-"}
          </Descriptions.Item>
        </Descriptions>
      </section>

      <section className="panel-card">
        <Typography.Title level={5} style={{ marginTop: 0 }}>钱包流水</Typography.Title>
        <Table
          rowKey="id"
          loading={walletLoading}
          dataSource={walletRows}
          locale={{ emptyText: "暂无钱包流水" }}
          scroll={{ x: "max-content" }}
          pagination={{
            current: walletPagination.current,
            pageSize: walletPagination.pageSize,
            total: walletPagination.total,
            showSizeChanger: true,
            pageSizeOptions: ["10", "20", "50"],
          }}
          onChange={(p) => loadWallet(p.current, p.pageSize)}
          columns={[
            {
              title: "时间",
              dataIndex: "created_at",
              key: "created_at",
              width: 180,
              render: (v) => (v ? new Date(v).toLocaleString("zh-CN") : "-"),
            },
            {
              title: "方向",
              dataIndex: "direction",
              key: "direction",
              width: 80,
              render: (v) => <Tag color={v === "credit" ? "green" : "red"}>{v === "credit" ? "充值" : "扣费"}</Tag>,
            },
            { title: "金额", dataIndex: "amount", key: "amount", width: 140, render: (v) => fmtCurrency(v) },
            { title: "变动前", dataIndex: "balance_before", key: "balance_before", width: 140, render: (v) => fmtCurrency(v) },
            { title: "变动后", dataIndex: "balance_after", key: "balance_after", width: 140, render: (v) => fmtCurrency(v) },
            {
              title: "来源",
              dataIndex: "operator_type",
              key: "operator_type",
              width: 80,
              render: (v) => (v === "admin" ? "管理员" : v === "system" ? "系统" : v || "-"),
            },
            { title: "模型", dataIndex: "model_public_name", key: "model_public_name", width: 180, render: (v) => v || "-" },
            { title: "Token", dataIndex: "total_tokens", key: "total_tokens", width: 110, render: (v) => fmt(v) },
            { title: "消费金额", dataIndex: "billable_amount", key: "billable_amount", width: 140, render: (v) => fmtCurrency(v) },
            { title: "备注", dataIndex: "note", key: "note", width: 160, render: (v) => v || "-" },
          ]}
        />
      </section>
    </Space>
  );
}
