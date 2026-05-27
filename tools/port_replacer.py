import os
import re

def replace_in_file(filepath, replacements):
    try:
        with open(filepath, 'r') as f:
            content = f.read()
            
        original_content = content
        for old, new in replacements:
            content = content.replace(old, new)
            
        if content != original_content:
            with open(filepath, 'w') as f:
                f.write(content)
            print(f"Updated {filepath}")
    except Exception as e:
        print(f"Error updating {filepath}: {e}")

# Base replacements
replacements_auth = [(":8002", ":8001"), ("port 8002", "port 8001"), ("localhost:8002", "localhost:8001")]
replacements_ai = [(":8003", ":8002"), ("port 8003", "port 8002"), ("localhost:8003", "localhost:8002")]
replacements_billing = [(":8004", ":8003"), ("port 8004", "port 8003"), ("localhost:8004", "localhost:8003")]
replacements_crypto = [(":8004", ":8101"), ("port 8004", "port 8101"), ("localhost:8004", "localhost:8101")]
replacements_umkm_acc = [(":9001", ":8201"), ("port 9001", "port 8201"), ("localhost:9001", "localhost:8201")]
replacements_umkm_chat = [(":9002", ":8202"), ("port 9002", "port 8202"), ("localhost:9002", "localhost:8202")]
replacements_campaign = [(":9002", ":8301"), ("port 9002", "port 8301"), ("localhost:9002", "localhost:8301")]
replacements_fe = [("5173", "3101"), ("5174", "3201"), ("5175", "3301")]

# Specific file updates
files_to_auth = ["services/auth-service/main.go", "scripts/e2e/main.go", "frontend/umkm-web/src/components/ResetPassword.vue", "frontend/umkm-web/src/components/Login.vue", "frontend/umkm-web/src/components/ForgotPassword.vue"]
files_to_ai = ["services/ai-gateway/main.go", "apps/crypto/worker/signal_engine.go", "apps/umkm/chatbot/main.go", "apps/umkm/automation/main.go", "scripts/loadtest/main.go", "scripts/e2e/main.go", "docker-compose.yml"]
files_to_billing = ["services/billing-service/main.go"]
files_to_crypto = ["apps/crypto/api/main.go"]
files_to_umkm_acc = ["apps/umkm/accounting/main.go", "apps/umkm/chatbot/main.go", "apps/umkm/scripts/register_tenant.go", "scripts/e2e/main.go", "frontend/umkm-web/src/components/Dashboard.vue", "frontend/umkm-web/src/components/Journal.vue", "frontend/umkm-web/src/components/Register.vue", "frontend/umkm-web/src/components/AdminDashboard.vue"]
files_to_umkm_chat = ["apps/umkm/chatbot/main.go"]
files_to_campaign = ["apps/campaign/api/main.go", "frontend/campaign-web/src/components/ReportGenerator.vue", "frontend/campaign-web/src/components/Voter.vue", "frontend/campaign-web/src/components/TaskBoard.vue", "frontend/campaign-web/src/components/Dashboard.vue", "frontend/campaign-web/src/components/Volunteer.vue", "frontend/campaign-web/src/components/NotificationBell.vue", "frontend/campaign-web/src/components/AccessManager.vue", "frontend/campaign-web/src/components/MapRegion.vue"]

base_dir = "/home/syahril/Desktop/dev/core_project"

for f in files_to_auth:
    replace_in_file(os.path.join(base_dir, f), replacements_auth)
for f in files_to_ai:
    replace_in_file(os.path.join(base_dir, f), replacements_ai)
for f in files_to_billing:
    replace_in_file(os.path.join(base_dir, f), replacements_billing)
for f in files_to_crypto:
    replace_in_file(os.path.join(base_dir, f), replacements_crypto)
for f in files_to_umkm_acc:
    replace_in_file(os.path.join(base_dir, f), replacements_umkm_acc)
for f in files_to_umkm_chat:
    replace_in_file(os.path.join(base_dir, f), replacements_umkm_chat)
for f in files_to_campaign:
    replace_in_file(os.path.join(base_dir, f), replacements_campaign)

# Update Makefile separately because it contains multiple
makefile_path = os.path.join(base_dir, "Makefile")
with open(makefile_path, 'r') as f:
    content = f.read()

# Makefile replacements
content = content.replace("port 8002", "port 8001")
content = content.replace("port 8003", "port 8002")
content = content.replace("port 9001", "port 8201")
content = content.replace("port 9002", "port 8202")  # UMKM chatbot
content = content.replace("port 8004", "port 8101")  # Crypto API
content = content.replace("5173", "3101")
content = content.replace("5174", "3201")
content = content.replace("5175", "3301")

content = content.replace(":8002", ":8001")
content = content.replace(":8003", ":8002")
content = content.replace(":8004", ":8101")
content = content.replace(":9001", ":8201")
content = content.replace(":9002", ":8202")

# Campaign API stop-all and billing might be missing in Makefile stop-all but let's add them
stop_all_additions = """
	@lsof -ti :8000 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8001 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8002 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8003 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8101 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8201 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8202 | xargs kill -9 2>/dev/null || true
	@lsof -ti :8301 | xargs kill -9 2>/dev/null || true
	@lsof -ti :3101 | xargs kill -9 2>/dev/null || true
	@lsof -ti :3201 | xargs kill -9 2>/dev/null || true
	@lsof -ti :3301 | xargs kill -9 2>/dev/null || true
"""

# Replace the block of lsof in Makefile
import re
content = re.sub(r'(@lsof -ti :.*?\n)+', stop_all_additions[1:], content) # remove first newline from stop_all_additions
with open(makefile_path, 'w') as f:
    f.write(content)
print("Updated Makefile")

# Update docker-compose.yml
dc_path = os.path.join(base_dir, "docker-compose.yml")
with open(dc_path, 'r') as f:
    dc_content = f.read()

dc_content = dc_content.replace('"8002:8002"', '"8001:8001"')
dc_content = dc_content.replace('"8003:8003"', '"8002:8002"')
dc_content = dc_content.replace('"9001:9001"', '"8201:8201"')
dc_content = dc_content.replace('"8004:8004"', '"8101:8101"')
dc_content = dc_content.replace('"9002:9002"', '"8301:8301"')

with open(dc_path, 'w') as f:
    f.write(dc_content)
print("Updated docker-compose.yml")
