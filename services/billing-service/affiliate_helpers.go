package main

import (
	"context"
)

func validateReferralPercentages(discount, commission float64) bool {
	return discount >= 0 && discount <= 100 && commission >= 0 && commission <= 100
}

func saveReferralConfig(ctx context.Context, req *struct {
	DiscountPercent     float64 `json:"discount_percent"`
	CommissionPercent   float64 `json:"commission_percent"`
	MinPurchaseRupiah   int64   `json:"min_purchase_rupiah"`
	MaxCommissionRupiah int64   `json:"max_commission_rupiah"`
	IsActive            bool    `json:"is_active"`
	ReferralLinkBase    string  `json:"referral_link_base"`
}) error {
	_, err := DB.Exec(ctx, `
		INSERT INTO referral_config (id, discount_percent, commission_percent, min_purchase_rupiah, max_commission_rupiah, is_active, referral_link_base, updated_at)
		VALUES (1, $1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (id)
		DO UPDATE SET discount_percent = EXCLUDED.discount_percent,
		              commission_percent = EXCLUDED.commission_percent,
		              min_purchase_rupiah = EXCLUDED.min_purchase_rupiah,
		              max_commission_rupiah = EXCLUDED.max_commission_rupiah,
		              is_active = EXCLUDED.is_active,
		              referral_link_base = EXCLUDED.referral_link_base,
		              updated_at = NOW()
	`, req.DiscountPercent, req.CommissionPercent, req.MinPurchaseRupiah, req.MaxCommissionRupiah, req.IsActive, req.ReferralLinkBase)
	return err
}
