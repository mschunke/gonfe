// Package certificado carrega e inspeciona certificados digitais ICP-Brasil do
// tipo A1, distribuídos em arquivos PKCS#12 (.pfx ou .p12).
//
// O pacote é escrito inteiramente em Go, sem CGO: o mesmo binário funciona em
// Windows, Linux e macOS, e o carregamento aceita tanto os algoritmos antigos
// (3DES/SHA-1) quanto os modernos (AES-256-CBC com PBES2), presentes nos
// certificados emitidos hoje.
//
// Certificados A3 (token ou cartão inteligente) e HSMs não são carregados aqui,
// mas podem ser usados na assinatura através da interface
// [github.com/mschunke/gonfe/xmldsig.Assinante].
package certificado

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

// Erros devolvidos pelo pacote.
var (
	// ErrSenhaIncorreta indica que o arquivo PKCS#12 não pôde ser decifrado.
	ErrSenhaIncorreta = errors.New("certificado: senha incorreta ou arquivo PKCS#12 corrompido")
	// ErrChaveNaoRSA indica uma chave privada de tipo não suportado pela
	// assinatura da NF-e, que exige RSA.
	ErrChaveNaoRSA = errors.New("certificado: a chave privada não é RSA")
	// ErrExpirado indica certificado fora do período de validade.
	ErrExpirado = errors.New("certificado: fora do período de validade")
)

// Identificadores de objeto da ICP-Brasil presentes no subjectAltName.
var (
	oidSubjectAltName = asn1.ObjectIdentifier{2, 5, 29, 17}
	// oidPessoaFisica traz, concatenados, data de nascimento, CPF, NIS e RG do
	// titular pessoa física.
	oidPessoaFisica = asn1.ObjectIdentifier{2, 16, 76, 1, 3, 1}
	// oidCNPJ traz o CNPJ da pessoa jurídica titular.
	oidCNPJ = asn1.ObjectIdentifier{2, 16, 76, 1, 3, 3}
	// oidResponsavel traz os dados da pessoa física responsável pela pessoa
	// jurídica titular.
	oidResponsavel = asn1.ObjectIdentifier{2, 16, 76, 1, 3, 4}
)

// Certificado é um certificado digital com a respectiva chave privada.
type Certificado struct {
	// Folha é o certificado do titular.
	Folha *x509.Certificate
	// Cadeia traz os certificados intermediários e a raiz, quando presentes no
	// arquivo. Alguns serviços da SEFAZ exigem a cadeia completa no handshake
	// TLS.
	Cadeia []*x509.Certificate

	chave crypto.PrivateKey
}

// Carregar decifra um arquivo PKCS#12 já lido em memória.
func Carregar(dados []byte, senha string) (*Certificado, error) {
	chave, folha, cadeia, err := pkcs12.DecodeChain(dados, senha)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSenhaIncorreta, err)
	}
	return De(chave, folha, cadeia...)
}

// CarregarArquivo lê e decifra um arquivo PKCS#12 do disco.
func CarregarArquivo(caminho, senha string) (*Certificado, error) {
	dados, err := os.ReadFile(caminho)
	if err != nil {
		return nil, fmt.Errorf("certificado: não foi possível ler %s: %w", caminho, err)
	}
	c, err := Carregar(dados, senha)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", caminho, err)
	}
	return c, nil
}

// CarregarDe decifra um arquivo PKCS#12 lido de um [io.Reader].
func CarregarDe(r io.Reader, senha string) (*Certificado, error) {
	dados, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("certificado: falha ao ler o PKCS#12: %w", err)
	}
	return Carregar(dados, senha)
}

// De monta um Certificado a partir de uma chave e de um certificado já
// carregados, útil para integrar chaves guardadas em outro lugar.
func De(chave crypto.PrivateKey, folha *x509.Certificate, cadeia ...*x509.Certificate) (*Certificado, error) {
	if folha == nil {
		return nil, errors.New("certificado: certificado do titular ausente")
	}
	if _, ok := chave.(*rsa.PrivateKey); !ok {
		return nil, fmt.Errorf("%w: %T", ErrChaveNaoRSA, chave)
	}
	return &Certificado{Folha: folha, Cadeia: cadeia, chave: chave}, nil
}

// ChavePrivada devolve a chave privada associada ao certificado.
func (c *Certificado) ChavePrivada() crypto.PrivateKey { return c.chave }

// ChaveRSA devolve a chave privada como chave RSA.
func (c *Certificado) ChaveRSA() *rsa.PrivateKey { return c.chave.(*rsa.PrivateKey) }

// Titular devolve o nome comum (CN) do certificado, que na ICP-Brasil traz a
// razão social ou o nome do titular seguido de ":" e do CNPJ ou CPF.
func (c *Certificado) Titular() string { return c.Folha.Subject.CommonName }

// RazaoSocial devolve o nome comum sem o sufixo ":CNPJ" da ICP-Brasil.
func (c *Certificado) RazaoSocial() string {
	nome := c.Titular()
	if i := strings.LastIndex(nome, ":"); i >= 0 {
		if sufixo := nome[i+1:]; ehNumerico(sufixo) {
			return strings.TrimSpace(nome[:i])
		}
	}
	return nome
}

// CNPJ devolve o CNPJ do titular, com 14 dígitos e sem pontuação, ou string
// vazia se o certificado for de pessoa física.
//
// A busca começa pela extensão subjectAltName, onde a ICP-Brasil registra o
// CNPJ sob o OID 2.16.76.1.3.3, e recorre ao sufixo do nome comum quando a
// extensão não está presente.
func (c *Certificado) CNPJ() string {
	if v := c.otherName(oidCNPJ); len(v) >= 14 {
		return v[:14]
	}
	if s := c.sufixoDoCN(); len(s) == 14 {
		return s
	}
	return ""
}

// CPF devolve o CPF do titular pessoa física, com 11 dígitos e sem pontuação,
// ou string vazia se o certificado for de pessoa jurídica.
func (c *Certificado) CPF() string {
	// O campo do OID 2.16.76.1.3.1 concatena data de nascimento (8 dígitos) e
	// CPF (11 dígitos), nessa ordem.
	if v := c.otherName(oidPessoaFisica); len(v) >= 19 {
		return v[8:19]
	}
	if s := c.sufixoDoCN(); len(s) == 11 {
		return s
	}
	return ""
}

// CPFResponsavel devolve o CPF da pessoa física responsável pela pessoa
// jurídica titular, registrado sob o OID 2.16.76.1.3.4.
func (c *Certificado) CPFResponsavel() string {
	if v := c.otherName(oidResponsavel); len(v) >= 19 {
		return v[8:19]
	}
	return ""
}

func (c *Certificado) sufixoDoCN() string {
	nome := c.Titular()
	i := strings.LastIndex(nome, ":")
	if i < 0 {
		return ""
	}
	sufixo := strings.TrimSpace(nome[i+1:])
	if !ehNumerico(sufixo) {
		return ""
	}
	return sufixo
}

// otherName devolve apenas os dígitos do valor registrado no subjectAltName sob
// o OID informado.
func (c *Certificado) otherName(oid asn1.ObjectIdentifier) string {
	for _, ext := range c.Folha.Extensions {
		if !ext.Id.Equal(oidSubjectAltName) {
			continue
		}
		var nomes []asn1.RawValue
		if _, err := asn1.Unmarshal(ext.Value, &nomes); err != nil {
			return ""
		}
		for _, rv := range nomes {
			if rv.Class != asn1.ClassContextSpecific || rv.Tag != 0 {
				continue
			}
			var outro struct {
				OID   asn1.ObjectIdentifier
				Valor asn1.RawValue `asn1:"tag:0,explicit"`
			}
			if _, err := asn1.UnmarshalWithParams(rv.FullBytes, &outro, "tag:0"); err != nil {
				continue
			}
			if !outro.OID.Equal(oid) {
				continue
			}
			return apenasDigitos(string(conteudoPrimitivo(outro.Valor.Bytes)))
		}
	}
	return ""
}

// conteudoPrimitivo descasca camadas de codificação DER até chegar ao conteúdo
// bruto. O valor de um otherName pode vir embrulhado em [0] EXPLICIT e ainda
// codificado como PrintableString, UTF8String ou OCTET STRING, dependendo da
// autoridade certificadora; sem descascar, os bytes de tag e comprimento podem
// se confundir com dígitos do valor.
func conteudoPrimitivo(dados []byte) []byte {
	for range 4 {
		var rv asn1.RawValue
		resto, err := asn1.Unmarshal(dados, &rv)
		if err != nil || len(resto) != 0 || len(rv.Bytes) == 0 {
			return dados
		}
		dados = rv.Bytes
		if !rv.IsCompound {
			return dados
		}
	}
	return dados
}

// Emissor devolve o nome comum da autoridade certificadora emissora.
func (c *Certificado) Emissor() string { return c.Folha.Issuer.CommonName }

// NumeroSerie devolve o número de série do certificado em hexadecimal.
func (c *Certificado) NumeroSerie() string { return fmt.Sprintf("%X", c.Folha.SerialNumber) }

// ValidoDe devolve o início do período de validade.
func (c *Certificado) ValidoDe() time.Time { return c.Folha.NotBefore }

// ValidoAte devolve o fim do período de validade.
func (c *Certificado) ValidoAte() time.Time { return c.Folha.NotAfter }

// ValidoEm confere se o certificado está dentro do período de validade no
// instante informado.
func (c *Certificado) ValidoEm(t time.Time) error {
	if t.Before(c.Folha.NotBefore) {
		return fmt.Errorf("%w: só passa a valer em %s", ErrExpirado, c.Folha.NotBefore.Format(time.RFC3339))
	}
	if t.After(c.Folha.NotAfter) {
		return fmt.Errorf("%w: venceu em %s", ErrExpirado, c.Folha.NotAfter.Format(time.RFC3339))
	}
	return nil
}

// Expirado informa se o certificado está fora da validade agora.
func (c *Certificado) Expirado() bool { return c.ValidoEm(time.Now()) != nil }

// DiasParaVencer devolve quantos dias faltam para o fim da validade,
// arredondado para baixo. O valor é negativo se o certificado já venceu.
func (c *Certificado) DiasParaVencer() int {
	return int(time.Until(c.Folha.NotAfter).Hours() / 24)
}

// DER devolve o certificado do titular codificado em DER.
func (c *Certificado) DER() []byte { return c.Folha.Raw }

// DERBase64 devolve o certificado do titular em base64, no formato exigido pelo
// elemento X509Certificate da assinatura.
func (c *Certificado) DERBase64() string {
	return base64.StdEncoding.EncodeToString(c.Folha.Raw)
}

// TLS monta o par certificado/chave para autenticação mútua nos serviços web da
// SEFAZ, incluindo a cadeia quando disponível.
func (c *Certificado) TLS() tls.Certificate {
	cadeia := make([][]byte, 0, 1+len(c.Cadeia))
	cadeia = append(cadeia, c.Folha.Raw)
	for _, ca := range c.Cadeia {
		cadeia = append(cadeia, ca.Raw)
	}
	return tls.Certificate{
		Certificate: cadeia,
		PrivateKey:  c.chave,
		Leaf:        c.Folha,
	}
}

// Assinar assina o resumo criptográfico informado com a chave privada, usando
// PKCS#1 v1.5 — o esquema exigido pelo padrão de assinatura da NF-e.
func (c *Certificado) Assinar(resumo []byte, hash crypto.Hash) ([]byte, error) {
	assinatura, err := rsa.SignPKCS1v15(rand.Reader, c.ChaveRSA(), hash, resumo)
	if err != nil {
		return nil, fmt.Errorf("certificado: falha ao assinar: %w", err)
	}
	return assinatura, nil
}

// Public implementa parte de [crypto.Signer], devolvendo a chave pública do
// certificado.
func (c *Certificado) Public() crypto.PublicKey { return c.Folha.PublicKey }

// Sign implementa [crypto.Signer], permitindo usar o certificado onde a
// biblioteca padrão espera um assinante genérico.
func (c *Certificado) Sign(rand io.Reader, resumo []byte, opts crypto.SignerOpts) ([]byte, error) {
	return rsa.SignPKCS1v15(rand, c.ChaveRSA(), opts.HashFunc(), resumo)
}

// Descrever devolve um resumo legível do certificado, útil em logs e em telas
// de configuração. Nenhum dado sigiloso é incluído.
func (c *Certificado) Descrever() string {
	doc := c.CNPJ()
	rotulo := "CNPJ"
	if doc == "" {
		doc, rotulo = c.CPF(), "CPF"
	}
	if doc == "" {
		doc, rotulo = "não identificado", "documento"
	}
	return fmt.Sprintf("%s (%s %s), emitido por %s, válido de %s até %s",
		c.RazaoSocial(), rotulo, doc, c.Emissor(),
		c.ValidoDe().Format("02/01/2006"), c.ValidoAte().Format("02/01/2006"))
}

// NomeDistinto devolve o subject completo do certificado.
func (c *Certificado) NomeDistinto() pkix.Name { return c.Folha.Subject }

func ehNumerico(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func apenasDigitos(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
