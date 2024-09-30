function $$(selector) {
  let elm: HTMLElement | null = null;
  if (typeof selector === "string") {
    elm = document.querySelector(selector);
  }

  if (selector instanceof HTMLElement) {
    elm = selector;
  }

  if (elm) {
    const cn = elm.getAttribute("s:cn");
    if (cn && cn != "" && typeof window[cn] === "function") {
      const component = new window[cn](elm);
      return new __sui_component(elm, component);
    }
  }
  return null;
}

function __sui_component_root(elm: Element, name: string) {
  return elm.closest(`[s\\:cn=${name}]`);
}

function __sui_state(component) {
  this.handlers = component.watch || {};
  this.Set = async function (key, value, target) {
    const handler = this.handlers[key];
    target = target || component.root;
    if (handler && typeof handler === "function") {
      const stateObj = {
        target: target,
        stopPropagation: function () {
          target.setAttribute("state-propagation", "true");
        },
      };
      await handler(value, stateObj);
      const isStopPropagation = target
        ? target.getAttribute("state-propagation") === "true"
        : false;
      if (isStopPropagation) {
        return;
      }

      let parent = component.root.parentElement?.closest(`[s\\:cn]`);
      if (parent == null) {
        return;
      }

      // Dispatch the state change custom event to parent component
      const event = new CustomEvent("state:change", {
        detail: { key: key, value: value, target: component.root },
      });
      parent.dispatchEvent(event);
    }
  };
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
  };

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
  };
}

function __sui_component(elm, component) {
  this.root = elm;
  this.store = new __sui_store(elm);
  this.props = new __sui_props(elm);
  this.state = component ? new __sui_state(component) : {};

  const __self = this;

  // @ts-ignore
  this.$root = new __Query(this.root);

  this.find = function (selector) {
    // @ts-ignore
    return new __Query(__self.root).find(selector);
  };

  this.query = function (selector) {
    return __self.root.querySelector(selector);
  };

  this.queryAll = function (selector) {
    return __self.root.querySelectorAll(selector);
  };

  this.emit = function (name, data) {
    const event = new CustomEvent(name, { detail: data });
    __self.root.dispatchEvent(event);
  };

  this.render = function (name, data, option) {
    // @ts-ignore
    const r = new __Render(__self, option);
    return r.Exec(name, data);
  };
}

function __sui_event_handler(event, dataKeys, jsonKeys, target, root, handler) {
  const data = {};
  target = target || null;
  if (target) {
    dataKeys.forEach(function (key) {
      const value = target.getAttribute("data:" + key);
      data[key] = value;
    });
    jsonKeys.forEach(function (key) {
      const value = target.getAttribute("json:" + key);
      data[key] = null;
      if (value && value != "") {
        try {
          data[key] = JSON.parse(value);
        } catch (e) {
          const message = e.message || e || "An error occurred";
          console.error(`[SUI] Event Handler Error: ${message} `, target);
        }
      }
    });
  }
  handler &&
    handler(event, data, {
      rootElement: root,
      targetElement: target,
    });
}

function __sui_event_init(elm: Element) {
  const bindEvent = (eventElm) => {
    const cn = eventElm.getAttribute("s:event-cn") || "";
    if (cn == "") {
      console.error("[SUI] Component name is required for event binding", elm);
      return;
    }

    // Data keys
    const events: Record<string, string> = {};
    const dataKeys: string[] = [];
    const jsonKeys: string[] = [];
    for (let i = 0; i < eventElm.attributes.length; i++) {
      if (eventElm.attributes[i].name.startsWith("data:")) {
        dataKeys.push(eventElm.attributes[i].name.replace("data:", ""));
      }
      if (eventElm.attributes[i].name.startsWith("json:")) {
        jsonKeys.push(eventElm.attributes[i].name.replace("json:", ""));
      }
      if (eventElm.attributes[i].name.startsWith("s:on-")) {
        const key = eventElm.attributes[i].name.replace("s:on-", "");
        events[key] = eventElm.attributes[i].value;
      }
    }

    // Bind the event
    for (const name in events) {
      const bind = events[name];
      if (cn == "__page") {
        const handler = window[bind];
        const root = document.body;
        const target = eventElm;
        eventElm.addEventListener(name, (event) => {
          __sui_event_handler(event, dataKeys, jsonKeys, target, root, handler);
        });
        continue;
      }

      const component = eventElm.closest(`[s\\:cn=${cn}]`);
      if (typeof window[cn] !== "function") {
        console.error(`[SUI] Component ${cn} not found`, eventElm);
        return;
      }

      // @ts-ignore
      const comp = new window[cn](component);
      const handler = comp[bind];
      const root = comp.root;
      const target = eventElm;
      eventElm.addEventListener(name, (event) => {
        __sui_event_handler(event, dataKeys, jsonKeys, target, root, handler);
      });
    }
  };

  const eventElms = elm.querySelectorAll("[s\\:event]");
  const jitEventElms = elm.querySelectorAll("[s\\:event-jit]");
  eventElms.forEach((eventElm) => bindEvent(eventElm));
  jitEventElms.forEach((eventElm) => bindEvent(eventElm));
}

function __sui_store(elm) {
  elm = elm || document.body;

  this.Get = function (key) {
    return elm.getAttribute("data:" + key);
  };

  this.Set = function (key, value) {
    elm.setAttribute("data:" + key, value);
  };

  this.GetJSON = function (key) {
    const value = elm.getAttribute("json:" + key);
    if (value && value != "") {
      try {
        const res = JSON.parse(value);
        return res;
      } catch (e) {
        const message = e.message || e || "An error occurred";
        console.error(`[SUI] Event Handler Error: ${message}`, elm);
        return null;
      }
    }
    return null;
  };

  this.SetJSON = function (key, value) {
    elm.setAttribute("json:" + key, JSON.stringify(value));
  };

  this.GetData = function () {
    return this.GetJSON("__component_data") || {};
  };
}

async function __sui_backend_call(
  route: string,
  headers: [string, string][] | Record<string, string> | Headers,
  method: string,
  ...args: any
): Promise<any> {
  const url = `/api/__yao/sui/v1/run${route}`;
  headers = {
    "Content-Type": "application/json",
    Referer: window.location.href,
    Cookie: document.cookie,
    ...headers,
  };
  const payload = { method, args };
  try {
    const body = JSON.stringify(payload);
    const response = await fetch(url, { method: "POST", headers, body: body });
    const text = await response.text();
    let data: any | null = null;
    if (text && text != "") {
      data = JSON.parse(text);
    }

    if (response.status >= 400) {
      const message = data.message
        ? data.message
        : `Failed to call ${route} ${method}`;
      const code = data.code ? data.code : 500;
      return Promise.reject({ message, code });
    }

    return Promise.resolve(data);
  } catch (e) {
    const message = e.message ? e.message : `Failed to call ${route} ${method}`;
    const code = e.code ? e.code : 500;
    console.error(`[SUI] Failed to call ${route} ${method}:`, e);
    return Promise.reject({ message, code });
  }
}

/**
 * SUI Render
 * @param component
 * @param name
 */
async function __sui_render(
  component: Component | string,
  name: string,
  data: Record<string, any>,
  option?: RenderOption
): Promise<string> {
  const comp = (
    typeof component === "object" ? component : $$(component)
  ) as Component;

  if (comp == null) {
    console.error(`[SUI] Component not found: ${component}`);
    return Promise.reject("Component not found");
  }

  const elms = comp.root.querySelectorAll(`[s\\:render=${name}]`);
  if (!elms.length) {
    console.error(`[SUI] No element found with s:render=${name}`);
    return Promise.reject("No element found");
  }

  // Set default options
  option = option || {};
  option.replace = option.replace === undefined ? true : option.replace;
  option.showLoader =
    option.showLoader === undefined ? false : option.showLoader;
  option.withPageData =
    option.withPageData === undefined ? false : option.withPageData;

  // Prepare loader
  let loader = `<span class="sui-render-loading">Loading...</span>`;
  if (option.showLoader && option.replace) {
    if (typeof option.showLoader === "string") {
      loader = option.showLoader;
    } else if (option.showLoader instanceof HTMLElement) {
      loader = option.showLoader.outerHTML;
    }
    elms.forEach((elm) => (elm.innerHTML = loader));
  }

  // Prepare data
  let _data = comp.store.GetData() || {};
  if (option.withPageData) {
    // @ts-ignore
    _data = { ..._data, ...__sui_data };
  }

  // get s:route attribute
  const elm = comp.root.closest("[s\\:route]");
  const routeAttr = elm ? elm.getAttribute("s:route") : false;
  const root = document.body.getAttribute("s:public") || "";
  const route = routeAttr ? `${root}${routeAttr}` : window.location.pathname;
  option.component = (routeAttr && comp.root.getAttribute("s:cn")) || "";

  const url = `/api/__yao/sui/v1/render${route}`;
  const payload = { name, data: _data, option };

  // merge the user data
  if (data) {
    for (const key in data) {
      payload.data[key] = data[key];
    }
  }
  const headers = {
    "Content-Type": "application/json",
    Cookie: document.cookie,
  };

  // Native post request to the server
  try {
    const body = JSON.stringify(payload);
    const response = await fetch(url, { method: "POST", headers, body: body });
    const text = await response.text();
    if (!option.replace) {
      return Promise.resolve(text);
    }

    // Set the response text to the elements
    elms.forEach((elm) => {
      elm.innerHTML = text;
      try {
        __sui_event_init(elm);
      } catch (e) {
        const message = e.message || "Failed to init events";
        Promise.reject(message);
      }
    });

    return Promise.resolve(text);
  } catch (e) {
    //Set the error message
    elms.forEach((elm) => {
      elm.innerHTML = `<span class="sui-render-error">Failed to render</span>`;
      console.error("Failed to render", e);
    });
    return Promise.reject("Failed to render");
  }
}

export type Component = {
  root: HTMLElement;
  state: ComponentState;
  store: ComponentStore;
  watch?: Record<string, (value: any, state?: State) => void>;
  Constants?: Record<string, any>;

  [key: string]: any;
};

export type RenderOption = {
  target?: HTMLElement; // default is same with s:render target
  showLoader?: HTMLElement | string | boolean; // default is false
  replace?: boolean; // default is true
  withPageData?: boolean; // default is false
  component?: string; // default is empty
};

export type ComponentState = {
  Set: (key: string, value: any) => void;
};

export type ComponentStore = {
  Get: (key: string) => string;
  Set: (key: string, value: any) => void;
  GetJSON: (key: string) => any;
  SetJSON: (key: string, value: any) => void;
  GetData: () => Record<string, any>;
};

export type State = {
  target: HTMLElement;
  stopPropagation();
};
