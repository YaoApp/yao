package core

import (
	"fmt"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/yaoapp/gou/runtime/transform"
	"github.com/yaoapp/yao/data"
)

var libsuicode = ""

// LibSUI return the libsui code
func LibSUI() ([]byte, []byte, error) {

	// Read source code from bindata
	index, err := data.Read("libsui/index.ts")
	if err != nil {
		return nil, nil, err
	}

	utils, err := data.Read("libsui/utils.ts")
	if err != nil {
		return nil, nil, err
	}

	yao, err := data.Read("libsui/yao.ts")
	if err != nil {
		return nil, nil, err
	}

	// Merge the source code
	source := fmt.Sprintf("%s\n%s\n%s", index, utils, yao)

	// Build the source code
	js, sm, err := transform.TypeScriptWithSourceMap(string(source), api.TransformOptions{
		Target:            api.ES2015,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		MinifyWhitespace:  true,
		Sourcefile:        "libsui.ts",
	})

	return js, sm, nil
}

const initScriptTmpl = `
	try {
		var __sui_data = %s;
	} catch (e) { console.log('init data error:', e); }

	document.addEventListener("DOMContentLoaded", function () {
		document.querySelectorAll("[s\\:ready]").forEach(function (element) {
			const method = element.getAttribute("s:ready");
			const cn = element.getAttribute("s:cn");
			if (method && typeof window[cn] === "function") {
				try {
					new window[cn](element);
				} catch (e) {
					const message = e.message || e || "An error occurred";
					console.error(` + "`[SUI] ${cn} Error: ${message}`" + `);
				}
			}
		});
		__sui_event_init(document.body);
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
	if (document.querySelector("[s\\:event=%s]")) {
		let elms = document.querySelectorAll("[s\\:event=%s]");
		elms.forEach(function (element) {
			element.addEventListener("%s", function (event) {
				const dataKeys = %s;
				const jsonKeys = %s;
				const root = document.body;
				__sui_event_handler(event, dataKeys, jsonKeys, element, root, window.%s);
			});
		});
	}
`

const compEventScriptTmpl = `
	if (document.querySelector("[s\\:event=%s]")) {
		let elms = document.querySelectorAll("[s\\:event=%s]");
		elms.forEach(function (element) {
			element.addEventListener("%s", function (event) {
				const dataKeys = %s;
				const jsonKeys = %s;
				const root = __sui_component_root(element, "%s");
				handler = new %s(root).%s;
				__sui_event_handler(event, dataKeys, jsonKeys, element, root, handler);
			});
		});
	}
`

const componentInitScriptTmpl = `
	this.root = %s;
	const __self = this;
	this.store = new __sui_store(this.root);
	this.state = new __sui_state(this);
	this.props = new __sui_props(this.root);
	this.$root = new __Query(this.root);
	
	this.find = function (selector) {
		return new __Query(__self.root).find(selector);
	};

	this.query = function (selector) {
		return __self.root.querySelector(selector);
	}

	this.queryAll = function (selector) {
		return __self.root.querySelectorAll(selector);
	}

	this.render = function(name, data, option) {
		const r = new __Render(__self, option);
  		return r.Exec(name, data);
	};

	this.emit = function (name, data) {
		const event = new CustomEvent(name, { detail: data });
		__self.root.dispatchEvent(event);
	};

	%s

	if (this.root.getAttribute("initialized") != 'true') {
		__self.root.setAttribute("initialized", 'true');
		__self.root.addEventListener("state:change", function (event) {
			const name = this.getAttribute("s:cn");
			const target = event.detail.target;
			const key = event.detail.key;
			const value = event.detail.value;
			const component = new window[name](this);
			const state = new __sui_state(component);
			state.Set(key, value, target)
		});
		__self.once && __self.once();
	}
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
	return fmt.Sprintf(pageEventScriptTmpl, eventID, eventID, eventName, dataKeys, jsonKeys, handler)
}

func compEventInjectScript(eventID, eventName, component, dataKeys, jsonKeys, handler string) string {
	return fmt.Sprintf(compEventScriptTmpl, eventID, eventID, eventName, dataKeys, jsonKeys, component, component, handler)
}

func componentInitScript(root string, source string) string {
	return fmt.Sprintf(componentInitScriptTmpl, root, source)
}

// BackendScript inject the backend script
func BackendScript(route string) string {
	return fmt.Sprintf(backendScriptTmpl, route)
}
