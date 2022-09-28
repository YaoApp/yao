package chart

// Lang for applying a language pack
func (dsl *DSL) Lang(trans func(widget string, inst string, value *string) bool) {
	widget := "form"
	trans(widget, dsl.ID, &dsl.Name)
}
