package digest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"sort"
	"time"

	"github.com/ProtoconNet/mitum-currency/v3/digest/config"
	dutil "github.com/ProtoconNet/mitum-currency/v3/digest/util"
	"github.com/ProtoconNet/mitum2/launch"
	"github.com/ProtoconNet/mitum2/network/quicstream"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/rs/zerolog"
)

var (
	DefaultDigestAPICache *url.URL
	DefaultDigestAPIBind  string
	//DefaultDigestAPIURL   string
)

func init() {
	DefaultDigestAPICache, _ = dutil.ParseURL("memory://", false)
	DefaultDigestAPIBind = "https://0.0.0.0:54320"
	//DefaultDigestAPIURL = "https://127.0.0.1:54320"
}

var (
	DefaultDigestURL  = "https://localhost:4430"
	DefaultDigestBind = "https://0.0.0.0:4430"
)

type YamlDigestDesign struct {
	NetworkYAML  *LocalNetwork         `yaml:"network,omitempty"`
	CacheYAML    *string               `yaml:"cache,omitempty"`
	DatabaseYAML *config.DatabaseYAML  `yaml:"database"`
	ConnInfo     []quicstream.ConnInfo `yaml:"conn_info,omitempty"`
	network      config.LocalNetwork
	database     config.BaseDatabase
	cache        *url.URL
}

func (d *YamlDigestDesign) Set(ctx context.Context) (context.Context, error) {
	e := util.StringError("set DigestDesign")

	nctx := context.WithValue(
		context.Background(),
		ContextValueLocalNetwork,
		config.EmptyBaseLocalNetwork(),
	)
	p := &LocalNetwork{}
	if *d.NetworkYAML != *p {
		var conf config.LocalNetwork
		if i, err := d.NetworkYAML.Set(nctx); err != nil {
			return ctx, e.Wrap(err)
		} else if err := util.LoadFromContext(i, ContextValueLocalNetwork, &conf); err != nil {
			return ctx, e.Wrap(err)
		} else {
			d.network = conf
		}
	}

	var ndesign launch.NodeDesign
	if err := util.LoadFromContext(ctx, launch.DesignContextKey, &ndesign); err != nil {
		return ctx, err
	}

	if d.network.Bind() == nil {
		_ = d.network.SetBind(DefaultDigestAPIBind)
	}

	if d.network.ConnInfo().URL() == nil {
		connInfo, _ := dutil.NewHTTPConnInfoFromString(DefaultDigestURL, ndesign.Network.TLSInsecure)
		_ = d.network.SetConnInfo(connInfo)
	}

	if certs := d.network.Certs(); len(certs) < 1 {
		priv, err := GenerateED25519PrivateKey()
		if err != nil {
			return ctx, e.Wrap(err)
		}

		host := "localhost"
		if d.network.ConnInfo().URL() != nil {
			host = d.network.ConnInfo().URL().Hostname()
		}

		ct, err := GenerateTLSCerts(host, priv)
		if err != nil {
			return ctx, e.Wrap(err)
		}

		if err := d.network.SetCerts(ct); err != nil {
			return ctx, e.Wrap(err)
		}
	}

	if d.CacheYAML == nil {
		d.cache = DefaultDigestAPICache
	} else {
		u, err := dutil.ParseURL(*d.CacheYAML, true)
		if err != nil {
			return ctx, e.Wrap(err)
		}
		d.cache = u
	}

	var st config.BaseDatabase
	if d.DatabaseYAML == nil {
		if err := st.SetURI(config.DefaultDatabaseURI); err != nil {
			return ctx, e.Wrap(err)
		} else if err := st.SetCache(config.DefaultDatabaseCache); err != nil {
			return ctx, e.Wrap(err)
		} else {
			d.database = st
		}
	} else {
		if err := st.SetURI(d.DatabaseYAML.URI); err != nil {
			return ctx, e.Wrap(err)
		}
		if d.DatabaseYAML.Cache != "" {
			err := st.SetCache(d.DatabaseYAML.Cache)
			if err != nil {
				return ctx, e.Wrap(err)
			}
		}
		d.database = st
	}

	return ctx, nil
}

func (d *YamlDigestDesign) Network() config.LocalNetwork {
	return d.network
}

func (d *YamlDigestDesign) Cache() *url.URL {
	return d.cache
}

func (d *YamlDigestDesign) Database() config.BaseDatabase {
	return d.database
}

func (d YamlDigestDesign) MarshalZerologObject(e *zerolog.Event) {
	e.
		Interface("network", d.network).
		Interface("database", d.database).
		Interface("cache", d.cache)
}

func (d YamlDigestDesign) Equal(b YamlDigestDesign) bool {
	if d.NetworkYAML != b.NetworkYAML {
		return false
	}

	if d.CacheYAML != b.CacheYAML {
		return false
	}

	if d.DatabaseYAML != b.DatabaseYAML {
		return false
	}

	if len(d.ConnInfo) != len(b.ConnInfo) {
		return false
	}

	sort.Slice(d.ConnInfo, func(i, j int) bool {
		return bytes.Compare([]byte(d.ConnInfo[i].String()), []byte(d.ConnInfo[j].String())) < 0
	})

	bConn := b.ConnInfo
	sort.Slice(bConn, func(i, j int) bool {
		return bytes.Compare([]byte(bConn[i].String()), []byte(bConn[j].String())) < 0
	})

	for i := range d.ConnInfo {
		if !(d.ConnInfo[i].String() == bConn[i].String()) {
			return false
		}
	}

	return true
}

func GenerateED25519PrivateKey() (ed25519.PrivateKey, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)

	return priv, err
}

func GenerateTLSCertsPair(host string, key ed25519.PrivateKey) (*pem.Block, *pem.Block, error) {
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{host},
		NotBefore:    time.Now().Add(time.Minute * -1),
		NotAfter:     time.Now().Add(time.Hour * 24 * 1825),
	}

	if i := net.ParseIP(host); i != nil {
		template.IPAddresses = []net.IP{i}
	}

	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		key.Public().(ed25519.PublicKey),
		key,
	)
	if err != nil {
		return nil, nil, err
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}

	return &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes},
		&pem.Block{Type: "CERTIFICATE", Bytes: certDER},
		nil
}

func GenerateTLSCerts(host string, key ed25519.PrivateKey) ([]tls.Certificate, error) {
	k, c, err := GenerateTLSCertsPair(host, key)
	if err != nil {
		return nil, err
	}

	certificate, err := tls.X509KeyPair(pem.EncodeToMemory(c), pem.EncodeToMemory(k))
	if err != nil {
		return nil, err
	}

	return []tls.Certificate{certificate}, nil
}
