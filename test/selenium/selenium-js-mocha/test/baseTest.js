import { Builder } from 'selenium-webdriver';

export async function createDriver() {
  const url = process.env.REMOTE_WEBDRIVER_URL;
  
  if (!url) throw new Error('REMOTE_WEBDRIVER_URL not set');
  const browser = process.env.BROWSER || 'chrome';
  
  return new Builder().usingServer(url).forBrowser(browser).build();
}