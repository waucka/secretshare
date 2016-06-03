/*\
|*|
|*|  :: cookies.js ::
|*|
|*|  A complete cookies reader/writer framework with full unicode support.
|*|
|*|  Revision #1 - September 4, 2014
|*|
|*|  https://developer.mozilla.org/en-US/docs/Web/API/document.cookie
|*|  https://developer.mozilla.org/User:fusionchess
|*|
|*|  This framework is released under the GNU Public License, version 3 or later.
|*|  http://www.gnu.org/licenses/gpl-3.0-standalone.html
|*|
|*|  Syntaxes:
|*|
|*|  * docCookies.setItem(name, value[, end[, path[, domain[, secure]]]])
|*|  * docCookies.getItem(name)
|*|  * docCookies.removeItem(name[, path[, domain]])
|*|  * docCookies.hasItem(name)
|*|  * docCookies.keys()
|*|
\*/

var docCookies = {
  getItem: function (sKey) {
    if (!sKey) { return null; }
    return decodeURIComponent(document.cookie.replace(new RegExp("(?:(?:^|.*;)\\s*" + encodeURIComponent(sKey).replace(/[\-\.\+\*]/g, "\\$&") + "\\s*\\=\\s*([^;]*).*$)|^.*$"), "$1")) || null;
  },
  setItem: function (sKey, sValue, vEnd, sPath, sDomain, bSecure) {
    if (!sKey || /^(?:expires|max\-age|path|domain|secure)$/i.test(sKey)) { return false; }
    var sExpires = "";
    if (vEnd) {
      switch (vEnd.constructor) {
        case Number:
          sExpires = vEnd === Infinity ? "; expires=Fri, 31 Dec 9999 23:59:59 GMT" : "; max-age=" + vEnd;
          break;
        case String:
          sExpires = "; expires=" + vEnd;
          break;
        case Date:
          sExpires = "; expires=" + vEnd.toUTCString();
          break;
      }
    }
    document.cookie = encodeURIComponent(sKey) + "=" + encodeURIComponent(sValue) + sExpires + (sDomain ? "; domain=" + sDomain : "") + (sPath ? "; path=" + sPath : "") + (bSecure ? "; secure" : "");
    return true;
  },
  removeItem: function (sKey, sPath, sDomain) {
    if (!this.hasItem(sKey)) { return false; }
    document.cookie = encodeURIComponent(sKey) + "=; expires=Thu, 01 Jan 1970 00:00:00 GMT" + (sDomain ? "; domain=" + sDomain : "") + (sPath ? "; path=" + sPath : "");
    return true;
  },
  hasItem: function (sKey) {
    if (!sKey) { return false; }
    return (new RegExp("(?:^|;\\s*)" + encodeURIComponent(sKey).replace(/[\-\.\+\*]/g, "\\$&") + "\\s*\\=")).test(document.cookie);
  },
  keys: function () {
    var aKeys = document.cookie.replace(/((?:^|\s*;)[^\=]+)(?=;|$)|^\s*|\s*(?:\=[^;]*)?(?:\1|$)/g, "").split(/\s*(?:\=[^;]*)?;\s*/);
    for (var nLen = aKeys.length, nIdx = 0; nIdx < nLen; nIdx++) { aKeys[nIdx] = decodeURIComponent(aKeys[nIdx]); }
    return aKeys;
  }
};

function SecretshareClient() {
  // Loads secretshare config from cookie. Returns true on success.
  this.loadConfig = function() {
    var configJson = docCookies.getItem("secretshareConfig");
    if (configJson === null) {
      console.log("No secretshareConfig cookie defined");
      return false;
    }
    this.config = JSON.parse(configJson);
    return true;
  };

  // Checks the config for problems and attempts to connect to the server.
  //
  // In case of success, we run `successCb`, and in case of failure we run
  // `failureCb`. In both cases, the callback is passed as its only argument
  // the data object we received from the server.
  this.checkConfig = function(successCb, failureCb) {
    if (this.config === undefined) {
      this.loadConfig();
    }

    var ret = ["endpointBaseUrl", "bucket", "bucketRegion", "authKey"].reduce(function(prev, cur) {
      if (!prev) {
        return false;
      }
      if (!this.config.hasOwnProperty(cur)) {
        console.log("No '" + cur + "' key in secretshareConfig cookie");
        return false;
      }
      return true;
    }, true);
    if (!ret) {
      return failureCb({});
    }

    this.apiPing(function(data) {
      if (data.pong) {
        return successCb(data);
      }
      return failureCb(data);
    }, failureCb);

    return;
  };

  this.apiPing = function(successCb, failureCb) {
    this.apiCall("ping", {secret_key: this.config.authKey}, successCb, failureCb);
  };

  this.apiCall = function(uri, reqBody, successCb, failureCb) {
    $.ajax({
      type: "POST",
      url: [this.config.endpointBaseUrl, uri].join("/"),
      data: JSON.stringify(reqBody),
      contentType: "application/json",
      success: successCb,
      error: failureCb
    });
  };

  return this;
};
