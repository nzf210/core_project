import { ref, computed, onMounted, watch } from 'vue'
import { superadminApi } from '../superadminApi'
import { api, API_BASE } from '../api'
import { useModalState } from '../utils/modalState'

export function useSuperAdmin() {
  const { openModal, closeModal } = useModalState()

  const verifierStatus = ref<'connected' | 'disconnected'>('disconnected')
  const verifierJID = ref('')
  const qrCode = ref('')
  const loadingQR = ref(false)
  const checkingStatus = ref(false)
  const disconnecting = ref(false)
  const tenants = ref<any[]>([])
  const deleting = ref(false)
  const deleteTarget = ref<any>(null)
  const deleteError = ref('')
  const loadingTenants = ref(false)

  const showAddTenant = ref(false)
  const formData = ref({
    name: '',
    username: '',
    password: '',
    email: '',
    phone_number: '',
    role: 'owner',
    plan: 'lite',
    subdomain: '',
    custom_domain: ''
  })

  const editTarget = ref<any>(null)
  const editForm = ref({
    tenant_id: '',
    name: '',
    business_name: '',
    wa_number: '',
    owner_phone: '',
    business_address: '',
    business_type: '',
    plan: 'lite',
    new_password: '',
    logo_url: '',
    subdomain: '',
    custom_domain: '',
    xendit_merchant_id: ''
  })
  const editLogoFile = ref<File | null>(null)
  const editLogoPreview = ref('')
  const uploadingLogo = ref(false)
  const savingProfile = ref(false)
  const profileError = ref('')

  // Track all modals for body blur
  watch(showAddTenant, (v) => { if (v) openModal(); else closeModal(); });
  watch(editTarget, (v) => { if (v) openModal(); else closeModal(); });
  watch(deleteTarget, (v) => { if (v) openModal(); else closeModal(); });

  const showGenerateVoucherModal = ref(false)
  const showVoucherListModal = ref(false)
  const showPlanEditor = ref(false)
  const showAddonEditor = ref(false)

  // F057: Feature Matrix
  const showFeatureMatrix = ref(false)
  const featureMatrixLoading = ref(false)
  const featureMatrixPlans = ref<Record<string, any>>({})
  const featureMatrixPlanIds = ref<string[]>([])
  const featureMatrixOrder = ref<string[]>([])
  const featureMatrixData = ref<Record<string, Record<string, any>>>({})
  const addonGatingLoading = ref(false)
  const addonGatingList = ref<any[]>([])

  watch(showGenerateVoucherModal, (v) => { if (v) openModal(); else closeModal(); });
  watch(showVoucherListModal, (v) => { if (v) openModal(); else closeModal(); });
  watch(showPlanEditor, (v) => { if (v) openModal(); else closeModal(); });
  watch(showAddonEditor, (v) => { if (v) openModal(); else closeModal(); });
  watch(showFeatureMatrix, (v) => { if (v) { openModal(); loadFeatureMatrix(); } else closeModal(); });

  const businessTypes = [
    { id: 'umum', name: 'Umum / General' },
    { id: 'warung', name: 'Warung / Toko Kelontong' },
    { id: 'laundry', name: 'Laundry' },
    { id: 'industri_kreatif', name: 'Industri Kreatif' },
    { id: 'toko_online', name: 'Toko Online / E-Commerce' },
    { id: 'restoran', name: 'Restoran / F&B' },
    { id: 'jasa', name: 'Jasa / Service' },
  ]

  const toast = ref({ visible: false, message: '', type: 'success' })
  const toastTop = ref(0)
  const showToast = (message: string, type: 'success' | 'error' = 'success') => {
    toast.value = { visible: true, message, type }
    toastTop.value = window.scrollY + 16
    setTimeout(() => { toast.value.visible = false }, 3000)
  }

  // Self-profile (super admin's own profile)
  const showMyProfile = ref(false)
  const myProfile = ref({ username: '', phone_number: '', old_password: '', new_password: '' })
  const savingMyProfile = ref(false)
  const role = computed(() => localStorage.getItem('role') || 'Super Admin')
  const myProfileError = ref('')

  watch(showMyProfile, (v) => { if (v) openModal(); else closeModal(); })

  const loadMyProfile = async () => {
    try {
      const data = await api.get('/api/profile')
      if (data.success && data.data) {
        myProfile.value = {
          username: data.data.username || '',
          phone_number: data.data.phone_number || '',
          old_password: '',
          new_password: '',
        }
      }
    } catch (e) {
      console.error('Failed to load my profile', e)
    }
  }

  const saveMyProfile = async () => {
    savingMyProfile.value = true
    myProfileError.value = ''
    try {
      const payload: any = {}
      if (myProfile.value.username) payload.username = myProfile.value.username
      if (myProfile.value.phone_number) payload.phone_number = myProfile.value.phone_number
      if (myProfile.value.new_password) {
        if (!myProfile.value.old_password) {
          myProfileError.value = 'Password lama harus diisi untuk mengganti password'
          savingMyProfile.value = false
          return
        }
        payload.old_password = myProfile.value.old_password
        payload.new_password = myProfile.value.new_password
      }
      const data = await api.put('/api/profile', payload)
      if (data.success) {
        showToast('Profil berhasil disimpan')
        myProfile.value.old_password = ''
        myProfile.value.new_password = ''
        showMyProfile.value = false
      } else {
        myProfileError.value = data.message || 'Gagal menyimpan'
      }
    } catch (e) {
      console.error('saveMyProfile failed', e)
      myProfileError.value = 'Kesalahan jaringan'
    } finally {
      savingMyProfile.value = false
    }
  }

  const planCounts = computed(() => {
    const counts = { free: 0, lite: 0, pro: 0, ultimate: 0 }
    tenants.value.forEach((t: any) => {
      const plan = t.plan as keyof typeof counts
      if (counts[plan] !== undefined) {
        counts[plan]++
      }
    })
    return counts
  })

  const fetchTenants = async () => {
    loadingTenants.value = true
    try {
      const data = await superadminApi.getTenants()
      if (data.success && data.data) {
        tenants.value = data.data
      }
    } catch (e) {
      console.error('Failed to fetch tenants', e)
    } finally {
      loadingTenants.value = false
    }
  }

  const confirmDelete = (tenant: any) => {
    deleteTarget.value = tenant
    deleteError.value = ''
  }

  // Get current superadmin's own tenant ID (stored after login)
  const myTenantId = computed(() => localStorage.getItem('tenant_id') || '')

  // Check if a tenant is the superadmin's own tenant
  const isMyOwnTenant = (tenant: any) => tenant.id === myTenantId.value

  const executeDelete = async () => {
    if (!deleteTarget.value) return
    deleting.value = true
    deleteError.value = ''

    try {
      const data = await superadminApi.deleteTenant(deleteTarget.value.id)
      if (data.success) {
        tenants.value = tenants.value.filter((t: any) => t.id !== deleteTarget.value.id)
        showToast('Tenant berhasil dihapus', 'success')
        deleteTarget.value = null
      } else {
        deleteError.value = data.message || 'Gagal menghapus tenant'
      }
    } catch (e) {
      console.error('deleteTenant failed', e)
      deleteError.value = 'Kesalahan jaringan saat menghapus tenant'
    } finally {
      deleting.value = false
    }
  }

  const checkVerifierStatus = async () => {
    checkingStatus.value = true
    try {
      const data = await superadminApi.getVerifierStatus()
      if (data.success && data.data) {
        verifierStatus.value = data.data.status === 'connected' ? 'connected' : 'disconnected'
        verifierJID.value = data.data.jid || ''
        if (verifierStatus.value === 'connected') {
          qrCode.value = ''
          showToast('WhatsApp Verifier berhasil terhubung!', 'success')
        }
      }
    } catch (e) {
      console.error('checkVerifierStatus failed', e)
      showToast('Gagal memeriksa status verifier', 'error')
    } finally {
      checkingStatus.value = false
    }
  }

  const connectVerifier = async () => {
    loadingQR.value = true
    try {
      const data = await superadminApi.getVerifierQR()
      if (data.success && data.data) {
        if (data.data.status === 'qr') {
          qrCode.value = data.data.qr_code
        } else if (data.data.status === 'connected') {
          verifierStatus.value = 'connected'
          verifierJID.value = data.data.jid || ''
          showToast('WhatsApp Verifier sudah terhubung!', 'success')
        }
      } else {
        showToast(data.message || 'Gagal mendapatkan QR code', 'error')
      }
    } catch (e) {
      console.error('connectVerifier failed', e)
      showToast('Gagal menghubungkan verifier', 'error')
    } finally {
      loadingQR.value = false
    }
  }

  const disconnectVerifier = async () => {
    disconnecting.value = true
    try {
      const data = await superadminApi.disconnectVerifier()
      if (data.success) {
        verifierStatus.value = 'disconnected'
        verifierJID.value = ''
        qrCode.value = ''
        showToast('WhatsApp Verifier telah diputuskan', 'success')
      } else {
        showToast(data.message || 'Gagal memutuskan verifier', 'error')
      }
    } catch (e) {
      console.error('disconnectVerifier failed', e)
      showToast('Gagal memutuskan verifier', 'error')
    } finally {
      disconnecting.value = false
    }
  }

  const openEditProfile = async (tenant: any) => {
    editTarget.value = tenant
    profileError.value = ''
    editLogoFile.value = null
    editLogoPreview.value = ''
    editForm.value.new_password = ''
    editForm.value.logo_url = ''

    try {
      const data = await superadminApi.getTenantProfile(tenant.id)
      if (data.success && data.data) {
        const p = data.data
        editForm.value.name = p.name || ''
        editForm.value.business_name = p.business_name || ''
        editForm.value.wa_number = p.wa_number || ''
        editForm.value.business_address = p.business_address || ''
        editForm.value.business_type = p.business_type || 'umum'
        editForm.value.plan = p.plan || 'lite'
        editForm.value.logo_url = p.logo_url || ''
        editForm.value.owner_phone = p.owner_phone || ''
        editForm.value.subdomain = p.subdomain || ''
        editForm.value.custom_domain = p.custom_domain || ''
        editForm.value.xendit_merchant_id = p.xendit_merchant_id || ''
      } else {
        profileError.value = 'Gagal memuat profil'
      }
    } catch (e) {
      console.error('openEditProfile getProfile failed', e)
      profileError.value = 'Kesalahan jaringan'
    }
  }

  const onLogoFileChange = (e: Event) => {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (!file) return
    editLogoFile.value = file
    editLogoPreview.value = URL.createObjectURL(file)
  }

  const uploadLogo = async () => {
    if (!editLogoFile.value || !editTarget.value) return
    uploadingLogo.value = true
    try {
      const result = await superadminApi.uploadTenantLogo(editTarget.value.id, editLogoFile.value)
      if (result.success) {
        editForm.value.logo_url = result.data?.logo_url || ''
        editLogoFile.value = null
        showToast('Logo berhasil diupload', 'success')
      } else {
        profileError.value = result.message || 'Gagal upload logo'
      }
    } catch (e) {
      console.error('uploadLogo failed', e)
      profileError.value = 'Kesalahan jaringan saat upload'
    } finally {
      uploadingLogo.value = false
    }
  }

  const saveProfile = async () => {
    if (!editTarget.value) return
    savingProfile.value = true
    profileError.value = ''

    try {
      if (editLogoFile.value) {
        const logoResult = await superadminApi.uploadTenantLogo(editTarget.value.id, editLogoFile.value)
        if (!logoResult.success) {
          profileError.value = logoResult.message || 'Gagal upload logo'
          savingProfile.value = false
          return
        }
        editForm.value.logo_url = logoResult.data?.logo_url || ''
        editLogoFile.value = null
      }

      const payload: any = {
        tenant_id: editTarget.value.id,
        name: editForm.value.name,
        business_name: editForm.value.business_name,
        wa_number: editForm.value.wa_number,
        owner_phone: editForm.value.owner_phone,
        business_address: editForm.value.business_address,
        business_type: editForm.value.business_type,
        plan: editForm.value.plan,
        subdomain: editForm.value.subdomain,
        custom_domain: editForm.value.custom_domain,
        xendit_merchant_id: editForm.value.xendit_merchant_id
      }
      if (editForm.value.new_password) {
        payload.new_password = editForm.value.new_password
      }

      const result = await superadminApi.updateTenantProfile(payload)
      if (result.success) {
        showToast('Profil tenant berhasil disimpan', 'success')
        editTarget.value = null
        fetchTenants()
      } else {
        profileError.value = result.message || 'Gagal menyimpan'
      }
    } catch (e) {
      console.error('saveProfile failed', e)
      profileError.value = 'Kesalahan jaringan'
    }
  }

  const closeAddModal = () => {
    showAddTenant.value = false
    formData.value = { name: '', username: '', password: '', email: '', phone_number: '', role: 'owner', plan: 'lite', subdomain: '', custom_domain: '' }
  }

  // ── Plan Editor ──────────────────────────────────────────────────────────────────

  const editablePlans = ref<any[]>([])
  const loadingPlans = ref(false)
  const savingPlans = ref(false)
  const planError = ref('')
  const planOptions = ref<any[]>([])

  // ── Voucher Generation ────────────────────────────────────────────────────────
  const generatingVoucher = ref(false)
  const voucherError = ref('')
  const voucherForm = ref({
    program_name: '',
    plan_id: '',
    quantity: 10,
    validity_days: 30,
    voucher_type: 'bonus_months',
    discount_value: 0,
    max_uses: null,
  })
  const voucherList = ref<any[]>([])
  const loadingVoucherList = ref(false)
  const voucherListFilter = ref({ used: '', plan_id: '' })
  const generatedVoucherCodes = ref<any[]>([])
  const deletingVoucherId = ref<string | null>(null)

  const openGenerateVoucher = async () => {
    voucherError.value = ''
    generatedVoucherCodes.value = []
    voucherForm.value = { program_name: '', plan_id: '', quantity: 10, validity_days: 30, voucher_type: 'bonus_months', discount_value: 0, max_uses: null }
    showGenerateVoucherModal.value = true
    // Fetch latest plan prices from backend
    try {
      const data = await superadminApi.getPlans()
      const plans = data.data || (data.success && data.data)
      if (plans && Array.isArray(plans)) {
        planOptions.value = plans
      }
    } catch (e) {
      console.error('Failed to fetch plan options', e)
    }
  }

  const openVoucherList = async () => {
    voucherListFilter.value = { used: '', plan_id: '' }
    fetchVoucherList()
    showVoucherListModal.value = true
    // Fetch latest plan prices from backend
    try {
      const data = await superadminApi.getPlans()
      const plans = data.data || (data.success && data.data)
      if (plans && Array.isArray(plans)) {
        planOptions.value = plans
      }
    } catch (e) {
      console.error('Failed to fetch plan options', e)
    }
  }

  const executeGenerateVoucher = async () => {
    if (!voucherForm.value.plan_id || !voucherForm.value.quantity || !voucherForm.value.validity_days) return
    generatingVoucher.value = true
    voucherError.value = ''
    try {
      const data = await superadminApi.generateVouchers({
        plan_id: voucherForm.value.plan_id,
        validity_days: voucherForm.value.validity_days,
        quantity: voucherForm.value.quantity,
        voucher_type: voucherForm.value.voucher_type,
        discount_value: voucherForm.value.discount_value,
        program_name: voucherForm.value.program_name || undefined,
        max_uses: voucherForm.value.max_uses || undefined,
      })
      if (data.success || data.status === 200) {
        generatedVoucherCodes.value = data.data?.codes || []
        showToast(`Berhasil generate ${data.data?.count || 0} voucher!`, 'success')
      } else {
        voucherError.value = data.message || 'Gagal generate voucher'
      }
    } catch (e) {
      console.error(e)
      voucherError.value = 'Kesalahan jaringan'
    } finally {
      generatingVoucher.value = false
    }
  }

  const copyToClipboard = async (text: string, msg?: string) => {
    try {
      await navigator.clipboard.writeText(text)
      showToast(msg || 'Berhasil disalin!', 'success')
    } catch (e) {
      console.error('clipboard copy failed', e)
      showToast('Gagal menyalin', 'error')
    }
  }

  const deleteVoucher = async (id: string, code: string) => {
    if (!confirm(`Hapus voucher "${code}"? Voucher yang belum terpakai akan dihapus permanen.`)) return
    deletingVoucherId.value = id
    try {
      const data = await superadminApi.deleteVoucher(id)
      if (data.success || data.status === 200) {
        voucherList.value = voucherList.value.filter(v => v.id !== id)
        showToast(`Voucher ${code} berhasil dihapus`, 'success')
      } else {
        showToast(data.message || 'Gagal menghapus voucher', 'error')
      }
    } catch (e) {
      console.error(e)
      showToast('Kesalahan jaringan', 'error')
    } finally {
      deletingVoucherId.value = null
    }
  }

  const downloadVoucherCSV = () => {
    if (!generatedVoucherCodes.value.length) return
    const header = 'code,validity_days\n'
    const rows = generatedVoucherCodes.value.map((v: any) => `${v.code},${v.days}`).join('\n')
    const csv = header + rows
    const blob = new Blob([csv], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `vouchers-${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  const copyText = (text: string) => {
    navigator.clipboard.writeText(text).then(() => {
      showToast('Kode berhasil disalin!', 'success')
    })
  }

  const fetchVoucherList = async () => {
    loadingVoucherList.value = true
    try {
      const data = await superadminApi.listVouchers({
        used: voucherListFilter.value.used || undefined,
        plan_id: voucherListFilter.value.plan_id || undefined,
        limit: 200,
      })
      if ((data.success || data.status === 200) && data.data?.codes) {
        voucherList.value = data.data.codes
      } else if (data.data && Array.isArray(data.data)) {
        voucherList.value = data.data
      } else {
        voucherList.value = []
      }
    } catch (e) {
      console.error('Failed to fetch voucher list', e)
      voucherList.value = []
    } finally {
      loadingVoucherList.value = false
    }
  }

  // Add-on state & functions
  const loadingAddons = ref(false)
  const savingAddons = ref(false)
  const addonOptions = ref<any[]>([])
  const addonSaveMsg = ref('')
  const showAddAddonForm = ref(false)
  const deletingAddon = ref<string | null>(null)
  const newAddon = ref({
    feature_key: '',
    feature_name: '',
    description: '',
    category: 'growth',
  })

  const openAddonEditor = async () => {
    showAddonEditor.value = true
    addonSaveMsg.value = ''
    loadingAddons.value = true
    try {
      const [featuresRes, gatingRes] = await Promise.all([
        superadminApi.getAvailableFeatures(),
        superadminApi.getAddonGating()
      ])
      const features = featuresRes.success ? (featuresRes.data || []) : []
      const gating: any[] = gatingRes.success ? (gatingRes.data || []) : []
      const gatingMap = new Map(gating.map((g: any) => [g.feature_key, g]))
      addonOptions.value = features.map((a: any) => ({
        ...a,
        price: Math.round((a.addon_price_rupiah || 0) ),
        min_tier: gatingMap.get(a.feature_key)?.min_tier || ''
      }))
      if (!featuresRes.success) addonSaveMsg.value = featuresRes.message || 'Gagal memuat features'
    } catch (e) {
      console.error('loadAddonOptions failed', e)
      addonSaveMsg.value = 'Kesalahan jaringan memuat add-on'
    } finally {
      loadingAddons.value = false
    }
  }

  const createAddon = async () => {
    if (!newAddon.value.feature_key || !newAddon.value.feature_name) {
      addonSaveMsg.value = 'Key dan nama addon wajib diisi'
      return
    }
    addonSaveMsg.value = ''
    try {
      const result = await superadminApi.upsertAvailableFeature({
        ...newAddon.value,
        is_addon: true,
        addon_price_rupiah: 0,
        addon_unit: 'month',
        default_enabled: [],
      })
      if (!result.success) {
        addonSaveMsg.value = result.message || 'Gagal menyimpan addon'
        return
      }
      addonSaveMsg.value = 'Addon berhasil dibuat'
      showAddAddonForm.value = false
      newAddon.value = {
        feature_key: '',
        feature_name: '',
        description: '',
        category: 'growth',
      }
      await openAddonEditor()
    } catch (e) {
      console.error('saveAddon failed', e)
      addonSaveMsg.value = 'Kesalahan jaringan saat menyimpan addon'
    }
  }

  const deleteAddon = async (addon: any) => {
    if (!confirm(`Hapus addon ${addon.addon_key}?`)) return
    deletingAddon.value = addon.addon_key
    try {
      const result = await superadminApi.deleteAvailableFeature(addon.addon_key)
      if (!result.success) {
        addonSaveMsg.value = result.message || 'Gagal menghapus addon'
        return
      }
      addonSaveMsg.value = 'Addon berhasil dihapus'
      addonOptions.value = addonOptions.value.filter((a: any) => a.addon_key !== addon.addon_key)
    } catch (e) {
      console.error('deleteAddon failed', e)
      addonSaveMsg.value = 'Kesalahan jaringan saat menghapus addon'
    } finally {
      deletingAddon.value = null
    }
  }

  const saveAddons = async () => {
    savingAddons.value = true
    addonSaveMsg.value = ''
    let hasError = false
    try {
      for (const addon of addonOptions.value) {
        const payload = {
          feature_name: addon.feature_name,
          description: addon.description,
          addon_price_rupiah: Math.round((addon.price || 0) * 100),
          addon_unit: addon.unit || 'month',
        }
        const r1 = await superadminApi.updateAvailableFeature(addon.addon_key, payload)
        if (!r1.success) hasError = true
        const r2 = await superadminApi.updateAddonGating({
          feature_key: addon.addon_key,
          min_tier: addon.min_tier || null,
          default_enabled: addon.default_enabled || [],
        })
        if (!r2.success) hasError = true
      }
      addonSaveMsg.value = hasError ? 'Beberapa perubahan gagal disimpan' : 'Semua perubahan berhasil disimpan'
      if (!hasError) await openAddonEditor()
    } catch (e) {
      console.error('saveAllAddons failed', e)
      addonSaveMsg.value = 'Kesalahan jaringan saat menyimpan'
    } finally {
      savingAddons.value = false
    }
  }

  const openPlanEditor = async () => {
    showPlanEditor.value = true
    planError.value = ''
    loadingPlans.value = true
    try {
      const data = await superadminApi.getPlans()
      const plans = data.data || (data.success && data.data)
      if (plans && Array.isArray(plans)) {
        editablePlans.value = plans.map((p: any) => ({
          ...p,
          price_monthly_display: Math.round((p.price_monthly || 0) ),
          price_yearly_display: Math.round((p.price_yearly || 0) ),
        }))
      } else {
        planError.value = 'Gagal memuat daftar paket'
      }
    } catch (e) {
      console.error('loadPlanEditor failed', e)
      planError.value = 'Kesalahan jaringan'
    } finally {
      loadingPlans.value = false
    }
  }

  const syncPlanPrice = (plan: any, kind: 'monthly' | 'yearly') => {
    if (kind === 'monthly') {
      plan.price_monthly = (plan.price_monthly_display || 0) * 100
    } else {
      plan.price_yearly = (plan.price_yearly_display || 0) * 100
    }
  }

  const savePlanPrices = async () => {
    savingPlans.value = true
    planError.value = ''
    try {
      let allOk = true
      for (const plan of editablePlans.value) {
        const result = await superadminApi.updatePlan(plan.id, {
          price_monthly: plan.price_monthly || 0,
          price_yearly: plan.price_yearly || 0,
          is_active: plan.is_active,
          sort_order: plan.sort_order || 0,
        })
        if (!result.success && result.status !== 200) {
          allOk = false
          planError.value = `Gagal menyimpan paket ${plan.name}: ${result.message}`
        }
      }
      if (allOk) {
        showToast('Harga paket berhasil diperbarui')
        showPlanEditor.value = false
      }
    } catch (e) {
      console.error('updatePlans failed', e)
      planError.value = 'Kesalahan jaringan'
    } finally {
      savingPlans.value = false
    }
  }

  // F057: Feature Matrix
  const openFeatureMatrix = async () => {
    showFeatureMatrix.value = true
  }

  const loadFeatureMatrix = async () => {
    featureMatrixLoading.value = true
    addonGatingLoading.value = true
    try {
      const [matrixRes, addonRes] = await Promise.all([
        superadminApi.getFeatureMatrix(),
        superadminApi.getAddonGating(),
      ])
      if (matrixRes?.success) {
        featureMatrixPlans.value = matrixRes.data?.plans || {}
        featureMatrixPlanIds.value = matrixRes.data?.plan_ids || []
        featureMatrixOrder.value = matrixRes.data?.feature_order || []
        featureMatrixData.value = matrixRes.data?.matrix || {}
      }
      if (addonRes?.success) {
        addonGatingList.value = addonRes.data || []
      }
    } catch (e) {
      console.error('Failed to load feature matrix', e)
    } finally {
      featureMatrixLoading.value = false
      addonGatingLoading.value = false
    }
  }

  const getFeatureEnabled = (planId: string, featureKey: string): boolean => {
    return featureMatrixData.value[planId]?.[featureKey]?.is_enabled ?? false
  }

  const isAddonFeature = (key: string): boolean => {
    return ['ai_vision', 'ai_audio', 'wa_blast', 'extra_store', 'extra_user'].includes(key)
  }

  const toggleFeature = async (planId: string, featureKey: string, event: Event) => {
    const target = event.target as HTMLInputElement
    const newVal = target.checked
    // Optimistic update
    if (!featureMatrixData.value[planId]) featureMatrixData.value[planId] = {}
    featureMatrixData.value[planId][featureKey] = { ...featureMatrixData.value[planId][featureKey], is_enabled: newVal }
    try {
      await superadminApi.toggleFeature({ plan_id: planId, feature_key: featureKey, is_enabled: newVal })
    } catch (e) {
      console.error('toggleFeature failed', e)
      // Revert on failure
      if (featureMatrixData.value[planId]) {
        featureMatrixData.value[planId][featureKey] = { ...featureMatrixData.value[planId][featureKey], is_enabled: !newVal }
      }
      target.checked = !newVal
      showToast('Gagal toggle fitur', 'error')
    }
  }

  const saveAddonMinTier = async (featureKey: string, minTier: string) => {
    const addon = addonGatingList.value.find(a => a.feature_key === featureKey)
    if (!addon) return
    try {
      await superadminApi.updateAddonGating({
        feature_key: featureKey,
        min_tier: minTier || undefined,
        default_enabled: addon.default_enabled || [],
      })
      addon.min_tier = minTier || null
      showToast('Addon gating disimpan')
    } catch (e) {
      console.error('saveAddonMinTier failed', e)
      showToast('Gagal menyimpan addon gating', 'error')
    }
  }

  const saveNewTenant = async () => {
    try {
      const data = await api.post('/api/umkm/admin/tenants', {
        name: formData.value.name,
        username: formData.value.username,
        email: formData.value.email,
        phone_number: formData.value.phone_number,
        plan: formData.value.plan,
        subdomain: formData.value.subdomain,
        custom_domain: formData.value.custom_domain
      })
      if (data.success) {
        showToast("Berhasil mendaftarkan UMKM baru!", "success")
        closeAddModal()
        fetchTenants()
      } else {
        showToast("Gagal: " + data.message, "error")
      }
    } catch (e) {
      console.error('generateVoucher failed', e)
      showToast("Terjadi kesalahan jaringan.", "error")
    }
  }

  onMounted(async () => {
    checkVerifierStatus()
    fetchTenants()
    loadMyProfile()
    // Load plan options for dropdowns
    try {
      const data = await superadminApi.getPlans()
      const plans = data.data || (data.success && data.data)
      if (plans && Array.isArray(plans)) {
        planOptions.value = plans
      }
    } catch (e) {
      console.error('Failed to fetch plan options', e)
    }
  })

  return {
    verifierStatus, verifierJID, qrCode, loadingQR, checkingStatus, disconnecting,
    tenants, deleting, deleteTarget, deleteError, loadingTenants,
    showAddTenant, formData, editTarget, editForm, editLogoFile, editLogoPreview,
    uploadingLogo, savingProfile, profileError,
    showGenerateVoucherModal, showVoucherListModal, showPlanEditor, showAddonEditor,
    showFeatureMatrix, featureMatrixLoading, featureMatrixPlans, featureMatrixPlanIds,
    featureMatrixOrder, featureMatrixData, addonGatingLoading, addonGatingList,
    businessTypes, toast, toastTop, showToast,
    showMyProfile, myProfile, savingMyProfile, saveMyProfile, role, myProfileError,
    planCounts, fetchTenants, confirmDelete, isMyOwnTenant, executeDelete,
    checkVerifierStatus, connectVerifier, disconnectVerifier,
    openEditProfile, onLogoFileChange, uploadLogo, saveProfile, closeAddModal,
    editablePlans, loadingPlans, savingPlans, planError, planOptions,
    generatingVoucher, voucherError, voucherForm, voucherList, loadingVoucherList,
    voucherListFilter, generatedVoucherCodes, deletingVoucherId,
    openGenerateVoucher, openVoucherList, executeGenerateVoucher, copyToClipboard,
    deleteVoucher, downloadVoucherCSV, copyText, fetchVoucherList,
    loadingAddons, savingAddons, addonOptions, addonSaveMsg, showAddAddonForm,
    deletingAddon, newAddon, openAddonEditor, createAddon, deleteAddon, saveAddons,
    openPlanEditor, syncPlanPrice, savePlanPrices,
    openFeatureMatrix, loadFeatureMatrix, getFeatureEnabled, isAddonFeature, toggleFeature,
    saveAddonMinTier, saveNewTenant,
    API_BASE,
  }
}
