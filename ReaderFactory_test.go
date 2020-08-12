package dashreader_test

import (
	"context"
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
		factory := dashreader.ReaderFactory{}
		rdr, err := factory.GetDASHReader("client1", "http://127.0.0.1/"+file, mpd)
		if err != nil {
			t.Errorf("Error getting reader : %v", err)
			t.Logf("================ %v =================", file)
			continue
		}
		representationSelector := dashreader.MinBWRepresentationSelector{}
		streamSelector := dashreader.StreamSelector{
			ID:          "1",
			ContentType: "video",
		}
		readCtx, err := rdr.MakeDASHReaderContext(nil, streamSelector, representationSelector)
		if err != nil {
			t.Errorf("Error getting context : %v", err)
			t.Logf("================ %v =================", file)
			continue
		}
		urlChan, err := readCtx.NextURLs(context.TODO())
		if err != nil {
			t.Errorf("Error getting urls : %v", err)
			t.Logf("================ %v =================", file)
			continue
		}
		for url := range urlChan {
			t.Logf("URL: %v", url)
		}
		t.Logf("================ %v =================", file)
	}
}
