package shimesaba_test

import (
	"testing"
	"time"

	"github.com/mashiike/shimesaba"
	"github.com/stretchr/testify/require"
)

func TestReliability(t *testing.T) {

	r := shimesaba.NewReliability(
		time.Date(2022, 1, 6, 9, 39, 0, 0, time.UTC),
		time.Hour,
		map[time.Time]bool{

			time.Date(2022, 1, 6, 8, 28, 0, 0, time.UTC): true,
			time.Date(2022, 1, 6, 8, 29, 0, 0, time.UTC): false,
			time.Date(2022, 1, 6, 8, 30, 0, 0, time.UTC): true,

			time.Date(2022, 1, 6, 9, 28, 0, 0, time.UTC): true,
			time.Date(2022, 1, 6, 9, 29, 0, 0, time.UTC): false,
			time.Date(2022, 1, 6, 9, 30, 0, 0, time.UTC): true,

			time.Date(2022, 1, 6, 9, 38, 0, 0, time.UTC): true,
			time.Date(2022, 1, 6, 9, 39, 0, 0, time.UTC): false,
			time.Date(2022, 1, 6, 9, 40, 0, 0, time.UTC): true,

			time.Date(2022, 1, 6, 10, 38, 0, 0, time.UTC): true,
			time.Date(2022, 1, 6, 10, 39, 0, 0, time.UTC): false,
			time.Date(2022, 1, 6, 10, 40, 0, 0, time.UTC): true,
		},
	)
	require.EqualValues(t, time.Date(2022, 1, 6, 10, 0, 0, 0, time.UTC), r.CursorAt(), "cursorAt 2022-1-6 10:00")
	require.EqualValues(t, time.Date(2022, 1, 6, 9, 0, 0, 0, time.UTC), r.TimeFrameStartAt(), "timeFrameStartAt 2022-1-6 9:00")
	require.EqualValues(t, time.Date(2022, 1, 6, 9, 59, 59, 999999999, time.UTC), r.TimeFrameEndAt(), "timeFrameEndAt 2022-1-6 9:59:59.999999999")
	require.EqualValues(t, 58*time.Minute, r.UpTime(), "upTime 58m")
	require.EqualValues(t, 2*time.Minute, r.FailureTime(), "failureTime 2m")
	require.True(t, r.UpTime()+r.FailureTime() == r.TimeFrame(), "upTime + failureTime = timeFrame")
}

func TestReliabilityMerge(t *testing.T) {
	r := shimesaba.NewReliability(
		time.Date(2022, 1, 6, 9, 39, 0, 0, time.UTC),
		time.Hour,
		map[time.Time]bool{
			time.Date(2022, 1, 6, 9, 38, 0, 0, time.UTC): true,
			time.Date(2022, 1, 6, 9, 39, 0, 0, time.UTC): false,
			time.Date(2022, 1, 6, 9, 40, 0, 0, time.UTC): false,
			time.Date(2022, 1, 6, 9, 41, 0, 0, time.UTC): true,
		},
	)
	other := shimesaba.NewReliability(
		time.Date(2022, 1, 6, 9, 39, 0, 0, time.UTC),
		time.Hour,
		map[time.Time]bool{
			time.Date(2022, 1, 6, 9, 37, 0, 0, time.UTC): true,
			time.Date(2022, 1, 6, 9, 38, 0, 0, time.UTC): false,
			time.Date(2022, 1, 6, 9, 39, 0, 0, time.UTC): false,
			time.Date(2022, 1, 6, 9, 40, 0, 0, time.UTC): true,
		},
	)
	actual, err := r.Merge(other)
	require.NoError(t, err)
	require.EqualValues(t, time.Date(2022, 1, 6, 10, 0, 0, 0, time.UTC), actual.CursorAt(), "cursorAt 2022-1-6 10:00")
	require.EqualValues(t, time.Date(2022, 1, 6, 9, 0, 0, 0, time.UTC), actual.TimeFrameStartAt(), "timeFrameStartAt 2022-1-6 9:00")
	require.EqualValues(t, time.Date(2022, 1, 6, 9, 59, 59, 999999999, time.UTC), actual.TimeFrameEndAt(), "timeFrameEndAt 2022-1-6 9:59:59.999999999")
	require.EqualValues(t, 57*time.Minute, actual.UpTime(), "upTime 57m")
	require.EqualValues(t, 3*time.Minute, actual.FailureTime(), "failureTime 3m")
	require.True(t, actual.UpTime()+actual.FailureTime() == actual.TimeFrame(), "upTime + failureTime = timeFrame")
}

func TestReliabilities(t *testing.T) {
	allTimeIsNoViolation := map[time.Time]bool{

		time.Date(2022, 1, 6, 8, 28, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 8, 29, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 8, 30, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 9, 28, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 9, 29, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 9, 30, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 9, 38, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 9, 39, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 9, 40, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 10, 38, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 10, 39, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 10, 40, 0, 0, time.UTC): false,
	}
	tumblingWindowTimeFrame := time.Hour
	c, err := shimesaba.NewReliabilities(
		[]*shimesaba.Reliability{
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 9, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				allTimeIsNoViolation,
			),
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 8, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				allTimeIsNoViolation,
			),
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 10, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				allTimeIsNoViolation,
			),
		},
	)
	require.NoError(t, err)
	require.True(t, c[0].CursorAt().UnixNano() > c[1].CursorAt().UnixNano(), "is desc? c[0].CursorAt > c[1].CursorAt")
	require.True(t, c[1].CursorAt().UnixNano() > c[2].CursorAt().UnixNano(), "is desc? c[1].CursorAt > c[2].CursorAt")

	upTime, failureTime, deltaFailureTime := c.CalcTime(0, 2)
	require.EqualValues(t, time.Date(2022, 1, 6, 11, 0, 0, 0, time.UTC), c.CursorAt(0), "1st CursorAt")
	require.EqualValues(t, (57+58)*time.Minute, upTime, "1st upTime")
	require.EqualValues(t, (3+2)*time.Minute, failureTime, "1st failureTime")
	require.EqualValues(t, 3*time.Minute, deltaFailureTime, "1st deltaFailureTime")
	upTime, failureTime, deltaFailureTime = c.CalcTime(1, 2)
	require.EqualValues(t, time.Date(2022, 1, 6, 10, 0, 0, 0, time.UTC), c.CursorAt(1), "2nd CursorAt")
	require.EqualValues(t, (58+59)*time.Minute, upTime, "2nd upTime")
	require.EqualValues(t, (2+1)*time.Minute, failureTime, "2nd failureTime")
	require.EqualValues(t, 2*time.Minute, deltaFailureTime, "2nd deltaFailureTime")
}

func TestReliabilitiesMerge(t *testing.T) {

	tumblingWindowTimeFrame := time.Hour
	baseAllTimeIsNoViolation := map[time.Time]bool{

		time.Date(2022, 1, 6, 8, 28, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 8, 29, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 8, 30, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 9, 28, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 9, 29, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 9, 30, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 9, 38, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 9, 39, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 9, 40, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 10, 38, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 10, 39, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 10, 40, 0, 0, time.UTC): false,
	}
	base, err := shimesaba.NewReliabilities(
		[]*shimesaba.Reliability{
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 9, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				baseAllTimeIsNoViolation,
			),
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 8, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				baseAllTimeIsNoViolation,
			),
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 10, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				baseAllTimeIsNoViolation,
			),
		},
	)
	require.NoError(t, err)
	otherAllTimeIsNoViolation := map[time.Time]bool{

		time.Date(2022, 1, 6, 7, 1, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 7, 2, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 7, 3, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 8, 1, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 8, 2, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 8, 3, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 9, 1, 0, 0, time.UTC): true,
		time.Date(2022, 1, 6, 9, 2, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 9, 3, 0, 0, time.UTC): true,

		time.Date(2022, 1, 6, 10, 1, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 10, 2, 0, 0, time.UTC): false,
		time.Date(2022, 1, 6, 10, 3, 0, 0, time.UTC): false,
	}
	other, err := shimesaba.NewReliabilities(
		[]*shimesaba.Reliability{
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 7, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				otherAllTimeIsNoViolation,
			),
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 9, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				otherAllTimeIsNoViolation,
			),
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 8, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				otherAllTimeIsNoViolation,
			),
			shimesaba.NewReliability(
				time.Date(2022, 1, 6, 10, 0, 0, 0, time.UTC),
				tumblingWindowTimeFrame,
				otherAllTimeIsNoViolation,
			),
		},
	)
	require.NoError(t, err)
	actual, err := base.Merge(other)
	require.NoError(t, err)
	require.Equal(t, 4, len(actual), "merged length 4")

	require.True(t, actual[0].CursorAt().UnixNano() > actual[1].CursorAt().UnixNano(), "is desc? c[0].CursorAt > c[1].CursorAt")
	require.True(t, actual[1].CursorAt().UnixNano() > actual[2].CursorAt().UnixNano(), "is desc? c[1].CursorAt > c[2].CursorAt")
	require.True(t, actual[2].CursorAt().UnixNano() > actual[3].CursorAt().UnixNano(), "is desc? c[2].CursorAt > c[3].CursorAt")

	upTime, failureTime, deltaFailureTime := actual.CalcTime(0, 3)
	require.EqualValues(t, time.Date(2022, 1, 6, 11, 0, 0, 0, time.UTC), actual.CursorAt(0), "1st CursorAt")
	require.EqualValues(t, (54+57+58)*time.Minute, upTime, "1st upTime")
	require.EqualValues(t, (6+3+2)*time.Minute, failureTime, "1st failureTime")
	require.EqualValues(t, 6*time.Minute, deltaFailureTime, "1st deltaFailureTime")

	upTime, failureTime, deltaFailureTime = actual.CalcTime(1, 3)
	require.EqualValues(t, time.Date(2022, 1, 6, 10, 0, 0, 0, time.UTC), actual.CursorAt(1), "2nd CursorAt")
	require.EqualValues(t, (57+58+59)*time.Minute, upTime, "2nd upTime")
	require.EqualValues(t, (3+2+1)*time.Minute, failureTime, "2nd failureTime")
	require.EqualValues(t, 3*time.Minute, deltaFailureTime, "2nd deltaFailureTime")

}
