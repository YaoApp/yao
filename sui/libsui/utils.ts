const $utils = {
  Store: (elm) => {
    if (typeof elm === "string") {
      elm = document.querySelector(elm);
    }
    return new __sui_store(elm);
  },

  RemoveClass: (element, className) => {
    const classes = Array.isArray(className) ? className : className.split(" ");
    classes.forEach((c) => {
      const v = c.replace(/[\n\r\s]/g, "");
      if (v === "") return;
      element.classList.remove(v);
    });
    return $utils;
  },

  AddClass: (element, className) => {
    const classes = Array.isArray(className) ? className : className.split(" ");
    classes.forEach((c) => {
      const v = c.replace(/[\n\r\s]/g, "");
      if (v === "") return;
      element.classList.add(v);
    });
    return $utils;
  },
};
