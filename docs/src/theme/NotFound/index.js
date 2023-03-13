import React, { useEffect, useState } from "react";
import posthog from "posthog-js";
import algoliasearch from "algoliasearch";
import Layout from "@theme/Layout";
import { useLocation } from "@docusaurus/router";
import { PageMetadata } from "@docusaurus/theme-common";
import Translate, { translate } from "@docusaurus/Translate";
import styles from "./styles.module.css";

export default function NotFound() {
  const [searchResults, setSearchResults] = useState([]);
  const [loading, setLoading] = useState(true);

  const client = algoliasearch(
    "QRREOKFLDE",
    "97a76158bf582346aa0c2605cbc593f6"
  );
  const index = client.initIndex("testkube");
  const location = useLocation();

  useEffect(() => {
    posthog.init("phc_iir7nEWDoXebZj2fxKs8ukJlgroN7bnKBTcT8deIuJb", {
      api_host: "https://app.posthog.com",
      autocapture: false,
      capture_pageview: false,
    });

    posthog.capture("page-not-found");
  }, []);

  useEffect(() => {
    const getSearchResults = async () => {
      const query = location.pathname;
      // Remove the no-no words from the query
      const removeList = ["testkube"];
      const parsedQuery = query
        .split("/")
        .filter((word) => !removeList.includes(word))
        .join(" ");

      const searchResults = await index.search(parsedQuery, {
        hitsPerPage: 5,
      });
      const hits = searchResults.hits;

      setSearchResults(hits);
      setLoading(false);
    };

    getSearchResults();
  }, []);

  return (
    <>
      <PageMetadata
        title={translate({
          id: "theme.NotFound.title",
          message: "Page Not Found",
        })}
      />
      <Layout>
        <main className="container margin-vert--xl">
          <div className="row">
            <div className="col col--6 col--offset-3">
              <h1>
                <Translate
                  id="theme.NotFound.title"
                  description="The title of the 404 page"
                >
                  You have found a broken link or the URL entered doesn't exist
                  in our docs.
                </Translate>
              </h1>
              <p>
                <Translate
                  id="theme.NotFound.p2"
                  description="The 2nd paragraph of the 404 page"
                >
                  {!loading && searchResults.length > 0
                    ? `Is there a chance one of these links will help?`
                    : ``}
                </Translate>
              </p>
              <ul className={styles["results"]}>
                {searchResults &&
                  searchResults.map((result) => (
                    <li key={result.objectID}>
                      <div>
                        <a href={result.url}>{result.hierarchy.lvl1}</a>
                      </div>
                    </li>
                  ))}
              </ul>
            </div>
          </div>
        </main>
      </Layout>
    </>
  );
}
