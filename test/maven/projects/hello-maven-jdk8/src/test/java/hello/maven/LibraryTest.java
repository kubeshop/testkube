package hello.maven;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class LibraryTest {
    @Test void runMavenTests() {
        String env = System.getenv("TESTKUBE_MAVEN");
        assertTrue(Boolean.parseBoolean(env), "TESTKUBE_MAVEN env should be true");
    }
}
