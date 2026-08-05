package services

import (
	"context"
	"errors"
	"slices"
	"strings"

	"novelhub/internal/dtos/request"
	"novelhub/internal/models"
	"novelhub/pkg/apperrors"
	"novelhub/pkg/constants"
	"novelhub/pkg/crypto"
	"novelhub/pkg/mailer"
)

var availableTLSModes = []string{mailer.TLSModeNone, mailer.TLSModeStartTLS, mailer.TLSModeImplicit}

func smtpSettingsFromRaw(raw map[string]any) models.SMTPSettings {
	port := 587
	if value, ok := strictInteger(raw["smtp.port"]); ok {
		port = int(value)
	}
	mode := rawString(raw, "smtp.tls_mode", mailer.TLSModeStartTLS)
	if !slices.Contains(availableTLSModes, mode) {
		mode = mailer.TLSModeStartTLS
	}
	return models.SMTPSettings{
		Enabled:              rawBool(raw, "smtp.enabled", false),
		Host:                 rawString(raw, "smtp.host", ""),
		Port:                 port,
		Username:             rawString(raw, "smtp.username", ""),
		FromEmail:            rawString(raw, "smtp.from_email", ""),
		TLSMode:              mode,
		AllowPrivateNetworks: rawBool(raw, "smtp.allow_private_networks", false),
		PasswordConfigured:   rawString(raw, "smtp.password", "") != "",
		AvailableTLSModes:    append([]string(nil), availableTLSModes...),
	}
}

func validateSMTPRaw(raw map[string]any) error {
	if value, ok := raw["smtp.port"]; ok {
		port, valid := strictInteger(value)
		if !valid || port < 1 || port > 65535 {
			return errors.New("Invalid SMTP port")
		}
	}
	if mode, ok := raw["smtp.tls_mode"]; ok {
		text, valid := mode.(string)
		if !valid || !slices.Contains(availableTLSModes, text) {
			return errors.New("Invalid SMTP TLS mode")
		}
	}

	settings := smtpSettingsFromRaw(raw)
	if strings.ContainsAny(settings.Host+settings.Username+settings.FromEmail, "\r\n") {
		return errors.New("SMTP settings must not contain newlines")
	}
	if strings.Contains(settings.Host, "/") {
		return errors.New("SMTP host must be a hostname, not a URL")
	}
	if settings.FromEmail != "" && !constants.EMAIL_REGEX.MatchString(settings.FromEmail) {
		return errors.New("Invalid SMTP sender address")
	}
	if !settings.Enabled {
		return nil
	}
	if settings.Host == "" || settings.FromEmail == "" {
		return errors.New("SMTP requires a host and a sender address when enabled")
	}
	if settings.Username == "" && settings.PasswordConfigured {
		return errors.New("SMTP password requires a username")
	}
	return nil
}

func (s *settingsService) smtpConfig(raw map[string]any) (mailer.SMTPConfig, error) {
	settings := smtpSettingsFromRaw(raw)
	password, err := crypto.DecryptAES(rawString(raw, "smtp.password", ""))
	if err != nil {
		return mailer.SMTPConfig{}, apperrors.New(apperrors.ErrInternalError, "Failed to read the stored SMTP password")
	}
	return mailer.SMTPConfig{
		Host:                 settings.Host,
		Port:                 settings.Port,
		Username:             settings.Username,
		Password:             password,
		FromEmail:            settings.FromEmail,
		TLSMode:              settings.TLSMode,
		AllowPrivateNetworks: settings.AllowPrivateNetworks,
	}, nil
}

func (s *settingsService) SMTP(ctx context.Context) (mailer.SMTPConfig, error) {
	s.mu.RLock()
	raw := s.raw
	s.mu.RUnlock()
	if !rawBool(raw, "smtp.enabled", false) {
		return mailer.SMTPConfig{}, apperrors.New(apperrors.ErrBadRequest, "Email delivery is not configured")
	}
	return s.smtpConfig(raw)
}

func (s *settingsService) TestSMTP(ctx context.Context, dto *request.SMTPTestDto) error {
	s.mu.RLock()
	raw := s.raw
	s.mu.RUnlock()

	config, err := s.smtpConfig(raw)
	if err != nil {
		return err
	}
	if dto != nil {
		if dto.Host != nil {
			config.Host = strings.TrimSpace(*dto.Host)
		}
		if dto.Port != nil {
			config.Port = *dto.Port
		}
		if dto.Username != nil {
			config.Username = strings.TrimSpace(*dto.Username)
		}
		if dto.Password != nil {
			config.Password = *dto.Password
		}
		if dto.FromEmail != nil {
			config.FromEmail = strings.TrimSpace(*dto.FromEmail)
		}
		if dto.TLSMode != nil {
			config.TLSMode = *dto.TLSMode
		}
		if dto.AllowPrivateNetworks != nil {
			config.AllowPrivateNetworks = *dto.AllowPrivateNetworks
		}
	}
	if config.Host == "" {
		return apperrors.New(apperrors.ErrBadRequest, "SMTP host is required")
	}
	if err := mailer.TestConnection(config); err != nil {
		return apperrors.New(apperrors.ErrBadRequest, err.Error())
	}
	return nil
}
