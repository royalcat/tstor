package config

// Config is the main config object
type Config struct {
	WebUi         WebUi         `koanf:"webUi"`
	TorrentClient TorrentClient `koanf:"torrent"`
	Mounts        Mounts        `koanf:"mounts"`
	Log           Log           `koanf:"log"`

	DataFolder string `koanf:"dataFolder"`
}

type WebUi struct {
	Port int    `koanf:"port"`
	IP   string `koanf:"ip"`
}

type Log struct {
	Debug      bool   `koanf:"debug"`
	MaxBackups int    `koanf:"max_backups"`
	MaxSize    int    `koanf:"max_size"`
	MaxAge     int    `koanf:"max_age"`
	Path       string `koanf:"path"`
}

type TorrentClient struct {
	ReadTimeout int `koanf:"read_timeout,omitempty"`
	AddTimeout  int `koanf:"add_timeout,omitempty"`

	DHTNodes    []string `koanf:"dhtnodes,omitempty"`
	DisableIPv6 bool     `koanf:"disable_ipv6,omitempty"`

	DataFolder     string `koanf:"data_folder,omitempty"`
	MetadataFolder string `koanf:"metadata_folder,omitempty"`

	// GlobalCacheSize int64 `koanf:"global_cache_size,omitempty"`

	Routes  []Route  `koanf:"routes"`
	Servers []Server `koanf:"servers"`
}

type Route struct {
	Name          string    `koanf:"name"`
	Torrents      []Torrent `koanf:"torrents"`
	TorrentFolder string    `koanf:"torrent_folder"`
}

type Server struct {
	Name       string   `koanf:"name"`
	Path       string   `koanf:"path"`
	Trackers   []string `koanf:"trackers"`
	TrackerURL string   `koanf:"tracker_url"`
}

type Torrent struct {
	MagnetURI   string `koanf:"magnet_uri,omitempty"`
	TorrentPath string `koanf:"torrent_path,omitempty"`
}

type Mounts struct {
	WebDAV WebDAV `koanf:"webdav"`
	HttpFs HttpFs `koanf:"httpfs"`
	Fuse   Fuse   `koanf:"fuse"`
	NFS    NFS    `koanf:"nfs"`
}

type NFS struct {
	Enabled bool `koanf:"enabled"`
	Port    int  `koanf:"port"`
}

type HttpFs struct {
	Enabled bool `koanf:"enabled"`
	Port    int  `koanf:"port"`
}

type WebDAV struct {
	Enabled bool   `koanf:"enabled"`
	Port    int    `koanf:"port"`
	User    string `koanf:"user"`
	Pass    string `koanf:"pass"`
}

type Fuse struct {
	Enabled    bool   `koanf:"enabled"`
	AllowOther bool   `koanf:"allow_other,omitempty"`
	Path       string `koanf:"path"`
}
