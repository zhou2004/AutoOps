package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type CertificateImportInput struct {
	Name        string `json:"name"`
	GroupID     uint   `json:"groupId"`
	Environment string `json:"environment"`
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"privateKey"`
}

type CertificateImportResult struct {
	Certificate model.Resource `json:"certificate"`
	Version     model.Revision `json:"version"`
}

// ImportNewCertificate creates the asset and its initial immutable version in
// one transaction. Invalid input or failed encryption/auditing leaves no asset.
func (s *Service) ImportNewCertificate(ctx context.Context, u uint, input CertificateImportInput) (CertificateImportResult, error) {
	result := CertificateImportResult{}
	r := model.Resource{Name: strings.TrimSpace(input.Name), GroupID: input.GroupID,
		Environment: strings.TrimSpace(input.Environment), Engine: "nginx"}
	if err := s.Module(ctx, u, "write"); err != nil {
		return result, err
	}
	if err := s.createAllowed(u, "certificates", r); err != nil {
		return result, err
	}
	meta, err := ParseCertificate(input.Certificate, input.PrivateKey, nil)
	if err != nil {
		return result, err
	}
	r.Spec = JSON(model.CertificateSpec{Domains: meta.Domains, Owner: "external"})
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txService := New(tx)
		asset, err := txService.Save(ctx, u, "certificates", r, "")
		if err != nil {
			return err
		}
		version, err := txService.storeCertificate(u, asset, input.Certificate, input.PrivateKey)
		if err != nil {
			return err
		}
		asset, err = txService.Get("certificates", asset.ID)
		if err != nil {
			return err
		}
		if err = tx.Create(&model.Event{UserID: u, Kind: "certificate-import",
			Message: fmt.Sprintf("certificate=%d version=%d", asset.ID, version.ID)}).Error; err != nil {
			return err
		}
		txService.Redact(u, "certificates", &asset)
		result = CertificateImportResult{Certificate: asset, Version: version}
		return nil
	})
	if err != nil {
		return CertificateImportResult{}, err
	}
	return result, nil
}
