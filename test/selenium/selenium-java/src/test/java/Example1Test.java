import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.Disabled;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class Example1Test extends BaseTest {

    @Test
    public void example_1_1_Test() {
        driver.get("https://testkube-test-page-lipsum.pages.dev/");
        assertEquals("Testkube test page - Lorem Ipsum", driver.getTitle());
    }

    @Test
    public void example_1_2_Test() throws InterruptedException {
        Thread.sleep(5000);
    }

    @Test
    @Disabled("Work in progress")
    public void example_1_3_Test() throws InterruptedException {
        Thread.sleep(5000);
        assertEquals(1, 1);
    }
}