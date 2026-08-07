package audio

import _ "embed"

//go:embed transcribe_schema.json
var TranscribeSchemaJSON []byte

//go:embed providers_schema.json
var ProvidersSchemaJSON []byte
