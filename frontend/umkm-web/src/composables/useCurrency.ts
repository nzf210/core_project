/**
 * useCurrency — Indonesian Rupiah formatting utilities
 *
 * Konvensi platform: semua harga di-DB dalam satuan SEN (1 rupiah = 100 sen).
 * Fungsi ini mengkonversi sen → rupiah dan format sesuai locale ID.
 *
 * Display format: "Rp 35.000" (Indonesian locale, ribuan separator = titik)
 */

export function formatRupiah(sen: number): string {
  if (!sen || sen <= 0) return 'Rp 0'
  const rupiah = sen / 100
  return `Rp ${rupiah.toLocaleString('id-ID')}`
}

export function formatRupiahShort(sen: number): string {
  if (!sen || sen <= 0) return 'Gratis'
  const rupiah = sen / 100
  if (rupiah >= 1_000_000) {
    return `Rp ${(rupiah / 1_000_000).toLocaleString('id-ID')}jt`
  }
  if (rupiah >= 1_000) {
    return `Rp ${(rupiah / 1_000).toLocaleString('id-ID')}rb`
  }
  return `Rp ${rupiah.toLocaleString('id-ID')}`
}
