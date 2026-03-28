package main

import (
	"fmt"
	"strings"
	"time"
)

type moduleState struct {
	PHPVersions        []PHPVersionInfo
	PHPIni             map[string]string
	DockerContainers   []DockerContainer
	DockerImages       []DockerImage
	DockerTemplates    []DockerAppTemplate
	DockerInstalled    []DockerInstalledApp
	DockerPackages     []DockerPackage
	Mailboxes          []Mailbox
	MailForwards       []MailForward
	MailCatchAll       map[string]MailCatchAll
	MailRouting        []MailRoutingRule
	MailDKIM           map[string]DKIMRecord
	DNSZones           []DNSZone
	DNSRecords         map[string][]DNSRecord
	DefaultNameservers DefaultNameservers
	FTPUsers           []TransferAccount
	SFTPUsers          []TransferAccount
	CronJobs           []CronJob
	OLSConfig          OLSTuningConfig
	MinIOBuckets       []string
	MinIOCredentials   map[string]MinIOCredential
	FederatedMode      FederatedMode
	FederatedNodes     []FederatedNode
	RuntimeApps        []RuntimeApp
	WordPressSites     []WordPressSite
	WordPressPlugins   map[string][]WordPressPlugin
	WordPressThemes    map[string][]WordPressTheme
	WordPressBackups   map[string][]WordPressBackup
	WordPressStaging   map[string][]WordPressStaging
	BackupDestinations []BackupDestination
	BackupSchedules    []BackupSchedule
	BackupSnapshots    []BackupSnapshot
	DBBackups          []DBBackupRecord
	ActivityLogs       []ActivityLogEntry
	ResellerQuotas     []ResellerQuota
	WhiteLabels        []WhiteLabel
	ACLPolicies        []ACLPolicy
	ACLAssignments     []ACLAssignment
	VirtualFiles       map[string]*virtualFile
	UploadedArchives   map[string]string
	MigrationAnalyses  map[string]MigrationAnalysis
	MigrationJobs      []MigrationJob
	SSLBindings        SSLBindings
	SSLCertificates    map[string]SSLCertificateDetail
	CloudflareZones    []CloudflareZone
	CloudflareDNS      map[string][]CloudflareDNSRecord
	CloudflareSettings map[string]cloudflareZoneConfig
	WebmailTokens      map[string]WebmailToken
}

type PHPVersionInfo struct {
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
	EOL       bool   `json:"eol"`
}

type DockerContainer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	Ports   string `json:"ports"`
	Created string `json:"created"`
}

type DockerImage struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
	Created    string `json:"created"`
}

type DockerAppTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Icon        string `json:"icon"`
	Category    string `json:"category"`
}

type DockerInstalledApp struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	Ports   string `json:"ports"`
	Package string `json:"package"`
}

type DockerPackage struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	MemoryLimit   string `json:"memory_limit"`
	CPULimit      string `json:"cpu_limit"`
	MaxContainers int    `json:"max_containers"`
}

type Mailbox struct {
	Address string `json:"address"`
	Domain  string `json:"domain"`
	User    string `json:"username"`
	Owner   string `json:"owner,omitempty"`
	QuotaMB int    `json:"quota_mb"`
	UsedMB  int    `json:"used_mb"`
}

type MailForward struct {
	Domain string `json:"domain"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type MailCatchAll struct {
	Domain  string `json:"domain"`
	Enabled bool   `json:"enabled"`
	Target  string `json:"target"`
}

type MailRoutingRule struct {
	ID       string `json:"id"`
	Domain   string `json:"domain"`
	Pattern  string `json:"pattern"`
	Target   string `json:"target"`
	Priority int    `json:"priority"`
}

type DKIMRecord struct {
	Domain    string `json:"domain"`
	Selector  string `json:"selector"`
	PublicKey string `json:"public_key"`
}

type DNSZone struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Records       int    `json:"records"`
	DNSSECEnabled bool   `json:"dnssec_enabled"`
}

type DNSRecord struct {
	RecordType string `json:"record_type"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	TTL        int    `json:"ttl"`
}

type DefaultNameservers struct {
	NS1 string `json:"ns1"`
	NS2 string `json:"ns2"`
}

type TransferAccount struct {
	Username  string `json:"username"`
	Domain    string `json:"domain,omitempty"`
	HomeDir   string `json:"home_dir"`
	CreatedAt int64  `json:"created_at"`
}

type CronJob struct {
	ID       string `json:"id"`
	User     string `json:"user"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
}

type OLSTuningConfig struct {
	MaxConnections       int  `json:"max_connections"`
	MaxSSLConnections    int  `json:"max_ssl_connections"`
	ConnTimeoutSecs      int  `json:"conn_timeout_secs"`
	KeepAliveTimeoutSecs int  `json:"keep_alive_timeout_secs"`
	MaxKeepAliveRequests int  `json:"max_keep_alive_requests"`
	GzipCompression      bool `json:"gzip_compression"`
	StaticCacheEnabled   bool `json:"static_cache_enabled"`
	StaticCacheMaxAgeSec int  `json:"static_cache_max_age_secs"`
}

type MinIOCredential struct {
	User      string `json:"user"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

type FederatedMode struct {
	Mode    string `json:"mode"`
	Primary bool   `json:"primary"`
}

type FederatedNode struct {
	NodeName  string `json:"node_name"`
	IPAddress string `json:"ip_address"`
	PubKey    string `json:"pub_key"`
}

type RuntimeApp struct {
	Runtime string `json:"runtime"`
	Dir     string `json:"dir"`
	AppName string `json:"app_name"`
	Status  string `json:"status"`
}

type WordPressSite struct {
	Domain           string `json:"domain"`
	Title            string `json:"title"`
	SiteURL          string `json:"site_url"`
	Docroot          string `json:"docroot"`
	Status           string `json:"status"`
	WordPressVersion string `json:"wordpress_version"`
	PHPVersion       string `json:"php_version"`
	Owner            string `json:"owner"`
	ActivePlugins    int    `json:"active_plugins"`
	TotalPlugins     int    `json:"total_plugins"`
	ActiveTheme      string `json:"active_theme"`
	DBEngine         string `json:"db_engine"`
	DBName           string `json:"db_name"`
	DBUser           string `json:"db_user"`
	DBHost           string `json:"db_host"`
	AdminEmail       string `json:"admin_email"`
}

type WordPressPlugin struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Update  string `json:"update"`
}

type WordPressTheme struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Update  string `json:"update"`
}

type WordPressBackup struct {
	ID         string `json:"id"`
	Domain     string `json:"domain"`
	FileName   string `json:"file_name"`
	BackupType string `json:"backup_type"`
	SizeBytes  int64  `json:"size_bytes"`
	CreatedAt  int64  `json:"created_at"`
	Path       string `json:"-"`
}

type WordPressStaging struct {
	ID            string `json:"id"`
	SourceDomain  string `json:"source_domain"`
	StagingDomain string `json:"staging_domain"`
	Owner         string `json:"owner"`
	CreatedAt     int64  `json:"created_at"`
	Status        string `json:"status"`
}

type BackupDestination struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	RemoteRepo string `json:"remote_repo"`
	Password   string `json:"password,omitempty"`
	Enabled    bool   `json:"enabled"`
}

type BackupSchedule struct {
	ID            string `json:"id"`
	Domain        string `json:"domain"`
	DestinationID string `json:"destination_id"`
	BackupPath    string `json:"backup_path"`
	Cron          string `json:"cron"`
	Incremental   bool   `json:"incremental"`
	Enabled       bool   `json:"enabled"`
}

type BackupSnapshot struct {
	ID         string   `json:"id"`
	ShortID    string   `json:"short_id"`
	Time       string   `json:"time"`
	Hostname   string   `json:"hostname"`
	Tags       []string `json:"tags"`
	Domain     string   `json:"domain"`
	BackupPath string   `json:"backup_path"`
}

type DBBackupRecord struct {
	ID        string `json:"id"`
	DBName    string `json:"db_name,omitempty"`
	Filename  string `json:"filename"`
	Engine    string `json:"engine"`
	Size      string `json:"size"`
	CreatedAt int64  `json:"created_at"`
	Path      string `json:"-"`
}

type ActivityLogEntry struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	User      string `json:"user"`
	Action    string `json:"action"`
	Detail    string `json:"detail"`
	IP        string `json:"ip"`
}

type ResellerQuota struct {
	Username    string `json:"username"`
	Plan        string `json:"plan"`
	DiskGB      int    `json:"disk_gb"`
	BandwidthGB int    `json:"bandwidth_gb"`
	MaxSites    int    `json:"max_sites"`
	UpdatedAt   int64  `json:"updated_at"`
}

type WhiteLabel struct {
	Username  string `json:"username"`
	PanelName string `json:"panel_name"`
	LogoURL   string `json:"logo_url"`
	UpdatedAt int64  `json:"updated_at"`
}

type ACLPolicy struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	UpdatedAt   int64    `json:"updated_at"`
}

type ACLAssignment struct {
	Username  string `json:"username"`
	PolicyID  string `json:"policy_id"`
	UpdatedAt int64  `json:"updated_at"`
}

type virtualFile struct {
	Path        string
	IsDir       bool
	Content     string
	Permissions string
	ModifiedAt  time.Time
}

type virtualFileEntry struct {
	Name        string `json:"name"`
	IsDir       bool   `json:"is_dir"`
	Size        int64  `json:"size"`
	Permissions string `json:"permissions"`
	Modified    int64  `json:"modified"`
}

type MigrationStats struct {
	FileCount     int `json:"file_count"`
	DatabaseCount int `json:"database_count"`
	EmailCount    int `json:"email_count"`
}

type MigrationAnalysis struct {
	SourceType      string         `json:"source_type"`
	Stats           MigrationStats `json:"stats"`
	MySQLDumps      []string       `json:"mysql_dumps"`
	EmailAccounts   []string       `json:"email_accounts"`
	VhostCandidates []string       `json:"vhost_candidates"`
	Warnings        []string       `json:"warnings"`
}

type MigrationSummary struct {
	ConvertedDBFiles []string `json:"converted_db_files"`
	EmailPlanFile    string   `json:"email_plan_file"`
	VhostPlanFile    string   `json:"vhost_plan_file"`
	SystemApply      bool     `json:"system_apply_enabled"`
}

type MigrationJob struct {
	ID        string           `json:"id"`
	Status    string           `json:"status"`
	Progress  int              `json:"progress"`
	Logs      []string         `json:"logs"`
	Summary   MigrationSummary `json:"summary"`
	PollCount int              `json:"-"`
}

type SSLBindings struct {
	HostnameSSLDomain string `json:"hostname_ssl_domain"`
	MailSSLDomain     string `json:"mail_ssl_domain"`
	UpdatedAt         int64  `json:"updated_at"`
}

type SSLCertificateDetail struct {
	Domain        string `json:"domain"`
	Status        string `json:"status"`
	Issuer        string `json:"issuer"`
	ExpiryDate    string `json:"expiry_date"`
	DaysRemaining int    `json:"days_remaining"`
	Wildcard      bool   `json:"wildcard"`
}

type CloudflareZone struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Plan        string   `json:"plan"`
	NameServers []string `json:"name_servers"`
}

type CloudflareDNSRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type cloudflareZoneConfig struct {
	SSLMode       string
	SecurityLevel string
	DevMode       bool
	AlwaysHTTPS   bool
}

type WebmailToken struct {
	Token     string
	Address   string
	ExpiresAt time.Time
}

func seedModuleState() moduleState {
	return moduleState{
		PHPIni:             map[string]string{},
		MailCatchAll:       map[string]MailCatchAll{},
		MailDKIM:           map[string]DKIMRecord{},
		DNSRecords:         map[string][]DNSRecord{},
		MinIOCredentials:   map[string]MinIOCredential{},
		WordPressPlugins:   map[string][]WordPressPlugin{},
		WordPressThemes:    map[string][]WordPressTheme{},
		WordPressBackups:   map[string][]WordPressBackup{},
		WordPressStaging:   map[string][]WordPressStaging{},
		VirtualFiles:       map[string]*virtualFile{},
		UploadedArchives:   map[string]string{},
		MigrationAnalyses:  map[string]MigrationAnalysis{},
		SSLCertificates:    map[string]SSLCertificateDetail{},
		CloudflareDNS:      map[string][]CloudflareDNSRecord{},
		CloudflareSettings: map[string]cloudflareZoneConfig{},
		WebmailTokens:      map[string]WebmailToken{},
	}
}

func (s *service) bootstrapModules() {
	if s.modules.PHPIni == nil {
		s.modules.PHPIni = map[string]string{}
	}
	if s.modules.MailCatchAll == nil {
		s.modules.MailCatchAll = map[string]MailCatchAll{}
	}
	if s.modules.MailDKIM == nil {
		s.modules.MailDKIM = map[string]DKIMRecord{}
	}
	if s.modules.DNSRecords == nil {
		s.modules.DNSRecords = map[string][]DNSRecord{}
	}
	if s.modules.MinIOCredentials == nil {
		s.modules.MinIOCredentials = map[string]MinIOCredential{}
	}
	if s.modules.WordPressPlugins == nil {
		s.modules.WordPressPlugins = map[string][]WordPressPlugin{}
	}
	if s.modules.WordPressThemes == nil {
		s.modules.WordPressThemes = map[string][]WordPressTheme{}
	}
	if s.modules.WordPressBackups == nil {
		s.modules.WordPressBackups = map[string][]WordPressBackup{}
	}
	if s.modules.WordPressStaging == nil {
		s.modules.WordPressStaging = map[string][]WordPressStaging{}
	}
	if s.modules.VirtualFiles == nil {
		s.modules.VirtualFiles = map[string]*virtualFile{}
	}
	if s.modules.UploadedArchives == nil {
		s.modules.UploadedArchives = map[string]string{}
	}
	if s.modules.MigrationAnalyses == nil {
		s.modules.MigrationAnalyses = map[string]MigrationAnalysis{}
	}
	if s.modules.SSLCertificates == nil {
		s.modules.SSLCertificates = map[string]SSLCertificateDetail{}
	}
	if s.modules.CloudflareDNS == nil {
		s.modules.CloudflareDNS = map[string][]CloudflareDNSRecord{}
	}
	if s.modules.CloudflareSettings == nil {
		s.modules.CloudflareSettings = map[string]cloudflareZoneConfig{}
	}
	if s.modules.WebmailTokens == nil {
		s.modules.WebmailTokens = map[string]WebmailToken{}
	}

	if len(s.modules.PHPVersions) == 0 {
		s.modules.PHPVersions = []PHPVersionInfo{
			{Version: "8.4", Installed: false, EOL: false},
			{Version: "8.3", Installed: true, EOL: false},
			{Version: "8.2", Installed: false, EOL: false},
			{Version: "8.1", Installed: false, EOL: false},
			{Version: "8.0", Installed: false, EOL: true},
			{Version: "7.4", Installed: false, EOL: true},
		}
	}
	for _, version := range []string{"8.3", "8.2"} {
		if _, ok := s.modules.PHPIni[version]; !ok {
			s.modules.PHPIni[version] = defaultPHPIni(version)
		}
	}
	if len(s.modules.DockerTemplates) == 0 {
		s.modules.DockerTemplates = []DockerAppTemplate{
			{ID: "redis", Name: "Redis", Description: "Low-latency cache for panel sites.", Image: "redis:7-alpine", Icon: "R", Category: "cache"},
			{ID: "meilisearch", Name: "Meilisearch", Description: "Fast search node for application workloads.", Image: "getmeili/meilisearch:v1.13", Icon: "M", Category: "search"},
			{ID: "n8n", Name: "n8n", Description: "Workflow automation service for integrations.", Image: "n8nio/n8n:latest", Icon: "N", Category: "automation"},
		}
	}
	if len(s.modules.DockerPackages) == 0 {
		s.modules.DockerPackages = []DockerPackage{
			{ID: "starter", Name: "Starter", MemoryLimit: "512 MB", CPULimit: "0.5", MaxContainers: 3},
			{ID: "pro", Name: "Pro", MemoryLimit: "2 GB", CPULimit: "2.0", MaxContainers: 12},
		}
	}
	if s.modules.FederatedMode.Mode == "" {
		s.modules.FederatedMode = FederatedMode{Mode: "standalone", Primary: true}
	}
	if s.modules.OLSConfig.MaxConnections == 0 {
		s.modules.OLSConfig = OLSTuningConfig{
			MaxConnections:       10000,
			MaxSSLConnections:    10000,
			ConnTimeoutSecs:      300,
			KeepAliveTimeoutSecs: 5,
			MaxKeepAliveRequests: 10000,
			GzipCompression:      true,
			StaticCacheEnabled:   true,
			StaticCacheMaxAgeSec: 3600,
		}
	}
	if len(s.modules.ACLPolicies) == 0 {
		s.modules.ACLPolicies = []ACLPolicy{
			{ID: "acl-sites", Name: "Site Manager", Description: "Website, mail and backup management.", Permissions: []string{"websites:view", "mail:manage", "backup:run"}, UpdatedAt: time.Now().UTC().Unix()},
			{ID: "acl-devops", Name: "DevOps", Description: "Runtime and deployment operations.", Permissions: []string{"apps:manage", "docker:view", "logs:view"}, UpdatedAt: time.Now().UTC().Unix()},
		}
	}

	s.ensureVirtualDirLocked("/")
	s.ensureVirtualDirLocked("/home")
	s.ensureVirtualDirLocked("/var")
	s.ensureVirtualDirLocked("/var/log")
	s.ensureVirtualDirLocked("/var/log/aurapanel")

	for _, site := range s.state.Websites {
		s.ensureDefaultSiteArtifactsLocked(site.Domain)
		if site.MailDomain {
			s.ensureMailArtifactsLocked(site)
		}
	}
}

func buildWordPressSite(domain, owner, email, phpVersion string) WordPressSite {
	return WordPressSite{
		Domain:           domain,
		Title:            "WordPress Site",
		SiteURL:          fmt.Sprintf("https://%s", domain),
		Docroot:          domainDocroot(domain),
		Status:           "active",
		WordPressVersion: "6.8",
		PHPVersion:       phpVersion,
		Owner:            owner,
		ActivePlugins:    2,
		TotalPlugins:     3,
		ActiveTheme:      "Astra",
		DBEngine:         "mariadb",
		DBName:           "example_app",
		DBUser:           "example_user",
		DBHost:           "localhost",
		AdminEmail:       email,
	}
}

func defaultPHPIni(version string) string {
	return fmt.Sprintf("; AuraPanel managed php.ini for PHP %s\nmemory_limit = 512M\nupload_max_filesize = 128M\npost_max_size = 128M\nmax_execution_time = 120\ndate.timezone = UTC\n", version)
}

func domainDocroot(domain string) string {
	domain = normalizeDomain(domain)
	if domain == "" {
		return "/home/public_html"
	}
	return fmt.Sprintf("/home/%s/public_html", domain)
}

func (s *service) appendActivityLocked(user, action, detail, ip string) {
	entry := ActivityLogEntry{
		ID:        generateSecret(6),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		User:      firstNonEmpty(strings.TrimSpace(user), "system"),
		Action:    firstNonEmpty(strings.TrimSpace(action), "update"),
		Detail:    detail,
		IP:        firstNonEmpty(strings.TrimSpace(ip), "127.0.0.1"),
	}
	s.modules.ActivityLogs = append([]ActivityLogEntry{entry}, s.modules.ActivityLogs...)
	if len(s.modules.ActivityLogs) > 250 {
		s.modules.ActivityLogs = s.modules.ActivityLogs[:250]
	}
}

func (s *service) recordIssuedCertificateLocked(domain, issuer string, wildcard bool) {
	key := normalizeDomain(domain)
	if key == "" {
		return
	}
	s.modules.SSLCertificates[key] = SSLCertificateDetail{
		Domain:        key,
		Status:        "issued",
		Issuer:        firstNonEmpty(strings.TrimSpace(issuer), "Let's Encrypt"),
		ExpiryDate:    time.Now().UTC().Add(90 * 24 * time.Hour).Format("2006-01-02"),
		DaysRemaining: 90,
		Wildcard:      wildcard,
	}
	s.modules.SSLBindings.UpdatedAt = time.Now().UTC().Unix()
}
