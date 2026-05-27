const API_GATEWAY_URL = 'http://localhost:8000';
async function test() {
  const credentials = {
    username: 'campaign_admin_demo',
    password: 'password123'
  };
  let res = await fetch(`${API_GATEWAY_URL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(credentials)
  });
  let data = await res.json();
  const token = data.data.accessToken;

  const taskRes = await fetch(`${API_GATEWAY_URL}/api/campaign/tasks`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
      'Origin': 'http://localhost:5173'
    },
    body: JSON.stringify({ title: 'Test Task', description: 'Testing', campaign_id: 'camp_123' })
  });
  
  console.log("Status:", taskRes.status);
  console.log("CORS Header:", taskRes.headers.get('access-control-allow-origin'));
}
test();
