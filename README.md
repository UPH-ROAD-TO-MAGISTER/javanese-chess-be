# Permainan Catur Jawa

## 📋 Aturan Permainan

### 🎲 Komponen
- **Papan:** 9×9 (indeks 1..9, internal 0..8)
- **Pemain:** 2–4 orang (minimal 1 Bot)
- **Kartu:** Setiap pemain mendapat 18 kartu `{1..9} × 2`, diacak menjadi deck
- **Hand:** Maksimal 3 kartu di tangan

### 🚀 Langkah Awal
1. **Urutan main diacak**
2. **Pemain pertama wajib menaruh kartu di tengah** `(5,5)`

### 🕹️ Cara Bermain
- Setelah menaruh kartu, ambil lagi dari deck hingga hand berisi 3 kartu (jika masih ada).
- Penempatan kartu (selain langkah pertama) hanya boleh di salah satu dari **8 sel tetangga** (ortogonal/diagonal) dari langkah terakhir yang dimainkan.
- **Menimpa:** Kartu boleh menimpa milik siapa pun (termasuk sendiri) **hanya jika nilai kartu yang diletakkan lebih besar** dari kartu di papan.

### 🏆 Menang Instan
- Jika setelah penempatan, ada **4 kartu berurutan** (horizontal/vertikal/diagonal) milik pemain yang sama, pemain tersebut menang langsung.

### ⏳ Akhir Permainan
- Game berakhir jika:
  - Ada pemenang 4-in-a-row
  - Semua pemain kehabisan kartu di hand dan deck, atau tidak ada langkah legal lagi (stuck total)

### 🥇 Penentuan Pemenang (Non-Instan)
- Hitung untuk tiap pemain **nilai penjumlahan terbesar** atas segmen berurutan (H/V/D) miliknya (panjang ≥ 2).
- Jika semua sendirian, ambil angka terbesar tunggal.
- **Tertinggi menang.** Jika seri:
  - Tiebreak: total seluruh nilai di papan untuk pemain itu
  - Jika masih seri: pemain yang giliran lebih awal kalah (bisa diubah sesuai kebutuhan)

### ⏭️ Skip Turn
- Jika tidak punya langkah legal dengan 3 kartu di tangan (misal semua target lebih besar dari kartu yang Anda punya), **lewati giliran** (tidak buang kartu).

---

## 💡 Contoh Interaksi

1. **Mulai Game:** Pilih jumlah pemain dan mulai permainan.
2. **Giliran Pemain:** Lihat kartu di tangan, pilih posisi sesuai aturan, dan letakkan kartu.
3. **Ambil Kartu:** Setelah menaruh, otomatis ambil kartu dari deck jika tersedia.
4. **Cek Menang:** Sistem akan otomatis cek apakah ada 4-in-a-row.
5. **Akhir Game:** Jika tidak ada langkah legal atau kartu habis, sistem akan menghitung skor dan menentukan pemenang.

---

## 📚 Referensi & Lisensi

- [MIT License](LICENSE)