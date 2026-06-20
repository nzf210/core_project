<template>
  <div class="product-catalog animate-fade-in">
    <div class="header-section flex items-center justify-between">
      <div>
        <h2>Katalog Produk</h2>
        <p class="text-muted">Kelola daftar produk, harga, dan deskripsi produk Anda.</p>
      </div>
      <div class="header-actions">
        <select v-model="selectedCategory" class="form-control"
          style="width: 200px; display: inline-block; margin-right: 1rem;">
          <option value="">Semua Kategori</option>
          <option v-for="cat in uniqueCategories" :key="cat" :value="cat">{{ cat }}</option>
        </select>

        <button class="btn btn-outline" @click="exportCSV" style="margin-right: 0.5rem;" :disabled="exporting">
          {{ exporting ? 'Mengekspor...' : 'Export CSV' }}
        </button>
        <button class="btn btn-outline" @click="exportXLSX" style="margin-right: 0.5rem;" :disabled="exporting">
          Export XLSX
        </button>

        <input type="file" ref="fileInput" accept=".csv,.xlsx" style="display: none" @change="handleFileUpload" />
        <button class="btn btn-outline" @click="($refs.fileInput as HTMLInputElement).click()"
          style="margin-right: 0.5rem;" :disabled="uploading">
          {{ uploading ? 'Mengimpor...' : 'Import (CSV/XLSX)' }}
        </button>

        <button class="btn btn-primary" @click="openAddModal">
          <span style="margin-right: 0.5rem;">+</span> Tambah Produk
        </button>
      </div>
    </div>

    <!-- Product Grid -->
    <div class="product-grid" v-if="filteredProducts.length > 0">
      <div v-for="product in filteredProducts" :key="product.id" class="product-card glass-card">
        <div class="product-image-container">
          <img v-if="product.photo_url" :src="product.photo_url" alt="Product Image" class="product-image" />
          <div v-else class="product-image-placeholder">
            <span>Tanpa Foto</span>
          </div>
        </div>
        <div class="product-details">
          <h3 class="product-name">{{ product.name }}</h3>
          <p class="product-price">{{ formatCurrency(product.price) }}</p>
          <div class="product-meta-tags flex gap-2" style="margin-bottom: 0.5rem;">
            <span class="badge">{{ product.category || 'Umum' }}</span>
            <span :class="['badge', product.stock_quantity <= 0 ? 'badge-danger' : 'badge-success']">
              Stok: {{ product.stock_quantity }}
            </span>
          </div>
          <p class="text-muted" style="margin-bottom: 1rem;">{{ product.description || 'Tidak ada deskripsi' }}</p>
          <div class="product-meta flex items-center justify-end">
            <div style="display: flex; gap: 0.5rem; width: 100%;">
              <button class="btn btn-primary btn-sm" style="flex: 1;" @click="openDetailModal(product)">Detail</button>
              <button class="btn btn-outline btn-sm" style="flex: 1;" @click="openEditModal(product)">Edit</button>
              <button class="btn btn-outline btn-sm" style="flex: 1; color: #ef4444; border-color: #ef4444;"
                @click="deleteProduct(product.id)" :disabled="deleting === product.id">
                {{ deleting === product.id ? 'Menghapus...' : 'Hapus' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else-if="loading" class="empty-state glass-card text-center">
      <h3>Memuat data produk...</h3>
    </div>

    <div v-else class="empty-state glass-card text-center">
      <div class="empty-icon">📦</div>
      <h3>Belum ada produk</h3>
      <p class="text-muted">Tambahkan produk pertama Anda untuk mulai berjualan.</p>
      <button class="btn btn-primary" style="margin-top: 1rem;" @click="openAddModal">Tambah Produk</button>
    </div>

    <!-- Detail Modal -->
    <Teleport to="body">
      <div v-if="showDetailModal" class="modal-overlay" @click="closeDetailModal">
        <div class="modal-content animate-fade-in" style="max-width: 600px; max-height: 90vh; overflow-y: auto;"
          @click.stop>
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem;">
            <h3 style="margin: 0;">Detail Produk</h3>
            <button class="btn btn-outline" style="padding: 0.2rem 0.5rem; border: none; font-size: 1.5rem;"
              @click="closeDetailModal">&times;</button>
          </div>

          <div v-if="selectedProduct" style="display: flex; flex-direction: column; gap: 1rem; max-width: 100%;">
            <div style="position: relative; max-width: 100%;">
              <button class="gallery-nav left" @click.stop="scrollGallery('left')">&lt;</button>
              <div ref="galleryRef" class="gallery-container"
                style="display: flex; gap: 1rem; overflow-x: auto; padding-bottom: 1rem; max-width: 100%; -webkit-overflow-scrolling: touch;">
                <img v-if="selectedProduct.photo_url" :src="selectedProduct.photo_url" class="gallery-img" />
                <div
                  v-if="!selectedProduct.photo_url && (!selectedProduct.additional_photos || selectedProduct.additional_photos.length === 0)"
                  class="gallery-img-placeholder">
                  Tanpa Foto
                </div>
                <img v-for="(photo, index) in selectedProduct.additional_photos" :key="index" :src="photo"
                  class="gallery-img" />
              </div>
              <button class="gallery-nav right" @click.stop="scrollGallery('right')">&gt;</button>
            </div>

            <h3>{{ selectedProduct.name }}</h3>
            <p style="color: var(--text-muted); white-space: pre-wrap;">{{ selectedProduct.description || `Tidak ada
              deskripsi.` }}</p>

            <div
              style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-top: 1rem; background: #f9fafb; padding: 1rem; border-radius: 8px;">
              <div>
                <strong style="color: var(--text-muted); font-size: 0.9em;">Harga</strong> <br /> <span
                  style="font-size: 1.1em; font-weight: bold;">{{ formatCurrency(selectedProduct.price) }}</span>
              </div>
              <div>
                <strong style="color: var(--text-muted); font-size: 0.9em;">Kategori</strong> <br /> {{
                  selectedProduct.category || 'Umum' }}
              </div>
              <div>
                <strong style="color: var(--text-muted); font-size: 0.9em;">Stok Saat Ini</strong> <br /> <span
                  :class="['badge', selectedProduct.stock_quantity <= 0 ? 'badge-danger' : 'badge-success']">{{
                    selectedProduct.stock_quantity }}</span>
              </div>
            </div>
          </div>

          <div style="margin-top: 2rem; display: flex; justify-content: flex-end;">
            <button class="btn btn-primary" @click="closeDetailModal">Tutup</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Add/Edit Modal -->
    <Teleport to="body">
      <div v-if="showModal" class="modal-overlay">
        <div class="modal-content animate-fade-in" style="max-height: 90vh; overflow-y: auto;">
          <h3 style="margin-bottom: 1.5rem;">{{ isEditing ? 'Edit Produk' : 'Tambah Produk Baru' }}</h3>

          <form @submit.prevent="saveProduct">
            <div class="form-group">
              <label>URL Foto Utama (Opsional)</label>
              <input type="url" v-model="formData.photo_url" class="form-control"
                placeholder="https://contoh.com/gambar.jpg" />
            </div>

            <div class="form-group">
              <label>Foto Tambahan (Opsional)</label>
              <div v-for="(_, index) in formData.additional_photos" :key="index"
                style="display: flex; gap: 0.5rem; margin-bottom: 0.5rem;">
                <input type="url" v-model="formData.additional_photos[index]"
                  placeholder="https://contoh.com/foto-tambahan.jpg" class="form-control" />
                <button class="btn btn-outline" style="color: #ef4444; border-color: #ef4444; padding: 0 0.75rem;"
                  @click.prevent="formData.additional_photos.splice(index, 1)">X</button>
              </div>
              <button class="btn btn-outline btn-sm" @click.prevent="formData.additional_photos.push('')"
                style="margin-top: 0.5rem; width: 100%;">+ Tambah Foto Lainnya</button>
            </div>

            <div class="form-group">
              <label>Nama Produk</label>
              <input type="text" v-model="formData.name" class="form-control" required
                placeholder="Contoh: Kopi Susu Aren" />
            </div>

            <div class="form-group">
              <label>Harga (Rp)</label>
              <input type="number" v-model="formData.price" class="form-control" required min="0" />
            </div>

            <div class="form-group">
              <label>Kategori</label>
              <input type="text" v-model="formData.category" class="form-control"
                placeholder="Contoh: Makanan, Minuman, Pakaian" />
            </div>

            <div class="form-group">
              <label>Stok Barang</label>
              <input type="number" v-model="formData.stock_quantity" class="form-control" required />
            </div>

            <div class="form-group">
              <label>Deskripsi (Opsional)</label>
              <textarea v-model="formData.description" class="form-control" placeholder="Deskripsi singkat produk"
                rows="3"></textarea>
            </div>

            <div class="modal-actions flex justify-end gap-2" style="margin-top: 2rem;">
              <button type="button" class="btn btn-outline" @click="closeModal" :disabled="saving">Batal</button>
              <button type="submit" class="btn btn-primary" :disabled="saving">
                {{ saving ? 'Menyimpan...' : 'Simpan' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { api } from '../api';
import { useModalState } from '../utils/modalState';

const { openModal } = useModalState()

interface Product {
  id: string;
  name: string;
  price: number;
  description: string;
  photo_url: string;
  category: string;
  stock_quantity: number;
  additional_photos: string[];
}

const products = ref<Product[]>([]);
const loading = ref(true);
const saving = ref(false);
const deleting = ref<string | null>(null);
const uploading = ref(false);
const exporting = ref(false);

const selectedCategory = ref('');

const uniqueCategories = computed(() => {
  const cats = new Set(products.value.map(p => p.category || 'Umum'));
  return Array.from(cats).sort();
});

const filteredProducts = computed(() => {
  if (!selectedCategory.value) return products.value;
  return products.value.filter(p => (p.category || 'Umum') === selectedCategory.value);
});

const showModal = ref(false);
const isEditing = ref(false);

const showDetailModal = ref(false);
const selectedProduct = ref<Product | null>(null);

watch(showModal, (v) => { if (v) openModal(); else closeModal(); });
watch(showDetailModal, (v) => { if (v) openModal(); else closeModal(); });

const galleryRef = ref<HTMLElement | null>(null);

const scrollGallery = (direction: 'left' | 'right') => {
  if (galleryRef.value) {
    const scrollAmount = 220; // image width + gap
    if (direction === 'left') {
      galleryRef.value.scrollBy({ left: -scrollAmount, behavior: 'smooth' });
    } else {
      galleryRef.value.scrollBy({ left: scrollAmount, behavior: 'smooth' });
    }
  }
};

const initialFormState = {
  id: '',
  name: '',
  price: 0,
  description: '',
  photo_url: '',
  category: 'Umum',
  stock_quantity: 0,
  additional_photos: [] as string[]
};

const formData = reactive({ ...initialFormState });

const formatCurrency = (amount: number) => {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0
  }).format(amount);
};

const fetchProducts = async () => {
  loading.value = true;
  try {
    const data = await api.get('/api/umkm/products');
    if (data.success) {
      products.value = data.data;
    }
  } catch (e) {
    console.error("Gagal memuat produk", e);
  } finally {
    loading.value = false;
  }
};

const openAddModal = () => {
  isEditing.value = false;
  Object.assign(formData, JSON.parse(JSON.stringify(initialFormState)));
  showModal.value = true;
};

const openEditModal = (product: Product) => {
  isEditing.value = true;
  Object.assign(formData, JSON.parse(JSON.stringify(product)));
  if (!formData.additional_photos) {
    formData.additional_photos = [];
  }
  showModal.value = true;
};

const openDetailModal = (product: Product) => {
  selectedProduct.value = product;
  showDetailModal.value = true;
};

const closeDetailModal = () => {
  showDetailModal.value = false;
  selectedProduct.value = null;
};

const closeModal = () => {
  showModal.value = false;
};

const saveProduct = async () => {
  saving.value = true;
  try {
    const body: any = {
      name: formData.name,
      price: Number(formData.price),
      description: formData.description,
      photo_url: formData.photo_url,
      category: formData.category,
      stock_quantity: Number(formData.stock_quantity),
      additional_photos: formData.additional_photos.filter((p: string) => p.trim() !== '')
    };
    if (isEditing.value) {
      body.id = (formData as any).id;
    }

    const data = isEditing.value
      ? await api.put('/api/umkm/products', body)
      : await api.post('/api/umkm/products', body);
    if (data.success) {
      await fetchProducts();
      closeModal();
    } else {
      alert(data.message || "Gagal menyimpan produk");
    }
  } catch (e) {
    alert("Terjadi kesalahan koneksi");
  } finally {
    saving.value = false;
  }
};

const deleteProduct = async (id: string) => {
  if (confirm('Apakah Anda yakin ingin menghapus produk ini?')) {
    deleting.value = id;
    try {
      const data = await api.del(`/api/umkm/products?id=${id}`);
      if (data.success) {
        products.value = products.value.filter(p => p.id !== id);
      } else {
        alert(data.message || "Gagal menghapus produk");
      }
    } catch (e) {
      alert("Terjadi kesalahan koneksi");
    } finally {
      deleting.value = null;
    }
  }
};

const handleFileUpload = async (event: any) => {
  const file = event.target.files[0];
  if (!file) return;

  uploading.value = true;
  try {
    // F022: use new /import/products endpoint (CSV + XLSX, upsert by SKU)
    const res = await api.importFile('/api/umkm/import/products', file);
    if (res.success && res.data) {
      const d = res.data;
      let msg = `Import selesai: ✅ ${d.imported} berhasil`;
      if (d.skipped > 0) msg += `, ⚠️ ${d.skipped} dilewati`;
      if (d.errors && d.errors.length > 0) {
        msg += `\n\nError (max 10):\n${d.errors.slice(0, 10).map((e: any) => `• Baris ${e.row}: ${e.error}`).join('\n')}`;
        if (d.errors.length > 10) msg += `\n... +${d.errors.length - 10} lainnya`;
      }
      alert(msg);
      await fetchProducts();
    } else {
      alert(res.message || 'Gagal import produk');
    }
  } catch (e: any) {
    console.error(e);
    alert('Terjadi kesalahan jaringan: ' + (e?.message || e));
  } finally {
    uploading.value = false;
    event.target.value = ''; // reset file input
  }
};

const exportCSV = async () => {
  exporting.value = true;
  try {
    // F022: use new /export/products endpoint (supports both formats)
    const blobUrl = await api.exportFile('/api/umkm/export/products', 'csv');
    const a = document.createElement('a');
    a.href = blobUrl;
    a.download = `products_export_${new Date().getTime()}.csv`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(blobUrl);
  } catch (e: any) {
    console.error(e);
    alert('Terjadi kesalahan jaringan saat export');
  } finally {
    exporting.value = false;
  }
};

const exportXLSX = async () => {
  exporting.value = true;
  try {
    const blobUrl = await api.exportFile('/api/umkm/export/products', 'xlsx');
    const a = document.createElement('a');
    a.href = blobUrl;
    a.download = `products_export_${new Date().getTime()}.xlsx`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(blobUrl);
  } catch (e: any) {
    console.error(e);
    alert('Terjadi kesalahan jaringan saat export');
  } finally {
    exporting.value = false;
  }
};

onMounted(() => {
  fetchProducts();
});
</script>

<style scoped>
.product-catalog {
  padding-bottom: 2rem;
}

.header-section {
  margin-bottom: 2rem;
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1.5rem;
}

.product-card {
  overflow: hidden;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  display: flex;
  flex-direction: column;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
}

.product-card:hover {
  transform: translateY(-4px);
  box-shadow: var(--shadow-lg);
}

.product-image-container {
  width: 100%;
  height: 200px;
  background-color: var(--surface-1);
  overflow: hidden;
  position: relative;
}

.product-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s ease;
}

.product-card:hover .product-image {
  transform: scale(1.05);
}

.product-image-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  background: linear-gradient(135deg, var(--surface-1) 0%, #e2e8f0 100%);
  font-weight: 500;
}

.product-details {
  padding: 1.25rem;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.product-name {
  font-size: 1.125rem;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
}

.product-price {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--accent-primary);
  margin-bottom: 1rem;
}

.product-meta {
  margin-top: auto;
}

.btn-sm {
  padding: 0.25rem 0.75rem;
  font-size: 0.875rem;
}

.empty-state {
  padding: 4rem 2rem;
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
  opacity: 0.5;
}

/* Mobile Responsiveness */
@media (max-width: 768px) {
  .product-grid {
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  .header-section.flex.items-center.justify-between {
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
  }

  .empty-state {
    padding: 2rem 1rem;
  }
}

.product-meta-tags .badge-success {
  background: rgba(16, 185, 129, 0.2);
  color: #059669;
  border-color: rgba(16, 185, 129, 0.3);
}

.gallery-img {
  width: 200px;
  height: 200px;
  object-fit: cover;
  border-radius: 8px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.gallery-img-placeholder {
  width: 200px;
  height: 200px;
  background: #f9fafb;
  border: 1px dashed #e5e7eb;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  flex-shrink: 0;
}

.gallery-container::-webkit-scrollbar {
  height: 8px;
}

.gallery-container::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.05);
  border-radius: 4px;
}

.gallery-container::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
}

.gallery-container::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.3);
}

.gallery-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid #e5e7eb;
  color: var(--text-primary);
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  font-size: 1.2rem;
  cursor: pointer;
  z-index: 10;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
}

.gallery-nav:hover {
  background: #f9fafb;
  transform: translateY(-50%) scale(1.1);
}

.gallery-nav.left {
  left: -18px;
}

.gallery-nav.right {
  right: -18px;
}

/* Sembunyikan tombol navigasi di perangkat mobile */
@media (max-width: 768px) {
  .gallery-nav {
    display: none;
  }
}
</style>
