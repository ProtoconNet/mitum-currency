package digest

import (
	"context"
	"crypto/tls"
	"strings"

	"github.com/ProtoconNet/mitum-currency/v3/digest/config"
	dutil "github.com/ProtoconNet/mitum-currency/v3/digest/util"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

type LocalNetwork struct {
	Bind        *string `yaml:"bind"`
	URL         *string `yaml:"url"`
	CertKeyFile *string `yaml:"cert-key,omitempty"`
	CertFile    *string `yaml:"cert,omitempty"`
	Cache       *string `yaml:",omitempty"`
	SealCache   *string `yaml:"seal-cache,omitempty"`
}

func (no LocalNetwork) Set(ctx context.Context) (context.Context, error) {
	var conf config.LocalNetwork
	if err := util.LoadFromContext(ctx, ContextValueLocalNetwork, &conf); err != nil {
		return ctx, err
	}

	if err := no.setConnInfo(conf); err != nil {
		return ctx, err
	}

	if no.Bind != nil {
		if err := conf.SetBind(*no.Bind); err != nil {
			return ctx, err
		}
	}

	if err := no.setCerts(conf); err != nil {
		return ctx, err
	}

	if no.Cache != nil {
		if err := conf.SetCache(*no.Cache); err != nil {
			return ctx, err
		}
	}

	if no.SealCache != nil {
		if err := conf.SetSealCache(*no.SealCache); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

func (no LocalNetwork) setConnInfo(conf config.LocalNetwork) error {
	if no.URL == nil {
		return nil
	}

	ci, err := dutil.NewHTTPConnInfoFromString(*no.URL, false)
	if err != nil {
		return err
	}

	if err := ci.IsValid(nil); err != nil {
		return err
	}

	return conf.SetConnInfo(ci)
}

func (no LocalNetwork) setCerts(conf config.LocalNetwork) error {
	switch {
	case (no.CertKeyFile != nil || no.CertFile != nil) && (no.CertKeyFile == nil || no.CertFile == nil):
		return errors.Errorf("Cert-key and cert should be given both")
	case no.CertKeyFile == nil || len(strings.TrimSpace(*no.CertKeyFile)) < 1:
		return nil
	case no.CertFile == nil || len(strings.TrimSpace(*no.CertFile)) < 1:
		return nil
	}

	c, err := tls.LoadX509KeyPair(*no.CertFile, *no.CertKeyFile)
	if err != nil {
		return err
	}

	return conf.SetCerts([]tls.Certificate{c})
}
