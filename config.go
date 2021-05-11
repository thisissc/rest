package rest

var (
	globalAppConfig AppConfig
)

type AppConfig struct {
	Debug          bool
	DebugUid       string
	DebugRoleId    string
	JWTTokenPrefix string
	JWTTokenSecret string
	AESSecret      string
	HMACSecret     string
	AllowedOrigin  string
	ReadTimeout    int64
	WriteTimeout   int64
}

func (c *AppConfig) Init() error {
	globalAppConfig = *c
	return nil
}
