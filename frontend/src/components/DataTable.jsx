import React from "react";
import { Table } from "antd";

/**
 * DataTable 是对 Ant Design Table 的高性能封装
 * 自动处理了 V3 规范中的数值对齐 (tabular-nums) 和标准间距
 */
export default function DataTable({ 
  columns = [], 
  dataSource = [], 
  loading = false, 
  rowKey = "id", 
  pagination,
  scroll = { x: "max-content" },
  ...props 
}) {
  // 防御性处理列定义
  const processedColumns = (columns || []).map(col => {
    return col;
  });

  return (
    <Table
      className="v3-data-table"
      columns={processedColumns}
      dataSource={dataSource || []}
      loading={loading}
      rowKey={rowKey}
      scroll={scroll}
      size="middle"
      pagination={pagination === false ? false : {
        showSizeChanger: true,
        showQuickJumper: true,
        showTotal: (total) => `共 ${total} 条数据`,
        ...pagination
      }}
      {...props}
    />
  );
}
