import java.util.concurrent.TimeUnit;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.BeforeEach;
import static org.junit.jupiter.api.Assertions.*;

public class TestSmokeFail {
    @Test
    public void test1() {
        assertEquals(1, 2);
    }

    @Test
    public void test2() {
        try {
            TimeUnit.SECONDS.sleep(2);
        }
        catch(Exception e) {
            System.out.println(e);
        }

        assertEquals(1, 1);
    }

    @Test
    public void test3() {
        try {
            TimeUnit.SECONDS.sleep(5);
        }
        catch(Exception e) {
            System.out.println(e);
        }

        assertEquals(1, 2);
    }

    @Test
    public void test4() {
        assertEquals(1, 2);
    }

    @Test
    public void test5() {
        try {
            TimeUnit.SECONDS.sleep(2);
        }
        catch(Exception e) {
            System.out.println(e);
        }

        assertEquals(1, 1);
    }

    @Test
    public void test6() {
        assertEquals(1, 1);
    }
}
