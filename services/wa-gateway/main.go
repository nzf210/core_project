package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "github.com/lib/pq"
)

var (
	// Map to store connected clients per tenant_id
	clientMap = make(map[string]*whatsmeow.Client)
	clientMu  sync.RWMutex
	
	// Postgres connection string will be built from env vars
)

func getDBURI() string {
	user := os.Getenv("DB_USER")
	if user == "" { user = "wch_admin" }
	pass := os.Getenv("DB_PASSWORD")
	if pass == "" { pass = "secure_postgres_password_123" }
	host := os.Getenv("DB_HOST")
	if host == "" { host = "localhost" }
	port := os.Getenv("DB_PORT")
	if port == "" { port = "5433" }
	dbname := os.Getenv("DB_NAME")
	if dbname == "" { dbname = "wch_platform" }
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" { sslmode = "disable" }
	
	return "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + dbname + "?sslmode=" + sslmode
}

func getTenantID(r *http.Request) string {
	// 1. Try URL Query parameter
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID != "" {
		return tenantID
	}

	// 2. Try Form value
	tenantID = r.FormValue("tenant_id")
	if tenantID != "" {
		return tenantID
	}

	// 3. Try X-Tenant-ID Header
	tenantID = r.Header.Get("X-Tenant-ID")
	if tenantID != "" {
		return tenantID
	}

	// 4. Try JSON body if Content-Type is application/json
	if r.Body != nil && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		// Read body
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil {
			// Restore r.Body so it can be read again if needed
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			var data map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &data); err == nil {
				if tID, ok := data["tenant_id"].(string); ok {
					return tID
				}
			}
		}
	}

	return ""
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}

func main() {
	// Load .env file
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../../.env")

	dbLog := waLog.Stdout("Database", "WARN", true)
	dbURI := getDBURI()
	// Initialize Postgres container store
	container, err := sqlstore.New(context.Background(), "postgres", dbURI, dbLog)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer container.Close()

	// Initialize our custom mapping table
	db, err := sql.Open("postgres", dbURI)
	if err == nil {
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS wa_tenant_sessions (tenant_id TEXT PRIMARY KEY, jid VARCHAR NOT NULL)`)
		if err != nil {
			log.Printf("Failed to create wa_tenant_sessions: %v", err)
		}
		// Migrate existing UUID column to TEXT if needed
		db.Exec(`ALTER TABLE wa_tenant_sessions ALTER COLUMN tenant_id TYPE TEXT`)
	} else {
		log.Printf("Failed to open DB for mapping: %v", err)
	}

	// Restore sessions
	if db != nil {
		rows, err := db.Query(`SELECT tenant_id, jid FROM wa_tenant_sessions`)
		if err == nil {
			for rows.Next() {
				var tID, jidStr string
				rows.Scan(&tID, &jidStr)
				jid, _ := types.ParseJID(jidStr)
				device, _ := container.GetDevice(context.Background(), jid)
				if device != nil {
					c := whatsmeow.NewClient(device, waLog.Stdout("Client-"+tID, "INFO", true))
					c.AddEventHandler(func(evt interface{}) { eventHandler(tID, evt, db) })
					err = c.Connect()
					if err == nil {
						clientMap[tID] = c
						log.Printf("Restored session for tenant %s", tID)
					}
				}
			}
			rows.Close()
		}
	}

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/api/wa/qr", func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantID(r)
		if tenantID == "" {
			http.Error(w, "tenant_id required", http.StatusBadRequest)
			return
		}

		clientMu.Lock()
		client, exists := clientMap[tenantID]
		if exists && client.Store.ID == nil {
			client.Disconnect()
			delete(clientMap, tenantID)
			exists = false
		}

		if !exists {
			// Initialize new client for this tenant
			newStore := container.NewDevice()
			
			clientLog := waLog.Stdout("Client-"+tenantID, "INFO", true)
			client = whatsmeow.NewClient(newStore, clientLog)
			client.AddEventHandler(func(evt interface{}) {
				eventHandler(tenantID, evt, db)
			})
			clientMap[tenantID] = client
		}
		clientMu.Unlock()

		if client.Store.ID != nil {
			// Already logged in
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "connected",
				"message": "Already connected",
			})
			return
		}

		// Generate QR
		qrChan, _ := client.GetQRChannel(context.Background())
		err := client.Connect()
		if err != nil {
			http.Error(w, "failed to connect", http.StatusInternalServerError)
			return
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				// Generate Base64 QR code image
				png, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
				if err != nil {
					http.Error(w, "failed to generate qr", http.StatusInternalServerError)
					return
				}
				base64QR := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "qr",
					"qr_code": base64QR,
					"raw_code": evt.Code,
				})
				
				// Launch a background routine to wait for login
				go func() {
					time.Sleep(60 * time.Second)
					// Disconnect if not logged in to save resources
					if client.Store.ID == nil {
						client.Disconnect()
						clientMu.Lock()
						if clientMap[tenantID] == client {
							delete(clientMap, tenantID)
						}
						clientMu.Unlock()
					}
				}()
				return
			} else {
				log.Printf("QR channel event: %s", evt.Event)
			}
		}
	})

	http.HandleFunc("/api/wa/status", func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantID(r)
		w.Header().Set("Content-Type", "application/json")
		
		clientMu.RLock()
		client, exists := clientMap[tenantID]
		clientMu.RUnlock()

		if !exists || client.Store.ID == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "disconnected",
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "connected",
			"jid": client.Store.ID.String(),
		})
	})
	
	http.HandleFunc("/api/wa/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		tenantID := getTenantID(r)
		target := r.FormValue("target")
		message := r.FormValue("message")

		if tenantID == "" || target == "" || message == "" {
			http.Error(w, "Missing fields", http.StatusBadRequest)
			return
		}

		clientMu.RLock()
		client, exists := clientMap[tenantID]
		clientMu.RUnlock()

		if !exists || client.Store.ID == nil {
			http.Error(w, "WhatsApp not connected for this tenant", http.StatusBadGateway)
			return
		}

		jid, err := types.ParseJID(target)
		if err != nil {
			http.Error(w, "Invalid target JID", http.StatusBadRequest)
			return
		}

		msg := &waE2E.Message{
			Conversation: proto.String(message),
		}

		_, err = client.SendMessage(context.Background(), jid, msg)
		if err != nil {
			log.Printf("[Tenant %s] Failed to send message to %s: %v", tenantID, jid.String(), err)
			http.Error(w, "Failed to send message", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	})

	http.HandleFunc("/api/wa/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		tenantID := getTenantID(r)
		if tenantID == "" {
			http.Error(w, "tenant_id required", http.StatusBadRequest)
			return
		}

		clientMu.Lock()
		client, exists := clientMap[tenantID]
		if exists {
			client.Disconnect()
			delete(clientMap, tenantID)
		}
		clientMu.Unlock()

		if db != nil {
			db.Exec(`DELETE FROM wa_tenant_sessions WHERE tenant_id = $1`, tenantID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	})

	// Apply CORS
	handler := func(h http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			h.ServeHTTP(w, r)
		}
	}

	port := ":8202"
	log.Printf("WA Gateway running on port %s", port)
	log.Fatal(http.ListenAndServe(port, handler(http.DefaultServeMux)))
}

func eventHandler(tenantID string, evt interface{}, db *sql.DB) {
	switch v := evt.(type) {
	case *events.Message:
		text := v.Message.GetConversation()
		if text == "" {
			text = v.Message.GetExtendedTextMessage().GetText()
		}
		if text == "" {
			return // ignore non-text messages for MVP
		}
		senderJID := v.Info.Sender.ToNonAD().String()
		
		// Prevent responding to our own messages
		if v.Info.IsFromMe {
			return
		}

		log.Printf("[Tenant %s] Received message from %s: %s", tenantID, senderJID, text)

		// Forward to Chatbot Webhook
		payload := map[string]interface{}{
			"sender": senderJID,
			"message": text,
		}
		jsonBody, _ := json.Marshal(payload)
		
		// Chatbot runs on 8203 internally
		webhookURL := "http://umkm-chatbot:8202/webhook/wa?tenant_id=" + tenantID
		resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			log.Printf("Failed to forward message to chatbot: %v", err)
		} else {
			resp.Body.Close()
		}

	case *events.Connected:
		log.Printf("[Tenant %s] Connected to WhatsApp!", tenantID)
		if db != nil {
			clientMu.RLock()
			c := clientMap[tenantID]
			clientMu.RUnlock()
			if c != nil && c.Store.ID != nil {
				_, err := db.Exec(`INSERT INTO wa_tenant_sessions (tenant_id, jid) VALUES ($1, $2) ON CONFLICT (tenant_id) DO UPDATE SET jid = EXCLUDED.jid`, tenantID, c.Store.ID.String())
				if err != nil {
					log.Printf("[Tenant %s] Failed to save session to DB: %v", tenantID, err)
				}
			}
		}
	}
}
