import java.util.concurrent.TimeUnit;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.BeforeEach;
import static org.junit.jupiter.api.Assertions.*;

public class TestSmoke1 {
    @BeforeEach
    public void beforeEach() {
        try {
            TimeUnit.SECONDS.sleep(1);
        }
        catch(Exception e) {
            System.out.println(e);
        }
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
        try {
            TimeUnit.SECONDS.sleep(3);
        }
        catch(Exception e) {
            System.out.println(e);
        }

        assertEquals(1, 1);
    }

    @Test
    public void test5() {
        assertEquals(1, 1);
    }

    @Test
    public void test6() {
        try {
            TimeUnit.SECONDS.sleep(6);
        }
        catch(Exception e) {
            System.out.println(e);
        }
        
        assertEquals(1, 1);
    }
}
