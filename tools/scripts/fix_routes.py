import re

with open("apps/umkm/accounting/main.go", "r") as f:
    content = f.read()

mux_setup = """	mux.HandleFunc("/checkout", handleCheckout)
	mux.HandleFunc("/transactions/status", handleTransactionStatus)
	mux.HandleFunc("/webhook/payment", handlePaymentWebhook)"""

new_content = content.replace('mux.HandleFunc("/checkout", handleCheckout)', mux_setup)

with open("apps/umkm/accounting/main.go", "w") as f:
    f.write(new_content)
