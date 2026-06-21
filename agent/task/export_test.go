package task

var (
	// Plan 1 exports
	ExportRowToTask        = rowToTask
	ExportMetaString       = metaString
	ExportGetString        = getString
	ExportGetStringDefault = getStringDefault
	ExportGetStringPtr     = getStringPtr
	ExportGetInt           = getInt
	ExportGetBool          = getBool
	ExportGetTime          = getTime

	// Plan 2 exports
	ExportMergeLayer     = mergeLayer
	ExportConfigReqToMap = configReqToMap
	ExportToStringMap    = toStringMap
)
