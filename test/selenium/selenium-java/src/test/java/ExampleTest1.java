import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class ExampleTest1 extends BaseTest {

    @Test
    public void exampleTest1_1() {
        driver.get("https://testkube-test-page-lipsum.pages.dev/");
        assertEquals("Testkube test page - Lipsum", driver.getTitle());
    }

    @Test
    public void exampleTest1_2() throws InterruptedException {
        Thread.sleep(500);
    }
}