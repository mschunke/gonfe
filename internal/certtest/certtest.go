// Package certtest gera certificados digitais sintéticos para os testes da
// biblioteca, imitando a estrutura de um certificado ICP-Brasil A1.
//
// Nada aqui deve ser usado em produção: as chaves são curtas, os certificados
// são autoassinados e nenhuma cadeia de confiança real é envolvida.
package certtest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/mschunke/gonfe/certificado"
	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

var (
	oidSubjectAltName = asn1.ObjectIdentifier{2, 5, 29, 17}
	oidPessoaFisica   = asn1.ObjectIdentifier{2, 16, 76, 1, 3, 1}
	oidCNPJ           = asn1.ObjectIdentifier{2, 16, 76, 1, 3, 3}
)

// Opcoes controla a geração do certificado.
type Opcoes struct {
	// RazaoSocial é o nome do titular; o CNPJ é anexado ao nome comum com ":".
	RazaoSocial string
	// CNPJ do titular pessoa jurídica, com 14 dígitos.
	CNPJ string
	// CPF do titular pessoa física, com 11 dígitos. Se preenchido, o
	// certificado é gerado como e-CPF e o CNPJ é ignorado.
	CPF string
	// ValidoDe e ValidoAte delimitam a validade; o padrão é de ontem até daqui
	// a um ano.
	ValidoDe, ValidoAte time.Time
	// BitsRSA é o tamanho da chave; o padrão é 2048.
	BitsRSA int
	// SemSubjectAltName omite a extensão com os OIDs da ICP-Brasil, para
	// exercitar o caminho de leitura pelo nome comum.
	SemSubjectAltName bool
}

func (o *Opcoes) aplicarPadroes() {
	if o.RazaoSocial == "" {
		o.RazaoSocial = "EMPRESA DE TESTE LTDA"
	}
	if o.CNPJ == "" && o.CPF == "" {
		o.CNPJ = "12345678000195"
	}
	if o.ValidoDe.IsZero() {
		o.ValidoDe = time.Now().Add(-24 * time.Hour)
	}
	if o.ValidoAte.IsZero() {
		o.ValidoAte = time.Now().Add(365 * 24 * time.Hour)
	}
	if o.BitsRSA == 0 {
		o.BitsRSA = 2048
	}
}

// NomeAC é o nome comum da autoridade certificadora fictícia que assina os
// certificados gerados por este pacote.
const NomeAC = "AC DE TESTE GONFE"

type autoridade struct {
	cert  *x509.Certificate
	chave *rsa.PrivateKey
}

var (
	acUmaVez sync.Once
	acRaiz   *autoridade
	acErro   error
)

// ac devolve a autoridade certificadora do pacote, criada uma única vez por
// processo para não pagar a geração de chave RSA em cada teste.
func ac() (*autoridade, error) {
	acUmaVez.Do(func() {
		chave, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			acErro = fmt.Errorf("certtest: geração da chave da AC: %w", err)
			return
		}
		modelo := x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName:   NomeAC,
				Country:      []string{"BR"},
				Organization: []string{"ICP-Brasil de Teste"},
			},
			NotBefore:             time.Now().Add(-365 * 24 * time.Hour),
			NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
			BasicConstraintsValid: true,
			IsCA:                  true,
		}
		der, err := x509.CreateCertificate(rand.Reader, &modelo, &modelo, &chave.PublicKey, chave)
		if err != nil {
			acErro = fmt.Errorf("certtest: criação do certificado da AC: %w", err)
			return
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			acErro = fmt.Errorf("certtest: releitura do certificado da AC: %w", err)
			return
		}
		acRaiz = &autoridade{cert: cert, chave: chave}
	})
	return acRaiz, acErro
}

// Gerar cria um certificado de titular assinado pela autoridade certificadora
// fictícia do pacote, com a chave privada correspondente e a cadeia anexada.
func Gerar(o Opcoes) (*certificado.Certificado, error) {
	o.aplicarPadroes()

	emissora, err := ac()
	if err != nil {
		return nil, err
	}

	chave, err := rsa.GenerateKey(rand.Reader, o.BitsRSA)
	if err != nil {
		return nil, fmt.Errorf("certtest: geração da chave: %w", err)
	}

	documento := o.CNPJ
	if o.CPF != "" {
		documento = o.CPF
	}

	modelo := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:   o.RazaoSocial + ":" + documento,
			Country:      []string{"BR"},
			Organization: []string{"ICP-Brasil"},
		},
		NotBefore:             o.ValidoDe,
		NotAfter:              o.ValidoAte,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	if !o.SemSubjectAltName {
		ext, err := extensaoICPBrasil(o.CNPJ, o.CPF)
		if err != nil {
			return nil, err
		}
		modelo.ExtraExtensions = []pkix.Extension{ext}
	}

	der, err := x509.CreateCertificate(rand.Reader, &modelo, emissora.cert, &chave.PublicKey, emissora.chave)
	if err != nil {
		return nil, fmt.Errorf("certtest: criação do certificado: %w", err)
	}
	folha, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("certtest: releitura do certificado: %w", err)
	}
	return certificado.De(chave, folha, emissora.cert)
}

// MustGerar é como [Gerar], mas entra em pânico em caso de erro.
func MustGerar(o Opcoes) *certificado.Certificado {
	c, err := Gerar(o)
	if err != nil {
		panic(err)
	}
	return c
}

// GerarPKCS12 devolve o certificado codificado como um arquivo PKCS#12
// protegido pela senha informada, junto com o certificado já carregado.
func GerarPKCS12(o Opcoes, senha string) ([]byte, *certificado.Certificado, error) {
	c, err := Gerar(o)
	if err != nil {
		return nil, nil, err
	}
	pfx, err := pkcs12.Modern.Encode(c.ChaveRSA(), c.Folha, c.Cadeia, senha)
	if err != nil {
		return nil, nil, fmt.Errorf("certtest: codificação PKCS#12: %w", err)
	}
	return pfx, c, nil
}

// extensaoICPBrasil monta o subjectAltName com o otherName do CNPJ ou do CPF,
// na mesma forma usada pelas autoridades certificadoras da ICP-Brasil.
func extensaoICPBrasil(cnpj, cpf string) (pkix.Extension, error) {
	type outroNome struct {
		OID   asn1.ObjectIdentifier
		Valor string `asn1:"tag:0,explicit,printable"`
	}

	var nomes []asn1.RawValue
	acrescentar := func(oid asn1.ObjectIdentifier, valor string) error {
		bruto, err := asn1.MarshalWithParams(outroNome{OID: oid, Valor: valor}, "tag:0")
		if err != nil {
			return fmt.Errorf("certtest: codificação do otherName %v: %w", oid, err)
		}
		nomes = append(nomes, asn1.RawValue{FullBytes: bruto})
		return nil
	}

	if cpf != "" {
		// Data de nascimento (8) + CPF (11) + NIS (11) + RG e órgão (15).
		conteudo := "01011990" + cpf + "00000000000" + "000000000000000"
		if err := acrescentar(oidPessoaFisica, conteudo); err != nil {
			return pkix.Extension{}, err
		}
	}
	if cnpj != "" {
		if err := acrescentar(oidCNPJ, cnpj); err != nil {
			return pkix.Extension{}, err
		}
	}

	valor, err := asn1.Marshal(nomes)
	if err != nil {
		return pkix.Extension{}, fmt.Errorf("certtest: codificação do subjectAltName: %w", err)
	}
	return pkix.Extension{Id: oidSubjectAltName, Value: valor}, nil
}
