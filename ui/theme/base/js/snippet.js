// Generates example request snippets directly in the browser without pulling
// in third-party runtime JavaScript.

const LANGUAGES = [
  { id: 'curl', label: 'cURL' },
  { id: 'javascript', label: 'JavaScript' },
  { id: 'node', label: 'Node.js' },
  { id: 'python', label: 'Python' },
  { id: 'go', label: 'Go' },
  { id: 'java', label: 'Java' },
  { id: 'csharp', label: 'C#' },
  { id: 'php', label: 'PHP' },
  { id: 'ruby', label: 'Ruby' },
  { id: 'c', label: 'C' },
  { id: 'swift', label: 'Swift' },
  { id: 'objc', label: 'Objective-C' },
  { id: 'kotlin', label: 'Kotlin' },
  { id: 'rust', label: 'Rust' },
  { id: 'powershell', label: 'PowerShell' },
  { id: 'dart', label: 'Dart' },
  { id: 'lua', label: 'Lua' },
];

function buildRequestUrl(req) {
  const url = new URL(req.url);
  (req.query || []).forEach(([name, value]) => {
    if (name) url.searchParams.set(name, String(value || ''));
  });
  return url.toString();
}

function requestHeaders(req, includeContentType) {
  const headers = [];
  const source = req.headers || [];
  source.forEach(([name, value]) => {
    if (name) headers.push([name, String(value == null ? '' : value)]);
  });
  if (includeContentType && req.bodyText && !hasHeader(headers, 'Content-Type')) {
    headers.push(['Content-Type', req.mimeType || 'application/json']);
  }
  return headers;
}

function hasHeader(headers, expectedName) {
  const target = expectedName.toLowerCase();
  return headers.some(function (entry) {
    return String(entry[0] || '').toLowerCase() === target;
  });
}

function indent(text, prefix) {
  return String(text || '').split('\n').map(function (line) {
    return prefix + line;
  }).join('\n');
}

function escapeSingleQuoted(value) {
  return String(value).replace(/\\/g, '\\\\').replace(/'/g, "\\'");
}

function escapeDoubleQuoted(value) {
  return String(value).replace(/\\/g, '\\\\').replace(/"/g, '\\"');
}

function escapeBackticks(value) {
  return String(value).replace(/\\/g, '\\\\').replace(/`/g, '\\`').replace(/\$\{/g, '\\${');
}

function escapeGoRaw(value) {
  return String(value).replace(/`/g, "` + \"`\" + `");
}

function escapePythonTriple(value) {
  return String(value).replace(/\\/g, '\\\\').replace(/"""/g, '\\"\\"\\"');
}

function escapePhpSingle(value) {
  return String(value).replace(/\\/g, '\\\\').replace(/'/g, "\\'");
}

function escapeRubySingle(value) {
  return String(value).replace(/\\/g, '\\\\').replace(/'/g, "\\\\'");
}

function escapeC(value) {
  return String(value)
    .replace(/\\/g, '\\\\')
    .replace(/"/g, '\\"')
    .replace(/\r/g, '\\r')
    .replace(/\n/g, '\\n');
}

function escapePowerShell(value) {
  return String(value).replace(/`/g, '``').replace(/"/g, '`"');
}

function escapeLuaLong(value) {
  return String(value).replace(/\]=\]/g, ']==] .. "]=\" .. [=[');
}

function objectLiteral(headers, options) {
  const quote = options && options.quote === 'double' ? '"' : "'";
  const pairs = headers.map(function (entry) {
    const key = quote === '"' ? escapeDoubleQuoted(entry[0]) : escapeSingleQuoted(entry[0]);
    const value = quote === '"' ? escapeDoubleQuoted(entry[1]) : escapeSingleQuoted(entry[1]);
    return quote + key + quote + ': ' + quote + value + quote;
  });
  return pairs.join(',\n');
}

function generateCurl(req, url, headers) {
  const parts = ['curl --request ' + req.method];
  headers.forEach(function (entry) {
    parts.push("--header '" + escapeSingleQuoted(entry[0] + ': ' + entry[1]) + "'");
  });
  if (req.bodyText) {
    parts.push("--data '" + escapeSingleQuoted(req.bodyText) + "'");
  }
  parts.push("'" + escapeSingleQuoted(url) + "'");
  return parts.join(' \\\n  ');
}

function generateJavaScript(req, url, headers) {
  const lines = [
    'const url = ' + "'" + escapeSingleQuoted(url) + "';",
    'const options = {',
    "  method: '" + escapeSingleQuoted(req.method) + "',",
  ];
  if (headers.length) {
    lines.push('  headers: {');
    lines.push(indent(objectLiteral(headers), '    '));
    lines.push('  },');
  }
  if (req.bodyText) {
    lines.push('  body: ' + '`' + escapeBackticks(req.bodyText) + '`,');
  }
  lines.push('};', '', 'const response = await fetch(url, options);', 'const data = await response.text();', 'console.log(response.status, data);');
  return lines.join('\n');
}

function generateNode(req, url, headers) {
  return generateJavaScript(req, url, headers);
}

function generatePython(req, url, headers) {
  const lines = ['import requests', '', "url = '" + escapeSingleQuoted(url) + "'"];
  if (headers.length) {
    lines.push('headers = {');
    headers.forEach(function (entry) {
      lines.push("    '" + escapeSingleQuoted(entry[0]) + "': '" + escapeSingleQuoted(entry[1]) + "',");
    });
    lines.push('}');
  } else {
    lines.push('headers = {}');
  }
  if (req.bodyText) {
    lines.push('', 'payload = """' + escapePythonTriple(req.bodyText) + '"""');
    lines.push("response = requests.request('" + escapeSingleQuoted(req.method) + "', url, headers=headers, data=payload)");
  } else {
    lines.push('', "response = requests.request('" + escapeSingleQuoted(req.method) + "', url, headers=headers)");
  }
  lines.push('print(response.status_code)', 'print(response.text)');
  return lines.join('\n');
}

function generateGo(req, url, headers) {
  const lines = [
    'package main',
    '',
    'import (',
    '  "fmt"',
    '  "io"',
    '  "net/http"',
  ];
  if (req.bodyText) {
    lines.push('  "strings"');
  }
  lines.push(')', '', 'func main() {');
  if (req.bodyText) {
    lines.push('  body := strings.NewReader(`' + escapeGoRaw(req.bodyText) + '`)');
    lines.push('  req, err := http.NewRequest("' + req.method + '", "' + url + '", body)');
  } else {
    lines.push('  req, err := http.NewRequest("' + req.method + '", "' + url + '", nil)');
  }
  lines.push('  if err != nil {', '    panic(err)', '  }');
  headers.forEach(function (entry) {
    lines.push('  req.Header.Set("' + escapeDoubleQuoted(entry[0]) + '", "' + escapeDoubleQuoted(entry[1]) + '")');
  });
  lines.push('', '  res, err := http.DefaultClient.Do(req)', '  if err != nil {', '    panic(err)', '  }', '  defer res.Body.Close()', '', '  data, err := io.ReadAll(res.Body)', '  if err != nil {', '    panic(err)', '  }', '', '  fmt.Println(res.StatusCode)', '  fmt.Println(string(data))', '}');
  return lines.join('\n');
}

function generateJava(req, url, headers) {
  const lines = [
    'OkHttpClient client = new OkHttpClient();',
    '',
    'Request.Builder builder = new Request.Builder()',
    '  .url("' + escapeDoubleQuoted(url) + '")',
  ];
  headers.forEach(function (entry) {
    lines.push('  .addHeader("' + escapeDoubleQuoted(entry[0]) + '", "' + escapeDoubleQuoted(entry[1]) + '")');
  });
  if (req.bodyText) {
    lines.push(';', '', 'MediaType mediaType = MediaType.parse("' + escapeDoubleQuoted(req.mimeType || 'application/json') + '");', 'RequestBody body = RequestBody.create("""' + escapeDoubleQuoted(req.bodyText) + '""", mediaType);', 'Request request = builder.method("' + req.method + '", body).build();');
  } else {
    lines.push(';', '', 'Request request = builder.method("' + req.method + '", null).build();');
  }
  lines.push('', 'try (Response response = client.newCall(request).execute()) {', '  System.out.println(response.code());', '  System.out.println(response.body() != null ? response.body().string() : "");', '}');
  return lines.join('\n');
}

function generateCSharp(req, url, headers) {
  const methodName = req.method.charAt(0) + req.method.slice(1).toLowerCase();
  const lines = [
    'using var client = new HttpClient();',
    'using var request = new HttpRequestMessage(HttpMethod.' + methodName + ', "' + escapeDoubleQuoted(url) + '");',
  ];
  headers.forEach(function (entry) {
    lines.push('request.Headers.TryAddWithoutValidation("' + escapeDoubleQuoted(entry[0]) + '", "' + escapeDoubleQuoted(entry[1]) + '");');
  });
  if (req.bodyText) {
    lines.push('request.Content = new StringContent("""' + escapeDoubleQuoted(req.bodyText) + '""", Encoding.UTF8, "' + escapeDoubleQuoted(req.mimeType || 'application/json') + '");');
  }
  lines.push('', 'using var response = await client.SendAsync(request);', 'var body = await response.Content.ReadAsStringAsync();', 'Console.WriteLine((int)response.StatusCode);', 'Console.WriteLine(body);');
  return lines.join('\n');
}

function generatePhp(req, url, headers) {
  const lines = [
    '$ch = curl_init();',
    'curl_setopt_array($ch, [',
    "    CURLOPT_URL => '" + escapePhpSingle(url) + "',",
    "    CURLOPT_CUSTOMREQUEST => '" + escapePhpSingle(req.method) + "',",
    '    CURLOPT_RETURNTRANSFER => true,',
  ];
  if (headers.length) {
    lines.push('    CURLOPT_HTTPHEADER => [');
    headers.forEach(function (entry) {
      lines.push("        '" + escapePhpSingle(entry[0] + ': ' + entry[1]) + "',");
    });
    lines.push('    ],');
  }
  if (req.bodyText) {
    lines.push("    CURLOPT_POSTFIELDS => '" + escapePhpSingle(req.bodyText) + "',");
  }
  lines.push(']);', '', '$response = curl_exec($ch);', 'echo curl_getinfo($ch, CURLINFO_RESPONSE_CODE) . PHP_EOL;', 'echo $response;', 'curl_close($ch);');
  return lines.join('\n');
}

function generateRuby(req, url, headers) {
  const klass = req.method.charAt(0) + req.method.slice(1).toLowerCase();
  const lines = ['require "uri"', 'require "net/http"', '', "uri = URI('" + escapeRubySingle(url) + "')", 'http = Net::HTTP.new(uri.host, uri.port)', 'http.use_ssl = uri.scheme == "https"', '', 'request = Net::HTTP::' + klass + '.new(uri)'];
  headers.forEach(function (entry) {
    lines.push("request['" + escapeRubySingle(entry[0]) + "'] = '" + escapeRubySingle(entry[1]) + "'");
  });
  if (req.bodyText) {
    lines.push("request.body = '" + escapeRubySingle(req.bodyText) + "'");
  }
  lines.push('', 'response = http.request(request)', 'puts response.code', 'puts response.body');
  return lines.join('\n');
}

function generateC(req, url, headers) {
  const lines = [
    '#include <curl/curl.h>',
    '',
    'int main(void) {',
    '  CURL *curl = curl_easy_init();',
    '  if (!curl) return 1;',
    '',
    '  struct curl_slist *headers = NULL;',
  ];
  headers.forEach(function (entry) {
    lines.push('  headers = curl_slist_append(headers, "' + escapeC(entry[0] + ': ' + entry[1]) + '");');
  });
  lines.push('  curl_easy_setopt(curl, CURLOPT_URL, "' + escapeC(url) + '");');
  lines.push('  curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "' + escapeC(req.method) + '");');
  if (headers.length) {
    lines.push('  curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);');
  }
  if (req.bodyText) {
    lines.push('  curl_easy_setopt(curl, CURLOPT_POSTFIELDS, "' + escapeC(req.bodyText) + '");');
  }
  lines.push('', '  curl_easy_perform(curl);', '  curl_slist_free_all(headers);', '  curl_easy_cleanup(curl);', '  return 0;', '}');
  return lines.join('\n');
}

function generateSwift(req, url, headers) {
  const lines = [
    'var request = URLRequest(url: URL(string: "' + escapeDoubleQuoted(url) + '")!)',
    'request.httpMethod = "' + escapeDoubleQuoted(req.method) + '"',
  ];
  headers.forEach(function (entry) {
    lines.push('request.setValue("' + escapeDoubleQuoted(entry[1]) + '", forHTTPHeaderField: "' + escapeDoubleQuoted(entry[0]) + '")');
  });
  if (req.bodyText) {
    lines.push('request.httpBody = """', req.bodyText, '""".data(using: .utf8)');
  }
  lines.push('', 'URLSession.shared.dataTask(with: request) { data, response, error in', '  if let error = error {', '    print(error)', '    return', '  }', '  let status = (response as? HTTPURLResponse)?.statusCode ?? 0', '  print(status)', '  print(String(data: data ?? Data(), encoding: .utf8) ?? "")', '}.resume()');
  return lines.join('\n');
}

function generateObjC(req, url, headers) {
  const lines = [
    'NSMutableURLRequest *request = [NSMutableURLRequest requestWithURL:[NSURL URLWithString:@"' + escapeDoubleQuoted(url) + '"]];',
    'request.HTTPMethod = @"' + escapeDoubleQuoted(req.method) + '";',
  ];
  headers.forEach(function (entry) {
    lines.push('[request setValue:@"' + escapeDoubleQuoted(entry[1]) + '" forHTTPHeaderField:@"' + escapeDoubleQuoted(entry[0]) + '"];');
  });
  if (req.bodyText) {
    lines.push('request.HTTPBody = [@"' + escapeDoubleQuoted(req.bodyText) + '" dataUsingEncoding:NSUTF8StringEncoding];');
  }
  lines.push('', 'NSURLSessionDataTask *task = [[NSURLSession sharedSession] dataTaskWithRequest:request', ' completionHandler:^(NSData *data, NSURLResponse *response, NSError *error) {', '  if (error) {', '    NSLog(@"%@", error);', '    return;', '  }', '  NSInteger status = [(NSHTTPURLResponse *)response statusCode];', '  NSString *body = [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];', '  NSLog(@"%ld", (long)status);', '  NSLog(@"%@", body);', '}];', '[task resume];');
  return lines.join('\n');
}

function generateKotlin(req, url, headers) {
  const lines = ['val client = OkHttpClient()'];
  if (req.bodyText) {
    lines.push('val body = """', req.bodyText, '""".trimIndent().toRequestBody("' + escapeDoubleQuoted(req.mimeType || 'application/json') + '".toMediaType())');
  }
  lines.push('val request = Request.Builder()', '  .url("' + escapeDoubleQuoted(url) + '")');
  headers.forEach(function (entry) {
    lines.push('  .addHeader("' + escapeDoubleQuoted(entry[0]) + '", "' + escapeDoubleQuoted(entry[1]) + '")');
  });
  lines.push(req.bodyText ? '  .method("' + req.method + '", body)' : '  .method("' + req.method + '", null)', '  .build()', '', 'client.newCall(request).execute().use { response ->', '  println(response.code)', '  println(response.body?.string().orEmpty())', '}');
  return lines.join('\n');
}

function generateRust(req, url, headers) {
  const lines = ['let client = reqwest::Client::new();', 'let response = client', '  .' + req.method.toLowerCase() + '("' + escapeDoubleQuoted(url) + '")'];
  headers.forEach(function (entry) {
    lines.push('  .header("' + escapeDoubleQuoted(entry[0]) + '", "' + escapeDoubleQuoted(entry[1]) + '")');
  });
  if (req.bodyText) {
    lines.push('  .body(r#"' + req.bodyText.replace(/"#/g, '"#""#') + '"#)');
  }
  lines.push('  .send()', '  .await?;', '', 'println!("{}", response.status());', 'println!("{}", response.text().await?);');
  return lines.join('\n');
}

function generatePowerShell(req, url, headers) {
  const lines = ['$headers = @{'];
  headers.forEach(function (entry) {
    lines.push('  "' + escapePowerShell(entry[0]) + '" = "' + escapePowerShell(entry[1]) + '"');
  });
  lines.push('}', '');
  if (req.bodyText) {
    lines.push('$body = @"', req.bodyText, '"@', '');
  }
  lines.push('$response = Invoke-RestMethod -Method ' + req.method + ' -Uri "' + escapePowerShell(url) + '" -Headers $headers' + (req.bodyText ? ' -Body $body -ContentType "' + escapePowerShell(req.mimeType || 'application/json') + '"' : ''), '$response | ConvertTo-Json -Depth 10');
  return lines.join('\n');
}

function generateDart(req, url, headers) {
  const lines = ["import 'package:http/http.dart' as http;", '', "final uri = Uri.parse('" + escapeSingleQuoted(url) + "');", 'final headers = <String, String>{'];
  headers.forEach(function (entry) {
    lines.push("  '" + escapeSingleQuoted(entry[0]) + "': '" + escapeSingleQuoted(entry[1]) + "',");
  });
  lines.push('};', '');
  if (req.bodyText) {
    lines.push("final response = await http." + req.method.toLowerCase() + "(uri, headers: headers, body: '''" + req.bodyText.replace(/'''/g, "'' '") + "''');");
  } else {
    lines.push("final response = await http." + req.method.toLowerCase() + "(uri, headers: headers);");
  }
  lines.push("print('Status: ${response.statusCode}');", 'print(response.body);');
  return lines.join('\n');
}

function generateLua(req, url, headers) {
  const lines = ['local http = require("socket.http")', 'local ltn12 = require("ltn12")', '', 'local response_body = {}', 'local headers = {'];
  headers.forEach(function (entry) {
    lines.push('  ["' + escapeDoubleQuoted(entry[0]) + '"] = "' + escapeDoubleQuoted(entry[1]) + '",');
  });
  lines.push('}', '');
  if (req.bodyText) {
    lines.push('local body = [==[' + escapeLuaLong(req.bodyText) + ']==]', '');
    lines.push('headers["Content-Length"] = tostring(#body)');
  }
  lines.push('local _, code = http.request{', '  url = "' + escapeDoubleQuoted(url) + '",', '  method = "' + escapeDoubleQuoted(req.method) + '",', '  headers = headers,');
  if (req.bodyText) {
    lines.push('  source = ltn12.source.string(body),');
  }
  lines.push('  sink = ltn12.sink.table(response_body),', '}', '', 'print("Status: " .. tostring(code))', 'print(table.concat(response_body))');
  return lines.join('\n');
}

const GENERATORS = {
  curl: generateCurl,
  javascript: generateJavaScript,
  node: generateNode,
  python: generatePython,
  go: generateGo,
  java: generateJava,
  csharp: generateCSharp,
  php: generatePhp,
  ruby: generateRuby,
  c: generateC,
  swift: generateSwift,
  objc: generateObjC,
  kotlin: generateKotlin,
  rust: generateRust,
  powershell: generatePowerShell,
  dart: generateDart,
  lua: generateLua,
};

async function generateSnippet(langId, req) {
  const lang = LANGUAGES.find(function (item) {
    return item.id === langId;
  });
  if (!lang) throw new Error('Unknown language: ' + langId);

  const headers = requestHeaders(req, true);
  const url = buildRequestUrl(req);
  return GENERATORS[lang.id](req, url, headers);
}
