package assistant

// search.go
var ExportParseSearchField = parseSearchField
var ExportShouldAutoSearch = (*Assistant).shouldAutoSearch
var ExportGetMergedSearchUses = (*Assistant).getMergedSearchUses

// build.go
var ExportBuildSystemPrompts = (*Assistant).buildSystemPrompts
var ExportBuildContextVariables = (*Assistant).buildContextVariables
var ExportShouldDisableGlobalPrompts = (*Assistant).shouldDisableGlobalPrompts
var ExportGetAssistantPrompts = (*Assistant).getAssistantPrompts
var ExportGetPromptPresetKey = (*Assistant).getPromptPresetKey

// history.go
var ExportFindOverlapIndex = (*Assistant).findOverlapIndex
var ExportMessagesMatch = (*Assistant).messagesMatch
var ExportGetHistorySize = getHistorySize

// chat.go
var ExportSanitizeCollectionID = sanitizeCollectionID
var ExportMergeChatMetadata = mergeChatMetadata

// permission.go
var ExportCheckPermissions = (*Assistant).checkPermissions

// load.go
var ExportLoadMap = loadMap

// loop.go
var ExportIsToolLoopDisabled = (*Assistant).isToolLoopDisabled
var ExportGetMaxToolLoopTurns = (*Assistant).getMaxToolLoopTurns
var ExportBuildToolLoopMessages = buildToolLoopMessages
var ExportBuildLoopFallbackMarkdown = buildLoopFallbackMarkdown
var ExportMessageText = messageText
