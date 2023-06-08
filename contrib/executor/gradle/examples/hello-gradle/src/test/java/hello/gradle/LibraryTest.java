package hello.gradle;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class LibraryTest {
    @Test void someLibraryMethodReturnsTrue() {
        String env = System.getenv("TESTKUBE_GRADLE");
        assertTrue(Boolean.parseBoolean(env), "TESTKUBE_GRADLE env should be true");
    }
}
