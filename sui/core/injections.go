package core

import (
	"fmt"
)

const suiLibScript = `

	function __sui_state(component) {
		this.handlers = component.watch || {};
		this.Set = async function (key, value) {
			const handler = this.handlers[key];
			if (handler && typeof handler === "function") {
				await handler(value);
			}
		}
	}

	function __sui_props(elm) {
		this.Get = function (key) {
			return elm && elm.getAttribute(key);
		}
	}

	function __sui_component(elm, component) {
		console.log('elm', component);
		this.root = elm;
		this.store = new __sui_store(elm);
		this.props = new __sui_props(elm);
		this.state = component ? new __sui_state(component) : {};
	}

	function __sui_event_handler(event, dataKeys, jsonKeys, elm, handler) {
		const data = {};
		dataKeys.forEach(function (key) {
			const value = elm.getAttribute("data:" + key);
			data[key] = value;
		})
		jsonKeys.forEach(function (key) {
			const value = elm.getAttribute("json:" + key);
			data[key] = null;
			if (value && value != "") {
				try {
					data[key] = JSON.parse(value);
				} catch (e) {
				 	const message = e.message || e || "An error occurred";
					console.error(` + "`[SUI] Event Handler Error: ${message}`" + `, elm);
				}
			}
		})

		const cn = elm.getAttribute("s:cn");
		let component = null
		if (cn != "") {
			component = new window[cn](elm);
		}
		handler && handler(event, data, new __sui_component(elm, component));
	};

	function __sui_store(elm) {
		elm = elm || document.body;

		this.Get = function (key) {
			return elm.getAttribute("data:" + key);
		}

		this.Set = function (key, value) {
			elm.setAttribute("data:" + key, value);
		}

		this.GetJSON = function (key) {
			const value = elm.getAttribute("json:" + key);
			if (value && value != "") {
				try {
					const res = JSON.parse(value);
					return res;
				} catch (e) {
					const message = e.message || e || "An error occurred";
					console.error(` + "`[SUI] Event Handler Error: ${message}`" + `, elm);
					return null;
				}
			}
			return null;
		}

		this.SetJSON = function (key, value) {
			elm.setAttribute("json:" + key, JSON.stringify(value));
		}
	}
`

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
		const dataKeys = %s;
		const jsonKeys = %s;
		__sui_event_handler(event, dataKeys, jsonKeys, this, %s);
	});
`

const compEventScriptTmpl = `
	document.querySelector("[s\\:event=%s]").addEventListener("%s", function (event) {
		const dataKeys = %s;
		const jsonKeys = %s;
		component = new %s(this).%s;
		__sui_event_handler(event, dataKeys, jsonKeys, this, handler);
	});
`

const componentInitScriptTmpl = `
	this.root = %s;
	this.store = new __sui_store(this.root);
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
	return fmt.Sprintf(pageEventScriptTmpl, eventID, eventName, dataKeys, jsonKeys, handler)
}

func compEventInjectScript(eventID, eventName, component, dataKeys, jsonKeys, handler string) string {
	return fmt.Sprintf(compEventScriptTmpl, eventID, eventName, dataKeys, jsonKeys, component, handler)
}

func componentInitScript(root string) string {
	return fmt.Sprintf(componentInitScriptTmpl, root)
}
