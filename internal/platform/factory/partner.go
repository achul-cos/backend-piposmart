package factory

import (
	"fmt"
	"time"
)

type PartnerType struct {
	Code        string
	Name        string
	Description string
}

type Partner struct {
	PartnerTypeCode string
	Code            string
	Name            string
	Phone           string
	Email           string
	Address         string
	BankAccount     string
	Status          string
}

type PartnerAssignment struct {
	PartnerCode string
	UserEmail   string
	Active      bool
}

type PartnerInteraction struct {
	PartnerCode     string
	InteractionType string
	InteractionAt   time.Time
	Note            string
}

type PartnerReferral struct {
	PartnerCode  string
	LeadCode     string
	ReferralDate time.Time
	Notes        string
}

func (f *Factory) BuildPartnerType(code, name, description string) PartnerType {
	return PartnerType{
		Code:        code,
		Name:        name,
		Description: description,
	}
}

func (f *Factory) BuildPartner(partnerTypeCode string, index int) Partner {
	codePrefix := map[string]string{
		"SUPPLIER":         "SUP",
		"DISTRIBUTOR":      "DIS",
		"AGENT":            "AGT",
		"REFERRAL_PARTNER": "REF",
	}[partnerTypeCode]
	if codePrefix == "" {
		codePrefix = "PTR"
	}
	return Partner{
		PartnerTypeCode: partnerTypeCode,
		Code:            fmt.Sprintf("%s-%03d", codePrefix, index),
		Name:            fmt.Sprintf("Mitra %s %03d", partnerTypeCode, index),
		Phone:           fmt.Sprintf("08123456%03d", index),
		Email:           fmt.Sprintf("partner.%s.%03d@demo.piposmart.id", codePrefix, index),
		Address:         fmt.Sprintf("Jl. Mitra Usaha No. %d, Jakarta", index),
		BankAccount:     fmt.Sprintf("52701234%04d", index),
		Status:          "ACTIVE",
	}
}
