package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aau-network-security/haaukins-store/model"
	pb "github.com/aau-network-security/haaukins-store/proto"
	_ "github.com/lib/pq"
)

const handleNullConversionError = "converting NULL to string is unsupported"

var (
	TimeFormat = "2006-01-02 15:04:05"
	OK         = "ok"
	Error      = int32(3)

	Running   = State(0)
	Suspended = State(1)
	Booked    = State(2)
	Closed    = State(3)
)

type State int32

type store struct {
	m  sync.Mutex
	db *sql.DB
}

type Store interface {
	AddEvent(*pb.AddEventRequest) (string, error)
	AddTeam(*pb.AddTeamRequest) (string, error)
	GetEvents(*pb.GetEventRequest) ([]model.Event, error)
	GetEventByUser(*pb.GetEventByUserReq) ([]model.Event, error)
	GetTeams(string) ([]model.Team, error)
	IsEventExists(*pb.GetEventByTagReq) (bool, error)
	DropEvent(req *pb.DropEventReq) (bool, error)
	GetCostsInTime() (map[string]int32, error)
	GetEventStatus(*pb.GetEventStatusRequest) (int32, error)
	SetEventStatus(*pb.SetEventStatusRequest) (int32, error)
	UpdateTeamSolvedChallenge(*pb.UpdateTeamSolvedChallengeRequest) (string, error)
	UpdateTeamLastAccess(*pb.UpdateTeamLastAccessRequest) (string, error)
	UpdateTeamPassword(in *pb.UpdateTeamPassRequest) error
	GetEventID(in *pb.GetEventIDReq) (int32, error)
	UpdateExercises(req *pb.UpdateExerciseRequest) (string, error)
	UpdateCloseEvent(*pb.UpdateEventRequest) (string, error)
	DelTeam(request *pb.DelTeamRequest) (string, error)
	AddProfile(request *pb.AddProfileRequest) (string, error)
	GetProfiles() ([]model.Profile, error)
	UpdateProfile(request *pb.UpdateProfileRequest) (string, error)
	DeleteProfile(request *pb.DelProfileRequest) (string, error)
}

func NewStore(conf *model.Config) (Store, error) {
	db, err := NewDBConnection(conf)

	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
		return nil, err
	}
	err = InitTables(db)
	if err != nil {
		log.Printf("failed to init tables: %v", err)
		return nil, err
	}
	return &store{db: db}, nil
}

func NewDBConnection(conf *model.Config) (*sql.DB, error) {

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		conf.DB.Host, conf.DB.Port, conf.DB.User, conf.DB.Pass, conf.DB.Name)
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

func (s *store) AddEvent(in *pb.AddEventRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	startTime, _ := time.Parse(TimeFormat, in.StartTime)
	finishTime, _ := time.Parse(TimeFormat, in.FinishedAt)
	expectedFinishTime, _ := time.Parse(TimeFormat, in.ExpectedFinishTime)

	_, err := s.db.Exec(AddEventQuery, in.Tag, in.Name, in.Available, in.Capacity, in.Frontends, in.Status, in.Exercises, startTime, expectedFinishTime, finishTime, in.CreatedBy, in.OnlyVPN, in.SecretKey, in.DisabledExercises)

	if err != nil {
		return "", err
	}
	return "Event correctly added!", nil
}

func (s *store) AddTeam(in *pb.AddTeamRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	now := time.Now()

	var eventId int
	if err := s.db.QueryRow(QueryEventId, in.EventTag).Scan(&eventId); err != nil {
		return "", err
	}

	_, err := s.db.Exec(AddTeamQuery, in.Id, eventId, in.Email, in.Name, in.Password, now, now, "[]")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Team [ %s ]  correctly added to event [ %s ]", in.Name, in.EventTag), nil
}

func (s *store) UpdateExercises(req *pb.UpdateExerciseRequest) (string, error) {
	var eventId int
	challenges := strings.TrimSpace(req.Challenges)
	if err := s.db.QueryRow(QueryEventId, req.EventTag).Scan(&eventId); err != nil {
		return "", err
	}
	_, err := s.db.Exec(UpdateExercises, eventId, challenges)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("The challenges [ %s ] is updated for event [ %s ]", challenges, req.EventTag), nil
}

func (s *store) DelTeam(req *pb.DelTeamRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	var eventId int
	if err := s.db.QueryRow(QueryEventId, req.EvTag).Scan(&eventId); err != nil {
		return "", err
	}
	_, err := s.db.Exec(DelTeamQuery, req.TeamId, eventId)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Team [ %s ] is deleted from event tag [ %s ]", req.TeamId, req.EvTag), nil

}

func (s *store) GetEvents(in *pb.GetEventRequest) ([]model.Event, error) {
	var rows *sql.Rows
	var err error
	s.m.Lock()
	defer s.m.Unlock()

	switch in.Status {

	case int32(Running):
		// query only running events
		rows, err = s.db.Query(QueryEventsByStatus, int32(Running))
		if err != nil {
			return nil, fmt.Errorf("query running events err %v", err)
		}
	case int32(Suspended):
		// query only suspended events
		rows, err = s.db.Query(QueryEventsByStatus, int32(Suspended))
		if err != nil {
			return nil, fmt.Errorf("query suspended events err %v", err)
		}
	case int32(Booked):
		// query only booked events
		rows, err = s.db.Query(QueryEventsByStatus, int32(Booked))
		if err != nil {
			return nil, fmt.Errorf("query boooked events err %v", err)
		}
		// query only closed events
	case int32(Closed):
		rows, err = s.db.Query(QueryEventsByStatus, int32(Closed))
		if err != nil {
			return nil, fmt.Errorf("query closed events err %v", err)
		}
	default:
		// all events
		rows, err = s.db.Query(QueryEventTable)
		if err != nil {
			return nil, fmt.Errorf("query running events err %v", err)
		}
	}

	return parseEvents(rows)
}

func (s *store) GetEventByUser(in *pb.GetEventByUserReq) ([]model.Event, error) {
	s.m.Lock()
	defer s.m.Unlock()
	rows, err := s.db.Query(QueryEventByUser, in.Status, in.User)
	if err != nil {
		return nil, fmt.Errorf("query suspended events err %v", err)
	}
	return parseEvents(rows)
}

func (s *store) GetTeams(tag string) ([]model.Team, error) {
	s.m.Lock()
	defer s.m.Unlock()

	var eventId int
	if err := s.db.QueryRow(QueryEventId, tag).Scan(&eventId); err != nil && !strings.Contains(err.Error(), "no rows in result set") {
		return nil, err
	}

	rows, err := s.db.Query(QueryEventTeams, eventId)
	if err != nil {
		return nil, err
	}

	var teams []model.Team
	for rows.Next() {

		team := new(model.Team)
		err := rows.Scan(&team.Id, &team.Tag, &team.EventId, &team.Email, &team.Name, &team.Password, &team.CreatedAt,
			&team.LastAccess, &team.SolvedChallenges)
		if err != nil && !strings.Contains(err.Error(), handleNullConversionError) {
			return nil, err
		}
		teams = append(teams, *team)
	}
	return teams, nil
}

func (s *store) GetCostsInTime() (map[string]int32, error) {
	s.m.Lock()
	defer s.m.Unlock()
	m, err := calculateCost(s.db)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *store) UpdateTeamSolvedChallenge(in *pb.UpdateTeamSolvedChallengeRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	type Challenge struct {
		Tag         string `json:"tag"`
		CompletedAt string `json:"completed-at"`
	}

	var solvedChallenges []Challenge
	var solvedChallengesDB string

	if err := s.db.QueryRow(QuerySolvedChls, in.TeamId).Scan(&solvedChallengesDB); err != nil {
		return "", err
	}

	if err := json.Unmarshal([]byte(solvedChallengesDB), &solvedChallenges); err != nil {
		return "", err
	}

	for _, sc := range solvedChallenges {
		if sc.Tag == in.Tag {
			return "", errors.New("challenge already solved")
		}
	}

	solvedChallenges = append(solvedChallenges, Challenge{
		Tag:         in.Tag,
		CompletedAt: in.CompletedAt,
	})

	newSolvedChallengesDB, _ := json.Marshal(solvedChallenges)

	_, err := s.db.Exec(UpdateTeamSolvedChl, in.TeamId, string(newSolvedChallengesDB))
	if err != nil {
		return "", err
	}

	return OK, nil
}

func (s *store) UpdateTeamPassword(in *pb.UpdateTeamPassRequest) error {
	_, err := s.db.Exec(UpdateTeamPassword, in.EncryptedPass, in.TeamID, in.EventID)
	if err != nil {
		return err
	}
	return nil
}

func (s *store) GetEventID(in *pb.GetEventIDReq) (int32, error) {

	var eventId int32
	if err := s.db.QueryRow(QueryEventId, in.EventTag).Scan(&eventId); err != nil && !strings.Contains(err.Error(), "no rows in result set") {
		return -1, err
	}
	return eventId, nil
}

func (s *store) UpdateTeamLastAccess(in *pb.UpdateTeamLastAccessRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	_, err := s.db.Exec(UpdateEventLastaccessedDate, in.TeamId, in.AccessAt)
	if err != nil {
		return "", err
	}

	return OK, nil
}

func (s *store) UpdateCloseEvent(in *pb.UpdateEventRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	_, err := s.db.Exec(UpdateCloseEvent, in.OldTag, in.NewTag, in.FinishedAt)
	if err != nil {
		return "", err
	}

	return OK, nil
}

func (s *store) GetEventStatus(in *pb.GetEventStatusRequest) (int32, error) {
	s.m.Lock()
	defer s.m.Unlock()

	var status int32
	if err := s.db.QueryRow(QueryEventStatus, in.EventTag).Scan(&status); err != nil {
		return Error, err
	}

	return status, nil

}

func (s *store) SetEventStatus(in *pb.SetEventStatusRequest) (int32, error) {
	s.m.Lock()
	defer s.m.Unlock()
	_, err := s.db.Exec(UpdateEventStatus, in.EventTag, in.Status)
	if err != nil {
		return Error, err
	}

	return in.Status, nil
}

func (s *store) IsEventExists(in *pb.GetEventByTagReq) (bool, error) {
	var isEventExists bool
	r := s.db.QueryRow(QueryIsEventExist, in.EventTag, in.Status)
	if err := r.Scan(&isEventExists); err != nil {
		return false, err
	}
	return isEventExists, nil
}

func (s *store) DropEvent(in *pb.DropEventReq) (bool, error) {
	r, err := s.db.Exec(DropEvent, in.Tag, in.Status)
	if err != nil {
		return false, err
	}
	count, err := r.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("affected number of rows error %v", err)
	}
	if count > 0 {
		return true, nil
	}
	return false, fmt.Errorf("either no such an event or something else happened")

}

func (s *store) AddProfile(in *pb.AddProfileRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()
	var rows *sql.Rows
	var err error
	//Check if profile exists
	var exists bool
	rows, err = s.db.Query(CheckProfileExistsQuery, in.Name)
	if err != nil {
		return "", fmt.Errorf("query checking if profile exists err %v", err)
	}
	for rows.Next() {
		err := rows.Scan(&exists)
		if err != nil && !strings.Contains(err.Error(), handleNullConversionError) {
			return "", err
		}
	}
	if exists {
		return "", fmt.Errorf("Secret profile with name \"%s\" already exists", in.Name)
	}

	type Challenge struct {
		Tag  string `json:"tag"`
		Name string `json:"name"`
	}
	var challenges []Challenge
	for _, c := range in.Challenges {
		challenges = append(challenges, Challenge{
			Tag:  c.Tag,
			Name: c.Name,
		})
	}
	challengesDB, _ := json.Marshal(challenges)
	//log.Printf("Adding the following profile to DB: %s \n %s", in.Name, string(challengesDB))
	_, err = s.db.Exec(AddProfileQuery, in.Name, in.Secret, string(challengesDB))
	if err != nil {
		return "", err
	}

	return OK, nil
}

func (s *store) GetProfiles() ([]model.Profile, error) {
	var rows *sql.Rows
	var err error
	s.m.Lock()
	defer s.m.Unlock()

	rows, err = s.db.Query(GetProfilesQuery)
	if err != nil {
		return nil, fmt.Errorf("query getting profiles err %v", err)
	}
	var profiles []model.Profile
	for rows.Next() {
		profile := new(model.Profile)
		err := rows.Scan(&profile.Id, &profile.Name, &profile.Secret, &profile.Challenges)
		if err != nil && !strings.Contains(err.Error(), handleNullConversionError) {
			return nil, err
		}
		profiles = append(profiles, *profile)
	}
	return profiles, nil
}

func (s *store) UpdateProfile(in *pb.UpdateProfileRequest) (string, error) {
	s.m.Lock()
	defer s.m.Unlock()

	type Challenge struct {
		Tag  string `json:"tag"`
		Name string `json:"name"`
	}
	var challenges []Challenge
	for _, c := range in.Challenges {
		challenges = append(challenges, Challenge{
			Tag:  c.Tag,
			Name: c.Name,
		})
	}
	challengesDB, _ := json.Marshal(challenges)
	_, err := s.db.Exec(UpdateProfileQuery, in.Secret, string(challengesDB), in.Name)
	if err != nil {
		return "", err
	}

	return OK, nil
}

func (s *store) DeleteProfile(in *pb.DelProfileRequest)(string, error){
	s.m.Lock()
	defer s.m.Unlock()
	_, err := s.db.Exec(DeleteProfileQuery, in.Name)
	if err != nil {
		return "", err
	}

	return OK, nil
}

func parseEvents(rows *sql.Rows) ([]model.Event, error) {
	var events []model.Event
	for rows.Next() {
		event := new(model.Event)
		err := rows.Scan(&event.Id, &event.Tag, &event.Name, &event.Available, &event.Capacity, &event.Status, &event.Frontends,
			&event.Exercises, &event.StartedAt, &event.ExpectedFinishTime, &event.FinishedAt, &event.CreatedBy, &event.OnlyVPN, &event.SecretKey, &event.DisabledExercises)
		if err != nil && !strings.Contains(err.Error(), handleNullConversionError) {
			return nil, err
		}
		events = append(events, *event)
	}
	return events, nil
}

//
//func (s *store) GetEventsByStatus () ([]model.Event, error) {
//	s.m.Lock()
//	defer s.m.Unlock()
//
//	rows, err := s.db.Exec(QueryEventsByStatus,)
//	if err != nil {
//		return nil, err
//	}
//	var events []model.Event
//	for rows.Next() {
//		event := new(model.Event)
//		err := rows.Scan(&event.Id, &event.Tag, &event.Name, &event.Available, &event.Capacity, &event.Status, &event.Frontends,
//			&event.Exercises, &event.StartedAt, &event.ExpectedFinishTime, &event.FinishedAt)
//		if err != nil && !strings.Contains(err.Error(), handleNullConversionError) {
//			return nil, err
//		}
//		events = append(events, *event)
//	}
//
//	return events, nil
//}
