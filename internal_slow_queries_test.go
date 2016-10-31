package newrelic

import (
	"strings"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
)

func TestSlowQueryBasic(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
	}})
}

func TestSlowQueryLocallyDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.DatastoreTracer.SlowQuery.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{})
}

func TestSlowQueryRemotelyDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	replyfn := func(reply *internal.ConnectReply) {
		reply.CollectTraces = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{})
}

func TestSlowQueryBelowThreshold(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 1 * time.Hour
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{})
}

func TestSlowQueryDatabaseProvided(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		DatabaseName:       "my_database",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "my_database",
		Host:         "",
		PortPathOrID: "",
	}})
}

func TestSlowQueryHostProvided(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		Host:               "db-server-1",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "db-server-1",
		PortPathOrID: "unknown",
	}})
	scope := "WebTransaction/Go/myName"
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/myName", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/operation/MySQL/INSERT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/users/INSERT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/users/INSERT", Scope: scope, Forced: false, Data: nil},
		{Name: "Datastore/instance/MySQL/db-server-1/unknown", Scope: "", Forced: false, Data: nil},
	})
}

func TestSlowQueryPortProvided(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		PortPathOrID:       "98021",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "unknown",
		PortPathOrID: "98021",
	}})
	scope := "WebTransaction/Go/myName"
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/myName", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/operation/MySQL/INSERT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/users/INSERT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/users/INSERT", Scope: scope, Forced: false, Data: nil},
		{Name: "Datastore/instance/MySQL/unknown/98021", Scope: "", Forced: false, Data: nil},
	})
}

func TestSlowQueryHostPortProvided(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		Host:               "db-server-1",
		PortPathOrID:       "98021",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "db-server-1",
		PortPathOrID: "98021",
	}})
	scope := "WebTransaction/Go/myName"
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/myName", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/operation/MySQL/INSERT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/users/INSERT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/users/INSERT", Scope: scope, Forced: false, Data: nil},
		{Name: "Datastore/instance/MySQL/db-server-1/98021", Scope: "", Forced: false, Data: nil},
	})
}

func TestSlowQueryAggregation(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}.End()
	DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
	}.End()
	DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastorePostgres,
		Collection:         "products",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO products (name, price) VALUES ($1, $2)",
	}.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        2,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
	}, {
		Count:        1,
		MetricName:   "Datastore/statement/Postgres/products/INSERT",
		Query:        "INSERT INTO products (name, price) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
	},
	})
}

func TestSlowQueryMissingQuery(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:  StartSegmentNow(txn),
		Product:    DatastoreMySQL,
		Collection: "users",
		Operation:  "INSERT",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "'INSERT' on 'users' using 'MySQL'",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
	}})
}

func TestSlowQueryMissingEverything(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime: StartSegmentNow(txn),
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/operation/Unknown/other",
		Query:        "'other' on 'unknown' using 'Unknown'",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
	}})
	scope := "WebTransaction/Go/myName"
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/myName", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/Unknown/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/Unknown/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/operation/Unknown/other", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/operation/Unknown/other", Scope: scope, Forced: false, Data: nil},
	})
}

func TestSlowQueryWithQueryParameters(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	params := map[string]interface{}{
		"str": "zap",
		"int": 123,
	}
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters:    params,
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
		Params:       params,
	}})
}

func TestSlowQueryHighSecurity(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.HighSecurity = true
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	params := map[string]interface{}{
		"str": "zap",
		"int": 123,
	}
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters:    params,
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
		Params:       nil,
	}})
}

func TestSlowQueryInvalidParameters(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	params := map[string]interface{}{
		"str":                               "zap",
		"int":                               123,
		"invalid_value":                     struct{}{},
		strings.Repeat("key-too-long", 100): 1,
		"long-key":                          strings.Repeat("A", 300),
	}
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters:    params,
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
		Params: map[string]interface{}{
			"str":      "zap",
			"int":      123,
			"long-key": strings.Repeat("A", 255),
		},
	}})
}

func TestSlowQueryParametersDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.DatastoreTracer.QueryParameters.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	params := map[string]interface{}{
		"str": "zap",
		"int": 123,
	}
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters:    params,
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
		Params:       nil,
	}})
}

func TestSlowQueryInstanceDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.DatastoreTracer.InstanceReporting.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		Host:               "db-server-1",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
	}})
	scope := "WebTransaction/Go/myName"
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/myName", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/operation/MySQL/INSERT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/users/INSERT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/users/INSERT", Scope: scope, Forced: false, Data: nil},
	})
}

func TestSlowQueryInstanceDisabledLocalhost(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.DatastoreTracer.InstanceReporting.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		Host:               "localhost",
		PortPathOrID:       "3306",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
	}})
	scope := "WebTransaction/Go/myName"
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/myName", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/myName", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/MySQL/allWeb", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/operation/MySQL/INSERT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/users/INSERT", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/MySQL/users/INSERT", Scope: scope, Forced: false, Data: nil},
	})
}

func TestSlowQueryDatabaseNameDisabled(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DatastoreTracer.SlowQuery.Threshold = 0
		cfg.DatastoreTracer.DatabaseNameReporting.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("myName", nil, helloRequest)
	s1 := DatastoreSegment{
		StartTime:          StartSegmentNow(txn),
		Product:            DatastoreMySQL,
		Collection:         "users",
		Operation:          "INSERT",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		DatabaseName:       "db-server-1",
	}
	s1.End()
	txn.End()

	app.ExpectSlowQueries(t, []internal.WantSlowQuery{{
		Count:        1,
		MetricName:   "Datastore/statement/MySQL/users/INSERT",
		Query:        "INSERT INTO users (name, age) VALUES ($1, $2)",
		TxnName:      "WebTransaction/Go/myName",
		TxnURL:       "/hello",
		DatabaseName: "",
		Host:         "",
		PortPathOrID: "",
	}})
}