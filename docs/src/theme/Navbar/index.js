import React from "react";
import Navbar from "@theme-original/Navbar";
import {useThemeConfig} from "@docusaurus/theme-common";
import {useAnnouncementBar, useNavbarMobileSidebar} from "@docusaurus/theme-common/internal";

import styles from "./styles.module.css";

export default function NavbarWrapper() {
  const {announcementBar} = useThemeConfig();
  const {isActive, close} = useAnnouncementBar();
  const mobileSidebar = useNavbarMobileSidebar();

  if (!isActive) {
    return <Navbar />;
  }

  return (
    <>
      <Navbar />
      <div
        id={announcementBar.id}
        className={styles.announcement}
        style={{
          '--announcementBar-color': announcementBar.textColor,
          '--announcementBar-background': announcementBar.backgroundColor,
        }}
        aria-disabled={mobileSidebar.shown}
      >
        <div className={styles.announcementContent} dangerouslySetInnerHTML={{__html: announcementBar.content}} />
        {announcementBar.isCloseable ? <button className={styles.announcementClose} onClick={close}>&times;</button> : null}
      </div>
    </>
  );
}
