package dashreader

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/PaesslerAG/gval"
	"github.com/rickb777/date/period"
)

// ReadMPDFromStream - Reads from an io.Reader interface into an MPD object returned.
// r - Must implement the io.Reader interface.
func ReadMPDFromStream(r io.Reader) (*MPDtype, error) {
	var mpd MPDtype
	d := xml.NewDecoder(r)
	err := d.Decode(&mpd)
	if err != nil {
		return nil, err
	}
	return &mpd, nil
}

// ReadMPDFromFile - Reads from a File strored into an MPD object returned.
func ReadMPDFromFile(filename string) (*MPDtype, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rdr := bufio.NewReader(f)
	return ReadMPDFromStream(rdr)
}

//GetFrameRate - Evaluate Framerate to float
func GetFrameRate(frameRate string) (float64, error) {
	value, err := gval.Evaluate(frameRate, nil)
	if err != nil {
		return 0.0, fmt.Errorf("Error evaluationg framerate %w", err)
	}
	switch v := reflect.ValueOf(value); v.Kind() {
	case reflect.Int:
		return float64(v.Int()), nil
	case reflect.Float32:
	case reflect.Float64:
		return v.Float(), nil
	default:
		err = fmt.Errorf("Error evaluationg framerate unexpected type %v", v.Kind())
	}
	return 0.0, err
}

//ParseDuration - Convert to time.Duration
func ParseDuration(durationStr string) (time.Duration, error) {
	period, err := period.Parse(durationStr)
	if err != nil {
		return 0 * time.Second, err
	}
	duration, precise := period.Duration()
	_ = precise //For now ignore
	return duration, nil
}

//IsPresentTime - Checks if Time field is Valid (Non-ZERO)
func IsPresentTime(val time.Time) bool {
	var ZEROTIME = time.Time{}
	if val == ZEROTIME {
		return false
	}
	return true
}

//IsPresentDuration - Checks if Time field is Valid (Non-ZERO)
func IsPresentDuration(durationStr string) bool {
	if len(durationStr) > 0 {
		return true
	}
	return false
}

//GetBoolFromConditionalUintType - returns true/false
func GetBoolFromConditionalUintType(v ConditionalUintType) bool {
	if strings.EqualFold(string(v), "true") {
		return true
	}
	return false
}
