# Frontend Briefing — Sprint 16c

## Tujuan

Acuan frontend mengenai perubahan struktur data modul Lead yang berpusat pada **Outlet** (*Outlet-Centric Architecture*).

## Konteks Perubahan

1. **Entitas Utama Lead = Outlet**:
   - Setiap baris pada tabel Lead kini merepresentasikan 1 **Outlet** spesifik.
   - Jika 1 Owner memiliki 2 atau lebih outlet, setiap outlet akan memiliki baris Lead sendiri.

2. **Dua Kolom Utama**:
   - Tabel Lead menampilkan kolom **Nama Outlet** dan **Nama Owner** secara berdampingan.

## Struktur Response Backend (`BackendLead`)

Frontend menerima objek `outlet` di dalam payload Lead:

```typescript
export interface BackendLead {
  id: number;
  code: string;
  owner: {
    id?: number;
    code: string;
    name: string;
    phone: string;
    brand_name: string;
    province: string;
    city: string;
  };
  outlet_id?: number;
  outlet?: {
    id?: number;
    code?: string;
    name?: string;
    phone?: string;
  };
  stage: string;
  status: string;
  current_score?: number;
  created_at: string;
  updated_at: string;
}
```

## Pemetaan Data di Frontend (`NasabahItem`)

- `namaOutlet`: prioritaskan `lead.outlet?.name`
- `namaOwner`: prioritaskan `lead.owner?.name`
- `kodeOutlet` / `kodeOwner`: prioritaskan `lead.outlet?.code || lead.owner?.code`
- `noHpOutlet` / `noHpOwner`: prioritaskan `lead.outlet?.phone || lead.owner?.phone`

## Rekomendasi Tampilan UI

1. **Tabel Utama Lead ([app/menu/lead/page.tsx](file:///d:/PT_piposmart/frontend/crm_piposmart/app/menu/lead/page.tsx))**:
   - Tampilkan kolom **Nama Outlet** dan **Nama Owner** secara berdampingan dalam kolom terpisah.
   - Jangan gunakan fallback `owner.name` untuk mengisi kolom `Nama Outlet`.

2. **Detail Lead ([app/menu/lead/[id]/page.tsx](file:///d:/PT_piposmart/frontend/crm_piposmart/app/menu/lead/[id]/page.tsx))**:
   - Judul detail memakai `Detail Lead: [Nama Outlet]`.

3. **Halaman Telepon/Call Lead ([app/menu/lead/call/page.tsx](file:///d:/PT_piposmart/frontend/crm_piposmart/app/menu/lead/call/page.tsx))**:
   - Profil customer utama mengedepankan **Nama Outlet** sebagai target utama follow-up.
