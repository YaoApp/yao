/**
 * YAO Pure JavaScript SDK
 * @author Max<max@iqka.com>
 * @maintainer https://yaoapps.com
 */

/**
 * Yao Object
 * @param {*} host
 */
function Yao(host) {
  this.host = `${
    host || window.location.protocol + "//" + window.location.host
  }/api`;
  this.query = {};
  new URLSearchParams(window.location.search).forEach((key, value) => {
    this.query[key] = value;
  });
}

/**
 * Get API
 * @param {*} path
 * @param {*} params
 */
Yao.prototype.Get = async function (path, params, headers) {
  return this.Fetch("GET", path, params, null, headers);
};

/**
 * Post API
 * @param {*} path
 * @param {*} data
 * @param {*} params
 * @param {*} headers
 */
Yao.prototype.Post = async function (path, data, params, headers) {
  return this.Fetch("POST", path, params, data, headers);
};

/**
 * Download API
 * @param {*} path
 * @param {*} params
 */
Yao.prototype.Download = async function (path, params, savefile, headers) {
  try {
    const blob = await this.Fetch("GET", path, params, null, headers, true);

    var objectUrl = window.URL.createObjectURL(blob);
    let anchor = document.createElement("a");
    document.body.appendChild(anchor);
    anchor.href = objectUrl;
    anchor.download = savefile;
    anchor.click();
    window.URL.revokeObjectURL(objectUrl);
  } catch (err) {
    alert("成功创建导出任务!");
  }
};

/**
 * Fetch API
 * @param {*} method
 * @param {*} path
 * @param {*} params
 * @param {*} data
 * @param {*} headers
 */
Yao.prototype.Fetch = async function (
  method,
  path,
  params,
  data,
  headers,
  isblob
) {
  params = params || {};
  headers = headers || {};
  data = data || null;
  var url = `${this.host}${path}`;
  var queryString = this.Serialize(params);
  if (queryString != "") {
    url = url.includes("?") ? `${url}&${queryString}` : `${url}?${queryString}`;
  }

  const token = this.Token();
  if (token != "") {
    headers["authorization"] = `Bearer ${token}`;
  }

  if (!headers["Content-Type"]) {
    headers["Content-Type"] = "application/json";
  }

  var options: any = {
    method: method,
    mode: "cors", // no-cors, *cors, same-origin
    cache: "no-cache", // *default, no-cache, reload, force-cache, only-if-cached
    credentials: "same-origin", // include, *same-origin, omit
    headers: headers,
    redirect: "follow", // manual, *follow, error
  };

  if (data != null) {
    options["body"] = JSON.stringify(data);
  }

  const resp = await fetch(url, options);
  const type = resp.headers.get("Content-Type") || "";
  if (type.includes("application/json")) {
    try {
      const data = await resp.json();
      return data;
    } catch (err) {
      return { code: resp.status, message: "empty return" };
    }
  } else if (isblob) {
    return resp.blob();
  } else if (type.includes("text/html") || type.includes("text/plain")) {
    return resp.text();
  }
  return resp.text();
};

/**
 * Token API
 * @param {*} path
 * @param {*} params
 */
Yao.prototype.Token = function () {
  var token = sessionStorage.getItem("token") || "";
  if (token == "") {
    return this.Cookie("__tk") || "";
  }
  return token;
};

/**
 * Get Cookie
 * @param {*} cookieName
 * @returns
 */
Yao.prototype.Cookie = function (cookieName) {
  var name = cookieName + "=";
  var decodedCookie = decodeURIComponent(document.cookie);
  var cookieArray = decodedCookie.split(";");

  for (var i = 0; i < cookieArray.length; i++) {
    var cookie = cookieArray[i].trim();
    if (cookie.indexOf(name) === 0) {
      return cookie.substring(name.length, cookie.length);
    }
  }
  return null;
};

Yao.prototype.SetCookie = function (cookieName, cookieValue, expireDays) {
  expireDays = expireDays || 30;
  var d = new Date();
  d.setTime(d.getTime() + expireDays * 24 * 60 * 60 * 1000);
  var expires = "expires=" + d.toUTCString();
  document.cookie = `${cookieName}=${cookieValue};${expires};path=/`;
};

Yao.prototype.DeleteCookie = function (cookieName) {
  document.cookie = `${cookieName}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
};

/**
 * Serialize To Query String
 * @param {*} obj
 * @returns
 */
Yao.prototype.Serialize = function (obj) {
  const str: string[] = [];
  for (const p in obj)
    if (obj.hasOwnProperty(p)) {
      str.push(encodeURIComponent(p) + "=" + encodeURIComponent(obj[p]));
    }
  return str.join("&");
};
