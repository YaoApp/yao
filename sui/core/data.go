package core

// Data get the data
func (page *Page) Data(request *Request) (map[string]interface{}, error) {
	return nil, nil
}

// RenderHTML render for the html
func (page *Page) renderData(html string, data map[string]interface{}, warnings []string) (string, error) {
	return html, nil
}
