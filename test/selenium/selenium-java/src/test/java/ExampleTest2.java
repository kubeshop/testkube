import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class ExampleTest2 extends BaseTest {

    @Test
    public void exampleTest2_1() {
        driver.get("https://testkube-test-page-lipsum.pages.dev/");
        assertEquals("Testkube test page - Lipsum", driver.getTitle());
    }

    @Test
    public void exampleTest2_2() throws InterruptedException {
        Thread.sleep(700);
    }

    @Test
    public void exampleTest2_3() throws InterruptedException {
        Thread.sleep(1000);
    }
}