package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func powerDNSDatabasePath() string {
	return envOr("AURAPANEL_POWERDNS_DB_PATH", "/var/lib/powerdns/aurapanel.sqlite3")
}

func powerDNSAvailable() bool {
	return commandExists("sqlite3") && fileExists(powerDNSDatabasePath())
}

func syncPowerDNSZone(domain string, records []DNSRecord, ns1 string, ns2 string) error {
	normalizedDomain := normalizeDomain(domain)
	if normalizedDomain == "" || !powerDNSAvailable() {
		return nil
	}

	dbPath := powerDNSDatabasePath()
	serial := time.Now().UTC().Format("2006010201")
	ns1 = normalizeDomain(firstNonEmpty(ns1, "ns1."+normalizedDomain))
	ns2 = normalizeDomain(firstNonEmpty(ns2, "ns2."+normalizedDomain))
	soaContent := fmt.Sprintf("%s hostmaster.%s %s 3600 900 1209600 300", ns1, normalizedDomain, serial)

	statements := []string{
		"BEGIN IMMEDIATE;",
		fmt.Sprintf(
			"INSERT INTO domains (name, type) SELECT '%s', 'NATIVE' WHERE NOT EXISTS (SELECT 1 FROM domains WHERE name='%s');",
			sqliteQuote(normalizedDomain),
			sqliteQuote(normalizedDomain),
		),
		fmt.Sprintf(
			"DELETE FROM records WHERE domain_id = (SELECT id FROM domains WHERE name='%s');",
			sqliteQuote(normalizedDomain),
		),
		insertPowerDNSRecordSQL(normalizedDomain, normalizedDomain, "SOA", soaContent, 3600, 0),
		insertPowerDNSRecordSQL(normalizedDomain, normalizedDomain, "NS", ns1, 3600, 0),
		insertPowerDNSRecordSQL(normalizedDomain, normalizedDomain, "NS", ns2, 3600, 0),
	}

	for _, record := range records {
		name := normalizePowerDNSRecordName(normalizedDomain, record.Name)
		if name == "" {
			continue
		}
		ttl := record.TTL
		if ttl <= 0 {
			ttl = 3600
		}
		prio := 0
		content := strings.TrimSpace(record.Content)
		if strings.EqualFold(record.RecordType, "MX") {
			prio = 10
			content = normalizePowerDNSRecordName(normalizedDomain, content)
		}
		statements = append(statements, insertPowerDNSRecordSQL(normalizedDomain, name, strings.ToUpper(strings.TrimSpace(record.RecordType)), content, ttl, prio))
	}
	statements = append(statements, "COMMIT;")

	return exec.Command("sqlite3", dbPath, strings.Join(statements, "\n")).Run()
}

func removePowerDNSZone(domain string) error {
	normalizedDomain := normalizeDomain(domain)
	if normalizedDomain == "" || !powerDNSAvailable() {
		return nil
	}
	dbPath := powerDNSDatabasePath()
	sql := fmt.Sprintf(
		"BEGIN IMMEDIATE; DELETE FROM records WHERE domain_id=(SELECT id FROM domains WHERE name='%s'); DELETE FROM domains WHERE name='%s'; COMMIT;",
		sqliteQuote(normalizedDomain),
		sqliteQuote(normalizedDomain),
	)
	return exec.Command("sqlite3", dbPath, sql).Run()
}

func insertPowerDNSRecordSQL(zoneName, recordName, recordType, content string, ttl int, prio int) string {
	auth := 1
	if strings.EqualFold(recordType, "NS") {
		auth = 0
	}
	return fmt.Sprintf(
		"INSERT INTO records (domain_id, name, type, content, ttl, prio, disabled, auth) VALUES ((SELECT id FROM domains WHERE name='%s'), '%s', '%s', '%s', %d, %d, 0, %d);",
		sqliteQuote(zoneName),
		sqliteQuote(recordName),
		sqliteQuote(recordType),
		sqliteQuote(content),
		maxInt(ttl, 60),
		maxInt(prio, 0),
		auth,
	)
}

func normalizePowerDNSRecordName(zoneName, name string) string {
	name = strings.TrimSpace(name)
	switch {
	case name == "", name == "@":
		return zoneName
	case strings.Contains(name, "."):
		return normalizeDomain(name)
	default:
		return normalizeDomain(name + "." + zoneName)
	}
}

func sqliteQuote(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "'", "''")
}
