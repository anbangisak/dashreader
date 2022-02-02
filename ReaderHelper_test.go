package dashreader_test

import (
	"net/url"
	"testing"

	"github.com/anbangisak/dashreader"
)

func TestAdjustURLPath(t *testing.T) {
	var newURL *url.URL
	var err error
	var exp string
	var adjustURL []dashreader.BaseURLType
	mpdURL := "http://127.0.0.1/default.mpd"
	refURL, err := url.Parse(mpdURL)
	if err != nil {
		t.Errorf("Supplied mpdURL(%v) not correct: %w", mpdURL, err)
	}
	newURL, err = dashreader.AdjustURLPath(*refURL, []dashreader.BaseURLType{}, "./")
	if err != nil {
		t.Errorf("Base Path test failed : %w", err)
	}
	exp = "http://127.0.0.1/"
	if newURL.String() != exp {
		t.Errorf("Base Path test failed Exp: %v Act: %v", exp, newURL.String())
	}
	t.Logf("Base Path : %v", newURL.String())

	adjustURL = []dashreader.BaseURLType{
		{Value: ""},
	}
	newURL, err = dashreader.AdjustURLPath(*refURL, adjustURL, "")
	if err != nil {
		t.Errorf("No Change test failed : %w", err)
	}
	exp = "http://127.0.0.1/default.mpd"
	if newURL.String() != exp {
		t.Errorf("No Change test failed Exp: %v Act: %v", exp, newURL.String())
	}
	t.Logf("No Change Path : %v", newURL.String())

	adjustURL = []dashreader.BaseURLType{
		{Value: "http://127.0.0.1/NewPath"},
	}
	newURL, err = dashreader.AdjustURLPath(*refURL, adjustURL, "")
	if err != nil {
		t.Errorf("Replace test failed : %w", err)
	}
	exp = "http://127.0.0.1/NewPath"
	if newURL.String() != exp {
		t.Errorf("Replace test failed Exp: %v Act: %v", exp, newURL.String())
	}
	t.Logf("Replace Path : %v", newURL.String())

	adjustURL = []dashreader.BaseURLType{
		{Value: "NewPath"},
	}
	newURL, err = dashreader.AdjustURLPath(*refURL, adjustURL, "")
	if err != nil {
		t.Errorf("Append test failed : %w", err)
	}
	exp = "http://127.0.0.1/NewPath"
	if newURL.String() != exp {
		t.Errorf("Append test failed Exp: %v Act: %v", exp, newURL.String())
	}
	t.Logf("Append Path : %v", newURL.String())

	_ = newURL
}
