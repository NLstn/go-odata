package main

import (
	"database/sql"
	"github.com/nlstn/go-odata/cmd/complianceserver/entities"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"net/url"
	"strings"
	"testing"
)

// Real dialect builders with a local connection for initialization; no SQL execution.
func TestApplyPipelineDialectSQL(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	dialects := map[string]gorm.Dialector{"sqlite": sqlite.New(sqlite.Config{Conn: conn}), "postgres": postgres.New(postgres.Config{Conn: conn}), "mysql": mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), "mariadb": mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), "sqlserver": sqlserver.New(sqlserver.Config{Conn: conn})}
	meta, err := metadata.AnalyzeEntity(&entities.Product{})
	if err != nil {
		t.Fatal(err)
	}
	for name, d := range dialects {
		t.Run(name, func(t *testing.T) {
			db, err := gorm.Open(d, &gorm.Config{DryRun: true, DisableAutomaticPing: true})
			if err != nil {
				t.Fatal(err)
			}
			for _, expr := range []string{"orderby(Price asc)/top(1)/aggregate(Price with sum as Total)", "compute(Price mul 2 as Twice)/filter(Twice gt 30)/aggregate(Twice with sum as Total)", "aggregate(Price with sum as Total)/compute(Total mul 2 as Twice)"} {
				opts, err := query.ParseQueryOptions(url.Values{"$apply": {expr}}, meta)
				if err != nil {
					t.Fatal(err)
				}
				var rows []map[string]interface{}
				r := query.ApplyQueryOptions(db.Session(&gorm.Session{}), opts, meta, nil).Find(&rows)
				if r.Error != nil {
					t.Fatal(r.Error)
				}
				s := r.Statement.SQL.String()
				if !strings.Contains(s, "FROM (SELECT") {
					t.Fatalf("missing boundary: %s", s)
				}
				if strings.Contains(expr, "top(1)") {
					if name == "sqlserver" {
						if !strings.Contains(s, "FETCH NEXT 1 ROWS ONLY") {
							t.Fatalf("missing SQL Server page: %s", s)
						}
					} else if !strings.Contains(s, "LIMIT") {
						t.Fatalf("missing page: %s", s)
					}
				}
			}
		})
	}
}
