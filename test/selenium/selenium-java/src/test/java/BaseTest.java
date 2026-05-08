import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.remote.RemoteWebDriver;
import org.openqa.selenium.remote.DesiredCapabilities;

import java.net.URL;

public abstract class BaseTest {

    protected WebDriver driver;

    @BeforeEach
    public void setUp() throws Exception {
        String remoteUrl = System.getenv("REMOTE_WEBDRIVER_URL");
        if (remoteUrl == null || remoteUrl.isBlank()) {
            throw new IllegalArgumentException("REMOTE_WEBDRIVER_URL ENV not set");
        }

        String browser = System.getenv().getOrDefault("BROWSER", "chrome");
        DesiredCapabilities capabilities = new DesiredCapabilities();
        capabilities.setBrowserName(browser);
        driver = new RemoteWebDriver(new URL(remoteUrl), capabilities);
    }

    @AfterEach
    public void tearDown() {
        if (driver != null) {
            driver.quit();
        }
    }
}