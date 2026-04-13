import { Typography } from "antd";

export default function PageHeaderCard({ eyebrow, title, description, actions }) {
  return (
    <section className="hero-card">
      <div>
        <Typography.Text className="hero-eyebrow">{eyebrow}</Typography.Text>
        <Typography.Title level={2} className="hero-title">
          {title}
        </Typography.Title>
        <Typography.Paragraph className="hero-description">
          {description}
        </Typography.Paragraph>
      </div>
      {actions ? <div className="hero-actions">{actions}</div> : null}
    </section>
  );
}
