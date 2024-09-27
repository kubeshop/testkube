package example.testsuite;

import java.util.concurrent.TimeUnit;

import org.junit.jupiter.api.Test;

public class TestSmokeFail {
    @Test
    public void test1() {
        assertEquals(1, 2);
    }

    @Test
    public void test2() {
        TimeUnit.SECONDS.sleep(2);
        assertEquals(1, 1);
    }

    @Test
    public void test3() {
        TimeUnit.SECONDS.sleep(5);
        assertEquals(1, 2);
    }
}
