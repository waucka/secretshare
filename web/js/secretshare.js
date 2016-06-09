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

  // Parses a `secretshare config` string and stores it in a cookie.
  //
  // In case of success, we run `successCb`, and in case of failure we run
  // `failureCb`. The latter receives an exception as its only argument.
  this.setConfig = function(configCmd, successCb, failureCb) {
    var pieces = configCmd.split(" ");
    var config = pieces.reduce(function(prev, cur) {
      switch (cur) {
        case "--endpoint":
          prev.keyToSet = "endpointBaseUrl";
          break;
        case "--bucket":
          prev.keyToSet = "bucket";
          break;
        case "--bucket-region":
          prev.keyToSet = "bucketRegion";
          break;
        case "--auth-key":
          prev.keyToSet = "authKey";
          break;
        default:
          if (prev.keyToSet !== undefined) {
            prev[prev.keyToSet] = cur;
          }
          delete prev.keyToSet;
          break;
      }
      return prev;
    }, {});

    var oldConfig = this.config;
    this.config = config;
    return this.checkConfig(
      function(resp) {
        docCookies.setItem("secretshareConfig", JSON.stringify(config));
        loadConfig();
        successCb();
      },
      function(resp, err) {
        this.config = oldConfig;
        failureCb(err);
      },
      config
    );
  }

  // Checks the config for problems and attempts to connect to the server.
  //
  // In case of success, we run `successCb`, and in case of failure we run
  // `failureCb`. In both cases, the callback is passed the data object we
  // received from the server. `failureCb` also receives an Error object.
  //
  // `config` may optionally be passed; otherwise we will load the config
  // from the client's cookie.
  this.checkConfig = function(successCb, failureCb) {
    var missing = ["endpointBaseUrl", "bucket", "bucketRegion", "authKey"].reduce(function(prev, cur) {
      if (!this.config.hasOwnProperty(cur)) {
        return prev.concat(cur);
      }
      return prev;
    }, []);

    if (missing.length !== 0) {
      return failureCb({}, new Error("Config is missing required option(s): [" + missing.join(", ") + "]"));
    }

    this.apiPing(function(data) {
      if (data.pong) {
        return successCb(data);
      }
      return failureCb(data, new Error("Received improper ping response from SecretShare server"));
    }, failureCb);

    return;
  };

  // Encodes binary data for human copy/pasting.
  this.encodeForHuman = function(bytes) {
    var str = String.fromCharCode.apply(null, bytes);
    var b64 = btoa(str);
    return b64.replace("/", "_").replace("=", "");
  };

  // Generates a random encryption key.
  //
  // Returns the key as both an array of ints and a human-readable string.
  this.generateKey = function() {
    var keyBytes = new Uint8Array(32);
    window.crypto.getRandomValues(keyBytes);
    return [keyBytes, this.encodeForHuman(keyBytes)];
  };

  // Generates an S3 object ID corresponding to the given encryption key.
  //
  // `successCb` receives as its only argument the (string) object ID.
  this.deriveId = function(keyBytes, successCb) {
    var keyBuf = new ArrayBuffer(32);
    var keyView = new Uint8Array(keyBuf);
    for (var i=0; i<32; i++) {
      keyView[i] = keyBytes[i];
    }

    var hashPromise = crypto.subtle.digest("SHA-256", keyBuf);
    return hashPromise.then(function(hashBuf) {
      console.log(new DataView(hashBuf).getFloat64(0));
      var hashView = new DataView(hashBuf);
      var hashBytes = new Uint8Array(32);
      for (var i=0; i<32; i++) {
        hashBytes[i] = hashView.getUint8(i);
      }
      var hashHuman = this.encodeForHuman(hashBytes);
      successCb(hashHuman);
    });
  };

  // Uploads the given file to the SecretShare server.
  //
  // On success, `successCb` is called with the `secretshare receive ...`
  // string as its only argument. On failure, `failureCb` is called with
  // an Error object as its only argument.
  this.uploadFile = function(fileName, fileContents, successCb, failureCb) {
    try {
      var keyBytes, keyHuman;
      [keyBytes, keyHuman] = this.generateKey();

      this.deriveId(keyBytes, function(objId) {
        console.log("x-bravo", objId);
      });
    } catch(e) {
      return failureCb(e);
    }
    return successCb("secretshare receive " + keyHuman);
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
