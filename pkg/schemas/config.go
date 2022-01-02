package schemas

type Config struct {
	Server  *Server  `json:"server"`
	Cluster *Cluster `json:"cluster"`
}

type Server struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type Cluster struct {
	Address string   `json:"address"`
	Port    int      `json:"port"`
	Routes  []*Route `json:"routes"`
}

type Route struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}
