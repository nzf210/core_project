import re

with open("apps/umkm/accounting/main.go", "r") as f:
    content = f.read()

# Find the start of handleTransactionStatus
start_idx = content.find("func handleTransactionStatus(w http.ResponseWriter, r *http.Request) {")

# It was inserted right before "mux := http.NewServeMux()". Let's find mux :=
mux_idx = content.find("mux := http.NewServeMux()")

if start_idx != -1 and mux_idx != -1:
    extracted = content[start_idx:mux_idx]
    # Remove it from there
    new_content = content[:start_idx] + content[mux_idx:]
    # Append to end of file
    new_content += "\n" + extracted
    
    with open("apps/umkm/accounting/main.go", "w") as f:
        f.write(new_content)
    print("Fixed functions")
else:
    print("Could not find blocks")
