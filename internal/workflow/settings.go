package workflow

const (
	MaxTriesDefault = 3
	TimeoutDefault  = "5s"
)

type Settings struct {
	MaxTries int    `yaml:"max-tries"`
	Timeout  string `yaml:"timeout"`
}

func NewSettings(maxTries int, timeout string) *Settings {
	if maxTries < 0 {
		maxTries = MaxTriesDefault
	}
	if timeout == "" {
		timeout = TimeoutDefault
	}
	return &Settings{MaxTries: maxTries, Timeout: timeout}
}
