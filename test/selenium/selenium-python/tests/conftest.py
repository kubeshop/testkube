import os
import pytest
from selenium.webdriver import Remote
from selenium.webdriver.chrome.options import Options as ChromeOptions
from selenium.webdriver.firefox.options import Options as FirefoxOptions
from selenium.webdriver.edge.options import Options as EdgeOptions

def _make_options(browser: str):
    b = browser.lower()
    if b == "chrome":
        o = ChromeOptions()
        o.add_argument("--headless=new")
        o.add_argument("--no-sandbox")
        o.add_argument("--disable-dev-shm-usage")
        return o
    if b == "firefox":
        o = FirefoxOptions()
        o.add_argument("-headless")
        return o
    raise ValueError(f"Unsupported BROWSER: {browser}")

@pytest.fixture
def driver():
    remote_url = os.getenv("REMOTE_WEBDRIVER_URL")
    if not remote_url:
        raise RuntimeError("REMOTE_WEBDRIVER_URL ENV not set")

    browser = os.getenv("BROWSER", "chrome")
    options = _make_options(browser)

    drv = Remote(command_executor=remote_url, options=options)
    try:
        yield drv
    finally:
        drv.quit()