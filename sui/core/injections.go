package core

import "fmt"

const initScriptTmpl = `
	try {
		var __sui_data = %s;
	} catch (e) { console.log('init data error:', e); }

	function __sui_findParentWithAttribute(element, attributeName) {
		while (element && element !== document) {
			if (element.hasAttribute(attributeName)) {
				return element.getAttribute(attributeName);
			}
			element = element.parentElement;
		}
		return null;
	}

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

			document.querySelectorAll("[s\\:click]").forEach(function (element) {
				const method = element.getAttribute("s:click");
				const cn = __sui_findParentWithAttribute(element, "s:cn");
				if (method && cn && typeof window[cn] === "function") {
					const obj = new window[cn]();
					if (typeof obj[method] === "function") {
						element.addEventListener("click", function (event) {
							try {
								obj[method](element, event);
							} catch (e) {
								const message = e.message || e || "An error occurred";
								console.error(` + "`[SUI] ${cn}.${method} Error: ${message}`" + `);
							}
						});
						return
					}
					console.error(` + "`[SUI] ${cn}.${method} Error: Method not found`" + `);
					return
				}

				if (method && typeof window[method] === "function") {
					element.addEventListener("click", function (event) {
						try {
							window[method](element, event);
						} catch (e) {
							const message = e.message || e || "An error occurred";
							console.error(` + "`[SUI] ${method} Error: ${message}`" + `);
						}
					});
				}

			});
		} catch (e) {}
	});
	%s
`

const i118nScriptTmpl = `
	function L(key) {
		return key;
	}
`

func bodyInjectionScript(jsonRaw string, debug bool) string {
	jsPrintData := ""
	if debug {
		jsPrintData = `console.log(__sui_data);`
	}
	return fmt.Sprintf(`<script type="text/javascript">`+initScriptTmpl+`</script>`, jsonRaw, jsPrintData)
}

func headInjectionScript() string {
	return fmt.Sprintf(`<script type="text/javascript">` + i118nScriptTmpl + `</script>`)
}
