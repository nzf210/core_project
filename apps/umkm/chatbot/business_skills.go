package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// enrichWithBusinessSkills dynamically injects operational knowledge and customer-specific
// live transaction context (laundry orders, service SPK, clinic appointments, restaurant info)
// into the LLM system prompt based on the tenant's business_type.
//
// F074: Dynamic Business Type Chatbot Skills & Function Calling Integration
func enrichWithBusinessSkills(ctx context.Context, tenantID, sender, prompt string) string {
	if DB == nil || tenantID == "" {
		return prompt
	}

	var businessType string
	err := DB.QueryRow(ctx, "SELECT business_type FROM tenants WHERE id = $1", tenantID).Scan(&businessType)
	if err != nil || businessType == "" {
		return prompt
	}

	cleanPhone := strings.Split(sender, "@")[0]
	cleanPhone = strings.TrimPrefix(cleanPhone, "+")
	if strings.HasPrefix(cleanPhone, "62") {
		cleanPhone = cleanPhone[2:]
	} else if strings.HasPrefix(cleanPhone, "0") {
		cleanPhone = cleanPhone[1:]
	}

	switch businessType {
	case "laundry":
		return enrichLaundrySkill(ctx, tenantID, cleanPhone, prompt)
	case "jasa":
		return enrichServiceSkill(ctx, tenantID, cleanPhone, prompt)
	case "clinic":
		return enrichClinicSkill(ctx, tenantID, cleanPhone, prompt)
	case "restoran":
		return enrichRestaurantSkill(ctx, tenantID, prompt)
	default:
		return prompt
	}
}

func enrichLaundrySkill(ctx context.Context, tenantID, cleanPhone, prompt string) string {
	skillPrompt := "\n\n[SPESIALISASI ASISTEN LAUNDRY]\n" +
		"Toko ini bergerak di bidang jasa cuci baju, dry clean, dan setrika. " +
		"Bantu pelanggan mengecek status cucian, estimasi biaya cuci kiloan/satuan, dan jam operasional.\n"

	if cleanPhone == "" {
		return prompt + skillPrompt
	}

	var orderNo, customerName, serviceType, rackLocation, status string
	var totalAmount int64
	var isPaid bool

	err := DB.QueryRow(ctx, `
		SELECT order_no, customer_name, service_type, rack_location, status, total_amount, is_paid
		FROM laundry_orders
		WHERE tenant_id = $1 AND customer_phone LIKE '%' || $2
		ORDER BY created_at DESC LIMIT 1`,
		tenantID, cleanPhone,
	).Scan(&orderNo, &customerName, &serviceType, &rackLocation, &status, &totalAmount, &isPaid)

	if err == nil && orderNo != "" {
		statusBayar := "Belum Lunas"
		if isPaid {
			statusBayar = "Lunas"
		}
		skillPrompt += fmt.Sprintf("\n[DATA CUCIAN PELANGGAN INI SAAT INI]:\n"+
			"- No. Nota: %s\n"+
			"- Nama: %s\n"+
			"- Layanan: %s\n"+
			"- Status Cucian: %s (antre/cuci/kering/setrika/siap_ambil/selesai)\n"+
			"- Lokasi Rak/Keranjang: %s\n"+
			"- Total Biaya: Rp %d (%s)\n"+
			"Instruksi: Jika pelanggan bertanya tentang cucian mereka, berikan informasi di atas secara akurat dan ramah.\n",
			orderNo, customerName, serviceType, status, rackLocation, totalAmount/100, statusBayar)
	}

	return prompt + skillPrompt
}

func enrichServiceSkill(ctx context.Context, tenantID, cleanPhone, prompt string) string {
	skillPrompt := "\n\n[SPESIALISASI ASISTEN BENGKEL & SERVIS (SPK)]\n" +
		"Toko ini bergerak di bidang jasa servis, perbaikan, dan perawatan unit/kendaraan/elektronik. " +
		"Bantu pelanggan mengetahui status perbaikan, keluhan, dan biaya pengerjaan.\n"

	if cleanPhone == "" {
		return prompt + skillPrompt
	}

	var orderNo, customerName, unitName, unitIdentifier, complaint, techName, status string
	var finalCost, estCost int64
	var isPaid bool

	err := DB.QueryRow(ctx, `
		SELECT order_no, customer_name, unit_name, unit_identifier, complaint, technician_name, status,
		       final_cost, estimated_cost, is_paid
		FROM service_orders
		WHERE tenant_id = $1 AND customer_phone LIKE '%' || $2
		ORDER BY created_at DESC LIMIT 1`,
		tenantID, cleanPhone,
	).Scan(&orderNo, &customerName, &unitName, &unitIdentifier, &complaint, &techName, &status, &finalCost, &estCost, &isPaid)

	if err == nil && orderNo != "" {
		cost := finalCost
		if cost == 0 {
			cost = estCost
		}
		statusBayar := "Belum Lunas"
		if isPaid {
			statusBayar = "Lunas"
		}
		skillPrompt += fmt.Sprintf("\n[DATA SERVIS PELANGGAN INI SAAT INI]:\n"+
			"- No. SPK: %s\n"+
			"- Nama: %s\n"+
			"- Unit: %s (%s)\n"+
			"- Keluhan: %s\n"+
			"- Teknisi: %s\n"+
			"- Status Pengerjaan: %s (received/diagnosing/working/testing/ready/completed)\n"+
			"- Biaya: Rp %d (%s)\n"+
			"Instruksi: Jika pelanggan bertanya tentang progres perbaikan unit mereka, jelaskan status dan rincian di atas.\n",
			orderNo, customerName, unitName, unitIdentifier, complaint, techName, status, cost/100, statusBayar)
	}

	return prompt + skillPrompt
}

func enrichClinicSkill(ctx context.Context, tenantID, cleanPhone, prompt string) string {
	skillPrompt := "\n\n[SPESIALISASI ASISTEN KLINIK & PRAKTEK MEDIS]\n" +
		"Klinik ini melayani pemeriksaan pasien dan antrean dokter/terapis. " +
		"Jawab pertanyaan pasien dengan empati, sopan, dan informatif.\n"

	if cleanPhone == "" {
		return prompt + skillPrompt
	}

	var apptNum, patientName, status string
	var scheduledAt *time.Time

	err := DB.QueryRow(ctx, `
		SELECT appointment_number, patient_name, status, scheduled_at
		FROM clinic_appointments
		WHERE tenant_id = $1 AND patient_phone LIKE '%' || $2
		ORDER BY created_at DESC LIMIT 1`,
		tenantID, cleanPhone,
	).Scan(&apptNum, &patientName, &status, &scheduledAt)

	if err == nil && apptNum != "" {
		schedStr := "Hari ini"
		if scheduledAt != nil {
			schedStr = scheduledAt.Format("02 Jan 2006 15:04")
		}
		skillPrompt += fmt.Sprintf("\n[DATA ANTREAN KLINIK PASIEN INI]:\n"+
			"- No. Antrean: %s\n"+
			"- Nama Pasien: %s\n"+
			"- Jadwal: %s\n"+
			"- Status: %s\n"+
			"Instruksi: Informasikan nomor antrean pasien ini jika mereka menanyakan pendaftaran atau nomor urut mereka.\n",
			apptNum, patientName, schedStr, status)
	}

	return prompt + skillPrompt
}

func enrichRestaurantSkill(_ context.Context, _, prompt string) string {
	return prompt + "\n\n[SPESIALISASI ASISTEN RESTORAN & CAFE]\n" +
		"Usaha ini bergerak di bidang kuliner/F&B. Bantu pelanggan dengan ramah mengenai rekomendasi menu favorit, " +
		"ketersediaan makanan/minuman, serta informasi reservasi meja dan jam operasional resto.\n"
}
