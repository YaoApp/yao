package importer

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/any"
)

// UnmarshalJSON for json marshalJSON
func (option *Option) UnmarshalJSON(source []byte) error {
	var data = map[string]interface{}{}
	err := jsoniter.Unmarshal(source, &data)
	if err != nil {
		return err
	}

	new, err := OptionOf(data)
	if err != nil {
		return err
	}

	*option = *new
	return nil
}

// OptionOf 解析配置
func OptionOf(data map[string]interface{}) (*Option, error) {
	option := &Option{
		UseTemplate:    true,
		ChunkSize:      500,
		MappingPreview: PreviewAuto,
		DataPreview:    PreviewAuto,
	}

	if autoMatching, ok := data["useTemplate"].(bool); ok {
		option.UseTemplate = autoMatching
	}

	chunkSize := any.Of(data["chunkSize"]).CInt()
	if chunkSize > 0 && chunkSize < 2000 {
		option.ChunkSize = chunkSize
	}

	if mappingPreview, ok := data["mappingPreview"].(string); ok {
		option.MappingPreview = getPreviewOption(mappingPreview)
	}

	if dataPreview, ok := data["dataPreview"].(string); ok {
		option.DataPreview = getPreviewOption(dataPreview)
	}

	return option, nil
}

func getPreviewOption(value string) string {
	if value != PreviewAlways && value != PreviewAuto && value != PreviewNever {
		return PreviewAuto
	}
	return value
}
