<template>
  <div class="modal-overlay" @click.self="$emit('update:showAddonEditor', false)">
    <div class="modal-card" style="max-width: 600px;">
      <h3 style="margin: 0 0 0.25rem 0;">Kelola Harga Add-on</h3>
      <div style="display: flex; justify-content: space-between; align-items: center; gap: 0.75rem; margin-bottom: 1rem;">
        <p style="color: var(--text-secondary); font-size: 0.85rem; margin: 0;">
          Ubah harga fitur add-on untuk marketplace wallet.
        </p>
        <button class="btn btn-primary" style="padding: 0.35rem 0.75rem; font-size: 0.8rem;" @click="$emit('create-addon')">
          + Buat Addon Baru
        </button>
      </div>

      <div v-if="showAddAddonForm" class="form-group" style="background: rgba(59,130,246,0.08); border: 1px solid rgba(59,130,246,0.3); padding: 1rem; border-radius: var(--radius-sm); margin-bottom: 1rem;">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem;">
          <strong>Buat Addon Baru</strong>
          <button class="btn" style="padding: 0.2rem 0.5rem; font-size: 0.75rem;" @click="$emit('update:showAddAddonForm', false)">Batal</button>
        </div>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
          <div>
            <label style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Key Addon
              <input v-model.trim="newAddon.feature_key" type="text" class="form-control" placeholder="extra_store" /></label>
          </div>
          <div>
            <label style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Nama
              <input v-model.trim="newAddon.feature_name" type="text" class="form-control" placeholder="Extra Store" /></label>
          </div>
        </div>
        <div style="margin-top: 0.75rem;">
          <label style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Kategori
            <input v-model.trim="newAddon.category" type="text" class="form-control" placeholder="growth" /></label>
        </div>
        <div style="margin-top: 0.75rem;">
          <label style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Deskripsi
            <input v-model.trim="newAddon.description" type="text" class="form-control" placeholder="Tambah jumlah toko" /></label>
        </div>
        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 0.75rem;">
          <button class="btn btn-secondary" @click="$emit('create-addon')">Simpan Addon</button>
        </div>
      </div>

      <div v-if="loadingAddons" style="text-align: center; padding: 2rem; color: var(--text-secondary);">
        Memuat data add-on...
      </div>
      <div v-else style="max-height: 60vh; overflow-y: auto; padding-right: 0.5rem;">
        <div v-for="addon in addonOptions" :key="addon.addon_key" class="addon-card">
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem;">
            <div style="font-weight: 600; text-transform: uppercase;">{{ addon.addon_key.replace(/_/g, ' ') }}</div>
            <label style="display: flex; align-items: center; gap: 0.5rem; margin: 0; font-size: 0.85rem; font-weight: 400;">
              <input type="checkbox" v-model="addon.is_active" /> Aktif
            </label>
          </div>

          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
            <div>
              <label style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Harga (Rp)
                <div style="position: relative;">
                  <span style="position: absolute; left: 0.75rem; top: 50%; transform: translateY(-50%); color: var(--text-secondary); font-size: 0.85rem;">Rp</span>
                  <input v-model.number="addon.price" type="number" class="form-control" style="padding-left: 2rem;" />
                </div>
              </label>
            </div>
            <div>
              <label style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Unit (mis: bulan, trx)
                <input v-model="addon.unit" type="text" class="form-control" /></label>
            </div>
          </div>
          <div style="margin-top: 0.75rem;">
            <label style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Deskripsi
              <input v-model="addon.description" type="text" class="form-control" /></label>
          </div>
          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; margin-top: 0.75rem;">
            <div>
              <label style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Tier minimum
                <select v-model="addon.min_tier" class="form-control">
                  <option value="">Semua tier</option>
                  <option value="lite">Lite</option>
                  <option value="pro">Pro</option>
                  <option value="ultimate">Ultimate</option>
                </select>
            </div>
            <div>
              <span style="font-size: 0.75rem; margin-bottom: 0.25rem; display: block;">Default aktif di tier</span>
              <div style="display: flex; gap: 0.5rem; flex-wrap: wrap;">
                <label v-for="t in ['lite', 'pro', 'ultimate']" :key="t" style="font-size: 0.75rem; display: flex; align-items: center; gap: 0.25rem;">
                  <input type="checkbox" :value="t" v-model="addon.default_enabled" />
                  <span>{{ t.charAt(0).toUpperCase() + t.slice(1) }}</span>
                </label>
              </div>
            </div>
          </div>
          <div style="margin-top: 0.75rem; text-align: right;">
            <button class="btn" style="background: #ef444420; color: #ef4444; border: 1px solid #ef444440; padding: 0.25rem 0.6rem; font-size: 0.75rem;" @click="$emit('delete-addon', addon)" :disabled="deletingAddon === addon.addon_key">
              {{ deletingAddon === addon.addon_key ? 'Menghapus...' : 'Hapus' }}
            </button>
          </div>
        </div>

        <div v-if="addonSaveMsg" :style="{ color: addonSaveMsg.includes('gagal') ? '#ef4444' : '#10b981', fontSize: '0.85rem', marginBottom: '1rem', textAlign: 'center' }">
          {{ addonSaveMsg }}
        </div>
      </div>

      <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 1.5rem;">
        <button class="btn btn-secondary" @click="$emit('update:showAddonEditor', false)">Tutup</button>
        <button class="btn btn-primary" @click="$emit('save')" :disabled="loadingAddons || savingAddons">
          {{ savingAddons ? 'Menyimpan...' : 'Simpan Perubahan' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  showAddonEditor: boolean
  loadingAddons: boolean
  savingAddons: boolean
  addonOptions: any[]
  addonSaveMsg: string
  showAddAddonForm: boolean
  deletingAddon: any
  newAddon: any
}>()

defineEmits<{
  'update:showAddonEditor': [value: boolean]
  'save': []
  'create-addon': []
  'delete-addon': [addon: any]
}>()
</script>

<style scoped>
.form-control {
  width: 100%;
  padding: 0.6rem 0.75rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.875rem;
  box-sizing: border-box;
}

.form-control:focus {
  outline: none;
  border-color: var(--accent-primary);
}

.addon-card {
  background: var(--surface-0);
  border: 1px solid var(--border-color);
  padding: 1rem;
  border-radius: var(--radius-sm);
  margin-bottom: 1rem;
}
</style>
