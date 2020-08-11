package dashreader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

//StreamSelectorList List of streams
type StreamSelectorList []StreamSelector

//NewStreamSelectorList - Loads the StreamSelector configuration from file
func NewStreamSelectorList(filename string) (*StreamSelectorList, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %v: %w", filename, err)
	}
	defer f.Close()
	rdr := bufio.NewReader(f)
	dec := json.NewDecoder(rdr)
	list := StreamSelectorList{}
	err = dec.Decode(&list)
	if err != nil {
		return nil, fmt.Errorf("error decoding json from file %v: %w", filename, err)
	}
	for i, entry := range list {
		if len(entry.ID) <= 0 {
			entry.ID = strconv.FormatInt(int64(i), 10)
		}
	}
	return &list, nil
}

//GetStream - Returns the StreamSelector for givne contentType
func (sl *StreamSelectorList) GetStream(contentType string) *StreamSelector {
	for _, StreamSelector := range *sl {
		if StreamSelector.ContentType == contentType {
			return &StreamSelector
		}
	}
	return &StreamSelector{ContentType: contentType, ID: time.Now().UTC().String()}
}
