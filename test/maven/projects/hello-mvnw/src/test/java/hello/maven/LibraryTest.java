package hello.maven;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class LibraryTest {
    @Test void runMavenWrapperTests() {
        String env = System.getenv("TESTKUBE_MAVEN_WRAPPER");
        assertTrue(Boolean.parseBoolean(env), "TESTKUBE_MAVEN_WRAPPER env should be true");
    }
}
