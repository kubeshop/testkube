import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class Example2Test extends BaseTest {

    @Test
    public void example_2_1_Test() {
        driver.get("https://testkube-test-page-lipsum.pages.dev/");
        assertEquals("Testkube test page - Lipsum", driver.getTitle());
    }

    @Test
    public void example_2_2_Test() throws InterruptedException {
        Thread.sleep(700);
    }

    @Test
    public void example_2_3_Test() throws InterruptedException {
        Thread.sleep(1000);
    }
}