import { useEffect, useState } from "react";
import {
  App,
  Button,
  Form,
  Input,
  Modal,
  Popconfirm,
  Space,
  Tag,
  Typography,
} from "antd";
import { useNavigate } from "react-router-dom";
import {
  createPortalClientKey,
  disablePortalClientKey,
  fetchPortalClientKeys,
  renamePortalClientKey,
} from "../api/client";
import PageHeaderCard from "../components/PageHeaderCard";
import DataTable from "../components/DataTable";

export default function PortalKeysPage() {
  const navigate = useNavigate();
  const { message, modal } = App.useApp();
  const [keys, setKeys] = useState([]);
  const [loading, setLoading] = useState(true);

  const [createOpen, setCreateOpen] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [createForm] = Form.useForm();

  const [editOpen, setEditOpen] = useState(false);
  const [editLoading, setEditLoading] = useState(false);
  const [editTarget, setEditTarget] = useState(null);
  const [editForm] = Form.useForm();

  const loadKeys = async () => {
    setLoading(true);
    try {
      const res = await fetchPortalClientKeys();
      setKeys(res.items || []);
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "加载客户端密钥失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadKeys(); }, []);

  const handleCreate = async (values) => {
    setCreateLoading(true);
    try {
      const created = await createPortalClientKey({ name: values.name.trim(), description: "" });
      setCreateOpen(false);
      createForm.resetFields();
      modal.success({
        title: "客户端密钥已创建",
        content: (
          <Space direction="vertical" size={12}>
            <Typography.Text>明文密钥只展示一次，关闭后无法再次查看。</Typography.Text>
            <Typography.Paragraph copyable style={{ marginBottom: 0 }}>
              {created.plain_api_key}
            </Typography.Paragraph>
          </Space>
        ),
      });
      await loadKeys();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "创建客户端密钥失败");
    } finally {
      setCreateLoading(false);
    }
  };

  const openEdit = (record) => {
    setEditTarget(record);
    editForm.setFieldsValue({ name: record.name });
    setEditOpen(true);
  };

  const handleEdit = async (values) => {
    setEditLoading(true);
    try {
      await renamePortalClientKey(editTarget.id, values.name.trim());
      message.success("密钥名称已更新");
      setEditOpen(false);
      setEditTarget(null);
      await loadKeys();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "更新密钥名称失败");
    } finally {
      setEditLoading(false);
    }
  };

  const handleDisable = async (record) => {
    try {
      await disablePortalClientKey(record.id);
      message.success("客户端密钥已停用");
      await loadKeys();
    } catch (error) {
      message.error(error.response?.data?.error?.message || error.message || "停用客户端密钥失败");
    }
  };

  function fmtCurrency(value) {
    const formatted = new Intl.NumberFormat("zh-CN", {
      style: "currency",
      currency: "USD",
      minimumFractionDigits: 4,
      maximumFractionDigits: 4,
    }).format(Number(value || 0));
    return <span style={{ fontVariantNumeric: "tabular-nums" }}>{formatted}</span>;
  }

  const columns = [
    { title: "名称", dataIndex: "name", key: "name", width: 160 },
    {
      title: "脱敏密钥",
      dataIndex: "masked_key",
      key: "masked_key",
      width: 200,
      render: (v) => <span style={{ fontVariantNumeric: "tabular-nums" }}>{v}</span>,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (v) => (
        <Tag bordered={false} className="tag-status" color={v === "active" ? "success" : "default"}>
          {v === "active" ? "启用" : "停用"}
        </Tag>
      ),
    },
    {
      title: "累计消费",
      key: "total_cost",
      width: 130,
      render: (_, r) => fmtCurrency(r.cost_usage?.total_cost_used),
    },
    {
      title: "最近使用",
      dataIndex: "last_used_at",
      key: "last_used_at",
      width: 180,
      render: (v) =>
        v ? (
          <span style={{ fontVariantNumeric: "tabular-nums" }}>{new Date(v).toLocaleString("zh-CN")}</span>
        ) : (
          "-"
        ),
    },
    {
      title: "过期时间",
      dataIndex: "expires_at",
      key: "expires_at",
      width: 180,
      render: (v) =>
        v ? (
          <span style={{ fontVariantNumeric: "tabular-nums" }}>{new Date(v).toLocaleString("zh-CN")}</span>
        ) : (
          "永不过期"
        ),
    },
    {
      title: "操作",
      key: "actions",
      width: 140,
      fixed: "right",
      render: (_, record) => (
        <Space size={8}>
          <Button size="small" onClick={(e) => { e.stopPropagation(); openEdit(record); }}>编辑</Button>
          <Popconfirm
            title="确定停用这把客户端密钥吗？"
            onConfirm={() => handleDisable(record)}
            onPopupClick={(e) => e.stopPropagation()}
            disabled={record.status !== "active"}
          >
            <Button size="small" disabled={record.status !== "active"} onClick={(e) => e.stopPropagation()}>停用</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Space direction="vertical" size={24} style={{ width: "100%", padding: 24 }}>
      <Typography.Title level={4} style={{ margin: 0 }}>Keys 管理</Typography.Title>

      <section className="panel-card">
        <Space style={{ width: "100%", justifyContent: "space-between", marginBottom: 16 }}>
          <Typography.Title level={5} style={{ margin: 0 }}>客户端密钥</Typography.Title>
          <Button type="primary" onClick={() => setCreateOpen(true)}>创建密钥</Button>
        </Space>
        <DataTable
          rowKey="id"
          loading={loading}
          dataSource={keys}
          pagination={false}
          columns={columns}
          onRow={(record) => ({
            onClick: () => navigate(`/portal/keys/${record.id}`, { state: { key: record } }),
            style: { cursor: "pointer" },
          })}
        />
      </section>

      <Modal
        open={createOpen}
        title="创建客户端密钥"
        onCancel={() => { setCreateOpen(false); createForm.resetFields(); }}
        onOk={() => createForm.submit()}
        confirmLoading={createLoading}
        okText="创建"
        cancelText="取消"
        destroyOnHide
      >
        <Form form={createForm} layout="vertical" onFinish={handleCreate} style={{ marginTop: 16 }}>
          <Form.Item
            label="密钥名称"
            name="name"
            rules={[{ required: true, message: "请输入密钥名称" }, { max: 64, message: "名称不超过 64 个字符" }]}
          >
            <Input placeholder="例如：my-app-prod" allowClear />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={editOpen}
        title="编辑密钥名称"
        onCancel={() => { setEditOpen(false); setEditTarget(null); }}
        onOk={() => editForm.submit()}
        confirmLoading={editLoading}
        okText="保存"
        cancelText="取消"
        destroyOnHide
      >
        <Form form={editForm} layout="vertical" onFinish={handleEdit} style={{ marginTop: 16 }}>
          <Form.Item
            label="密钥名称"
            name="name"
            rules={[{ required: true, message: "请输入密钥名称" }, { max: 64, message: "名称不超过 64 个字符" }]}
          >
            <Input allowClear />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
