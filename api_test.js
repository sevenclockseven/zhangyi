const http = require('http');
function req(method, path, data, token) {
  return new Promise((resolve, reject) => {
    const body = data ? JSON.stringify(data) : null;
    const r = http.request({ hostname: 'localhost', port: 8080, path, method,
      headers: { 'Content-Type': 'application/json',
        ...(body && { 'Content-Length': Buffer.byteLength(body) }),
        ...(token && { 'Authorization': 'Bearer ' + token }) }
    }, res => { let d = ''; res.on('data', c => d += c); res.on('end', () => resolve({ s: res.statusCode, d: d })); });
    r.on('error', reject);
    if (body) r.write(body); r.end();
  });
}
async function main() {
  // Register superadmin
  let r = await req('POST', '/api/auth/register', { username: 'testadmin3', password: 'test123', role: 'superadmin' });
  console.log('Register:', r.s, r.d.substring(0, 100));

  let login = await req('POST', '/api/auth/login', { username: 'testadmin3', password: 'test123' });
  const loginData = JSON.parse(login.d);
  const token = loginData.token;
  console.log('Login:', token ? 'OK ' + loginData.role : login.d);

  if (!token) return;

  // List books
  r = await req('GET', '/api/books', null, token);
  const books = JSON.parse(r.d);
  console.log('Books:', books.length);
  if (books.length === 0) {
    // Create one
    r = await req('POST', '/api/books', { name: 'APITest', taxpayer_type: 'general', accounting_standard: 'business' }, token);
    console.log('Create:', r.s, r.d.substring(0, 200));
    const cb = JSON.parse(r.d);
    const bid = cb.data?.id || cb.id;
    if (!bid) { console.log('Still no book'); return; }

    // Toggle test
    console.log('\nTesting book', bid);
    r = await req('GET', '/api/books/' + bid, null, token);
    console.log('Before:', JSON.parse(r.d).data?.status);

    r = await req('PUT', '/api/books/' + bid, { status: 'inactive' }, token);
    console.log('PUT:', r.s, r.d.substring(0, 200));

    r = await req('GET', '/api/books/' + bid, null, token);
    console.log('After inactive:', JSON.parse(r.d).data?.status);

    r = await req('PUT', '/api/books/' + bid, { status: 'active' }, token);
    r = await req('GET', '/api/books/' + bid, null, token);
    console.log('After active:', JSON.parse(r.d).data?.status);
  } else {
    const bid = books[0].id;
    console.log('\nTesting book', bid, books[0].name, 'status=' + books[0].status);
    r = await req('PUT', '/api/books/' + bid, { status: 'inactive' }, token);
    console.log('PUT:', r.s, r.d.substring(0, 200));
    r = await req('GET', '/api/books/' + bid, null, token);
    console.log('After:', JSON.parse(r.d).data?.status);
  }
}
main().catch(e => console.error(e));
