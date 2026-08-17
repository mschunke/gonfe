package nfce_test

import (
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"strings"
	"testing"

	"github.com/mschunke/gonfe/nfce"
	"github.com/mschunke/gonfe/nfe"
	"github.com/mschunke/gonfe/uf"
)

const chaveExemplo = "43260312345678000195650010000012341876543214"

func cscExemplo() nfce.CSC {
	return nfce.CSC{Id: "1", Codigo: "ABCDEF12-3456-7890-ABCD-EF1234567890"}
}

func TestQRCodeOnlineTemOsQuatroParametros(t *testing.T) {
	qr, err := nfce.QRCodeOnline(chaveExemplo, nfe.Homologacao, cscExemplo(),
		"https://www.sefaz.rs.gov.br/NFCE/NFCE-COM.aspx")
	if err != nil {
		t.Fatalf("QRCodeOnline: %v", err)
	}

	_, p, achou := strings.Cut(qr, "?p=")
	if !achou {
		t.Fatalf("o QR Code não tem o parâmetro p: %s", qr)
	}
	partes := strings.Split(p, "|")
	if len(partes) != 5 {
		t.Fatalf("%d campos no parâmetro p, queria 5: %v", len(partes), partes)
	}
	if partes[0] != chaveExemplo {
		t.Errorf("chave = %q", partes[0])
	}
	if partes[1] != nfce.VersaoQRCode {
		t.Errorf("versão = %q, queria %q", partes[1], nfce.VersaoQRCode)
	}
	if partes[2] != string(nfe.Homologacao) {
		t.Errorf("ambiente = %q", partes[2])
	}
	if partes[3] != "000001" {
		t.Errorf("identificador do CSC = %q, queria 000001 com zeros à esquerda", partes[3])
	}
	if len(partes[4]) != 40 {
		t.Errorf("hash com %d caracteres, queria 40", len(partes[4]))
	}
	if _, err := hex.DecodeString(partes[4]); err != nil {
		t.Errorf("hash não é hexadecimal: %v", err)
	}
}

func TestHashEhSHA1DosParametrosMaisOCSC(t *testing.T) {
	csc := cscExemplo()
	qr, err := nfce.QRCodeOnline(chaveExemplo, nfe.Producao, csc, "https://exemplo.gov.br/qr")
	if err != nil {
		t.Fatalf("QRCodeOnline: %v", err)
	}

	dados := chaveExemplo + "|2|1|000001"
	esperado := sha1.Sum([]byte(dados + csc.Codigo))
	if !strings.HasSuffix(qr, "|"+hex.EncodeToString(esperado[:])) {
		t.Errorf("o hash não é o SHA-1 de %q concatenado ao CSC:\n%s", dados, qr)
	}
	if strings.Contains(qr, csc.Codigo) {
		t.Error("o código do CSC nunca pode aparecer na URL do QR Code")
	}
}

func TestConferirQRCode(t *testing.T) {
	csc := cscExemplo()
	qr, err := nfce.QRCodeOnline(chaveExemplo, nfe.Homologacao, csc, "https://exemplo.gov.br/qr")
	if err != nil {
		t.Fatalf("QRCodeOnline: %v", err)
	}

	if err := nfce.ConferirQRCode(qr, csc.Codigo); err != nil {
		t.Errorf("o QR Code recém-gerado deveria conferir: %v", err)
	}
	if err := nfce.ConferirQRCode(qr, "outro-csc"); err == nil {
		t.Error("com outro CSC o hash não deveria conferir")
	}

	adulterado := strings.Replace(qr, "|2|", "|1|", 1)
	if err := nfce.ConferirQRCode(adulterado, csc.Codigo); err == nil {
		t.Error("alterar um parâmetro deveria invalidar o hash")
	}
	if err := nfce.ConferirQRCode("https://exemplo.gov.br/qr", csc.Codigo); err == nil {
		t.Error("URL sem o parâmetro p deveria falhar")
	}
}

func TestQRCodeRejeitaEntradasInvalidas(t *testing.T) {
	csc := cscExemplo()
	casos := []struct {
		nome  string
		chave string
		csc   nfce.CSC
		url   string
	}{
		{"chave curta", "123", csc, "https://x"},
		{"chave com DV errado", chaveExemplo[:43] + "9", csc, "https://x"},
		{"CSC sem identificador", chaveExemplo, nfce.CSC{Codigo: "x"}, "https://x"},
		{"CSC sem código", chaveExemplo, nfce.CSC{Id: "1"}, "https://x"},
		{"identificador não numérico", chaveExemplo, nfce.CSC{Id: "A1", Codigo: "x"}, "https://x"},
		{"URL vazia", chaveExemplo, csc, ""},
	}
	for _, c := range casos {
		if qr, err := nfce.QRCodeOnline(c.chave, nfe.Producao, c.csc, c.url); err == nil {
			t.Errorf("%s: devolveu %q, queria erro", c.nome, qr)
		}
	}
}

func TestMontarQRCodeRespeitaURLComQuery(t *testing.T) {
	comQuery := nfce.MontarQRCode("https://exemplo.gov.br/qr?v=1", []string{"a", "b"}, "csc")
	if !strings.Contains(comQuery, "?v=1&p=a|b|") {
		t.Errorf("URL com query deveria usar &: %s", comQuery)
	}
	semQuery := nfce.MontarQRCode("https://exemplo.gov.br/qr", []string{"a", "b"}, "csc")
	if !strings.Contains(semQuery, "/qr?p=a|b|") {
		t.Errorf("URL sem query deveria usar ?: %s", semQuery)
	}
}

func TestURLsPorUF(t *testing.T) {
	unidades := nfce.UFsComEndereco()
	if len(unidades) != 27 {
		t.Errorf("%d UFs com endereço cadastrado, queria 27", len(unidades))
	}
	for _, u := range unidades {
		for _, amb := range []nfe.Ambiente{nfe.Producao, nfe.Homologacao} {
			qr, err := nfce.URLQRCode(u, amb)
			if err != nil {
				t.Errorf("URLQRCode(%s, %s): %v", u, amb, err)
				continue
			}
			if _, err := url.ParseRequestURI(qr); err != nil {
				t.Errorf("URLQRCode(%s, %s) = %q não é uma URL válida", u, amb, qr)
			}
			consulta, err := nfce.URLConsulta(u, amb)
			if err != nil {
				t.Errorf("URLConsulta(%s, %s): %v", u, amb, err)
				continue
			}
			if _, err := url.ParseRequestURI(consulta); err != nil {
				t.Errorf("URLConsulta(%s, %s) = %q não é uma URL válida", u, amb, consulta)
			}
		}
	}
	if _, err := nfce.URLQRCode(uf.UF("XX"), nfe.Producao); err == nil {
		t.Error("UF desconhecida deveria falhar")
	}
	if _, err := nfce.URLQRCode(uf.RS, nfe.Ambiente("9")); err == nil {
		t.Error("ambiente desconhecido deveria falhar")
	}
}
