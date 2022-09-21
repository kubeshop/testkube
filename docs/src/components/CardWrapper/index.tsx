import React from "react";

export default function CardWrapper({ children, color }) {
  return (
    <div
      style={{
        marginTop: 24,
        marginBottom: 24,
        display: "grid",
        gridTemplateColumns: "repeat(auto-fill, minmax(290px, 1fr))",
        gap: 32,
      }}
    >
      {children}
    </div>
  );
}
