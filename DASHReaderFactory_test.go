package dashreader_test

import (
	"testing"

	"github.com/eswarantg/dashreader"
)

func TestFactory(t *testing.T) {
	files := getFiles()
	for _, file := range files {
		mpd, err := dashreader.ReadMPDFromFile(file)
		if err != nil {
			t.Errorf("Error reading %s:%v", file, err)
		}
		t.Logf("================ %v =================", file)
		factory := dashreader.Factory{}
		_, err = factory.GetDASHReader(mpd)
		if err != nil {
			t.Errorf("Error getting reader : %v", err)
		}
		t.Logf("================ %v =================", file)
	}
}
