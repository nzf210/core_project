import re

with open("apps/umkm/accounting/main.go", "r") as f:
    content = f.read()

# Replace the Fonnte send WA Notification block
old_wa_block = """	// Send WA Notification
	var waNumber, fonnteToken *string
	DB.QueryRow(ctx, "SELECT wa_number, fonnte_token FROM tenants WHERE id = $1", tenantID).Scan(&waNumber, &fonnteToken)
	if waNumber != nil && *waNumber != "" && fonnteToken != nil && *fonnteToken != "" {
		go func(phone, token, ref string, amount float64) {
			msg := fmt.Sprintf("✅ *PEMBAYARAN DITERIMA* ✅\\n\\nRef: %s\\nNominal: Rp %.0f\\nMetode: QRIS\\n\\nTerima kasih, dana telah masuk ke rekening Anda dan sistem telah mencatat transaksi ini.", ref, amount)
			
			// Fonnte API call
			data := map[string]string{"target": phone, "message": msg}
			jsonData, _ := json.Marshal(data)
			req, _ := http.NewRequest("POST", "https://api.fonnte.com/send", strings.NewReader(string(jsonData)))
			req.Header.Set("Authorization", token)
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 10 * time.Second}
			client.Do(req)
		}(*waNumber, *fonnteToken, req.Reference, totalAmount)
	}"""

new_wa_block = """	// Send WA Notification
	var waNumber *string
	DB.QueryRow(ctx, "SELECT wa_number FROM tenants WHERE id = $1", tenantID).Scan(&waNumber)
	if waNumber != nil && *waNumber != "" {
		go func(tenantID, phone, ref string, amount float64) {
			msg := fmt.Sprintf("✅ *PEMBAYARAN DITERIMA* ✅\\n\\nRef: %s\\nNominal: Rp %.0f\\nMetode: QRIS\\n\\nTerima kasih, dana telah masuk ke rekening Anda dan sistem telah mencatat transaksi ini.", ref, amount)
			
			// Format phone to JID
			target := phone
			if strings.HasPrefix(target, "0") {
				target = "62" + target[1:]
			}
			if !strings.Contains(target, "@") {
				target = target + "@s.whatsapp.net"
			}

			// Internal WA Gateway call
			data := url.Values{}
			data.Set("tenant_id", tenantID)
			data.Set("target", target)
			data.Set("message", msg)

			req, _ := http.NewRequest("POST", "http://wa-gateway:8202/api/wa/send", strings.NewReader(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			client := &http.Client{Timeout: 10 * time.Second}
			client.Do(req)
		}(tenantID, *waNumber, req.Reference, totalAmount)
	}"""

content = content.replace(old_wa_block, new_wa_block)

# We need to make sure "net/url" is imported if not already. Let's add it carefully.
if '"net/url"' not in content:
    content = content.replace('"net/http"', '"net/http"\n\t"net/url"')

with open("apps/umkm/accounting/main.go", "w") as f:
    f.write(content)
