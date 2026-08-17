package certificado_test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mschunke/gonfe/certificado"
	"github.com/mschunke/gonfe/internal/certtest"
)

func TestCarregarPKCS12(t *testing.T) {
	const senha = "senha-do-teste"
	pfx, original, err := certtest.GerarPKCS12(certtest.Opcoes{
		RazaoSocial: "COMERCIO EXEMPLO LTDA",
		CNPJ:        "12345678000195",
	}, senha)
	if err != nil {
		t.Fatalf("GerarPKCS12: %v", err)
	}

	c, err := certificado.Carregar(pfx, senha)
	if err != nil {
		t.Fatalf("Carregar: %v", err)
	}
	if c.Titular() != original.Titular() {
		t.Errorf("titular = %q, queria %q", c.Titular(), original.Titular())
	}
	if c.CNPJ() != "12345678000195" {
		t.Errorf("CNPJ = %q", c.CNPJ())
	}
	if c.RazaoSocial() != "COMERCIO EXEMPLO LTDA" {
		t.Errorf("RazaoSocial = %q", c.RazaoSocial())
	}
}

func TestCarregarSenhaErrada(t *testing.T) {
	pfx, _, err := certtest.GerarPKCS12(certtest.Opcoes{}, "certa")
	if err != nil {
		t.Fatalf("GerarPKCS12: %v", err)
	}
	_, err = certificado.Carregar(pfx, "errada")
	if err == nil {
		t.Fatal("Carregar com senha errada deveria falhar")
	}
	if !errors.Is(err, certificado.ErrSenhaIncorreta) {
		t.Errorf("erro = %v, queria ErrSenhaIncorreta", err)
	}
}

func TestCarregarArquivo(t *testing.T) {
	const senha = "1234"
	pfx, _, err := certtest.GerarPKCS12(certtest.Opcoes{CNPJ: "99999999000191"}, senha)
	if err != nil {
		t.Fatalf("GerarPKCS12: %v", err)
	}
	caminho := filepath.Join(t.TempDir(), "certificado.pfx")
	if err := os.WriteFile(caminho, pfx, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	c, err := certificado.CarregarArquivo(caminho, senha)
	if err != nil {
		t.Fatalf("CarregarArquivo: %v", err)
	}
	if c.CNPJ() != "99999999000191" {
		t.Errorf("CNPJ = %q", c.CNPJ())
	}

	if _, err := certificado.CarregarArquivo(filepath.Join(t.TempDir(), "ausente.pfx"), senha); err == nil {
		t.Error("arquivo inexistente deveria falhar")
	}
}

func TestCNPJDoSubjectAltName(t *testing.T) {
	c := certtest.MustGerar(certtest.Opcoes{
		RazaoSocial: "INDUSTRIA XPTO SA",
		CNPJ:        "11222333000181",
	})
	if got := c.CNPJ(); got != "11222333000181" {
		t.Errorf("CNPJ = %q, queria 11222333000181", got)
	}
	if got := c.CPF(); got != "" {
		t.Errorf("CPF = %q, queria vazio em certificado de pessoa jurídica", got)
	}
}

func TestCPFDoSubjectAltName(t *testing.T) {
	// O campo do OID 2.16.76.1.3.1 tem 45 caracteres; o comprimento DER cai em
	// 0x2D, mas mesmo comprimentos que coincidem com dígitos ASCII precisam ser
	// descascados corretamente para não contaminar o CPF extraído.
	c := certtest.MustGerar(certtest.Opcoes{
		RazaoSocial: "JOSE DA SILVA",
		CPF:         "52998224725",
	})
	if got := c.CPF(); got != "52998224725" {
		t.Errorf("CPF = %q, queria 52998224725", got)
	}
	if got := c.CNPJ(); got != "" {
		t.Errorf("CNPJ = %q, queria vazio em certificado de pessoa física", got)
	}
}

func TestDocumentoPeloNomeComumQuandoNaoHaSAN(t *testing.T) {
	pj := certtest.MustGerar(certtest.Opcoes{
		RazaoSocial:       "SEM SAN LTDA",
		CNPJ:              "12345678000195",
		SemSubjectAltName: true,
	})
	if got := pj.CNPJ(); got != "12345678000195" {
		t.Errorf("CNPJ pelo CN = %q", got)
	}
	if got := pj.RazaoSocial(); got != "SEM SAN LTDA" {
		t.Errorf("RazaoSocial = %q", got)
	}

	pf := certtest.MustGerar(certtest.Opcoes{
		RazaoSocial:       "MARIA SOUZA",
		CPF:               "52998224725",
		SemSubjectAltName: true,
	})
	if got := pf.CPF(); got != "52998224725" {
		t.Errorf("CPF pelo CN = %q", got)
	}
}

func TestValidade(t *testing.T) {
	agora := time.Now()
	c := certtest.MustGerar(certtest.Opcoes{
		ValidoDe:  agora.Add(-48 * time.Hour),
		ValidoAte: agora.Add(48 * time.Hour),
	})

	if err := c.ValidoEm(agora); err != nil {
		t.Errorf("deveria estar válido agora: %v", err)
	}
	if err := c.ValidoEm(agora.Add(-72 * time.Hour)); err == nil {
		t.Error("antes do início da validade deveria falhar")
	} else if !errors.Is(err, certificado.ErrExpirado) {
		t.Errorf("erro = %v, queria ErrExpirado", err)
	}
	if err := c.ValidoEm(agora.Add(72 * time.Hour)); err == nil {
		t.Error("depois do fim da validade deveria falhar")
	}
	if c.Expirado() {
		t.Error("Expirado() deveria ser falso")
	}
	if d := c.DiasParaVencer(); d != 1 && d != 2 {
		t.Errorf("DiasParaVencer = %d, queria 1 ou 2", d)
	}

	vencido := certtest.MustGerar(certtest.Opcoes{
		ValidoDe:  agora.Add(-72 * time.Hour),
		ValidoAte: agora.Add(-24 * time.Hour),
	})
	if !vencido.Expirado() {
		t.Error("certificado vencido deveria ser reportado como expirado")
	}
	if vencido.DiasParaVencer() >= 0 {
		t.Errorf("DiasParaVencer = %d, queria negativo", vencido.DiasParaVencer())
	}
}

func TestAssinarEVerificar(t *testing.T) {
	c := certtest.MustGerar(certtest.Opcoes{})

	mensagem := []byte("<infNFe>conteúdo canônico</infNFe>")
	resumo := sha1.Sum(mensagem)

	assinatura, err := c.Assinar(resumo[:], crypto.SHA1)
	if err != nil {
		t.Fatalf("Assinar: %v", err)
	}
	pub := c.Folha.PublicKey.(*rsa.PublicKey)
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA1, resumo[:], assinatura); err != nil {
		t.Errorf("a assinatura não confere: %v", err)
	}

	// Uma alteração de um byte na mensagem invalida a assinatura.
	outro := sha1.Sum([]byte("<infNFe>conteudo canônico</infNFe>"))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA1, outro[:], assinatura); err == nil {
		t.Error("a assinatura deveria falhar para outra mensagem")
	}
}

func TestSignerInterface(t *testing.T) {
	c := certtest.MustGerar(certtest.Opcoes{})
	var s crypto.Signer = c

	resumo := sha1.Sum([]byte("x"))
	assinatura, err := s.Sign(rand.Reader, resumo[:], crypto.SHA1)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if err := rsa.VerifyPKCS1v15(s.Public().(*rsa.PublicKey), crypto.SHA1, resumo[:], assinatura); err != nil {
		t.Errorf("a assinatura não confere: %v", err)
	}
}

func TestRejeitaChaveNaoRSA(t *testing.T) {
	chave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	modelo := certtest.MustGerar(certtest.Opcoes{})
	_, err = certificado.De(chave, modelo.Folha)
	if !errors.Is(err, certificado.ErrChaveNaoRSA) {
		t.Errorf("erro = %v, queria ErrChaveNaoRSA", err)
	}

	if _, err := certificado.De(nil, nil); err == nil {
		t.Error("certificado nulo deveria falhar")
	}
}

func TestTLSIncluiFolha(t *testing.T) {
	c := certtest.MustGerar(certtest.Opcoes{})
	par := c.TLS()
	if len(par.Certificate) != 2 {
		t.Fatalf("cadeia com %d certificados, queria 2 (titular + AC)", len(par.Certificate))
	}
	if string(par.Certificate[0]) != string(c.Folha.Raw) {
		t.Error("o primeiro certificado da cadeia deveria ser o do titular")
	}
	if par.PrivateKey == nil {
		t.Error("a chave privada deveria acompanhar o par TLS")
	}
}

func TestDERBase64(t *testing.T) {
	c := certtest.MustGerar(certtest.Opcoes{})
	b64 := c.DERBase64()
	if b64 == "" {
		t.Fatal("DERBase64 vazio")
	}
	for _, r := range b64 {
		if r == '\n' || r == '\r' || r == ' ' {
			t.Fatal("o base64 do X509Certificate não pode conter quebras de linha")
		}
	}
}

func TestDescrever(t *testing.T) {
	c := certtest.MustGerar(certtest.Opcoes{
		RazaoSocial: "PADARIA DO ZE LTDA",
		CNPJ:        "12345678000195",
	})
	d := c.Descrever()
	for _, trecho := range []string{"PADARIA DO ZE LTDA", "CNPJ", "12345678000195", certtest.NomeAC} {
		if !strings.Contains(d, trecho) {
			t.Errorf("Descrever() = %q, faltou %q", d, trecho)
		}
	}
}
