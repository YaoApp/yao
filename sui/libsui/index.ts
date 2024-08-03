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

function __sui_component_root(elm, name) {
  while (elm && elm.getAttribute("s:cn") !== name) {
    elm = elm.parentElement;
  }
  return elm;
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

      let parent = component.root.parentElement;
      while (parent && !parent.getAttribute("s:cn")) {
        parent = parent.parentElement;
      }
      if (parent == document.body || parent == null) {
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
