(function () {
  const documentURL = window.__DOCS_DOCUMENT_URL__ || "/openapi.json";
  const app = document.getElementById("app");

  if (!app) {
    return;
  }

  if (!window.Scalar || typeof window.Scalar.createApiReference !== "function") {
    app.innerHTML = [
      '<div class="scalar-fallback">',
      '  <h1>Scalar runtime unavailable</h1>',
      '  <p>Open the raw OpenAPI document directly if the CDN script is blocked.</p>',
      '  <p><a href="' + documentURL + '" target="_blank" rel="noreferrer">Open schema</a></p>',
      '</div>'
    ].join("");
    return;
  }

  // Mount the Scalar reference directly so the Scalar provider renders only
  // the API docs view instead of the custom landing-page shell.
  window.Scalar.createApiReference('#app', {
    url: documentURL,
    theme: "purple",
    layout: "modern",
    darkMode: true,
    hideClientButton: false,
    searchHotKey: "k",
    showSidebar: true,
    defaultHttpClient: {
      targetKey: "js",
      clientKey: "fetch"
    }
  });
})();
