package artifacts

import "testing"

func TestTarProcessorsAllowAnEmptyArtifactMatch(t *testing.T) {
	for _, processor := range []Processor{
		NewTarProcessor("artifacts.tar.gz"),
		NewTarCachedProcessor("artifacts.tar.gz", t.TempDir()+"/artifacts.tar.gz"),
	} {
		if err := processor.Start(); err != nil {
			t.Fatalf("start empty artifact processor: %v", err)
		}
		if err := processor.End(); err != nil {
			t.Fatalf("end empty artifact processor: %v", err)
		}
	}
}
