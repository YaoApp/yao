package table

// Lang for applying a language pack
func (dsl *DSL) Lang(trans func(widget string, inst string, value *string) bool) {
	widget := "table"
	trans(widget, dsl.ID, &dsl.Name)
}
