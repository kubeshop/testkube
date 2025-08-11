import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class Example2Test extends BaseTest {

    @Test
    public void example_2_1_Test() throws InterruptedException {
        driver.get("https://testkube-test-page-lipsum.pages.dev/");
        Thread.sleep(10000); // just to make the test longer
        assertEquals("Testkube test page - Lorem Ipsum", driver.getTitle());
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