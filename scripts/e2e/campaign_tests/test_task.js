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
    console.log("Login failed, registering...");
    // 2. Register
    res = await fetch(`${API_GATEWAY_URL}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials)
    });
    data = await res.json();
    console.log("Register response:", data);
    
    // 3. Login again
    res = await fetch(`${API_GATEWAY_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: credentials.username, password: credentials.password })
    });
    data = await res.json();
  }
  
  console.log("Login response:", data);
  if (!data.data || !data.data.accessToken) {
    console.error("Failed to get access token");
    return;
  }
  
  const token = data.data.accessToken;
  console.log("Got token");
  
  // Fetch campaigns to get a valid campaign ID
  const campRes = await fetch(`${API_GATEWAY_URL}/api/campaign/campaigns`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  const campData = await campRes.json();
  let campaignId = '';
  if (campData.success && campData.data && campData.data.length > 0) {
    campaignId = campData.data[0].id;
  }
  
  console.log("Using campaignId:", campaignId);
  
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
  console.log("Task creation response:", taskData, "Status:", taskRes.status);
}

test();
