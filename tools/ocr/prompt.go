package ocr

import "fmt"

// buildOCRSystemPrompt constructs the system prompt for VLM-OCR based on type, format, mode, and language.
func buildOCRSystemPrompt(ocrType, outputFormat, mode, language string) string {
	base := typePrompt(ocrType)

	if outputFormat == "markdown" {
		base += "\n\n请以 Markdown 格式输出识别结果。保留原始文档的层级结构，表格用 Markdown 表格，标题用 # 标记。"
	} else if outputFormat == "json" {
		base += "\n\n" + jsonFormatInstruction(ocrType)
	}

	if mode == "accurate" {
		base += "\n\n请确保最高识别精度，逐字校对，不要遗漏任何文字。"
	} else if mode == "standard" {
		base += "\n\n请抓取主要内容，简洁输出。"
	}

	if language != "" {
		base += fmt.Sprintf("\n\n文档主要语言为: %s。请优先按此语言识别。", language)
	}

	return base
}

// typePrompt returns the base system prompt for a given OCR type.
func typePrompt(ocrType string) string {
	switch ocrType {
	case "table":
		return "你是一个专业的表格 OCR 引擎。请准确提取图片中所有表格的内容，保留表格的行列结构。"
	case "handwriting":
		return "你是一个手写文字识别引擎。请准确识别图片中的手写文字内容，注意区分潦草字迹。"
	case "invoice":
		return "你是一个发票识别引擎。请提取发票中的所有关键字段：发票号码、日期、金额、税额、购方/销方信息等。"
	case "receipt":
		return "你是一个小票/收据识别引擎。请提取收据中的商户名称、日期、商品明细、金额等关键信息。"
	case "id_card":
		return "你是一个身份证识别引擎。请提取身份证中的姓名、性别、民族、出生日期、住址、身份证号码等字段。"
	case "bank_card":
		return "你是一个银行卡识别引擎。请提取银行卡上的卡号、有效期、持卡人姓名、发卡行等信息。"
	case "license":
		return "你是一个营业执照识别引擎。请提取执照中的公司名称、统一社会信用代码、法定代表人、注册资本、成立日期、经营范围等字段。"
	case "vehicle_license":
		return "你是一个行驶证识别引擎。请提取行驶证中的车牌号、车辆类型、所有人、VIN码、发动机号、注册日期等字段。"
	case "passport":
		return "你是一个护照识别引擎。请提取护照中的姓名、国籍、护照号码、出生日期、有效期、签发机关等字段。"
	case "license_plate":
		return "你是一个车牌识别引擎。请提取图片中的车牌号码和车牌颜色。"
	case "document":
		return "你是一个文档识别引擎。请完整提取文档中的所有文字内容，保留段落和章节结构。"
	default: // "general"
		return "你是一个精确的 OCR 文字识别引擎。请完整准确地提取图片中的所有文字内容，保持原始排版顺序。"
	}
}

// jsonFormatInstruction returns format-specific JSON output instructions.
func jsonFormatInstruction(ocrType string) string {
	switch ocrType {
	case "invoice", "receipt", "id_card", "bank_card", "license",
		"vehicle_license", "passport", "license_plate":
		return `请以 JSON 格式输出识别结果，包含结构化字段。格式：
{"fields": {"字段名1": "值1", "字段名2": "值2", ...}, "text": "完整文本"}
只输出 JSON，不要添加任何解释文字。`
	case "table":
		return `请以 JSON 格式输出识别结果。格式：
{"text": "完整文本", "blocks": [{"text": "单元格文本", "page": 1}]}
只输出 JSON，不要添加任何解释文字。`
	default:
		return `请以 JSON 格式输出识别结果。格式：
{"text": "完整文本", "blocks": [{"text": "一行文字", "page": 1}]}
只输出 JSON，不要添加任何解释文字。`
	}
}
