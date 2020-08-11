package dashreader_test

import (
	"testing"

	"github.com/eswarantg/dashreader"
)

func getFiles() []string {
	return []string{
		"test/live_Man30supd.mpd",
		"test/live_noManupd.mpd",
		"test/live_SegTimeline.mpd",
		"test/ll_number_default.mpd",
		"test/ll_time_default.mpd",
	}
}

func printMPDFields(t *testing.T, level string, mpd dashreader.MPDtype) {
	t.Logf("\t Profile:%v", mpd.Profiles)
	t.Logf("\t PT:%v AST:%v ASTE:%v", mpd.PublishTime, mpd.AvailabilityStartTime, mpd.AvailabilityEndTime)
	tsb, err := dashreader.ParseDuration(mpd.TimeShiftBufferDepth)
	if err != nil {
		t.Errorf("Error Prasing TimeShiftBufferDepth %v : %v", mpd.TimeShiftBufferDepth, err)
	} else {
		t.Logf("%v TSB:%v ", level, tsb)
	}
	mup, err := dashreader.ParseDuration(mpd.MinimumUpdatePeriod)
	if err != nil {
		t.Errorf("Error Prasing MinimumUpdatePeriod %v : %v", mpd.MinimumUpdatePeriod, err)
	} else {
		t.Logf("%v MUP:%v ", level, mup)
	}
	mbt, err := dashreader.ParseDuration(mpd.MinBufferTime)
	if err != nil {
		t.Errorf("Error Prasing MinBufferTime %v : %v", mpd.MinBufferTime, err)
	} else {
		t.Logf("%v MBT:%v ", level, mbt)
	}
	if len(mpd.SuggestedPresentationDelay) > 0 {
		spd, err := dashreader.ParseDuration(mpd.SuggestedPresentationDelay)
		if err != nil {
			t.Errorf("Error Prasing SuggestedPresentationDelay %v : %v", mpd.SuggestedPresentationDelay, err)
		} else {
			t.Logf("%v SPD:%v ", level, spd)
		}
	}
}

func printPeriodFields(t *testing.T, level string, period dashreader.PeriodType) {
	ps, err := dashreader.ParseDuration(period.Start)
	if err != nil {
		t.Errorf("Error Prasing Period.Start %v : %v", period.Start, err)
	} else {
		t.Logf("%v PS:%v ", level, ps)
	}
}

func printAdaptationSetFields(t *testing.T, level string, adaptSet dashreader.AdaptationSetType) {
}

func TestNewDashClient(t *testing.T) {
	files := getFiles()
	for _, file := range files {
		mpd, err := dashreader.ReadMPDFromFile(file)
		if err != nil {
			t.Errorf("Error reading %s:%v", file, err)
		}
		t.Logf("================ %v =================", file)
		t.Logf("MPD:")
		t.Logf("\t Type:%v", mpd.Type)
		printMPDFields(t, "\t", *mpd)
		for _, period := range mpd.Period {
			t.Logf("\t Period: %v", period.Id)
			printPeriodFields(t, "\t\t", period)
			for _, adaptSet := range period.AdaptationSet {
				t.Logf("\t\t AdaptSet: %v", adaptSet.Id)
				printAdaptationSetFields(t, "\t\t\t", adaptSet)
			}
		}
		t.Logf("================ %v =================", file)
	}
}
