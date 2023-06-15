package driver

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/dbal/query"
	"github.com/yaoapp/xun/dbal/schema"
	"github.com/yaoapp/yao/share"
)

// Connector the store driver
type Connector struct {
	Connector string
	Table     string
	Reload    bool
	Widget    string
	query     query.Query
	schema    schema.Schema
}

// NewConnector create a new stroe driver
func NewConnector(widgetID string, connectorName string, tableName string, reload bool) (*Connector, error) {
	if connectorName == "" {
		connectorName = "default"
	}

	if tableName == "" {
		tableName = fmt.Sprintf("__yao_dsl_%s", widgetID)
	}

	store := &Connector{Widget: widgetID, Connector: connectorName, Reload: reload, Table: tableName}
	if store.Connector == "default" {
		store.query = capsule.Global.Query()
		store.schema = capsule.Global.Schema()

	} else {

		conn, err := connector.Select(connectorName)
		if err != nil {
			return nil, err
		}

		if !conn.Is(connector.DATABASE) {
			return nil, fmt.Errorf("The connector %s is not a database connector", connectorName)
		}

		store.query, err = conn.Query()
		if err != nil {
			return nil, err
		}

		store.schema, err = conn.Schema()
		if err != nil {
			return nil, err
		}
	}

	err := store.init()
	if err != nil {
		return nil, err
	}

	return store, nil
}

// Walk load the widget instances
func (app *Connector) Walk(cb func(string, map[string]interface{})) error {

	rows, err := app.query.
		Table(app.Table).
		Select("file", "source").
		Limit(5000).
		Get()

	if err != nil {
		return err
	}

	messages := []string{}
	for _, row := range rows {

		source := map[string]interface{}{}
		data := []byte(row["source"].(string))
		file := row["file"].(string)

		id := share.ID("", file)
		err := application.Parse(row["file"].(string), data, &source)
		if err != nil {
			messages = append(messages, err.Error())
			continue
		}
		cb(id, source)
	}

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}

	return nil
}

// Save save the widget DSL
func (app *Connector) Save(file string, source map[string]interface{}) error {

	bytes, err := jsoniter.Marshal(source)
	if err != nil {
		return err
	}

	content := string(bytes)

	has, err := app.query.Table(app.Table).Where("file", file).Exists()
	if err != nil {
		return err
	}

	if has {
		_, err = app.query.Table(app.Table).Where("file", file).Update(map[string]interface{}{"source": content})
	} else {
		err = app.query.Table(app.Table).Insert(map[string]interface{}{"file": file, "source": content})
	}

	return err
}

// Remove remove the widget DSL
func (app *Connector) Remove(file string) error {
	_, err := app.query.Table(app.Table).Where("file", file).Delete()
	return err
}

// init the widget store
func (app *Connector) init() error {

	has, err := app.schema.HasTable(app.Table)
	if err != nil {
		return err
	}

	// create the table
	if !has {
		err = app.schema.CreateTable(app.Table, func(table schema.Blueprint) {
			table.ID("id")                     // The ID field
			table.String("file", 255).Unique() // The file name
			table.Text("source").Null()
			table.TimestampTz("created_at").SetDefaultRaw("NOW()").Index()
			table.TimestampTz("updated_at").Null().Index()
			table.TimestampTz("expired_at").Null().Index()
		})

		if err != nil {
			return err
		}
		log.Trace("Create the conversation table: %s", app.Table)
	}

	// validate the table
	tab, err := app.schema.GetTable(app.Table)
	if err != nil {
		return err
	}

	fields := []string{"id", "file", "source", "created_at", "updated_at", "expired_at"}
	for _, field := range fields {
		if !tab.HasColumn(field) {
			return fmt.Errorf("%s table %s field %s is required", app.Widget, app.Table, field)
		}
	}

	return nil
}
