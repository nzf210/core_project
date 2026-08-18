package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// KTPData represents parsed KTP information
type KTPData struct {
	NIK     string `json:"nik"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Gender  string `json:"gender"`
	Age     int    `json:"age"`
}

func callVisionOCR(imageURL string) (string, error) {
	visionPayload := map[string]string{
		"image_url": imageURL,
		"prompt":    "Extract KTP data",
	}
	body, _ := json.Marshal(visionPayload)

	resp, err := http.Post(visionGatewayURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var visionResp struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&visionResp); err != nil {
		return "", err
	}

	if !visionResp.Success {
		return "", nil
	}

	return visionResp.Data.Text, nil
}

func parseKTPData(text string) (*KTPData, error) {
	var ktpData KTPData
	if err := json.Unmarshal([]byte(text), &ktpData); err != nil {
		return nil, err
	}
	return &ktpData, nil
}

func upsertCitizenFromKTP(ctx context.Context, tx pgx.Tx, ktp *KTPData) (string, error) {
	var citizenID string
	query := `
		INSERT INTO citizens (nik, name, address, gender, age)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (nik) DO UPDATE SET
			name = EXCLUDED.name,
			address = EXCLUDED.address,
			updated_at = NOW()
		RETURNING id
	`
	err := tx.QueryRow(ctx, query, ktp.NIK, ktp.Name, ktp.Address, ktp.Gender, ktp.Age).Scan(&citizenID)
	return citizenID, err
}

func recordKTPEndorsement(ctx context.Context, tx pgx.Tx, citizenID, tenantID, campaignID, imageURL string) error {
	query := `
		INSERT INTO endorsements (citizen_id, tenant_id, campaign_id, proof_image_url, status)
		VALUES ($1, $2, $3, $4, 'valid')
	`
	_, err := tx.Exec(ctx, query, citizenID, tenantID, campaignID, imageURL)
	return err
}
