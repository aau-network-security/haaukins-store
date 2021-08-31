package database

var (
	CreateEventTable = "CREATE TABLE IF NOT EXISTS Event(" +
		"id serial primary key, " +
		"tag varchar (50), " +
		"name varchar (150), " +
		"available integer, " +
		"capacity integer, " +
		"status integer, " +
		"frontends text, " +
		"exercises text, " +
		"started_at timestamp, " +
		"finish_expected timestamp, " +
		"finished_at timestamp," +
		"createdBy text," +
		"onlyVPN integer," +
		"secretKey text," +
		"disabledExercises text);"

	CreateTeamsTable = "CREATE TABLE IF NOT EXISTS Team(" +
		"id serial primary key, " +
		"tag varchar (50), " +
		"event_id integer, " +
		"email varchar (50), " +
		"name varchar (50), " +
		"password varchar (250), " +
		"created_at timestamp, " +
		"last_access timestamp, " +
		"solved_challenges text);"

	CreateProfilesTable = "CREATE TABLE IF NOT EXISTS Profiles("+
		"id serial primary key, "+
		"name varchar (50), "+
		"secret boolean, "+
		"challenges text);"

	AddProfileQuery = "INSERT INTO profiles (name, secret, challenges) VALUES ($1, $2, $3)"

	GetProfilesQuery = "SELECT * FROM profiles ORDER BY id asc"

	UpdateProfileQuery = "UPDATE profiles SET secret = $1, challenges = $2 WHERE name = $3"

	DeleteProfileQuery = "DELETE FROM profiles WHERE name = $1"

	AddTeamQuery = "INSERT INTO team (tag, event_id, email, name, password, created_at, last_access, solved_challenges)" +
		"VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"

	DelTeamQuery = "DELETE FROM team WHERE tag=$1 and event_id = $2;"

	AddEventQuery = "INSERT INTO event (tag, name, available, capacity, frontends, status, exercises, started_at, finish_expected, finished_at, createdby, onlyvpn,secretKey, disabledExercises)" +
		"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,$11,$12,$13,$14)"

	UpdateCloseEvent            = "UPDATE event SET tag = $2, finished_at = $3 WHERE tag = $1"
	UpdateEventStatus           = "UPDATE event SET status = $2 WHERE tag = $1 "
	UpdateExercises             = "UPDATE event SET exercises = (SELECT (SELECT exercises FROM event WHERE id = $1) || $2) WHERE id=$1"
	UpdateEventLastaccessedDate = "UPDATE team SET last_access = $2 WHERE tag = $1"
	UpdateTeamSolvedChl         = "UPDATE team SET solved_challenges = $2 WHERE tag = $1"
	UpdateTeamPassword          = "UPDATE team SET password = $1 WHERE tag = $2 and event_id = $3"

	QuerySolvedChls = "SELECT solved_challenges FROM team WHERE tag=$1"
	QueryEventTable = "SELECT * FROM event"

	// finished_at '0001-01-01 00:00:00 means event does not finished yet '
	QueryEventId    = "SELECT id FROM event WHERE tag=$1 and finished_at = date('0001-01-01 00:00:00'); "
	QueryEventTeams = "SELECT * FROM team WHERE event_id=$1"
	QueryTeamCount  = "SELECT count(team.id) FROM team WHERE team.event_id=$1"

	QueryEventStatus           = "SELECT status FROM event WHERE tag=$1"
	QueryAllEventsExceptClosed = "SELECT * FROM event WHERE status!=3"
	QueryEventsByStatus        = "SELECT * FROM event WHERE status=$1"
	QueryEventByUser           = "SELECT * FROM event WHERE status!=$1 and createdby=$2"
	QueryIsEventExist          = "SELECT EXISTS (select tag from event where tag=$1 and status!=$2)"
	// finished_at '0001-01-01 00:00:00 means event does not finished yet '
	EarliestDate = "SELECT started_at FROM event WHERE started_at=(SELECT MIN(started_at) FROM event) and finished_at = date('0001-01-01 00:00:00');"
	LatestDate   = "SELECT finish_expected FROM event WHERE finish_expected =(SELECT max(finish_expected) FROM event) and finished_at = date('0001-01-01 00:00:00');"
	// DropEvent is used in dropping booked events
	DropEvent = "DELETE FROM event WHERE tag=$1 and status=$2"
)
