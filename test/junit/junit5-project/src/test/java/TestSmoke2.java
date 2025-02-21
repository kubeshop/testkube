import java.util.concurrent.TimeUnit;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.BeforeEach;
import static org.junit.jupiter.api.Assertions.*;

public class TestSmoke2 {
    @Test
    public void testAaa() {
        try {
            TimeUnit.SECONDS.sleep(1);
        }
        catch(Exception e) {
            System.out.println(e);
        }
        
        assertEquals(1, 1);
    }

    @Test
    public void testBbb() {
        assertEquals(1, 1);
    }

    @Test
    public void testCcc() {
        try {
            TimeUnit.SECONDS.sleep(2);
        }
        catch(Exception e) {
            System.out.println(e);
        }

        assertEquals(1, 1);
    }

    @Test
    public void testDdd() {
        assertEquals(1, 1);
    }
}
