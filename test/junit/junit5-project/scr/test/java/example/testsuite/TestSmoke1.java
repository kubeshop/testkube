package example.testsuite;

import java.util.concurrent.TimeUnit;

import org.junit.jupiter.api.Test;

public class TestSmoke1 {
    @BeforeEach
    public void beforeEach() {
        TimeUnit.SECONDS.sleep(1);
    }

    @Test
    public void test1() {
        assertEquals(1, 1);
    }

    @Test
    public void test2() {
        assertEquals(1, 1);
    }

    @Test
    public void test3() {
        assertEquals(1, 1);
    }

    @Test
    public void test4() {
        TimeUnit.SECONDS.sleep(3);
        assertEquals(1, 1);
    }

    @Test
    public void test5() {
        assertEquals(1, 1);
    }

    @Test
    public void test6() {
        TimeUnit.SECONDS.sleep(6);
        assertEquals(1, 1);
    }
}
