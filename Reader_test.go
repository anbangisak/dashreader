package dashreader_test

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eswarantg/dashreader"
	"github.com/eswarantg/statzagg"
)

func getMPD(t *testing.T, url string) (mpd *dashreader.MPDtype, err error) {
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("Error getting \"%v\" : %v", url, err)
		return
	}
	defer resp.Body.Close()
	mpd, err = dashreader.ReadMPDFromStream(resp.Body)
	if err != nil {
		t.Errorf("Error reading %s:%v", url, err)
	}
	return
}

func getReader(t *testing.T, mpd *dashreader.MPDtype, url string) (rdr dashreader.Reader, err error) {
	factory := dashreader.ReaderFactory{}
	rdr, err = factory.GetDASHReader("client1", url, mpd)
	if err != nil {
		t.Errorf("Error finding reader : %v", err)
		return
	}
	return
}

func getReaderCtx(t *testing.T, rdr dashreader.Reader,
	oldReaderCtx dashreader.ReaderContext,
	streamSelector dashreader.StreamSelector,
	representationSelector dashreader.RepresentationSelector) (readCtx dashreader.ReaderContext, err error) {
	readCtx, err = rdr.MakeDASHReaderContext(oldReaderCtx, streamSelector, representationSelector)
	if err != nil {
		t.Errorf("Error getting context : %v", err)
		return
	}
	return
}

func printURLs(t *testing.T, readCtx dashreader.ReaderContext) (err error) {
	urlChan, err := readCtx.NextURLs(context.TODO())
	if err != nil {
		if err != io.EOF {
			t.Errorf("Error getting urls : %v", err)
		}
		return
	}
	for url := range urlChan {
		t.Logf("URL: %v", url)
	}
	return
}

func TestOneManifest(t *testing.T) {
	var url string = "https://livesim.dashif.org/livesim/segtimeline_1/testpic_2s/Manifest.mpd"
	representationSelector := dashreader.MinBWRepresentationSelector{}
	streamSelector := dashreader.StreamSelector{
		ID:          "1",
		ContentType: "video",
	}
	t.Logf("================ %v =================", url)
	defer t.Logf("================ %v =================", url)
	mpd, err := getMPD(t, url)
	if err != nil {
		return
	}
	rdr, err := getReader(t, mpd, url)
	if err != nil {
		return
	}
	readCtx, err := getReaderCtx(t, rdr, nil, streamSelector, representationSelector)
	if err != nil {
		return
	}
	err = printURLs(t, readCtx)
	if err != nil {
		return
	}
}

func TestNManifest(t *testing.T) {
	var urlTest string = "https://livesim.dashif.org/livesim/segtimeline_1/testpic_2s/Manifest.mpd"
	representationSelector := dashreader.MinBWRepresentationSelector{}
	streamSelector := dashreader.StreamSelector{
		ID:          "1",
		ContentType: "video",
	}
	logStatzAgg := statzagg.NewLogStatzAgg(os.Stdout)
	t.Logf("================ %v =================", urlTest)
	mpd, err := getMPD(t, urlTest)
	if err != nil {
		return
	}
	wcRef := time.Now()
	rdr, err := getReader(t, mpd, urlTest)
	if err != nil {
		return
	}
	rdr.SetStatzAgg(logStatzAgg)
	var wg sync.WaitGroup
	ch := make(chan bool)
	wg.Add(1)
	go func(ch <-chan bool) {
		i := 0
		defer wg.Done()
		var readCtx dashreader.ReaderContext
		var newRdrCtx dashreader.ReaderContext
		var err error
		for {
			select {
			case _, ok := <-ch:
				if !ok {
					return
				}
			}
			i++
			t.Logf("\n Run %v", i)
			t.Logf("---------------- %v ----------------", urlTest)
			newRdrCtx, err = getReaderCtx(t, rdr, readCtx, streamSelector, representationSelector)
			readCtx = newRdrCtx
			if err == nil {
				err = printURLs(t, readCtx)
			}
			t.Logf("---------------- %v ----------------", urlTest)
		}
	}(ch)
	N := 20
	d := 2 * time.Second
	if len(mpd.MinimumUpdatePeriod) > 0 {
		v, err := dashreader.ParseDuration(mpd.MinimumUpdatePeriod)
		if err != nil {
			t.Errorf("Unable to parse MinimumUpdatePeriod")
		} else {
			t.Logf("MUD: %v", v)
			if v > d {
				d = v
			}
		}
	}
	for i := 0; i < N; i++ {
		curWc := time.Now()
		sleepDur := d - curWc.Sub(wcRef)
		wcRef = curWc
		time.Sleep(sleepDur)
		if i < N {
			mpd, err = getMPD(t, urlTest)
			if err != nil {
				return
			}
			updated, err := rdr.Update(mpd)
			if err != nil {
				t.Errorf("Unable to update MPD %v", err)
				continue
			}
			if updated {
				ch <- true
				p, err := url.Parse(urlTest)
				if err != nil {
					t.Logf("Unable to Parser URL %v : %v", urlTest, err)
					continue
				}
				filename := "test/" + p.Host + "_" + strings.ReplaceAll(p.Path, "/", "_") + strconv.Itoa(i)
				f, err := os.Create(filename)
				if err != nil {
					t.Logf("Unable to Create file %v : %v", filename, err)
					continue
				}
				enc := xml.NewEncoder(f)
				enc.Indent("  ", "    ")
				if err := enc.Encode(mpd); err != nil {
					fmt.Printf("error: %v\n", err)
				}
				f.Close()
			}
		}
	}
	close(ch)
	wg.Wait()
	t.Logf("================ %v =================", urlTest)
}
