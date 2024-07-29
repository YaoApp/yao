package core

import (
	"fmt"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/yaoapp/gou/runtime/transform"
)

var libsuicode = ""

func libsui(minify bool) (string, error) {
	if libsuicode != "" {
		return libsuicode, nil
	}

	option := api.TransformOptions{Target: api.ES2015}
	if minify {
		option.MinifyIdentifiers = true
		option.MinifySyntax = true
		option.MinifyWhitespace = true
	}
	var err error
	libsuicode, err = transform.JavaScript(libsuisource, option)
	if err != nil {
		return "", fmt.Errorf("libsui error: %w", err)
	}
	return libsuicode, nil
}

const libsuisource = `

	function __sui_component_root(elm, name) {
		while (elm && elm.getAttribute("s:cn") !== name) {
			elm = elm.parentElement;
		}
		return elm;
	}

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
			if (!elm || typeof elm.getAttribute !== "function") {
				return null;
			}
			const k = "prop:" + key;
			const v = elm.getAttribute(k);
			const json = elm.getAttribute("json-attr-prop:" + key) === "true";
			if (json) {
				try {
					return JSON.parse(v);
				} catch (e) {
					return null;
				}
			}
			return v;
		}

		this.List = function () {
			const props = {};
			if (!elm || typeof elm.getAttribute !== "function") {
				return props;
			}

			const attrs = elm.attributes;
			for (let i = 0; i < attrs.length; i++) {
				const attr = attrs[i];
				if (attr.name.startsWith("prop:")) {
					const k = attr.name.replace("prop:", "");
					const json = elm.getAttribute("json-attr-prop:" + k) === "true";
					if (json) {
						try {
							props[k] = JSON.parse(attr.value);
						} catch (e) {
							props[k] = null;
						}
						continue;
					}
					props[k] = attr.value;
				}
			}
			return props;
		}
	}

	function __sui_component(elm, component) {
		this.root = elm;
		this.store = new __sui_store(elm);
		this.props = new __sui_props(elm);
		this.state = component ? new __sui_state(component) : {};
	}

	function $$(selector) {
		elm = null;
		if (typeof selector === "string" ){
			 elm = document.querySelector(selector);
		}

		if (selector instanceof HTMLElement) {
			elm = selector;
		}
		
		if (elm) {
			cn = elm.getAttribute("s:cn");
			if (cn != "" && typeof window[cn] === "function") {
				const component = new window[cn](elm);
				return new __sui_component(elm, component);
			}
		}
		return null;
	}

	function __sui_event_handler(event, dataKeys, jsonKeys, target, root, handler) {
		const data = {};
		target = target || null;
		if (target) {
			dataKeys.forEach(function (key) {
				const value = target.getAttribute("data:" + key);
				data[key] = value;
			})
			jsonKeys.forEach(function (key) {
				const value = target.getAttribute("json:" + key);
				data[key] = null;
				if (value && value != "") {
					try {
						data[key] = JSON.parse(value);
					} catch (e) {
						const message = e.message || e || "An error occurred";
						console.error(` + "`[SUI] Event Handler Error: ${message}`" + `, target);
					}
				}
			})
		}
		handler && handler(event, data, root, target);
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

		this.GetData = function () {
			return this.GetJSON("__component_data") || {};
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
		const root = document.body;
		const target = this;
		__sui_event_handler(event, dataKeys, jsonKeys, target, root, %s);
	});
`

const compEventScriptTmpl = `
	if (document.querySelector("[s\\:event=%s]")) {
		document.querySelector("[s\\:event=%s]").addEventListener("%s", function (event) {
			const dataKeys = %s;
			const jsonKeys = %s;
			const root = __sui_component_root(this, "%s");
			handler = new %s(root).%s;
			const target = event.target || null;
			__sui_event_handler(event, dataKeys, jsonKeys, target, root, handler);
		});
	}
`

const componentInitScriptTmpl = `
	this.root = %s;
	this.store = new __sui_store(this.root);
	this.props = new __sui_props(this.root);
`

// Inject code
const backendScriptTmpl = `
this.__sui_page = '%s';
this.__sui_constants = {};
this.__sui_helpers = [];

if (typeof Helpers === 'object') {
	this.__sui_helpers = Object.keys(Helpers);
}

if (typeof Constants === 'object') {
	this.__sui_constants = Constants;
}
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
	return fmt.Sprintf(compEventScriptTmpl, eventID, eventID, eventName, dataKeys, jsonKeys, component, component, handler)
}

func componentInitScript(root string) string {
	return fmt.Sprintf(componentInitScriptTmpl, root)
}

// BackendScript inject the backend script
func BackendScript(route string) string {
	return fmt.Sprintf(backendScriptTmpl, route)
}
