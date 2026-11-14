package config

// TaiConfig controls Yao's Tai control channel and transport server.
type TaiConfig struct {
	Control TaiControlConfig `json:"control,omitempty"`
	Server  TaiServerConfig  `json:"server,omitempty"`
}

// TaiControlConfig holds the system-level SSH key used by Yao to control
// cloud hosts. The private key is never returned by APIs or stored in models.
type TaiControlConfig struct {
	KeyID          string `json:"key_id,omitempty" env:"YAO_TAI_CONTROL_KEY_ID"`
	PrivateKey     string `json:"private_key,omitempty" env:"YAO_TAI_CONTROL_PRIVATE_KEY"`
	PrivateKeyFile string `json:"private_key_file,omitempty" env:"YAO_TAI_CONTROL_PRIVATE_KEY_FILE"`
	PublicKey      string `json:"public_key,omitempty" env:"YAO_TAI_CONTROL_PUBLIC_KEY"`
}

// TaiServerConfig controls the embedded tai-golang transport server.
type TaiServerConfig struct {
	Enabled       string `json:"enabled,omitempty" env:"YAO_TAI_SERVER" envDefault:"false"`
	TransportPort int    `json:"transport_port,omitempty" env:"YAO_TAI_TRANSPORT_PORT" envDefault:"6099"`
}
