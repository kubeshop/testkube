import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class Example1Test extends BaseTest {

    @Test
    public void example_1_1_Test() {
        driver.get("https://testkube-test-page-lipsum.pages.dev/");
        assertEquals("Testkube test page - Lipsum", driver.getTitle());
    }

    @Test
    public void example_1_2_Test() throws InterruptedException {
        Thread.sleep(500);
    }
}