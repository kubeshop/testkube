import React, { useEffect } from "react";
import NotFound from "@theme-original/NotFound";
import posthog from "posthog-js";

export default function NotFoundWrapper(props) {
  useEffect(() => {
    posthog.init("phc_iir7nEWDoXebZj2fxKs8ukJlgroN7bnKBTcT8deIuJb", {
      api_host: "https://app.posthog.com",
      autocapture: false,
      capture_pageview: false,
    });

    posthog.capture("page-not-found");
  }, []);

  return (
    <>
      <NotFound {...props} />
    </>
  );
}
