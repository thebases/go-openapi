// Fetches a Swagger 2.0 / OpenAPI document and normalizes it into a plain
// model the rest of the app can render without knowing about $ref, tags, etc.

const METHODS = ['get', 'post', 'put', 'patch', 'delete', 'options', 'head', 'trace', 'query'];

/**
 * Normalizes an OpenAPI server URL into an absolute URL without a trailing slash.
 * Falls back to the derived legacy base URL when the server entry is relative.
 */
function normalizeServerUrl(rawUrl, fallbackBaseUrl, variables) {
  const template = String(rawUrl || '').trim();
  if (!template) return '';

  const expanded = template.replace(/\{([^}]+)\}/g, (_, name) => {
    const variable = variables && typeof variables === 'object' ? variables[name] : null;
    if (variable && variable.default !== undefined) return String(variable.default);
    return `{${name}}`;
  });

  try {
    return new URL(expanded, `${fallbackBaseUrl}/`).href.replace(/\/$/, '');
  } catch {
    return expanded.replace(/\/$/, '');
  }
}

function resolveRef(node, spec) {
  if (node && typeof node === 'object' && typeof node.$ref === 'string') {
    const path = node.$ref.replace(/^#\//, '').split('/');
    let target = spec;
    for (const seg of path) target = target?.[decodeURIComponent(seg.replace(/~1/g, '/').replace(/~0/g, '~'))];
    return target ? { ...target } : {};
  }
  return node;
}

function resolveExample(node, spec) {
  if (!node) return null;
  const resolved = resolveRef(node, spec);
  if (!resolved || typeof resolved !== 'object') return null;
  return resolved;
}

function schemaName(node) {
  if (node && typeof node.$ref === 'string') return node.$ref.split('/').pop();
  return null;
}

function firstExampleEntry(examples, spec) {
  if (!examples || typeof examples !== 'object') return null;
  for (const [name, exampleNode] of Object.entries(examples)) {
    const resolved = resolveExample(exampleNode, spec);
    if (!resolved) continue;
    // 3.2 prefers dataValue/serializedValue over the legacy value/externalValue.
    if (resolved.dataValue !== undefined) {
      return { key: name, label: resolved.summary || name, value: resolved.dataValue, isAuto: false };
    }
    if (resolved.serializedValue !== undefined) {
      return { key: name, label: resolved.summary || name, value: resolved.serializedValue, isAuto: false };
    }
    if (resolved.value !== undefined) {
      return { key: name, label: resolved.summary || name, value: resolved.value, isAuto: false };
    }
    if (resolved.externalValue) {
      return { key: name, label: resolved.summary || name, value: resolved.externalValue, isAuto: false };
    }
  }
  return null;
}

function schemaInlineExample(schema) {
  if (!schema) return undefined;
  if (schema.example !== undefined) return schema.example;
  if (schema.default !== undefined) return schema.default;
  if (Array.isArray(schema.enum) && schema.enum.length) return schema.enum[0];
  return undefined;
}

function mediaTypeExample(mediaType, spec) {
  if (!mediaType) return null;
  const named = firstExampleEntry(mediaType.examples, spec);
  if (named) return named;
  if (mediaType.example !== undefined) return { label: 'media example', value: mediaType.example, isAuto: false };
  const schema = resolveRef(mediaType.schema, spec);
  const inline = schemaInlineExample(schema);
  if (inline !== undefined) return { label: 'schema example', value: inline, isAuto: false };
  return null;
}

function parameterExample(parameter, spec) {
  const named = firstExampleEntry(parameter.examples, spec);
  if (named) return named;
  if (parameter.example !== undefined) return { label: 'example', value: parameter.example, isAuto: false };
  const schema = resolveRef(parameter.schema, spec);
  const inline = schemaInlineExample(schema);
  if (inline !== undefined) return { label: 'schema example', value: inline, isAuto: false };
  return null;
}

// Builds a JSON example value from a (possibly $ref'd) JSON-schema-ish node.
function generateExample(node, spec, seen = new Set()) {
  if (!node) return null;
  const name = schemaName(node);
  if (name) {
    if (seen.has(name)) return {};
    seen = new Set(seen).add(name);
  }
  const schema = resolveRef(node, spec);
  if (!schema) return null;
  const inline = schemaInlineExample(schema);
  if (inline !== undefined) return inline;

  if (schema.type === 'array' || schema.items) {
    return [generateExample(schema.items || {}, spec, seen)];
  }
  if (schema.properties || schema.type === 'object') {
    const out = {};
    for (const [key, propSchema] of Object.entries(schema.properties || {})) {
      out[key] = generateExample(propSchema, spec, seen);
    }
    return out;
  }
  switch (schema.type) {
    case 'integer':
      return schema.minimum ?? 0;
    case 'number':
      return schema.minimum ?? 0;
    case 'boolean':
      return true;
    case 'string':
      if (schema.format === 'date-time') return new Date().toISOString();
      if (schema.format === 'date') return new Date().toISOString().slice(0, 10);
      if (schema.format === 'byte') return 'base64==';
      return schema.title || 'string';
    default:
      return null;
  }
}

// Flattens a schema into a list of {path, type, required, description, enum}
// rows for the "Schema" tree view. Depth-limited to avoid runaway recursion.
function schemaRows(node, spec, { depth = 0, prefix = '', seen = new Set(), required = false, selfRow = true } = {}) {
  if (!node || depth > 4) return [];
  const name = schemaName(node);
  if (name) {
    if (seen.has(name)) return [];
    seen = new Set(seen).add(name);
  }
  const schema = resolveRef(node, spec);
  const rows = [];
  const contentNote = schema.contentMediaType || schema.contentEncoding
    ? [schema.contentMediaType, schema.contentEncoding && `encoding:${schema.contentEncoding}`].filter(Boolean).join(', ')
    : '';
  const discriminatorNote = schema.discriminator?.propertyName ? `discriminator: ${schema.discriminator.propertyName}` : '';

  // Tuple validation (JSON Schema 2020-12): fixed-position items before the
  // open-ended `items` schema for any remaining entries.
  if (Array.isArray(schema.prefixItems)) {
    if (selfRow) rows.push({ path: prefix || 'items', type: 'tuple', required, description: schema.description || '' });
    schema.prefixItems.forEach((item, index) => {
      rows.push(...schemaRows(item, spec, { depth: depth + 1, prefix: `${prefix}[${index}]`, seen, required: false }));
    });
    if (schema.items) rows.push(...schemaRows(schema.items, spec, { depth: depth + 1, prefix: `${prefix}[n]`, seen, required: false }));
    return rows;
  }
  if (schema.type === 'array' || schema.items) {
    if (selfRow) rows.push({ path: prefix || 'items', type: `array`, required, description: schema.description || '' });
    rows.push(...schemaRows(schema.items || {}, spec, { depth: depth + 1, prefix: `${prefix}[]`, seen, required: false }));
    return rows;
  }
  if (schema.properties || schema.type === 'object' || schema.patternProperties) {
    const requiredSet = new Set(schema.required || []);
    for (const [key, propSchema] of Object.entries(schema.properties || {})) {
      const resolved = resolveRef(propSchema, spec);
      const path = prefix ? `${prefix}.${key}` : key;
      rows.push({
        path,
        type: resolved.type || (resolved.$ref ? schemaName(propSchema) : 'any') || 'any',
        format: resolved.format || '',
        enum: resolved.enum,
        required: requiredSet.has(key),
        description: resolved.description || '',
      });
      if (resolved.type === 'object' || resolved.properties || resolved.type === 'array' || resolved.items) {
        rows.push(...schemaRows(propSchema, spec, { depth: depth + 1, prefix: path, seen, required: requiredSet.has(key), selfRow: false }));
      }
    }
    for (const [pattern, propSchema] of Object.entries(schema.patternProperties || {})) {
      const resolved = resolveRef(propSchema, spec);
      rows.push({
        path: prefix ? `${prefix}.<${pattern}>` : `<${pattern}>`,
        type: resolved.type || 'any',
        format: resolved.format || '',
        required: false,
        description: resolved.description || '',
      });
    }
    return rows;
  }
  rows.push({
    path: prefix || 'value',
    type: schema.type || 'any',
    format: schema.format || '',
    enum: schema.enum,
    required,
    description: [schema.description, contentNote, discriminatorNote].filter(Boolean).join(' — '),
  });
  return rows;
}

function paramRow(p) {
  const example = parameterExample(p, this);
  return {
    name: p.name,
    in: p.in,
    required: !!p.required,
    type: p.type || (p.schema && p.schema.type) || 'string',
    format: p.format || '',
    description: p.description || '',
    default: p.default,
    enum: p.enum,
    example: example ? example.value : undefined,
    exampleLabel: example ? example.label : '',
  };
}

function normalizedSecurityDefinitions(spec) {
  return spec.securityDefinitions || spec.components?.securitySchemes || {};
}

function buildSecurity(op, spec) {
  const defs = normalizedSecurityDefinitions(spec);
  const reqs = op.security || spec.security || [];
  const out = [];
  for (const req of reqs) {
    for (const key of Object.keys(req)) {
      const def = defs[key];
      if (def) out.push({ key, ...def });
    }
  }
  return out;
}

function normalizeOperation(spec, path, method, opRaw) {
  const parameters = { path: [], query: [], header: [], formData: [], body: null };
  for (const raw of opRaw.parameters || []) {
    const p = resolveRef(raw, spec);
    if (p.in === 'body') {
      const content = { schema: p.schema };
      const example = mediaTypeExample(content, spec);
      parameters.body = {
        required: !!p.required,
        schema: p.schema,
        description: p.description || '',
        contentType: 'application/json',
        example: example ? example.value : generateExample(p.schema, spec),
        exampleLabel: example ? example.label : 'auto',
        exampleIsAuto: !example,
      };
    } else if (parameters[p.in]) {
      parameters[p.in].push(paramRow.call(spec, p));
    }
  }

  if (opRaw.requestBody) {
    const requestBody = resolveRef(opRaw.requestBody, spec) || {};
    const [contentType, content] = Object.entries(requestBody.content || {})[0] || [];
    if (contentType && content) {
      const example = mediaTypeExample(content, spec);
      parameters.body = {
        required: !!requestBody.required,
        schema: content.schema || null,
        description: requestBody.description || '',
        contentType,
        encoding: content.encoding || null,
        itemSchema: content.itemSchema || null,
        example: example ? example.value : generateExample(content.schema, spec),
        exampleLabel: example ? example.label : 'auto',
        exampleIsAuto: !example,
      };
    }
  }

  const responses = Object.entries(opRaw.responses || {})
    .map(([status, rawResponse]) => {
      const response = resolveRef(rawResponse, spec) || {};
      const [contentType, content] = Object.entries(response.content || {})[0] || [];
      const example = content ? mediaTypeExample(content, spec) : null;
      return {
        status,
        summary: response.summary || '',
        description: response.description || '',
        contentType: contentType || 'application/json',
        schema: content?.schema || response.schema || null,
        encoding: content?.encoding || null,
        headers: normalizeHeaders(response.headers, spec),
        links: normalizeLinks(response.links, spec),
        example: example ? example.value : (content?.schema ? generateExample(content.schema, spec) : null),
        exampleLabel: example ? example.label : 'auto',
        exampleIsAuto: !example && !!content?.schema,
      };
    })
    .sort((a, b) => {
      const na = status2num(a.status), nb = status2num(b.status);
      return na - nb;
    });

  const callbacks = normalizeCallbacks(opRaw.callbacks, spec);

  const method_up = method.toUpperCase();
  const id = (opRaw.operationId || `${method}-${path}`).replace(/[^a-zA-Z0-9]+/g, '-').replace(/^-+|-+$/g, '').toLowerCase();

  return {
    id,
    method: method_up,
    path,
    tag: (opRaw.tags && opRaw.tags[0]) || 'default',
    summary: opRaw.summary || opRaw.operationId || `${method_up} ${path}`,
    description: opRaw.description || '',
    deprecated: !!opRaw.deprecated,
    produces: opRaw.produces || spec.produces || ['application/json'],
    consumes: opRaw.consumes || spec.consumes || (parameters.body ? ['application/json'] : []),
    parameters,
    responses,
    callbacks,
    externalDocs: opRaw.externalDocs || null,
    security: buildSecurity(opRaw, spec),
  };
}

// Header Object shares almost all fields with Parameter (minus name/in).
function normalizeHeaders(headers, spec) {
  if (!headers || typeof headers !== 'object') return [];
  return Object.entries(headers).map(([name, raw]) => {
    const h = resolveRef(raw, spec) || {};
    const example = parameterExample({ ...h, in: 'header' }, spec);
    return {
      name,
      required: !!h.required,
      deprecated: !!h.deprecated,
      type: h.schema?.type || 'string',
      description: h.description || '',
      example: example ? example.value : undefined,
    };
  });
}

function normalizeLinks(links, spec) {
  if (!links || typeof links !== 'object') return [];
  return Object.entries(links).map(([name, raw]) => {
    const l = resolveRef(raw, spec) || {};
    return {
      name,
      operationId: l.operationId || '',
      operationRef: l.operationRef || '',
      description: l.description || '',
      parameters: l.parameters || {},
    };
  });
}

// Callback Object: name -> Path Item keyed by runtime expression. Flattened
// into {name, expression, method, description}[] for a simple list view.
function normalizeCallbacks(callbacks, spec) {
  if (!callbacks || typeof callbacks !== 'object') return [];
  const out = [];
  for (const [name, callbackMap] of Object.entries(callbacks)) {
    const resolvedMap = resolveRef(callbackMap, spec) || {};
    for (const [expression, pathItemRaw] of Object.entries(resolvedMap)) {
      const pathItem = resolveRef(pathItemRaw, spec) || {};
      for (const method of METHODS) {
        if (!pathItem[method]) continue;
        out.push({
          name,
          expression,
          method: method.toUpperCase(),
          summary: pathItem[method].summary || '',
          description: pathItem[method].description || '',
        });
      }
    }
  }
  return out;
}

function status2num(status) {
  if (status === 'default') return 999;
  const n = parseInt(status, 10);
  return Number.isNaN(n) ? 998 : n;
}

async function loadOpenApiDoc(url) {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`Failed to fetch spec: HTTP ${res.status}`);
  const spec = await res.json();

  const scheme = (spec.schemes && spec.schemes.includes('https')) ? 'https' : (spec.schemes?.[0] || 'https');
  const host = spec.host || location.host;
  const basePath = spec.basePath || '';
  const derivedBaseUrl = `${scheme}://${host}${basePath}`.replace(/\/$/, '');
  const servers = Array.isArray(spec.servers) && spec.servers.length
    ? spec.servers
      .map((server) => ({
        name: server?.name || '',
        url: normalizeServerUrl(server?.url, derivedBaseUrl, server?.variables),
        description: String(server?.description || '').trim(),
      }))
      .filter((server) => server.url)
    : [{ name: '', url: derivedBaseUrl, description: '' }];
  const baseUrl = servers[0]?.url || derivedBaseUrl;

  const tagOrder = (spec.tags || []).map(t => t.name);
  const tagMeta = new Map((spec.tags || []).map(t => [t.name, t]));

  const operations = [];
  for (const [path, rawPathItem] of Object.entries(spec.paths || {})) {
    const pathItem = resolveRef(rawPathItem, spec) || {};
    for (const method of METHODS) {
      if (pathItem[method]) operations.push(normalizeOperation(spec, path, method, pathItem[method]));
    }
    // 3.2 additionalOperations: custom HTTP verbs (e.g. COPY, LOCK) keyed by name.
    for (const [verb, opRaw] of Object.entries(pathItem.additionalOperations || {})) {
      operations.push(normalizeOperation(spec, path, verb.toLowerCase(), opRaw));
    }
  }

  const webhooks = [];
  for (const [name, rawPathItem] of Object.entries(spec.webhooks || {})) {
    const pathItem = resolveRef(rawPathItem, spec) || {};
    for (const method of METHODS) {
      if (pathItem[method]) webhooks.push(normalizeOperation(spec, name, method, pathItem[method]));
    }
  }

  const groupNames = [...new Set([...tagOrder, ...operations.map(o => o.tag)])];
  const groups = groupNames.map(name => {
    const meta = tagMeta.get(name);
    return {
      id: name.replace(/[^a-zA-Z0-9]+/g, '-').toLowerCase(),
      name,
      summary: meta?.summary || '',
      description: meta?.description || '',
      parent: meta?.parent || '',
      kind: meta?.kind || '',
      items: operations.filter(o => o.tag === name),
    };
  }).filter(g => g.items.length);

  return {
    spec,
    info: spec.info || {},
    baseUrl,
    derivedBaseUrl,
    servers,
    securityDefinitions: normalizedSecurityDefinitions(spec),
    groups,
    webhooks,
    resolveRef: (node) => resolveRef(node, spec),
    generateExample: (node) => generateExample(node, spec),
    schemaRows: (node) => schemaRows(node, spec),
    findOperation: (id) => operations.find(o => o.id === id) || webhooks.find(o => o.id === id),
    firstOperation: () => operations[0],
  };
}
