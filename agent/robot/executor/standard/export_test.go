package standard

import (
	agentcontext "github.com/yaoapp/yao/agent/context"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

var (
	ExtractLLMContentFn     = extractLLMContent
	MergeManifestFilesFn    = mergeManifestFiles
	DetectNeedMoreInfoFn    = detectNeedMoreInfo
	GetEffectiveLocaleFn    = getEffectiveLocale
	GetLocalizedMessageFn   = getLocalizedMessage
	ExtractGoalNameFn       = extractGoalName
	StripMarkdownFmtFn      = stripMarkdownFormatting
	FormatTaskProgressFn    = formatTaskProgressName
	GenerateSummaryFn       = generateSummary
	ExtractKeyOutputsFn     = extractKeyOutputs
	FlattenOutputFn         = flattenOutput
	FormatOutputAsTextFn    = formatOutputAsText
	MimeFromExtFn           = mimeFromExt
	BoolMarkFn              = boolMark
	SkipFillerPrefixesFn    = skipFillerPrefixes
	CapSliceFn              = capSlice
	HasValidOutputFn        = (*Validator).hasValidOutput
	ConvertStringRuleFn     = (*Validator).convertStringRule
	HasAgentRulesFn         = (*Validator).hasAgentRules
	GetSemanticRulesFn      = (*Validator).getSemanticRules
	GenerateFeedbackReplyFn = (*Validator).generateFeedbackReply
)

type ExportedCallResult = CallResult
type ExportedManifestFile = ManifestFile
type ExportedCompletionResponse = agentcontext.CompletionResponse
type ExportedValidationResult = robottypes.ValidationResult
