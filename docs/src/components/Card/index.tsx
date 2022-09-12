import React from "react";
import Link from "@docusaurus/Link";
import { useHotkeys } from "react-hotkeys-hook";
import useBaseUrl from "@docusaurus/useBaseUrl";

export default function Card({ children, title, hotkey, color, link }) {
  const redirectLink = useBaseUrl(link);
  hotkey &&
    useHotkeys(`shift+${hotkey}`, () => window.location.assign(redirectLink));
  return (
    <Link className="card category__card" to={link}>
      <h2
        className="text--truncate cardTitle_node_modules-@docusaurus-theme-classic-lib-theme-DocCard-styles-module"
        title={title}
      >
        <div
          style={{
            padding: "8px 10px 8px 10px",
            borderRadius: 4,
            fontSize: "0.6rem",
            border: `1px solid ${color}`,
            color: color,
            display: "inline-block",
            marginRight: 12,
          }}
        >
          {hotkey}
        </div>
        <div>{title}</div>
      </h2>
      <p className=" cardDescription_node_modules-@docusaurus-theme-classic-lib-theme-DocCard-styles-module">
        {children}
      </p>
    </Link>
  );
}
