"use strict";
// OAuth2 popup landing page for the Try It authorize flow (see js/app.js).
// Reads the token/code (or error) the auth server put in the URL, hands it
// back to the window that opened this popup via postMessage, then closes.
(function () {
  function parseParams(raw) {
    var out = {};
    new URLSearchParams(raw).forEach(function (value, key) {
      out[key] = value;
    });
    return out;
  }

  function run() {
    // Implicit flow returns token params in the hash; authorization code
    // and error responses use the query string. Merge both, query first.
    var query = parseParams(location.search.replace(/^\?/, ""));
    var hash = parseParams(location.hash.replace(/^#/, ""));
    var params = Object.assign({}, query, hash);

    var payload = {
      source: "oauth2-redirect",
      state: params.state || null,
      code: params.code || null,
      accessToken: params.access_token || null,
      tokenType: params.token_type || null,
      expiresIn: params.expires_in || null,
      scope: params.scope || null,
      error: params.error || null,
      errorDescription: params.error_description || null,
      errorUri: params.error_uri || null,
    };

    if (window.opener) {
      try {
        window.opener.postMessage(payload, window.location.origin);
      } catch (e) {
        // Opener may have already navigated away or been closed; nothing
        // more we can do from here.
      }
      window.close();
      return;
    }

    // Opened directly (no opener) - leave a readable message instead of a
    // blank page.
    document.body.textContent = payload.error
      ? "Authorization failed: " + payload.error + (payload.errorDescription ? " (" + payload.errorDescription + ")" : "")
      : "Authorization complete. You can close this window.";
  }

  if (document.readyState !== "loading") run();
  else document.addEventListener("DOMContentLoaded", run);
})();
