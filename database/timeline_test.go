package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/aau-network-security/haaukins-store/model"
)

type fakeEvent struct {
	tag       string
	available int
	capacity  int
	sT        time.Time
	fT        time.Time
}

// helper functions

func createDBConnection() (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		"localhost", 5432, "postgres", "postgres", "dummydb")
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func setup() (*sql.DB, error) {
	db, err := createDBConnection()
	if err != nil {
		return nil, fmt.Errorf("error on creating databse connection %v", err)
	}
	if err := InitTables(db); err != nil {
		return nil, fmt.Errorf("initialization of db tables error %v", err)
	}
	if err := cleanRecords(db); err != nil {
		return nil, fmt.Errorf("cleaning existing records error %v", err)
	}
	return db, nil
}

func insertTeamEvent(eid int, db *sql.DB) error {

	_, err := db.Exec(AddTeamQuery, "", eid, "random@email.com", "randomteam", "12345", time.Now(), time.Now(), 0, "[]", "[]")
	if err != nil {
		return err
	}
	return err
}

func insertFakeEvent(event fakeEvent, db *sql.DB) error {
	_, err := db.Exec(AddEventQuery, event.tag, "", event.available, event.capacity, "kali", 1, "ftp,sql", event.sT.UTC(), event.fT.UTC(), time.Date(0001, 01, 01, 00, 00, 00, 0000, time.UTC).Format(time.RFC3339), "tester", false)
	if err != nil {
		return err
	}
	return nil
}
func cleanRecords(db *sql.DB) error {
	// initially delete all records
	_, err := db.Query("DELETE FROM event;")
	if err != nil {
		return err
	}
	_, err = db.Query("DELETE FROM team;")
	if err != nil {
		return err
	}
	return nil
}

// tests starts here

func TestIsInvalidDate(t *testing.T) {

	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		{name: "Invalid date format", t: time.Date(0001, 01, 01, 00, 00, 00, 0000, time.UTC), want: true},
		{name: "Valid date format", t: time.Now(), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInvalidDate(tt.t); got != tt.want {
				t.Errorf("isInvalidDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDaysInDates(t *testing.T) {

	tests := []struct {
		name string
		sT   time.Time
		fT   time.Time
		want int
	}{
		{name: "2 Days", sT: time.Now(), fT: time.Now().AddDate(0, 0, 2), want: 2},
		{name: "1 Day", sT: time.Now(), fT: time.Now().AddDate(0, 0, 1), want: 1},
		{name: "20 Day", sT: time.Now(), fT: time.Now().AddDate(0, 0, 20), want: 20},
		{name: "0 Day", sT: time.Now(), fT: time.Now().AddDate(0, 0, 0), want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := daysInDates(tt.sT, tt.fT); got != tt.want {
				t.Errorf("daysInDates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDates(t *testing.T) {
	tests := []struct {
		name string
		sT   time.Time
		fT   time.Time
		want []time.Time
	}{
		{name: "3 Days",
			sT: time.Date(2020, 05, 19, 00, 00, 00, 00000, time.UTC),
			fT: time.Date(2020, 05, 21, 00, 00, 00, 00000, time.UTC),
			want: []time.Time{
				time.Date(2020, 05, 19, 00, 00, 00, 00000, time.UTC),
				time.Date(2020, 05, 20, 00, 00, 00, 00000, time.UTC),
				time.Date(2020, 05, 21, 00, 00, 00, 00000, time.UTC)},
		},
		{name: "1 Days",
			sT: time.Date(2020, 05, 19, 00, 00, 00, 00000, time.UTC),
			fT: time.Date(2020, 05, 19, 00, 00, 00, 00000, time.UTC),
			want: []time.Time{
				time.Date(2020, 05, 19, 00, 00, 00, 00000, time.UTC),
			},
		},
		{name: "2 Days",
			sT: time.Date(2020, 05, 19, 00, 00, 00, 00000, time.UTC),
			fT: time.Date(2020, 05, 20, 00, 00, 00, 00000, time.UTC),
			want: []time.Time{
				time.Date(2020, 05, 19, 00, 00, 00, 00000, time.UTC),
				time.Date(2020, 05, 20, 00, 00, 00, 00000, time.UTC),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDates(tt.sT, tt.fT); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZeroTime(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want time.Time
	}{
		{name: "19 May 1919", t: time.Date(1919, 05, 19, 05, 04, 24, 4152, time.UTC),
			want: time.Date(1919, 05, 19, 00, 00, 00, 0000, time.UTC)},
		{name: "23 April 1920", t: time.Date(1920, 04, 23, 12, 24, 54, 1251, time.UTC),
			want: time.Date(1920, 04, 23, 00, 00, 00, 0000, time.UTC)},
		{name: "9  November 1938 :( ", t: time.Date(1938, 11, 9, 9, 05, 05, 0005, time.UTC),
			want: time.Date(1938, 11, 9, 00, 00, 00, 0000, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zeroTime(tt.t); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("zeroTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEvents(t *testing.T) {
	var eventID int
	db, err := setup()
	if err != nil {
		t.Fatalf("Setting up env error %v ", err)
	}
	startTime, _ := time.Parse(TimeFormat, "1919-05-19 19:19:19")
	expectedFinishTime, _ := time.Parse(TimeFormat, "1923-04-23 09:00:00")

	if err := insertFakeEvent(fakeEvent{tag: "zafer", sT: startTime, fT: expectedFinishTime}, db); err != nil {
		t.Fatalf("Executing add event error %v", err)
	}

	if err = db.QueryRow("SELECT event.id FROM event where event.tag='zafer'").Scan(&eventID); err != nil {
		t.Fatalf("Error on getting event id  %v", err)
	}
	eventIds := make(map[string]uint)
	eventIds["zafer"] = uint(eventID)
	events := []model.Event{model.Event{
		Id:                 eventIds["zafer"],
		Tag:                "zafer",
		Name:               "",
		Frontends:          "kali",
		Exercises:          "ftp,sql",
		Available:          0, // fakeEvent does not need available or capacity, hence setting it to zero is ok
		Capacity:           0, // fakeEvent does not need available or capacity, hence setting it to zero is ok
		Status:             1,
		StartedAt:          startTime.Format(time.RFC3339),
		ExpectedFinishTime: expectedFinishTime.Format(time.RFC3339),
		FinishedAt:         time.Date(0001, 01, 01, 00, 00, 00, 0000, time.UTC).Format(time.RFC3339),
		CreatedBy:          "tester",
		OnlyVPN:            false,
	}}

	if got := getEvents(db); !reflect.DeepEqual(got, events) {
		t.Errorf("getEvents() = %v, want %v", got, events)
	}

}

func TestGetLatestDate(t *testing.T) {
	db, err := setup()
	if err != nil {
		t.Fatalf("Setting up env error %v ", err)
	}
	sT, _ := time.Parse(TimeFormat, "1919-05-19 19:19:19")
	fT, _ := time.Parse(TimeFormat, "1923-04-23 09:00:00")
	fEvent := fakeEvent{sT: sT, fT: fT, capacity: 10, tag: "19May"}
	if err := insertFakeEvent(fEvent, db); err != nil {
		t.Fatalf("insertFakeEvention error on events for event %s %v", fEvent.tag, err)
	}
	sT, _ = time.Parse(TimeFormat, "1923-10-29 19:19:19")
	fT, _ = time.Parse(TimeFormat, "1938-11-10 09:05:00")
	fEvent = fakeEvent{sT: sT, fT: fT, capacity: 10, tag: "23April"}
	if err := insertFakeEvent(fEvent, db); err != nil {
		t.Fatalf("insertFakeEvention error on events for event %s %v", fEvent.tag, err)
	}

	got, err := getLastDate(db)
	if err != nil {
		t.Errorf("getLastDate() error = %v", err)
		return
	}
	// latest date is fT
	// UTC is trailing that's why Local() called
	want := fT.Local()
	if !reflect.DeepEqual(got.Local(), want) {
		t.Errorf("getLastDate() got = %v, want %v", got, want)
	}
}

func TestGetEarliestDate(t *testing.T) {
	db, err := setup()
	if err != nil {
		t.Fatalf("Setting up env error %v ", err)
	}
	sT, _ := time.Parse(TimeFormat, "1919-05-19 19:19:19")
	minST := sT
	fT, _ := time.Parse(TimeFormat, "1923-04-23 09:00:00")
	fEvent := fakeEvent{sT: sT, fT: fT, capacity: 10, tag: "19May"}
	if err := insertFakeEvent(fEvent, db); err != nil {
		t.Fatalf("insertFakeEvention error on events for event %s %v", fEvent.tag, err)
	}
	sT, _ = time.Parse(TimeFormat, "1923-10-29 19:19:19")
	fT, _ = time.Parse(TimeFormat, "1938-11-10 09:05:00")
	fEvent = fakeEvent{sT: sT, fT: fT, capacity: 10, tag: "23April"}
	if err := insertFakeEvent(fEvent, db); err != nil {
		t.Fatalf("insertFakeEvention error on events for event %s %v", fEvent.tag, err)
	}

	got, err := getEarliestDate(db)
	if err != nil {
		t.Errorf("getLastDate() error = %v", err)
		return
	}
	// latest date is fT
	// UTC is trailing that's why Local() called
	want := minST.Local()
	if !reflect.DeepEqual(got.Local(), want) {
		t.Errorf("getLastDate() got = %v, want %v", got, want)
	}
}

// used for calculateCost
func addFakeEvents(db *sql.DB) error {
	cleanRecords(db)
	sT, _ := time.Parse(TimeFormat, "2020-05-19 19:19:19")
	eFT, _ := time.Parse(TimeFormat, "2020-05-23 09:00:00")

	// test1 								// test2
	// 	sT: 2020-05-19 19:19:19   			   // sT: 2020-05-20 19:19:19
	//	   fT: 2020-05-23 09:00:00				  // fT: 2020-05-30 09:00:00

	fEvents := []fakeEvent{{tag: "test1", available: 5, capacity: 10, sT: sT, fT: eFT}, {tag: "test2", available: 7, capacity: 15, sT: sT.AddDate(0, 0, 1), fT: eFT.AddDate(0, 0, 7)}}

	for _, e := range fEvents {
		if err := insertFakeEvent(e, db); err != nil {
			return err
		}
	}
	var eventId int
	if err := db.QueryRow(QueryEventId, "test1").Scan(&eventId); err != nil {
		return err
	}
	// add 5 teams to event test1
	for i := 500; i < 505; i++ {
		if err := insertTeamEvent(eventId, db); err != nil {
			return err
		}
	}
	if err := db.QueryRow(QueryEventId, "test2").Scan(&eventId); err != nil {
		return err
	}
	// add 10 teams to event test2
	for i := 900; i < 910; i++ {
		if err := insertTeamEvent(eventId, db); err != nil {
			return err
		}
	}
	return nil

}

func TestCalculateCost(t *testing.T) {
	db, err := setup()
	if err != nil {
		t.Fatalf("Setting up env error %v ", err)
	}
	// add events & teams
	if err := addFakeEvents(db); err != nil {
		t.Fatalf("error on generating fake events %v", err)
	}

	expectedResult := map[string]int32{
		"2020-05-19 00:00:00": 10,
		"2020-05-20 00:00:00": 10 + 17,
		"2020-05-21 00:00:00": 10 + 17,
		"2020-05-22 00:00:00": 10 + 17,
		"2020-05-23 00:00:00": 10 + 17,
		"2020-05-24 00:00:00": 17,
		"2020-05-25 00:00:00": 17,
		"2020-05-26 00:00:00": 17,
		"2020-05-27 00:00:00": 17,
		"2020-05-28 00:00:00": 17,
		"2020-05-29 00:00:00": 17,
		"2020-05-30 00:00:00": 17,
	}
	got, err := calculateCost(db)
	if err != nil {
		t.Errorf("calculateCost() error = %v", err)
		return
	}

	if !reflect.DeepEqual(got, expectedResult) {
		t.Errorf("calculateCost() got = %v, want %v", got, expectedResult)
	}
}
