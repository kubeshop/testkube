import java.util.concurrent.TimeUnit;

import org.junit.jupiter.api.Test;

public class SmokeTest2 {
    @Test
    public void testAaa() {
        TimeUnit.SECONDS.sleep(1);
        assertEquals(1, 1);
    }

    @Test
    public void testBbb() {
        assertEquals(1, 1);
    }

    @Test
    public void testCcc() {
        TimeUnit.SECONDS.sleep(2);
        assertEquals(1, 1);
    }

    @Test
    public void testDdd() {
        assertEquals(1, 1);
    }
}
