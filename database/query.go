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
		"onlyVPN boolean);"

	CreateTeamsTable = "CREATE TABLE IF NOT EXISTS Team(" +
		"id serial primary key, " +
		"tag varchar (50), " +
		"event_id integer, " +
		"email varchar (50), " +
		"name varchar (50), " +
		"password varchar (250), " +
		"created_at timestamp, " +
		"last_access timestamp, " +
		"step integer, " +
		"skipped_challenges text, " +
		"solved_challenges text);"

	AddTeamQuery = "INSERT INTO team (tag, event_id, email, name, password, created_at, last_access, step, skipped_challenges, solved_challenges)" +
		"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"

	AddEventQuery = "INSERT INTO event (tag, name, available, capacity, frontends, status, exercises, started_at, finish_expected, finished_at, createdby, onlyvpn)" +
		"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)"

	UpdateCloseEvent  = "UPDATE event SET tag = $2, finished_at = $3 WHERE tag = $1"
	UpdateEventStatus = "UPDATE event SET status = $2 WHERE tag = $1 "

	UpdateEventLastaccessedDate = "UPDATE team SET last_access = $2 WHERE tag = $1"
	UpdateTeamSolvedChl         = "UPDATE team SET solved_challenges = $2 WHERE tag = $1"
	UpdateTeamSkippedChl        = "UPDATE team SET skipped_challenges = $2 WHERE tag = $1"
	UpdateTeamStep              = "UPDATE team SET step = $2 WHERE tag = $1"

	QuerySolvedChls = "SELECT solved_challenges FROM team WHERE tag=$1"
	QueryEventTable = "SELECT * FROM event"

	// finished_at '0001-01-01 00:00:00 means event does not finished yet '
	QueryEventId    = "SELECT id FROM event WHERE tag=$1 and finished_at = date('0001-01-01 00:00:00'); "
	QueryEventTeams = "SELECT * FROM team WHERE event_id=$1"
	QueryTeamCount  = "SELECT count(team.id) FROM team WHERE team.event_id=$1"

	QueryEventStatus    = "SELECT status FROM event WHERE tag=$1"
	QueryEventsByStatus = "SELECT * FROM event WHERE status=$1"
	QueryEventByUser    = "SELECT * FROM event WHERE status!=$1 and createdby=$2"
	QueryIsEventExist   = "SELECT EXISTS (select tag from event where tag=$1 and status!=$2)"
	// finished_at '0001-01-01 00:00:00 means event does not finished yet '
	EarliestDate = "SELECT started_at FROM event WHERE started_at=(SELECT MIN(started_at) FROM event) and finished_at = date('0001-01-01 00:00:00');"
	LatestDate   = "SELECT finish_expected FROM event WHERE finish_expected =(SELECT max(finish_expected) FROM event) and finished_at = date('0001-01-01 00:00:00');"
	// DropEvent is used in dropping booked events
	DropEvent = "DELETE FROM event WHERE tag=$1 and status=$2"
)
