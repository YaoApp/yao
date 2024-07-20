package core

import "fmt"

const initScriptTmpl = `
	try {
		var __sui_data = %s;
	} catch (e) { console.log('init data error:', e); }

	document.addEventListener("DOMContentLoaded", function () {
		try {
			document.querySelectorAll("[s\\:ready]").forEach(function (element) {
				const method = element.getAttribute("s:ready");
				const cn = element.getAttribute("s:cn");
				if (method && typeof window[cn] === "function") {
					try {
						window[cn](element);
					} catch (e) {
						const message = e.message || e || "An error occurred";
						console.error(` + "`[SUI] ${cn} Error: ${message}`" + `);
					}
				}
			});
		} catch (e) {}
	});
	%s
`

const i118nScriptTmpl = `
	let __sui_locale = {};
	try {
		__sui_locale = %s;
	} catch (e) { __sui_locale = {}  }

	function __m(message, fmt) {
		if (fmt && typeof fmt === "function") {
			return fmt(message, __sui_locale);
		}
		return __sui_locale[message] || message;
	}
`

const pageEventScriptTmpl = `
	document.querySelector("[s\\:event=%s]").addEventListener("%s", function (event) {
		let data = {};
		const dataKeys = %s;
		const jsonKeys = %s;

		const elm = this;
		dataKeys.forEach(function (key) {
			const value = elm.getAttribute("data:" + key);
			data[key] = value;
		})

		jsonKeys.forEach(function (key) {
			const value = elm.getAttribute("json:" + key);
			data[key] = null;
			if (value && value != "") {
				data[key] = JSON.parse(value);
			}
		})

		%s && %s(event, data, this);
	});
`

func bodyInjectionScript(jsonRaw string, debug bool) string {
	jsPrintData := ""
	if debug {
		jsPrintData = `console.log(__sui_data);`
	}
	return fmt.Sprintf(`<script type="text/javascript">`+initScriptTmpl+`</script>`, jsonRaw, jsPrintData)
}

func headInjectionScript(jsonRaw string) string {
	return fmt.Sprintf(`<script type="text/javascript">`+i118nScriptTmpl+`</script>`, jsonRaw)
}

func pageEventInjectScript(eventID, eventName, dataKeys, jsonKeys, handler string) string {
	return fmt.Sprintf(pageEventScriptTmpl, eventID, eventName, dataKeys, jsonKeys, handler, handler)
}
