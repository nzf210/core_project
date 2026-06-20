<template>
  <div class="wa-setup-page">
    <div class="header-actions" style="margin-bottom: 1.5rem;">
      <h2>📱 WhatsApp & AI CS</h2>
      <p>Hubungkan WhatsApp dan atur kepribadian AI Customer Service toko Anda di satu tempat.</p>
    </div>

    <!-- TABS -->
    <div class="tabs" style="display: flex; gap: 1rem; border-bottom: 1px solid var(--border-color); margin-bottom: 1.5rem; padding-bottom: 0.5rem;">
      <button 
        class="tab-btn" 
        :class="{ active: activeTab === 'connection' }" 
        @click="activeTab = 'connection'">
        Koneksi & Provider
      </button>
      <button 
        class="tab-btn" 
        :class="{ active: activeTab === 'ai_config' }" 
        @click="activeTab = 'ai_config'">
        Pengaturan AI CS
      </button>
    </div>

    <div v-if="loading" class="glass-card" style="padding: 2rem; text-align: center;">
      <p>Memuat status...</p>
    </div>
    
    <!-- TAB 1: KONEKSI & PROVIDER -->
    <div v-else-if="activeTab === 'connection'" class="setup-layout" style="display: grid; grid-template-columns: 2fr 1fr; gap: 1.5rem;">
      <div class="glass-card" style="padding: 1.5rem;">
        <h3 style="margin-bottom: 1rem;">Provider Utama</h3>
        <p style="font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 1rem;">
          Pilih provider yang ingin Anda gunakan. Mode Auto akan memprioritaskan Cloud API untuk notifikasi penting (jika aktif) dan Whatsmeow untuk chatbot.
        </p>

        <div style="display: flex; flex-direction: column; gap: 0.75rem; margin-bottom: 2rem;">
          <label class="provider-card" :class="{ active: provider === 'auto' }">
            <input type="radio" value="auto" v-model="provider" @change="saveProvider" />
            <div class="provider-info">
              <strong>⚡ Auto (Hybrid) — Rekomendasi</strong>
              <span class="desc">Notifikasi tagihan/OTP via Cloud API, Chatbot via Whatsmeow.</span>
            </div>
          </label>
          
          <label class="provider-card" :class="{ active: provider === 'whatsmeow' }">
            <input type="radio" value="whatsmeow" v-model="provider" @change="saveProvider" />
            <div class="provider-info">
              <strong>📱 Whatsmeow Only</strong>
              <span class="desc">Gratis, tanpa kuota. <strong>Dilarang/diblokir untuk fitur Broadcast Massal</strong>.</span>
            </div>
          </label>
          
          <label class="provider-card" :class="{ active: provider === 'cloud_api', locked: !waSetupState.can_use_cloud_api }">
            <input type="radio" value="cloud_api" v-model="provider" :disabled="!waSetupState.can_use_cloud_api" @change="saveProvider" />
            <div class="provider-info">
              <strong>☁️ Cloud API (Meta Official)</strong>
              <span class="desc">Resmi, bebas blokir. Wajib untuk Broadcast Massal. Butuh saldo wallet.</span>
              <span v-if="!waSetupState.can_use_cloud_api" class="badge locked-badge">Butuh Add-on</span>
            </div>
          </label>
        </div>

        <h3 style="margin-bottom: 1rem;">Status Whatsmeow</h3>
        <div class="status-box" :class="waSetupState.whatsmeow.status">
          <div class="status-header">
            <span class="status-dot"></span>
            <strong>{{ waSetupState.whatsmeow.connected ? 'Terhubung' : (waSetupState.whatsmeow.status === 'qr_pending' ? 'Menunggu Scan QR' : 'Terputus') }}</strong>
          </div>
          <p class="status-desc">
            Koneksi pihak ketiga ke WhatsApp Web. Harus terhubung agar chatbot bisa membalas.
          </p>
          <div v-if="!waSetupState.whatsmeow.connected" class="action-row">
            <button class="btn btn-primary" @click="requestQR">Generate QR Code</button>
          </div>
        </div>

        <h3 style="margin-bottom: 1rem; margin-top: 2rem;">Status Cloud API</h3>
        <div class="status-box" :class="waSetupState.cloud_api.active ? 'connected' : 'disconnected'">
           <div class="status-header">
            <span class="status-dot"></span>
            <strong>{{ waSetupState.cloud_api.active ? 'Aktif' : 'Belum Dikonfigurasi' }}</strong>
          </div>
          <div v-if="waSetupState.cloud_api.active" style="margin-top: 1rem; background: var(--bg-primary); padding: 1rem; border-radius: 0.5rem;">
            <div style="display: flex; justify-content: space-between; margin-bottom: 0.5rem;">
              <span>Kredit Tersedia:</span>
              <strong style="color: var(--success);">Rp {{ formatPrice(waSetupState.cloud_api.credit_balance_cents) }}</strong>
            </div>
            <div style="display: flex; justify-content: space-between;">
              <span>Pemakaian Bulan Ini:</span>
              <strong style="color: var(--warning);">Rp {{ formatPrice(waSetupState.cloud_api.credit_used_cents) }}</strong>
            </div>
            <button class="btn btn-secondary" style="margin-top: 1rem; width: 100%;">Top Up Kredit</button>
          </div>
          <div v-else class="action-row">
            <button class="btn btn-primary" :disabled="!waSetupState.can_use_cloud_api" @click="openCloudApiModal">
              Hubungkan ke Meta
            </button>
          </div>
        </div>
      </div>

      <div class="glass-card preview-card" style="padding: 1.25rem;">
        <h4 style="margin-bottom: 1rem; color: #f59e0b;">⚠️ Peringatan Broadcast</h4>
        <p style="font-size: 0.85rem; color: var(--text-secondary); line-height: 1.5; margin-bottom: 1rem;">
          Whatsmeow adalah koneksi tidak resmi (unofficial). Sesuai kebijakan keamanan, <strong>pengiriman broadcast promosi massal diblokir pada jalur Whatsmeow</strong> untuk mencegah nomor WhatsApp Anda dibanned secara permanen.
        </p>
        <p style="font-size: 0.85rem; color: var(--text-secondary); line-height: 1.5;">
          Untuk melakukan Blast / Broadcast massal, Anda diwajibkan melakukan upgrade dan menghubungkan nomor melalui jalur Meta Cloud API yang resmi.
        </p>

        <h4 style="margin-bottom: 0.75rem; margin-top: 2rem; color: #3b82f6;">💳 Tarif Cloud API</h4>
        <ul style="font-size: 0.85rem; color: var(--text-secondary); padding-left: 1.2rem; line-height: 1.6;">
          <li>Pesan Percakapan (Chatbot): ~Rp 450 / pesan</li>
          <li>Notifikasi Otentikasi (OTP): ~Rp 300 / pesan</li>
          <li>Broadcast Marketing: ~Rp 600 / pesan</li>
        </ul>
      </div>
    </div>

    <!-- TAB 2: AI CONFIG -->
    <div v-else-if="activeTab === 'ai_config'" class="config-layout" style="display: grid; grid-template-columns: 2fr 1fr; gap: 1.5rem;">
      <div class="glass-card" style="padding: 1.5rem;">
        <!-- Stepper Inside AI Config -->
        <div class="stepper" style="display: flex; gap: 0.5rem; margin-bottom: 2rem;">
          <div v-for="(label, i) in steps" :key="i" class="step-pill" :class="{ active: currentStep === i, done: currentStep > i }" @click="goToStep(i)">
            <span class="step-num">{{ currentStep > i ? '✓' : i + 1 }}</span>
            <span class="step-label">{{ label }}</span>
          </div>
        </div>

        <!-- STEP 1 — Identitas Bot -->
        <div v-show="currentStep === 0">
          <h3 style="margin-bottom: 1rem;">Step 1 — Identitas Bot</h3>
          <div style="display: flex; flex-direction: column; gap: 1rem;">
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Nama Bot (opsional)</span>
              <input v-model="form.bot_name" type="text" class="form-control" placeholder="Contoh: CS Toko Barokah" />
            </label>
            <div>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.5rem;">Bahasa</span>
              <div style="display: flex; gap: 1rem;">
                <label class="radio-pill">
                  <input type="radio" value="id" v-model="form.language" /> 🇮🇩 Indonesia
                </label>
                <label class="radio-pill">
                  <input type="radio" value="en" v-model="form.language" /> 🇬🇧 English
                </label>
              </div>
            </div>
            <div>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.5rem;">Tone / Gaya Bicara</span>
              <select v-model="form.tone" class="form-control">
                <option value="friendly">Ramah & Hangat</option>
                <option value="formal">Formal</option>
                <option value="casual">Santai & Akrab</option>
                <option value="professional">Profesional & Solutif</option>
              </select>
            </div>
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">System Prompt Custom (opsional, advanced)</span>
              <textarea v-model="form.system_prompt" class="form-control" rows="3" placeholder="Biarkan kosong untuk pakai default. Isi jika ingin instruksi spesifik, misal 'Kamu selalu jawab pakai emoji ✨'."></textarea>
            </label>
          </div>
        </div>

        <!-- STEP 2 — Jam Operasional & Escalation -->
        <div v-show="currentStep === 1">
          <h3 style="margin-bottom: 1rem;">Step 2 — Jam Operasional & Auto-Eskalasi</h3>
          <div style="display: flex; flex-direction: column; gap: 1rem;">
            <div style="display: flex; gap: 1rem;">
              <label style="flex: 1;">
                <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Jam Buka</span>
                <input v-model="form.business_hours_start" type="time" class="form-control" />
              </label>
              <label style="flex: 1;">
                <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Jam Tutup</span>
                <input v-model="form.business_hours_end" type="time" class="form-control" />
              </label>
            </div>
            <div>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.5rem;">Hari Operasional</span>
              <div style="display: flex; gap: 0.5rem; flex-wrap: wrap;">
                <label v-for="d in dayList" :key="d.value" class="day-pill" :class="{ active: form.business_days.includes(d.value) }">
                  <input type="checkbox" :value="d.value" v-model="form.business_days" hidden />
                  {{ d.short }}
                </label>
              </div>
            </div>
            <hr style="border: 0; border-top: 1px solid var(--border-color); margin: 0.5rem 0;" />
            <label class="toggle-row">
              <input type="checkbox" v-model="form.escalation_enabled" />
              <span>Aktifkan auto-eskalasi ke admin</span>
            </label>
            <div v-if="form.escalation_enabled">
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Kata Kunci Eskalasi (pisahkan dengan Enter)</span>
              <div class="keyword-input">
                <span v-for="(kw, i) in form.escalation_keywords" :key="i" class="kw-tag">
                  {{ kw }}
                  <button type="button" @click="form.escalation_keywords.splice(i, 1)">×</button>
                </span>
                <input
                  type="text"
                  v-model="newKeyword"
                  @keydown.enter.prevent="addKeyword"
                  @keydown.,.prevent="addKeyword"
                  placeholder="Tekan Enter untuk tambah"
                  class="form-control kw-input"
                />
              </div>
            </div>
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Auto-eskalasi setelah berapa menit tanpa jawaban?</span>
              <input v-model.number="form.auto_escalate_after_minutes" type="number" min="0" max="60" class="form-control" style="max-width: 120px;" />
            </label>
          </div>
        </div>

        <!-- STEP 3 — Kalimat & Channel -->
        <div v-show="currentStep === 2">
          <h3 style="margin-bottom: 1rem;">Step 3 — Kalimat & Channel</h3>
          <div style="display: flex; flex-direction: column; gap: 1rem;">
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Pesan Sambutan (welcome)</span>
              <textarea v-model="form.welcome_message" class="form-control" rows="2" />
            </label>
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Pesan Fallback (kalau bot bingung)</span>
              <textarea v-model="form.fallback_message" class="form-control" rows="2" />
            </label>
            <label>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Pesan di Luar Jam Operasional</span>
              <textarea v-model="form.outside_hours_message" class="form-control" rows="2" />
            </label>
            <hr style="border: 0; border-top: 1px solid var(--border-color); margin: 0.5rem 0;" />
            <div>
              <span style="display: block; font-weight: 600; font-size: 0.9rem; margin-bottom: 0.5rem;">AI Modality</span>
              <label class="toggle-row" style="margin-bottom: 0.25rem;">
                <input type="checkbox" v-model="form.enable_vision" />
                <span><strong>Enable Vision</strong> (Process image messages)</span>
              </label>
              <label class="toggle-row" style="margin-bottom: 0.25rem;">
                <input type="checkbox" v-model="form.enable_voice_reply" />
                <span><strong>Enable Voice Reply</strong> (Reply with voice notes)</span>
              </label>
              <label v-if="form.enable_voice_reply" style="display: block; margin-top: 0.5rem;">
                <span style="display: block; font-size: 0.85rem; margin-bottom: 0.25rem;">Voice Model</span>
                <select v-model="form.voice_model" class="form-control">
                  <option value="id-ID-ArdiNeural">id-ID-ArdiNeural (Laki-laki)</option>
                  <option value="id-ID-GadisNeural">id-ID-GadisNeural (Perempuan)</option>
                </select>
              </label>
            </div>
            <hr style="border: 0; border-top: 1px solid var(--border-color); margin: 0.5rem 0;" />

            <div>
              <span style="display: block; font-size: 0.85rem; margin-bottom: 0.5rem;">Channel Aktif</span>
              <div style="display: flex; gap: 1rem; flex-wrap: wrap;">
                <label class="channel-pill" :class="{ active: form.channels_enabled.includes('whatsapp'), locked: true }">
                  <input type="checkbox" value="whatsapp" v-model="form.channels_enabled" disabled /> 📱 WhatsApp
                </label>
                <label class="channel-pill" :class="{ active: form.channels_enabled.includes('telegram') }">
                  <input type="checkbox" value="telegram" v-model="form.channels_enabled" /> ✈️ Telegram
                </label>
                <label class="channel-pill" :class="{ active: form.channels_enabled.includes('webchat') }">
                  <input type="checkbox" value="webchat" v-model="form.channels_enabled" /> 💬 Web Chat
                </label>
              </div>
              <p style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.5rem;">
                WhatsApp selalu aktif. Channel lain aktif jika Anda centang.
              </p>
            </div>
            <hr style="border: 0; border-top: 1px solid var(--border-color); margin: 0.5rem 0;" />

            <label class="toggle-row">
              <input type="checkbox" v-model="form.is_active" />
              <span><strong>Aktifkan AI CS</strong> (jika nonaktif, customer akan dapat pesan di luar jam)</span>
            </label>
            <button class="btn btn-secondary" @click="openTestModal" :disabled="testing || !form.is_active">
              {{ testing ? 'Mengirim...' : '🧪 Test Bot' }}
            </button>
          </div>
        </div>

        <!-- Navigation buttons -->
        <div class="nav-row" style="display: flex; justify-content: space-between; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="prev" :disabled="currentStep === 0">← Kembali</button>
          <div>
            <button v-if="currentStep < steps.length - 1" class="btn btn-primary" @click="next">Lanjut →</button>
            <button v-else class="btn btn-primary" @click="save" :disabled="saving">
              {{ saving ? 'Menyimpan...' : 'Simpan & Aktifkan' }}
            </button>
          </div>
        </div>
        <p v-if="errorMsg" style="color: #dc2626; margin-top: 0.5rem;">{{ errorMsg }}</p>
      </div>

      <!-- PREVIEW COLUMN -->
      <div class="glass-card preview-card" style="padding: 1.25rem;">
        <h4 style="margin-bottom: 0.75rem;">Preview</h4>
        <div class="preview-row"><span>Bot</span><b>{{ form.bot_name || 'CS Toko Anda' }}</b></div>
        <div class="preview-row"><span>Bahasa</span><b>{{ form.language === 'en' ? '🇬🇧 English' : '🇮🇩 Indonesia' }}</b></div>
        <div class="preview-row"><span>Tone</span><b>{{ toneLabel }}</b></div>
        <div class="preview-row"><span>Aktif</span><b>{{ form.is_active ? '✅ Ya' : '❌ Tidak' }}</b></div>
        <hr style="border: 0; border-top: 1px solid var(--border-color); margin: 0.75rem 0;" />
        <div class="preview-row"><span>Jam</span><b>{{ form.business_hours_start }}–{{ form.business_hours_end }}</b></div>
        <div class="preview-row"><span>Hari</span><b>{{ dayShortList || 'Setiap hari' }}</b></div>
        <div class="preview-row"><span>Escalation</span><b>{{ form.escalation_enabled ? '✅ On' : '❌ Off' }}</b></div>
        <div class="preview-row"><span>Channel</span><b>{{ form.channels_enabled.join(', ') || '-' }}</b></div>
      </div>
    </div>

    <!-- QR Modal -->
    <div v-if="qrModal" class="modal-backdrop" @click.self="qrModal = false; stopQRPolling()">
      <div class="modal-content glass-card" style="max-width: 360px; padding: 1.5rem; text-align: center;">
        <h3 style="margin-bottom: 0.75rem;">📱 Scan QR Code</h3>
        <p style="font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 1rem;">
          Buka WhatsApp di HP Anda → Settings → Linked Devices → Link a Device
        </p>
        <div v-if="qrStatus === 'loading'" style="padding: 2rem; color: var(--text-secondary);">
          Memuat QR Code...
        </div>
        <div v-else-if="qrStatus === 'qr' && qrImage">
          <img :src="qrImage" alt="QR Code" style="width: 220px; height: 220px; border: 1px solid var(--border-color); border-radius: 8px;" />
          <p style="font-size: 0.8rem; color: var(--text-secondary); margin-top: 0.5rem;">QR code berlaku 60 detik. Refresh otomatis.</p>
        </div>
        <div v-else-if="qrStatus === 'connected'" style="padding: 2rem; color: #10b981;">
          ✅ WhatsApp terhubung!
        </div>
        <div v-else-if="qrStatus === 'error'" style="padding: 1rem; color: #dc2626; font-size: 0.85rem;">
          {{ qrError }}
        </div>
        <div style="display: flex; justify-content: center; gap: 0.5rem; margin-top: 1rem;">
          <button class="btn btn-secondary" @click="qrModal = false; stopQRPolling()">Tutup</button>
          <button v-if="qrStatus === 'qr'" class="btn btn-primary" @click="requestQR">🔄 Refresh QR</button>
        </div>
      </div>
    </div>

    <!-- Cloud API Credential Modal -->
    <div v-if="cloudApiModal" class="modal-backdrop" @click.self="closeCloudApiModal()">
      <div class="modal-content glass-card" style="max-width: 520px; padding: 1.5rem;">
        <h3 style="margin-bottom: 0.5rem;">☁️ Hubungkan Meta Cloud API</h3>
        <p style="font-size: 0.8rem; color: var(--text-secondary); margin-bottom: 0.75rem; line-height: 1.5;">
          <strong>Step 1:</strong> Masukkan credential → klik <em>Validate</em> untuk uji coba koneksi ke Meta.<br/>
          <strong>Step 2:</strong> Jika valid, klik <em>Simpan</em> untuk menyimpan.
        </p>
        <div style="display: flex; flex-direction: column; gap: 0.85rem;">
          <label>
            <span style="display: block; font-size: 0.82rem; margin-bottom: 0.2rem; color: var(--text-secondary);">Phone Number ID <span style="color: #dc2626;">*</span></span>
            <input v-model="cloudApiForm.phone_number_id" type="text" class="form-control" placeholder="例: 123456789012345" />
          </label>
          <label>
            <span style="display: block; font-size: 0.82rem; margin-bottom: 0.2rem; color: var(--text-secondary);">WABA ID (WhatsApp Business Account)</span>
            <input v-model="cloudApiForm.waba_id" type="text" class="form-control" placeholder="例: 987654321098765" />
          </label>
          <label>
            <span style="display: block; font-size: 0.82rem; margin-bottom: 0.2rem; color: var(--text-secondary);">Permanent Access Token <span style="color: #dc2626;">*</span></span>
            <input v-model="cloudApiForm.access_token" type="password" class="form-control" placeholder="Token dari Meta Developers" />
          </label>
          <label>
            <span style="display: block; font-size: 0.82rem; margin-bottom: 0.2rem; color: var(--text-secondary);">Webhook Verify Token (opsional)</span>
            <input v-model="cloudApiForm.verify_token" type="text" class="form-control" placeholder="Kosongkan untuk auto-generate" />
          </label>
        </div>
        <!-- Validation status badge -->
        <div v-if="cloudApiValidationResult === 'valid'" class="success-text" style="margin-top: 0.75rem; padding: 0.5rem 0.75rem; background: rgba(16,185,129,0.1); border-radius: 0.5rem;">
          ✅ Credential tervalidasi! Silakan klik Simpan.
        </div>
        <div v-if="cloudApiValidationResult === 'invalid'" class="error-text" style="margin-top: 0.75rem; padding: 0.5rem 0.75rem; background: rgba(220,38,38,0.1); border-radius: 0.5rem;">
          ❌ {{ cloudApiError || 'Credential tidak valid. Periksa kembali.' }}
        </div>
        <p v-if="cloudApiError && cloudApiValidationResult !== 'invalid'" class="error-text" style="margin-top: 0.75rem;">{{ cloudApiError }}</p>
        <p v-if="cloudApiSuccess" class="success-text" style="margin-top: 0.75rem;">{{ cloudApiSuccess }}</p>
        <div style="display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 1.25rem;">
          <button class="btn btn-secondary" @click="closeCloudApiModal()">Batal</button>
          <button v-if="!cloudApiValidated" class="btn btn-primary" :disabled="cloudApiLoading || !cloudApiForm.access_token.trim() || !cloudApiForm.phone_number_id.trim()" @click="validateAndSaveCredential">
            {{ cloudApiValidating ? '⏳ Validasi...' : '🔍 Validate' }}
          </button>
          <button v-if="cloudApiValidated" class="btn btn-primary" :disabled="cloudApiLoading" @click="saveCloudApiCredential">
            {{ cloudApiLoading ? 'Menyimpan...' : '💾 Simpan' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Test modal -->
    <div v-if="testOpen" class="modal-backdrop" @click.self="testOpen = false">
      <div class="modal-content glass-card" style="max-width: 480px; padding: 1.5rem;">
        <h3 style="margin-bottom: 0.75rem;">🧪 Test Bot</h3>
        <p style="font-size: 0.85rem; color: var(--text-secondary);">Coba kirim pesan ke bot untuk lihat bagaimana dia akan menjawab dengan konfigurasi saat ini.</p>
        <input v-model="testInput" type="text" class="form-control" placeholder='Misal: "Halo, ada diskon?"' @keydown.enter="runTest" style="margin-top: 0.5rem;" />
        <div v-if="testReply" class="test-reply" style="margin-top: 1rem; padding: 0.75rem; background: var(--bg-tertiary); border-radius: 0.5rem;">
          <p style="margin: 0 0 0.5rem; font-size: 0.9rem;">{{ testReply }}</p>
          <p v-if="testWouldEscalate" style="margin: 0; font-size: 0.75rem; color: #f59e0b;">⚠️ Pesan ini akan di-eskalasi ke admin (kata kunci cocok).</p>
        </div>
        <p v-if="testError" style="color: #dc2626; font-size: 0.85rem; margin-top: 0.5rem;">{{ testError }}</p>
        <div style="display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 1rem;">
          <button class="btn btn-secondary" @click="testOpen = false">Tutup</button>
          <button class="btn btn-primary" @click="runTest" :disabled="testing || !testInput.trim()">{{ testing ? 'Mengirim...' : 'Kirim' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { api } from '../api'

const activeTab = ref<'connection'|'ai_config'>('connection')

// --- CONNECTION STATE ---
const loading = ref(true)
const provider = ref('auto')
const waSetupState = ref({
  wa_provider_preference: 'auto',
  can_use_cloud_api: false,
  whatsmeow: { connected: false, status: 'disconnected' },
  cloud_api: { active: false, credit_balance_cents: 0, credit_used_cents: 0 }
})

const formatPrice = (sen: number) => {
  return (sen / 100).toLocaleString('id-ID')
}

// --- AI CONFIG STATE ---
const steps = ['Identitas', 'Jam & Eskalasi', 'Kalimat & Channel']
const currentStep = ref(0)
const saving = ref(false)
const testing = ref(false)
const errorMsg = ref('')

const testOpen = ref(false)
const testInput = ref('')
const testReply = ref('')
const testWouldEscalate = ref(false)
const testError = ref('')
const newKeyword = ref('')

const form = reactive<any>({
  bot_name: '',
  language: 'id',
  tone: 'friendly',
  system_prompt: '',
  business_hours_start: '08:00',
  business_hours_end: '22:00',
  business_days: [1, 2, 3, 4, 5, 6],
  escalation_enabled: true,
  escalation_keywords: ['bicara cs', 'hubungi admin', 'operator', 'manusia', 'human'],
  auto_escalate_after_minutes: 5,
  welcome_message: 'Halo! Ada yang bisa saya bantu?',
  fallback_message: 'Maaf, saya belum bisa menjawab pertanyaan tersebut. Apakah Anda ingin dihubungkan dengan CS kami?',
  outside_hours_message: 'Terima kasih telah menghubungi kami. Saat ini di luar jam operasional. Pesan Anda akan dibalas saat jam kerja.',
  channels_enabled: ['whatsapp'],
  wa_provider_preference: 'auto',
  is_active: true,
  enable_vision: false,
  enable_voice_reply: false,
  voice_model: 'id-ID-GadisNeural',
})

const dayList = [
  { value: 0, short: 'Min' },
  { value: 1, short: 'Sen' },
  { value: 2, short: 'Sel' },
  { value: 3, short: 'Rab' },
  { value: 4, short: 'Kam' },
  { value: 5, short: 'Jum' },
  { value: 6, short: 'Sab' },
]

const toneLabel = computed(() => {
  const map: Record<string, string> = {
    friendly: 'Ramah & Hangat',
    formal: 'Formal',
    casual: 'Santai & Akrab',
    professional: 'Profesional & Solutif',
  }
  return map[form.tone] || form.tone
})

const dayShortList = computed(() => {
  return dayList
    .filter((d) => form.business_days.includes(d.value))
    .map((d) => d.short)
    .join(', ')
})


const loadData = async () => {
  loading.value = true
  try {
    // 1. Load WA Setup
    const resWA = await api.getWASetup()
    if (resWA.success) {
      waSetupState.value = resWA.data
      provider.value = resWA.data.wa_provider_preference
    }
    // 2. Load AI Config
    const resAI = await api.getChatbotConfig()
    if (resAI.success && resAI.data) {
      const d = resAI.data
      if (d.language) form.language = d.language
      if (d.tone) form.tone = d.tone
      if (d.system_prompt) form.system_prompt = d.system_prompt
      if (d.welcome_message) form.welcome_message = d.welcome_message
      if (d.fallback_message) form.fallback_message = d.fallback_message
      if (d.outside_hours_message) form.outside_hours_message = d.outside_hours_message
      if (d.business_hours_start) form.business_hours_start = d.business_hours_start
      if (d.business_hours_end) form.business_hours_end = d.business_hours_end
      if (d.business_days) form.business_days = d.business_days
      form.escalation_enabled = !!d.escalation_enabled
      if (d.escalation_keywords) form.escalation_keywords = d.escalation_keywords
      if (d.auto_escalate_after_minutes) form.auto_escalate_after_minutes = d.auto_escalate_after_minutes
      if (d.channels_enabled) form.channels_enabled = d.channels_enabled
      if (d.wa_provider_preference) form.wa_provider_preference = d.wa_provider_preference
      form.is_active = d.is_active !== false
    }
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

const saveProvider = async () => {
  try {
    const res = await api.updateWAProvider(provider.value as 'auto' | 'whatsmeow' | 'cloud_api')
    if (res.success) {
      const toast = document.createElement('div')
      toast.textContent = '✅ Preferensi provider berhasil disimpan'
      toast.style.cssText = 'position:fixed;bottom:20px;right:20px;background:#10b981;color:white;padding:12px 20px;border-radius:8px;z-index:9999;box-shadow:0 4px 12px rgba(0,0,0,.15);'
      document.body.appendChild(toast)
      setTimeout(() => toast.remove(), 2500)
    }
  } catch (e) {
    console.error(e)
    alert('Gagal menyimpan preferensi')
  }
}

function closeCloudApiModal() {
  cloudApiModal.value = false
  cloudApiError.value = ''
  cloudApiSuccess.value = ''
  cloudApiValidated.value = false
  cloudApiValidationResult.value = ''
  cloudApiForm.phone_number_id = ''
  cloudApiForm.waba_id = ''
  cloudApiForm.access_token = ''
  cloudApiForm.verify_token = ''
}

function validateAndSaveCredential() {
  saveCloudApiCredential()
}

async function openCloudApiModal() {
  cloudApiModal.value = true
  cloudApiError.value = ''
  cloudApiSuccess.value = ''
  cloudApiForm.phone_number_id = ''
  cloudApiForm.waba_id = ''
  cloudApiForm.access_token = ''
  cloudApiForm.verify_token = ''
  try {
    const res = await api.getCloudAPICredential()
    if (res.success && res.data) {
      cloudApiForm.phone_number_id = res.data.phone_number_id || ''
      cloudApiForm.waba_id = res.data.waba_id || ''
      cloudApiForm.verify_token = res.data.verify_token || ''
      // access_token tidak pernah di-return (security)
    }
  } catch {}
}

async function saveCloudApiCredential() {
  if (!cloudApiForm.phone_number_id.trim() || !cloudApiForm.access_token.trim()) {
    cloudApiError.value = 'Phone Number ID dan Access Token wajib diisi.'
    return
  }

  // Step 1: Validasi credential dulu sebelum save
  if (!cloudApiValidated.value) {
    cloudApiValidating.value = true
    cloudApiError.value = ''
    cloudApiSuccess.value = ''
    try {
      const res = await api.validateCloudAPICredential({
        access_token: cloudApiForm.access_token.trim(),
        phone_number_id: cloudApiForm.phone_number_id.trim(),
        waba_id: cloudApiForm.waba_id.trim(),
      })
      if (res.success) {
        cloudApiValidated.value = true
        cloudApiValidationResult.value = 'valid'
        cloudApiSuccess.value = '✅ Credential valid! Silakan klik "Simpan" untuk menyimpan.'
        cloudApiLoading.value = false
      } else {
        cloudApiValidationResult.value = 'invalid'
        cloudApiError.value = res.message || 'Credential tidak valid. Periksa access token dan phone number ID.'
        cloudApiValidated.value = false
        cloudApiLoading.value = false
        return
      }
    } catch (e: any) {
      cloudApiError.value = e?.message || 'Gagal validasi credential'
      cloudApiValidated.value = false
      cloudApiLoading.value = false
      return
    } finally {
      cloudApiValidating.value = false
    }
    return // Jangan save dulu, tunggu user klik "Simpan" setelah validasi sukses
  }

  // Step 2: Simpan credential setelah validasi sukses
  cloudApiLoading.value = true
  cloudApiError.value = ''
  cloudApiSuccess.value = ''
  try {
    const res = await api.saveCloudAPICredential({
      phone_number_id: cloudApiForm.phone_number_id.trim(),
      waba_id: cloudApiForm.waba_id.trim(),
      access_token: cloudApiForm.access_token.trim(),
      verify_token: cloudApiForm.verify_token.trim(),
    })
    if (res.success) {
      cloudApiSuccess.value = '✅ Cloud API credential berhasil disimpan!'
      // Refresh WA setup state
      const resWA = await api.getWASetup()
      if (resWA.success) waSetupState.value = resWA.data
      // Reset state
      cloudApiForm.access_token = ''
      cloudApiValidated.value = false
      cloudApiValidationResult.value = ''
      setTimeout(() => { cloudApiModal.value = false; cloudApiSuccess.value = '' }, 2000)
    } else {
      cloudApiError.value = res.message || 'Gagal menyimpan credential'
      cloudApiValidated.value = false
    }
  } catch (e: any) {
    cloudApiError.value = e?.message || 'Terjadi kesalahan'
    cloudApiValidated.value = false
  } finally {
    cloudApiLoading.value = false
  }
}

// --- QR STATE ---
const qrModal = ref(false)
const qrImage = ref('')
const qrStatus = ref<'loading'|'qr'|'connected'|'error'>('loading')
const qrError = ref('')
let qrPollInterval: ReturnType<typeof setInterval> | null = null

// --- CLOUD API CREDENTIAL STATE ---
const cloudApiModal = ref(false)
const cloudApiLoading = ref(false)
const cloudApiValidating = ref(false)
const cloudApiError = ref('')
const cloudApiSuccess = ref('')
const cloudApiValidated = ref(false)
const cloudApiValidationResult = ref('')
const cloudApiForm = reactive({
  phone_number_id: '',
  waba_id: '',
  access_token: '',
  verify_token: '',
})

function stopQRPolling() {
  if (qrPollInterval) { clearInterval(qrPollInterval); qrPollInterval = null }
}

async function requestQR() {
  qrModal.value = true
  qrImage.value = ''
  qrStatus.value = 'loading'
  qrError.value = ''
  stopQRPolling()

  const poll = async () => {
    try {
      const res = await api.wa('qr')
      if (res.status === 'qr' && res.qr_code) {
        qrImage.value = res.qr_code
        qrStatus.value = 'qr'
        // Poll every 3s until connected or error
        qrPollInterval = setInterval(poll, 3000)
      } else if (res.status === 'connected') {
        qrStatus.value = 'connected'
        qrImage.value = ''
        stopQRPolling()
        // Auto-close after 2s
        setTimeout(() => { qrModal.value = false }, 2000)
        // Refresh WA status
        const resWA = await api.getWASetup()
        if (resWA.success) waSetupState.value = resWA.data
      } else if (res.status === 'busy') {
        qrStatus.value = 'error'
        qrError.value = res.message || 'Gateway sibuk, coba lagi sebentar.'
        stopQRPolling()
      } else if (res.error) {
        qrStatus.value = 'error'
        qrError.value = res.error
        stopQRPolling()
      }
    } catch (e: any) {
      qrStatus.value = 'error'
      qrError.value = e?.message || 'Gagal mengambil QR code'
      stopQRPolling()
    }
  }

  // Stop after 5 minutes to avoid infinite polling
  setTimeout(() => { stopQRPolling() }, 5 * 60 * 1000)
  await poll()
}

// AI Config Actions
function goToStep(i: number) {
  if (i <= currentStep.value) currentStep.value = i
}

function next() {
  if (currentStep.value < steps.length - 1) {
    currentStep.value++
    saveDraft()
  }
}
function prev() {
  if (currentStep.value > 0) currentStep.value--
}

function addKeyword() {
  const k = newKeyword.value.trim()
  if (k && !form.escalation_keywords.includes(k)) {
    form.escalation_keywords.push(k)
  }
  newKeyword.value = ''
}

function saveDraft() {
  try {
    sessionStorage.setItem('chatbot_config_draft', JSON.stringify(form))
  } catch {}
}

function loadDraft() {
  try {
    const raw = sessionStorage.getItem('chatbot_config_draft')
    if (raw) Object.assign(form, JSON.parse(raw))
  } catch {}
}

async function save() {
  errorMsg.value = ''
  if (form.business_hours_start >= form.business_hours_end) {
    errorMsg.value = 'Jam buka harus lebih awal dari jam tutup.'
    return
  }
  if (form.escalation_enabled && form.escalation_keywords.length === 0) {
    errorMsg.value = 'Minimal 1 kata kunci eskalasi jika escalation aktif.'
    return
  }
  if (form.channels_enabled.length === 0) {
    errorMsg.value = 'Minimal 1 channel harus aktif.'
    return
  }
  saving.value = true
  try {
    const payload = {
      language: form.language,
      tone: form.tone,
      system_prompt: form.system_prompt,
      welcome_message: form.welcome_message,
      fallback_message: form.fallback_message,
      outside_hours_message: form.outside_hours_message,
      business_hours_start: form.business_hours_start,
      business_hours_end: form.business_hours_end,
      business_days: form.business_days,
      escalation_enabled: form.escalation_enabled,
      escalation_keywords: form.escalation_keywords,
      auto_escalate_after_minutes: form.auto_escalate_after_minutes,
      channels_enabled: form.channels_enabled,
      is_active: form.is_active,
    }
    const res = await api.updateChatbotConfig(payload)
    if (res.success) {
      sessionStorage.removeItem('chatbot_config_draft')
      const toast = document.createElement('div')
      toast.textContent = '✅ Konfigurasi tersimpan & AI CS aktif'
      toast.style.cssText = 'position:fixed;bottom:20px;right:20px;background:#10b981;color:white;padding:12px 20px;border-radius:8px;z-index:9999;box-shadow:0 4px 12px rgba(0,0,0,.15);'
      document.body.appendChild(toast)
      setTimeout(() => toast.remove(), 2500)
    } else {
      errorMsg.value = res.message || 'Gagal menyimpan'
    }
  } catch (e: any) {
    errorMsg.value = 'Error: ' + (e?.message || e)
  } finally {
    saving.value = false
  }
}

function openTestModal() {
  testOpen.value = true
  testReply.value = ''
  testError.value = ''
  testInput.value = ''
}

async function runTest() {
  if (!testInput.value.trim()) return
  testing.value = true
  testError.value = ''
  try {
    const res = await api.testChatbotConfig(testInput.value)
    if (res.success && res.data) {
      testReply.value = res.data.reply
      testWouldEscalate.value = !!res.data.would_escalate
    } else {
      testError.value = res.message || 'Gagal menjalankan test'
    }
  } catch (e: any) {
    testError.value = 'Error: ' + (e?.message || e)
  } finally {
    testing.value = false
  }
}

onMounted(() => {
  loadDraft()
  loadData()
})

onUnmounted(() => {
  stopQRPolling()
})
</script>

<style scoped>
.tabs {
  margin-bottom: 2rem;
}
.tab-btn {
  background: none;
  border: none;
  padding: 0.5rem 1rem;
  cursor: pointer;
  font-size: 1rem;
  color: var(--text-secondary);
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}
.tab-btn.active {
  color: #4f46e5;
  border-bottom: 2px solid #4f46e5;
  font-weight: 600;
}
.provider-card {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1rem;
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  background: var(--bg-tertiary);
  cursor: pointer;
  transition: all 0.2s;
}
.provider-card:hover:not(.locked) {
  border-color: #4f46e5;
}
.provider-card.active {
  border-color: #4f46e5;
  background: rgba(79, 70, 229, 0.1);
}
.provider-card.locked {
  opacity: 0.6;
  cursor: not-allowed;
}
.provider-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.provider-info .desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
}
.locked-badge {
  background: #dc2626;
  color: white;
  font-size: 0.7rem;
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  width: fit-content;
  margin-top: 0.25rem;
}
.status-box {
  padding: 1rem;
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  background: var(--bg-tertiary);
}
.status-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}
.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #6b7280;
}
.status-box.connected .status-dot { background: #10b981; }
.status-box.qr_pending .status-dot { background: #f59e0b; }
.status-box.disconnected .status-dot { background: #dc2626; }
.status-desc {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-bottom: 1rem;
}

/* AI Config Styles */
.step-pill {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 999px;
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  transition: all 0.2s;
}
.step-pill.active {
  background: #4f46e5;
  color: white;
}
.step-pill.done {
  background: #10b981;
  color: white;
}
.step-num {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.25);
  font-size: 0.75rem;
  font-weight: 600;
}
.radio-pill,
.day-pill,
.channel-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.5rem 0.9rem;
  border-radius: 0.5rem;
  border: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  cursor: pointer;
  font-size: 0.85rem;
  user-select: none;
  transition: all 0.15s;
}
.radio-pill input,
.channel-pill input { margin: 0; }
.day-pill.active,
.channel-pill.active {
  background: #4f46e5;
  color: white;
  border-color: #4f46e5;
}
.channel-pill.locked {
  opacity: 0.6;
  cursor: not-allowed;
}
.toggle-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.9rem;
}
.preview-row {
  display: flex;
  justify-content: space-between;
  padding: 0.35rem 0;
  font-size: 0.85rem;
  border-bottom: 1px dashed var(--border-color);
}
.preview-row:last-child { border-bottom: 0; }
.preview-row span { color: var(--text-secondary); }
.keyword-input {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  align-items: center;
  padding: 0.4rem;
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  background: var(--bg-primary);
}
.kw-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.2rem 0.6rem;
  background: #4f46e5;
  color: white;
  border-radius: 999px;
  font-size: 0.8rem;
}
.kw-tag button {
  background: rgba(255, 255, 255, 0.3);
  border: 0;
  color: white;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  cursor: pointer;
  font-size: 0.9rem;
  line-height: 1;
}
.kw-input {
  flex: 1;
  min-width: 120px;
  border: 0 !important;
  background: transparent !important;
  padding: 0.3rem !important;
}
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-content {
  width: 90%;
  max-width: 480px;
}
@media (max-width: 768px) {
  .config-layout, .setup-layout {
    grid-template-columns: 1fr !important;
  }
  .stepper {
    flex-direction: column;
  }
}
</style>
