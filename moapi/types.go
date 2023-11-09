package moapi

// Mirror is the mirror info
type Mirror struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Area    string `json:"area"`    // area code
	Latency int    `json:"latency"` // ms
	Status  string `json:"status"`  // on, slow, off,
}

// App is the app info
type App struct {
	Name        string   `json:"name"`
	UpdatedAt   int64    `json:"updated_at"`
	CreatedAt   int64    `json:"created_at"`
	Country     string   `json:"country"`
	Creator     string   `json:"creator"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Short       string   `json:"short"`
	Icon        string   `json:"icon"`
	Homepage    string   `json:"homepage"`
	Images      []string `json:"images,omitempty"`
	Videos      []string `json:"videos,omitempty"`
	Stat        AppStat  `json:"stat,omitempty"`
	Languages   []string `json:"languages"`
}

// AppStat is the app stat info
type AppStat struct {
	Downloads int `json:"downloads"`
	Stars     int `json:"stars"`
}
