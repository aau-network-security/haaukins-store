package model

type Event struct {
	Id                 uint //DB Primary Key
	Tag                string
	Name               string
	Frontends          string
	Exercises          string
	Available          uint
	Capacity           uint
	Status             int32
	StartedAt          string
	ExpectedFinishTime string
	FinishedAt         string
	CreatedBy          string
	OnlyVPN            bool
}

type Team struct {
	Id                uint //DB Primary key
	Tag               string
	EventId           uint //DB Primary key of the event
	Email             string
	Name              string
	Password          string
	CreatedAt         string
	LastAccess        string
	SolvedChallenges  string
	Step              uint
	SkippedChallenges string
}

type Config struct {
	Host      string `yaml:"host"`
	AuthKey   string `yaml:"auth-key"`
	SigninKey string `yaml:"signin-key"`
	DB        struct {
		Host string `yaml:"host"`
		User string `yaml:"user"`
		Pass string `yaml:"pass"`
		Name string `yaml:"db_name"`
		Port uint   `yaml:"db_port"`
	} `yaml:"db"`
	TLS struct {
		Enabled  bool   `yaml:"enabled"`
		CertFile string `yaml:"certfile"`
		CertKey  string `yaml:"certkey"`
		CAFile   string `yaml:"cafile"`
	} `tls:"tls,omitempty"`
}
