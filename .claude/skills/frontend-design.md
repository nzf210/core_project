# frontend-design

Guidance untuk UI design WCH Platform (Dashboard SaaS, bukan landing page).

## When to invoke

- User mengetik `/frontend-design`
- Sebelum implement UI/UX baru
- Saat redesign existing components
- Untuk konsistensi design system

## Design Principles untuk WCH Platform

WCH Platform adalah **B2B SaaS Dashboard** dengan 3 produk:
1. **UMKM Dashboard** - Accounting, POS, Chatbot management
2. **Campaign Dashboard** - Relawan, Real Count, Data Pemilih
3. **Superadmin Dashboard** - Tenant management, Billing

**Target audience:** Pemilik UMKM, Tim Kampanye, Internal Admin (bukan consumer/end-user)

## Design System

### Typography
- **Font:** System font stack atau Inter
- **Sizes:** 
  - Heading: 24px, 20px, 18px
  - Body: 16px (default), 14px (secondary)
  - Small: 12px (labels, captions)

### Colors (Tailwind)
- **Primary:** blue-600 (actions, links)
- **Success:** green-600 (completed, active)
- **Warning:** yellow-600 (pending, alerts)
- **Danger:** red-600 (errors, delete)
- **Neutral:** gray-50 to gray-900 (backgrounds, text)

### Spacing
- **Consistent:** 4px increments (4, 8, 12, 16, 24, 32, 48)
- **Component padding:** 16px (default), 24px (cards)
- **Section spacing:** 32px, 48px

## Component Patterns

### Data Table
- Zebra striping untuk readability
- Sticky header saat scroll
- Actions di kolom terakhir (icon buttons)
- Pagination di bottom-right
- Search/filter di top-left

```vue
<template>
  <div class="bg-white rounded-lg shadow">
    <div class="p-4 border-b flex justify-between">
      <input type="search" placeholder="Cari..." class="px-3 py-2 border rounded" />
      <button class="px-4 py-2 bg-blue-600 text-white rounded">+ Tambah</button>
    </div>
    <table class="w-full">
      <thead class="bg-gray-50 sticky top-0">
        <tr>
          <th class="px-4 py-3 text-left">Nama</th>
          <th class="px-4 py-3 text-left">Status</th>
          <th class="px-4 py-3 text-right">Aksi</th>
        </tr>
      </thead>
      <tbody>
        <tr class="border-b hover:bg-gray-50">
          <td class="px-4 py-3">...</td>
          <td class="px-4 py-3">...</td>
          <td class="px-4 py-3 text-right">
            <button class="text-blue-600 hover:text-blue-800">Edit</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
```

### Form
- Labels di atas input
- Required fields dengan `*`
- Inline validation dengan error message
- Actions di bottom-right (Cancel | Submit)

### Modal
- Backdrop semi-transparent (bg-black/50)
- Max-width: 600px (default), 800px (large)
- Close button di top-right
- Actions di bottom-right

### Card
- White background dengan shadow
- Rounded corners (8px)
- Padding: 16px atau 24px
- Border-bottom untuk sections

## Responsive Design

- **Desktop-first** (primary target adalah desktop users)
- **Breakpoints:** 
  - Mobile: < 640px
  - Tablet: 640px - 1024px
  - Desktop: > 1024px

## Accessibility

- **Color contrast:** WCAG AA minimum
- **Focus states:** Visible outline
- **Keyboard navigation:** Tab order logical
- **Screen reader:** Proper ARIA labels

## Anti-Patterns (Avoid)

❌ **JANGAN:**
- AI-purple gradients everywhere
- Glassmorphism tanpa alasan
- Infinite animations
- 3-column equal cards (generic landing page pattern)
- Centered hero with mesh background

✅ **LAKUKAN:**
- Clean, professional B2B aesthetic
- Consistent spacing & typography
- Fast, functional interactions
- Clear information hierarchy
- Data-dense tables yang readable

## Integration

Auto-invoke saat:
- Implement new UI components
- Redesign existing pages
- Before commit frontend changes
