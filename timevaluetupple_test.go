package shimesaba_test

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

type timeValueTuple struct {
	Time  time.Time
	Value interface{}
}

func loadTupleFromCSV(t *testing.T, path string) []timeValueTuple {
	t.Helper()
	fp, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer fp.Close()

	reader := csv.NewReader(fp)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	header := records[0]
	var timeIndex, valueIndex int
	for i, h := range header {
		if strings.HasPrefix(h, "time") {
			timeIndex = i
			continue
		}
		if strings.HasSuffix(h, "value") {
			valueIndex = i
			continue
		}
	}
	ret := make([]timeValueTuple, 0, len(records)-1)
	for _, record := range records[1:] {

		tt, err := time.Parse(time.RFC3339Nano, record[timeIndex])
		if err != nil {
			t.Fatal(err)
		}
		tv, err := strconv.ParseFloat(record[valueIndex], 64)
		if err != nil {
			t.Fatal(err)
		}
		ret = append(ret, timeValueTuple{
			Time:  tt,
			Value: tv,
		})
	}
	return ret
}
