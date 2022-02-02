package dashreader_test

import (
	"testing"

	"github.com/anbangisak/dashreader"
)

func TestFactory(t *testing.T) {
	files := getFiles()
	for _, file := range files {
		mpd, err := dashreader.ReadMPDFromFile(file)
		if err != nil {
			t.Errorf("Error reading %s:%v", file, err)
		}
		t.Logf("================ %v =================", file)
		factory := dashreader.ReaderFactory{}
		_, err = factory.GetDASHReader("client1", "http://127.0.0.1/"+file, mpd)
		if err != nil {
			t.Errorf("Error getting reader : %v", err)
			t.Logf("================ %v =================", file)
			continue
		}
		t.Logf("================ %v =================", file)
	}
}
