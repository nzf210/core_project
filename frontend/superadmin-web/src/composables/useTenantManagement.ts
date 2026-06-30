import { ref, computed } from 'vue'
import { api } from '../api/client'

export function useTenantManagement() {
  const tenants = ref<any[]>([])
  const loading = ref(false)
  const deleting = ref(false)
  const deleteTarget = ref<any>(null)
  const deleteError = ref('')

  const showAddTenant = ref(false)
  const savingAddTenant = ref(false)
  // Password is NOT sent — backend auto-generates. Username is auto-lowercased on input.
  const formData = ref({
    name: '', username: '', email: '',
    phone_number: '', plan: 'lite', subdomain: '', custom_domain: '',
  })

  const editTarget = ref<any>(null)
  const savingProfile = ref(false)
  const profileError = ref('')
  const editForm = ref({
    name: '', business_name: '', wa_number: '', owner_phone: '',
    business_address: '', business_type: 'umum', plan: 'lite',
    subdomain: '', custom_domain: '', xendit_merchant_id: '', logo_url: '',
    owner_email: '',
  })
  const editFormRaw = ref({ new_password: '' })
  const editLogoFile = ref<File | null>(null) // used internally only
  const editLogoPreview = ref('')

  const planCounts = computed(() => {
    const counts: Record<string, number> = {}
    for (const t of tenants.value) {
      const p = t.plan || 'unknown'
      counts[p] = (counts[p] || 0) + 1
    }
    return counts
  })

  const planOptions = ref<any[]>([])

  const fetchTenants = async () => {
    loading.value = true
    try {
      const data = await api.getTenants()
      if (data.success && data.data) tenants.value = data.data
      // ponytail: fetch plans here so modal always has options ready
      if (!planOptions.value.length) {
        const plansRes = await api.listPlans()
        if (plansRes.success && plansRes.data) planOptions.value = plansRes.data
      }
    } finally {
      loading.value = false
    }
  }

  const openEditProfile = async (tenant: any) => {
    editTarget.value = tenant
    profileError.value = ''
    editLogoFile.value = null
    editLogoPreview.value = ''  // ponytail: clear old blob URL
    editFormRaw.value.new_password = ''

    try {
      const res = await api.getTenantProfile(tenant.id)
      if (res.success && res.data) {
        const p = res.data
        editForm.value = {
          name: p.name || '', business_name: p.business_name || '',
          wa_number: p.wa_number || '', owner_phone: p.owner_phone || '',
          business_address: p.business_address || '',
          business_type: p.business_type || 'umum', plan: p.plan || 'lite',
          subdomain: p.subdomain || '', custom_domain: p.custom_domain || '',
          xendit_merchant_id: p.xendit_merchant_id || '', logo_url: p.logo_url || '',
          owner_email: p.owner_email || '',
        }
        // Don't set editLogoPreview — let template fallback to editForm.logo_url
      }
    } catch {
      profileError.value = 'Kesalahan jaringan'
    }
  }

  const onLogoFileChange = (e: Event) => {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (!file) return
    editLogoFile.value = file
    editLogoPreview.value = URL.createObjectURL(file)
  }

  const saveProfile = async () => {
    if (!editTarget.value) return
    savingProfile.value = true
    profileError.value = ''
    try {
      // ponytail: save profile first — logo upload only after user confirms
      const payload: any = { tenant_id: editTarget.value.id, ...editForm.value }
      if (editFormRaw.value.new_password) payload.new_password = editFormRaw.value.new_password

      const result = await api.updateTenantProfile(payload)
      if (!result.success) {
        profileError.value = result.message || 'Gagal menyimpan'
        savingProfile.value = false
        return
      }

      // Upload logo AFTER profile saved successfully
      if (editLogoFile.value) {
        const logoResult = await api.uploadTenantLogo(editTarget.value.id, editLogoFile.value)
        if (logoResult.success) {
          editForm.value.logo_url = logoResult.data?.logo_url || ''
          editLogoPreview.value = logoResult.data?.logo_url || ''
          editLogoFile.value = null
        } else {
          // Profile saved, logo failed — non-fatal, warn user
          profileError.value = 'Profil tersimpan. Logo gagal diupload: ' + (logoResult.message || 'error')
          savingProfile.value = false
          return
        }
      }

      editTarget.value = null
      fetchTenants()
    } catch {
      profileError.value = 'Kesalahan jaringan'
    } finally {
      savingProfile.value = false
    }
  }

  const saveNewTenant = async () => {
    savingAddTenant.value = true
    try {
      // No password — backend auto-generates
      const res = await api.createTenant({
        name: formData.value.name,
        username: formData.value.username,
        email: formData.value.email,
        phone_number: formData.value.phone_number,
        plan: formData.value.plan,
        subdomain: formData.value.subdomain,
        custom_domain: formData.value.custom_domain,
      })
      if (res.success) {
        showAddTenant.value = false
        formData.value = { name: '', username: '', email: '', phone_number: '', plan: 'lite', subdomain: '', custom_domain: '' }
        fetchTenants()
      }
    } finally {
      savingAddTenant.value = false
    }
  }

  const confirmDelete = (tenant: any) => { deleteTarget.value = tenant; deleteError.value = '' }
  const executeDelete = async () => {
    if (!deleteTarget.value) return
    deleting.value = true; deleteError.value = ''
    try {
      const data = await api.deleteTenant(deleteTarget.value.id)
      if (data.success) {
        tenants.value = tenants.value.filter((t: any) => t.id !== deleteTarget.value.id)
        deleteTarget.value = null
      } else {
        deleteError.value = data.message || 'Gagal menghapus'
      }
    } catch {
      deleteError.value = 'Kesalahan jaringan'
    } finally {
      deleting.value = false
    }
  }

  return {
    tenants, loading, deleting, deleteTarget, deleteError,
    showAddTenant, formData, savingAddTenant,
    planCounts, fetchTenants,
    openEditProfile, editTarget, editForm, editFormRaw,
    editLogoFile, editLogoPreview, profileError, savingProfile,
    onLogoFileChange, saveProfile, saveNewTenant,
    confirmDelete, executeDelete,
    planOptions,
    businessTypes: [
      { id: 'umum', name: 'Umum / General' },
      { id: 'warung', name: 'Warung / Toko Kelontong' },
      { id: 'restoran', name: 'Restoran / F&B' },
      { id: 'clinic', name: 'Klinik' },
      { id: 'laundry', name: 'Laundry' },
      { id: 'jasa', name: 'Jasa / Service' },
      { id: 'industri_kreatif', name: 'Industri Kreatif' },
      { id: 'toko_online', name: 'Toko Online' },
    ],
  }
}
