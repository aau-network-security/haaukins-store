package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aau-network-security/haaukins-store/model"
	pb "github.com/aau-network-security/haaukins-store/proto"
	_ "github.com/lib/pq"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	timeFormat = "2006-01-02 15:04:05"
	OK =  "ok"
)

type store struct {
	m sync.Mutex
	db *sql.DB
}

type Store interface {
	AddEvent(*pb.AddEventRequest) (string, error)
	AddTeam(*pb.AddTeamRequest) (string, error)
	GetEvents() ([]model.Event, error)
	GetTeams(string) ([]model.Team, error)
	UpdateTeamSolvedChallenge(*pb.UpdateTeamSolvedChallengeRequest) (string, error)
	UpdateTeamLastAccess(*pb.UpdateTeamLastAccessRequest) (string, error)
	UpdateEventFinishDate(*pb.UpdateEventRequest) (string, error)
}

func NewStore() (Store, error){
	db, err := NewDBConnection()

	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	err = InitTables(db)
	if err != nil {
		log.Printf("failed to init tables: %v", err)
	}
	return &store{ db: db }, nil
}

func NewDBConnection() (*sql.DB, error){

	// todo: optimize this approach, looks ugly
	host 		:= os.Getenv("DATABASE_HOST")
	portString	:= os.Getenv("DB_PORT")
	dbUser     := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName     := os.Getenv("POSTGRES_DB")

	port, err := strconv.Atoi(portString)
	if err != nil {
		return nil, err
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, dbUser, dbPassword, dbName)
	db, err := sql.Open("postgres", psqlInfo)


	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (s store) AddEvent(in *pb.AddEventRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	_, err := s.db.Exec(ADD_EVENT_QUERY, in.Tag, in.Name, in.Available, in.Capacity, in.Frontends, in.Exercises, in.StartTime, in.ExpectedFinishTime)

	if err != nil {
		return "", err
	}
	return "Event correctly added!", nil
}

func (s store) AddTeam(in *pb.AddTeamRequest) (string, error){
	s.m.Lock()
	defer s.m.Unlock()

	now := time.Now()
	nowString := now.Format(timeFormat)
	_, err := s.db.Exec(ADD_TEAM_QUERY, in.Id, in.EventTag, in.Email, in.Name, in.Password, nowString, nowString, "[]")
	if err != nil {
		return "", err
	}
	return "Team correctly added!", nil
}

func (s store) GetEvents() ([]model.Event, error) {

	s.m.Lock()
	defer s.m.Unlock()

	rows, err := s.db.Query(QUERY_EVENT_TABLE)
	if err != nil{
		return []model.Event{}, err
	}
	var events []model.Event
	for rows.Next() {

		// todo  : optimize this one, looks ugly
		var tag 				string
		var name 				string
		var frontends 			string
		var exercises 			string
		var available 			uint
		var capacity 			uint
		var startedAt			string
		var expectedFinishTime 	string
		var finishedAt			string
		rows.Scan(&tag, &name, &available, &capacity, &frontends, &exercises, &startedAt, &expectedFinishTime, &finishedAt)
		events = append(events, model.Event{
			Tag:                tag,
			Name:               name,
			Frontends:          frontends,
			Exercises:          exercises,
			Available:          available,
			Capacity:           capacity,
			StartedAt:          startedAt,
			ExpectedFinishTime: expectedFinishTime,
			FinishedAt:         finishedAt,
		})
	}

	return events, nil
}

func (s store) GetTeams(tag string) ([]model.Team, error) {
	s.m.Lock()
	defer s.m.Unlock()

	rows, err := s.db.Query(QUERY_EVENT_TEAMS, tag)
	if err != nil{
		return []model.Team{}, err
	}

	// todo  : optimize this one, looks ugly
	var teams []model.Team
	for rows.Next(){
		var id 					string
		var eventTag			string
		var email				string
		var name				string
		var password			string
		var createdAt			string
		var lastAccess			string
		var solvedChallenges	string

		rows.Scan(&id, &eventTag, &email, &name, &password, &createdAt, &lastAccess, &solvedChallenges)
		teams = append(teams, model.Team{
			Id:               id,
			EventTag:         eventTag,
			Email:            email,
			Name:             name,
			Password:         password,
			CreatedAt:        createdAt,
			LastAccess:       lastAccess,
			SolvedChallenges: solvedChallenges,
		})
	}
	return teams, nil
}

func (s store) UpdateTeamSolvedChallenge(in *pb.UpdateTeamSolvedChallengeRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	type Challenge struct {
		Tag  		string		`json:"tag"`
		CompletedAt string		`json:"completed-at"`
	}

	var solvedChallenges []Challenge
	var solvedChallengesDB string

	if err := s.db.QueryRow(QUERY_SOLVED_CHLS, in.TeamId).Scan(&solvedChallengesDB); err != nil {
		return "", err
	}

	if err := json.Unmarshal([]byte(solvedChallengesDB), &solvedChallenges); err != nil{
		return "", err
	}

	for _, sc := range solvedChallenges{
		if sc.Tag == in.Tag{
			return "", errors.New("challenge already solved")
		}
	}

	solvedChallenges = append(solvedChallenges, Challenge{
		Tag:         in.Tag,
		CompletedAt: in.CompletedAt,
	})

	newSolvedChallengesDB, _ := json.Marshal(solvedChallenges)

	_, err := s.db.Exec(UPDATE_TEAM_SOLVED_CHL, in.TeamId, string(newSolvedChallengesDB))
	if err != nil {
		return "", err
	}

	return OK, nil
}

func (s store) UpdateTeamLastAccess(in *pb.UpdateTeamLastAccessRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	_, err := s.db.Exec(UPDATE_EVENT_LASTACCESSED_DATE, in.TeamId, in.AccessAt)
	if err != nil {
		return "", err
	}

	return OK, nil
}

func (s store) UpdateEventFinishDate(in *pb.UpdateEventRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	_, err := s.db.Exec(UPDATE_EVENT_FINISH_DATE, in.EventId, in.FinishedAt)
	if err != nil {
		return "", err
	}

	return OK, nil
}