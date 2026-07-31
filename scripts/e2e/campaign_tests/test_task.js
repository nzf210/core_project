const API_GATEWAY_URL = 'http://localhost:8000';

async function test() {
  const credentials = {
    username: 'campaign_admin_demo',
    email: 'admin_demo@campaign.local',
    password: 'password123'
  };

  // 1. Try to login
  let res = await fetch(`${API_GATEWAY_URL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: credentials.username, password: credentials.password })
  });
  let data = await res.json();
  
  if (!data.success) {
    process.stdout.write("Login failed, registering...\n");
    // 2. Register
    res = await fetch(`${API_GATEWAY_URL}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials)
    });
    data = await res.json();
    process.stdout.write("Register response: " + JSON.stringify(data) + "\n");

    // 3. Login again
    res = await fetch(`${API_GATEWAY_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: credentials.username, password: credentials.password })
    });
    data = await res.json();
  }

  process.stdout.write("Login response: " + JSON.stringify(data) + "\n");
  if (!data.data || !data.data.accessToken) {
    process.stderr.write("Failed to get access token\n");
    return;
  }

  const token = data.data.accessToken;
  process.stdout.write("Got token\n");

  // Fetch campaigns to get a valid campaign ID
  const campRes = await fetch(`${API_GATEWAY_URL}/api/campaign/campaigns`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  const campData = await campRes.json();
  let campaignId = '';
  if (campData.success && campData.data && campData.data.length > 0) {
    campaignId = campData.data[0].id;
  }

  process.stdout.write("Using campaignId: " + campaignId + "\n");

  // 4. Create Task
  const taskRes = await fetch(`${API_GATEWAY_URL}/api/campaign/tasks`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({ title: 'Test Task', description: 'Testing task creation', campaign_id: campaignId })
  });

  const taskData = await taskRes.json();
  process.stdout.write("Task creation response: " + JSON.stringify(taskData) + " Status: " + taskRes.status + "\n");
}

test();
