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
  const formData = ref({
    name: '', username: '', password: '', email: '',
    phone_number: '', plan: 'lite', subdomain: '', custom_domain: '',
  })

  const editTarget = ref<any>(null)
  const savingProfile = ref(false)
  const profileError = ref('')
  const editForm = ref({
    name: '', business_name: '', wa_number: '', owner_phone: '',
    owner_username: '', business_address: '', business_type: 'umum', plan: 'lite',
    subdomain: '', custom_domain: '', xendit_merchant_id: '', logo_url: '',
  })
  const editFormRaw = ref({ new_password: '' })
  const editLogoFile = ref<File | null>(null)
  const editLogoPreview = ref('')

  const planCounts = computed(() => {
    const counts: Record<string, number> = {}
    for (const t of tenants.value) {
      const p = t.plan || 'unknown'
      counts[p] = (counts[p] || 0) + 1
    }
    return counts
  })

  const fetchTenants = async () => {
    loading.value = true
    try {
      const data = await api.getTenants()
      if (data.success && data.data) tenants.value = data.data
    } finally {
      loading.value = false
    }
  }

  const openEditProfile = async (tenant: any) => {
    editTarget.value = tenant
    profileError.value = ''
    editLogoFile.value = null
    editFormRaw.value.new_password = ''

    try {
      const res = await api.getTenantProfile(tenant.id)
      if (res.success && res.data) {
        const p = res.data
        editForm.value = {
          name: p.name || '', business_name: p.business_name || '',
          wa_number: p.wa_number || '', owner_phone: p.owner_phone || '',
          owner_username: p.owner_username || '', business_address: p.business_address || '',
          business_type: p.business_type || 'umum', plan: p.plan || 'lite',
          subdomain: p.subdomain || '', custom_domain: p.custom_domain || '',
          xendit_merchant_id: p.xendit_merchant_id || '', logo_url: p.logo_url || '',
        }
        editLogoPreview.value = p.logo_url || ''
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
      if (editLogoFile.value) {
        const logoResult = await api.uploadTenantLogo(editTarget.value.id, editLogoFile.value)
        if (!logoResult.success) {
          profileError.value = logoResult.message || 'Gagal upload logo'
          savingProfile.value = false
          return
        }
        editForm.value.logo_url = logoResult.data?.logo_url || ''
        editLogoFile.value = null
      }
      const payload: any = { tenant_id: editTarget.value.id, ...editForm.value }
      if (editFormRaw.value.new_password) payload.new_password = editFormRaw.value.new_password
      const result = await api.updateTenantProfile(payload)
      if (result.success) {
        editTarget.value = null
        fetchTenants()
      } else {
        profileError.value = result.message || 'Gagal menyimpan'
      }
    } catch {
      profileError.value = 'Kesalahan jaringan'
    } finally {
      savingProfile.value = false
    }
  }

  const saveNewTenant = async () => {
    savingAddTenant.value = true
    try {
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
        formData.value = { name: '', username: '', password: '', email: '', phone_number: '', plan: 'lite', subdomain: '', custom_domain: '' }
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
