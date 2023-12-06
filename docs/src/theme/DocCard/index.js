import React from "react";
import clsx from "clsx";
import Link from "@docusaurus/Link";
import {
  findFirstCategoryLink,
  useDocById,
} from "@docusaurus/theme-common/internal";
import isInternalUrl from "@docusaurus/isInternalUrl";
import { translate } from "@docusaurus/Translate";
import styles from "./styles.module.css";
import { useColorMode } from "@docusaurus/theme-common";

function CardContainer({ href, children }) {
  return (
    <Link
      href={href}
      className={clsx("card padding--md", styles.cardContainer)}
    >
      {children}
    </Link>
  );
}
function CardLayout({ href, icon, logo, title, description }) {
  return (
    <CardContainer href={href}>
      <h2 className={clsx("text--truncate", styles.cardTitle)} title={title}>
        {icon || (
          <img
            src={logo}
            width={32}
            height={32}
            style={
              useColorMode().colorMode !== "dark"
                ? {
                    WebkitFilter: "invert(1)",
                    filter: "invert(1)",
                  }
                : undefined
            }
          />
        )}{" "}
        {title}
      </h2>
      {description && (
        <p
          className={clsx("text--truncate", styles.cardDescription)}
          title={description}
        >
          {description}
        </p>
      )}
    </CardContainer>
  );
}
function CardCategory({ item }) {
  const href = findFirstCategoryLink(item);
  // Unexpected: categories that don't have a link have been filtered upfront
  if (!href) {
    return null;
  }
  return (
    <CardLayout
      href={href}
      icon="🗃️"
      title={item.label}
      description={translate(
        {
          message: "{count} items",
          id: "theme.docs.DocCard.categoryDescription",
          description:
            "The default description for a category card in the generated index about how many items this category includes",
        },
        { count: item.items.length }
      )}
    />
  );
}

const testExecutorLogo = new Map([
  [
    "/test-types/executor-artillery",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/artilleryIcon.svg",
  ],
  [
    "/test-types/executor-curl",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/curlIcon.svg",
  ],
  [
    "/test-types/executor-cypress",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/cypressIcon.svg",
  ],
  [
    "/test-types/executor-ginkgo",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/ginkgoIcon.svg",
  ],
  [
    "/test-types/executor-gradle",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/gradleIcon.svg",
  ],
  [
    "/test-types/executor-jmeter",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/jmeterIcon.svg",
  ],
  [
    "/test-types/executor-k6",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/k6Icon.svg",
  ],
  [
    "/test-types/executor-kubepug",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/kubepug.svg",
  ],
  [
    "/test-types/executor-maven",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/mavenIcon.svg",
  ],
  [
    "/test-types/executor-postman",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/postmanIcon.svg",
  ],
  [
    "/test-types/executor-soapui",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/soapIcon.svg",
  ],
  [
    "/test-types/executor-playwright",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/playwrightIcon.svg",
  ],
  [
    "/test-types/executor-zap",
    "https://raw.githubusercontent.com/kubeshop/testkube-dashboard/main/packages/web/src/assets/images/zapIcon.svg",
  ],
]);

function CardLink({ item }) {
  const executorLogo = testExecutorLogo.get(item.href);
  const icon = executorLogo
    ? undefined
    : isInternalUrl(item.href)
    ? "📄️"
    : "🔗";
  const doc = useDocById(item.docId ?? undefined);
  return (
    <CardLayout
      href={item.href}
      icon={icon}
      logo={executorLogo}
      title={item.label}
      description={doc?.description}
    />
  );
}
export default function DocCard({ item }) {
  switch (item.type) {
    case "link":
      return <CardLink item={item} />;
    case "category":
      return <CardCategory item={item} />;
    case "intro":
      return <CardIntro item={item} />;
    case "tool_icons":
      return <CardToolIcons item={item} />;
    default:
      throw new Error(`unknown item type ${JSON.stringify(item)}`);
  }
}

function CardIntro({ item }) {
  return (
    <CardLayout
      href={item.href}
      icon={item.icon}
      title={item.label}
      description={item.description}
    />
  );
}

function CardToolIcons({ item }) {
  const executorLogo = testExecutorLogo.get(item.href);
  const icon = executorLogo
  return (
    <CardLayout
      href={item.href}
      logo={executorLogo}
      description={item.description}
    />
  );
}
